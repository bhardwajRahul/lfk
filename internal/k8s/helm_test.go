package k8s

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	k8sfake "k8s.io/client-go/kubernetes/fake"

	"github.com/janosmiko/lfk/internal/model"
)

// newFakeHelmReleaseSecret builds a kubernetes Secret object matching the
// shape helm v3 writes to the cluster. The release blob is supplied as the
// base64(gzip(json(blob))) encoding that helm uses. All fixtures in this
// file share the "default" namespace to keep call sites readable.
func newFakeHelmReleaseSecret(t *testing.T, name, release, status, version string, blob helmReleaseBlob, created time.Time) *corev1.Secret {
	t.Helper()
	data := buildHelmReleaseSecretData(t, blob)
	return &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              name,
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: created},
			Labels: map[string]string{
				"owner":   "helm",
				"name":    release,
				"status":  status,
				"version": version,
			},
		},
		Data: map[string][]byte{"release": data},
		Type: "helm.sh/release.v1",
	}
}

func makeHelmBlob(name, chart, chartVer, appVer, status, description string, revision int) helmReleaseBlob {
	b := helmReleaseBlob{Name: name, Version: revision, Namespace: "default"}
	b.Chart.Metadata.Name = chart
	b.Chart.Metadata.Version = chartVer
	b.Chart.Metadata.AppVersion = appVer
	b.Info.Status = status
	b.Info.Description = description
	b.Info.LastDeployed = time.Now().Format(time.RFC3339)
	return b
}

// makeHelmBlobWithManifest is makeHelmBlob plus an inline multi-document YAML
// manifest, which is the field getHelmManagedResources now consults to
// enumerate the resources a release owns.
func makeHelmBlobWithManifest(name, chart, chartVer, appVer, status, description, manifest string, revision int) helmReleaseBlob {
	b := makeHelmBlob(name, chart, chartVer, appVer, status, description, revision)
	b.Manifest = manifest
	return b
}

func TestGetHelmReleases_PopulatesChartColumns(t *testing.T) {
	now := time.Now()
	secret := newFakeHelmReleaseSecret(
		t, "sh.helm.release.v1.my-release.v3", "my-release", "deployed", "3",
		makeHelmBlob("my-release", "nginx", "15.4.2", "1.25.3", "deployed", "Upgrade complete", 3),
		now,
	)

	cs := k8sfake.NewClientset(secret)
	c := newFakeClient(cs, nil)

	items, err := c.GetHelmReleases(context.Background(), "", "default")
	require.NoError(t, err)
	require.Len(t, items, 1)

	it := items[0]
	assert.Equal(t, "my-release", it.Name)
	assert.Equal(t, "HelmRelease", it.Kind)
	assert.Equal(t, "Deployed", it.Status)

	keys := map[string]string{}
	for _, kv := range it.Columns {
		keys[kv.Key] = kv.Value
	}
	assert.Equal(t, "nginx", keys["Chart"])
	assert.Equal(t, "15.4.2", keys["Chart Version"])
	assert.Equal(t, "1.25.3", keys["App Version"])
	assert.Equal(t, "3", keys["Revision"])
	assert.Equal(t, "Deployed", keys["Status"])
	assert.Contains(t, keys, "Last Deployed")
	assert.Equal(t, "Upgrade complete", keys["Description"])

	// Extra retained for backward compatibility with legacy readers.
	assert.Equal(t, "v3", it.Extra)
}

func TestGetHelmReleases_GracefulBlobFailure(t *testing.T) {
	// Broken release blob should not crash; the item should still be returned
	// using the label-only fallback with no Chart columns populated.
	now := time.Now()
	secret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "sh.helm.release.v1.broken.v1",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: now},
			Labels: map[string]string{
				"owner":   "helm",
				"name":    "broken",
				"status":  "failed",
				"version": "1",
			},
		},
		Data: map[string][]byte{"release": []byte("###not-valid-base64###")},
	}

	cs := k8sfake.NewClientset(secret)
	c := newFakeClient(cs, nil)

	items, err := c.GetHelmReleases(context.Background(), "", "default")
	require.NoError(t, err)
	require.Len(t, items, 1)

	it := items[0]
	assert.Equal(t, "broken", it.Name)
	assert.Equal(t, "Failed", it.Status)
	// No chart columns should have been populated.
	for _, kv := range it.Columns {
		assert.NotEqual(t, "Chart Version", kv.Key)
		assert.NotEqual(t, "App Version", kv.Key)
	}
}

func TestGetHelmReleases_StripsControlCharsFromDescription(t *testing.T) {
	// A release whose description contains ANSI escape sequences or other
	// control characters must not smuggle them into the TUI render layer.
	now := time.Now()
	// \x1b[31m = red, \x07 = BEL, \x1b]0;title\x07 = OSC title-set
	evil := "Upgrade\x1b[31m complete\x07\x1b]0;hijack\x07"
	secret := newFakeHelmReleaseSecret(
		t, "sh.helm.release.v1.ansi.v1", "ansi", "deployed", "1",
		makeHelmBlob("ansi", "app", "1.0.0", "1.0.0", "deployed", evil, 1),
		now,
	)
	cs := k8sfake.NewClientset(secret)
	c := newFakeClient(cs, nil)

	items, err := c.GetHelmReleases(context.Background(), "", "default")
	require.NoError(t, err)
	require.Len(t, items, 1)

	var desc string
	for _, kv := range items[0].Columns {
		if kv.Key == "Description" {
			desc = kv.Value
		}
	}
	assert.Equal(t, "Upgrade[31m complete]0;hijack", desc)
	assert.NotContains(t, desc, "\x1b")
	assert.NotContains(t, desc, "\x07")
}

// ciliumLikeManifest is a trimmed multi-document YAML manifest similar in
// shape to what the Cilium chart produces: it spans workloads, networking,
// RBAC, and cluster-scoped kinds that the old label-based discovery missed.
// Workloads are placed in the release's own namespace ("default") so the
// live-status merge path is exercised by this fixture; RBAC and the separate
// Namespace resource cover the cross-namespace and cluster-scoped cases.
const ciliumLikeManifest = `---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cilium
  namespace: default
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cilium-config
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: cilium-config-agent
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cilium-config-agent
  namespace: default
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRole
metadata:
  name: cilium
---
apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: cilium
---
apiVersion: gateway.networking.k8s.io/v1
kind: GatewayClass
metadata:
  name: cilium
---
apiVersion: v1
kind: Namespace
metadata:
  name: cilium-secrets
---
apiVersion: apps/v1
kind: DaemonSet
metadata:
  name: cilium
  namespace: default
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cilium-operator
  namespace: default
`

// TestGetHelmManagedResources_ManifestPath is the regression guard for charts
// whose resources cannot be discovered via the standard instance labels.
// The release manifest enumerates every kind, including cluster-scoped
// resources (ClusterRole, ClusterRoleBinding, Namespace) and custom
// resources (GatewayClass) that the old label-based implementation missed.
func TestGetHelmManagedResources_ManifestPath(t *testing.T) {
	now := time.Now()
	blob := makeHelmBlobWithManifest(
		"cilium", "cilium", "1.15.0", "1.15.0",
		"deployed", "Install complete", ciliumLikeManifest, 1,
	)
	secret := newFakeHelmReleaseSecret(
		t, "sh.helm.release.v1.cilium.v1", "cilium", "deployed", "1", blob, now,
	)
	// The fake clientset holds the helm release secret plus a live Deployment
	// so we can also verify live status enrichment (Ready column) below.
	liveDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "cilium-operator",
			Namespace: "default",
		},
		Spec: appsv1.DeploymentSpec{Replicas: int32Ptr(2)},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 2,
			ReadyReplicas:     2,
		},
	}
	cs := k8sfake.NewClientset(secret, liveDep)
	c := newFakeClient(cs, nil)

	items, err := c.getHelmManagedResources(context.Background(), "", "default", "cilium")
	require.NoError(t, err)

	kinds := map[string]int{}
	for _, it := range items {
		kinds[it.Kind]++
	}
	// All ten manifest kinds must appear at least once.
	require.Equal(t, 1, kinds["ServiceAccount"], "ServiceAccount missing from child resources")
	require.Equal(t, 1, kinds["ConfigMap"])
	require.Equal(t, 1, kinds["Role"])
	require.Equal(t, 1, kinds["RoleBinding"])
	require.Equal(t, 1, kinds["ClusterRole"], "cluster-scoped ClusterRole missing")
	require.Equal(t, 1, kinds["ClusterRoleBinding"], "cluster-scoped ClusterRoleBinding missing")
	require.Equal(t, 1, kinds["GatewayClass"], "custom GatewayClass kind missing")
	require.Equal(t, 1, kinds["Namespace"])
	require.Equal(t, 1, kinds["DaemonSet"])
	require.Equal(t, 1, kinds["Deployment"])

	// Live status enrichment: the Deployment that exists in the fake clientset
	// should have a Ready column populated from its live status.
	var depItem *model.Item
	for i := range items {
		if items[i].Kind == "Deployment" && items[i].Name == "cilium-operator" {
			depItem = &items[i]
			break
		}
	}
	require.NotNil(t, depItem, "expected cilium-operator Deployment in results")
	assert.Equal(t, "2/2", depItem.Ready, "live Deployment ready count must be merged into manifest ref")

	// Manifest-derived items should expose KIND and APIVERSION as Columns so
	// the explorer renders them as table columns (matching the ArgoCD child
	// pattern), and must NOT carry a per-kind Icon.
	for _, it := range items {
		assert.Empty(t, it.Icon, "manifest-derived items must not carry a per-kind icon: %s/%s", it.Kind, it.Name)
		var hasKind, hasAPIVersion bool
		for _, kv := range it.Columns {
			switch kv.Key {
			case "KIND":
				hasKind = true
				assert.Equal(t, it.Kind, kv.Value)
			case "APIVERSION":
				hasAPIVersion = true
				assert.NotEmpty(t, kv.Value)
			}
		}
		assert.True(t, hasKind, "%s/%s missing KIND column", it.Kind, it.Name)
		assert.True(t, hasAPIVersion, "%s/%s missing APIVERSION column", it.Kind, it.Name)
	}
}

// TestGetHelmManagedResources_EmptyManifestFallsBackToLabels verifies that
// when the manifest field is empty the code falls back to the pre-existing
// label-based discovery path so legacy releases keep working.
func TestGetHelmManagedResources_EmptyManifestFallsBackToLabels(t *testing.T) {
	now := time.Now()
	blob := makeHelmBlobWithManifest(
		"legacy", "legacy", "1.0.0", "1.0.0", "deployed", "Install", "", 1,
	)
	secret := newFakeHelmReleaseSecret(
		t, "sh.helm.release.v1.legacy.v1", "legacy", "deployed", "1", blob, now,
	)
	labelledDep := &appsv1.Deployment{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "legacy-app",
			Namespace: "default",
			Labels:    map[string]string{"app.kubernetes.io/instance": "legacy"},
		},
		Spec: appsv1.DeploymentSpec{Replicas: int32Ptr(1)},
		Status: appsv1.DeploymentStatus{
			AvailableReplicas: 1,
			ReadyReplicas:     1,
		},
	}
	cs := k8sfake.NewClientset(secret, labelledDep)
	c := newFakeClient(cs, nil)

	items, err := c.getHelmManagedResources(context.Background(), "", "default", "legacy")
	require.NoError(t, err)
	// Fallback path must still pick up the labelled Deployment.
	var found bool
	for _, it := range items {
		if it.Kind == "Deployment" && it.Name == "legacy-app" {
			found = true
			assert.Equal(t, "1/1", it.Ready)
			break
		}
	}
	assert.True(t, found, "fallback label-based discovery must surface the labelled Deployment")
}

// TestGetHelmManagedResources_DecodeFailureFallsBack confirms that if the
// helm release blob itself cannot be decoded (corrupt secret data), the
// function falls back to the label-based path instead of returning an error.
func TestGetHelmManagedResources_DecodeFailureFallsBack(t *testing.T) {
	now := time.Now()
	brokenSecret := &corev1.Secret{
		ObjectMeta: metav1.ObjectMeta{
			Name:              "sh.helm.release.v1.broken.v1",
			Namespace:         "default",
			CreationTimestamp: metav1.Time{Time: now},
			Labels: map[string]string{
				"owner":   "helm",
				"name":    "broken",
				"status":  "deployed",
				"version": "1",
			},
		},
		Data: map[string][]byte{"release": []byte("!!! not a helm release blob !!!")},
	}
	labelledSvc := &corev1.Service{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "broken-svc",
			Namespace: "default",
			Labels:    map[string]string{"app.kubernetes.io/instance": "broken"},
		},
		Spec: corev1.ServiceSpec{Ports: []corev1.ServicePort{{Port: 80}}},
	}
	cs := k8sfake.NewClientset(brokenSecret, labelledSvc)
	c := newFakeClient(cs, nil)

	items, err := c.getHelmManagedResources(context.Background(), "", "default", "broken")
	require.NoError(t, err)
	// Expect label-based fallback to surface the labelled Service.
	var found bool
	for _, it := range items {
		if it.Kind == "Service" && it.Name == "broken-svc" {
			found = true
			break
		}
	}
	assert.True(t, found, "decode-failure path must fall back to label-based discovery")
}

// TestGetHelmManagedResources_LatestSecretWins verifies that when multiple
// helm release secrets exist for the same release name, the manifest from the
// newest secret is the one consulted.
func TestGetHelmManagedResources_LatestSecretWins(t *testing.T) {
	older := time.Now().Add(-2 * time.Hour)
	newer := time.Now()
	oldManifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: old-cm
  namespace: default
`
	newManifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: new-cm
  namespace: default
`
	oldBlob := makeHelmBlobWithManifest("stacked", "app", "0.1.0", "0.1.0", "superseded", "Old", oldManifest, 1)
	newBlob := makeHelmBlobWithManifest("stacked", "app", "0.2.0", "0.2.0", "deployed", "New", newManifest, 2)

	oldSec := newFakeHelmReleaseSecret(t, "sh.helm.release.v1.stacked.v1", "stacked", "superseded", "1", oldBlob, older)
	newSec := newFakeHelmReleaseSecret(t, "sh.helm.release.v1.stacked.v2", "stacked", "deployed", "2", newBlob, newer)

	cs := k8sfake.NewClientset(oldSec, newSec)
	c := newFakeClient(cs, nil)

	items, err := c.getHelmManagedResources(context.Background(), "", "default", "stacked")
	require.NoError(t, err)

	var names []string
	for _, it := range items {
		if it.Kind == "ConfigMap" {
			names = append(names, it.Name)
		}
	}
	assert.Contains(t, names, "new-cm")
	assert.NotContains(t, names, "old-cm")
}

func TestGetHelmReleases_LatestVersionWins(t *testing.T) {
	earlier := time.Now().Add(-2 * time.Hour)
	later := time.Now()

	oldSecret := newFakeHelmReleaseSecret(
		t, "sh.helm.release.v1.multi.v1", "multi", "superseded", "1",
		makeHelmBlob("multi", "app", "0.1.0", "0.1.0", "superseded", "Install", 1),
		earlier,
	)
	newSecret := newFakeHelmReleaseSecret(
		t, "sh.helm.release.v1.multi.v2", "multi", "deployed", "2",
		makeHelmBlob("multi", "app", "0.2.0", "0.2.0", "deployed", "Upgrade", 2),
		later,
	)

	cs := k8sfake.NewClientset(oldSecret, newSecret)
	c := newFakeClient(cs, nil)

	items, err := c.GetHelmReleases(context.Background(), "", "default")
	require.NoError(t, err)
	require.Len(t, items, 1)

	it := items[0]
	assert.Equal(t, "multi", it.Name)
	keys := map[string]string{}
	for _, kv := range it.Columns {
		keys[kv.Key] = kv.Value
	}
	assert.Equal(t, "0.2.0", keys["Chart Version"])
	assert.Equal(t, "2", keys["Revision"])
}
