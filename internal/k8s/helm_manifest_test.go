package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestParseHelmManifest_HappyPath verifies that a multi-document YAML manifest
// produces the expected refs in document order, with namespaces preserved or
// left empty for cluster-scoped resources.
func TestParseHelmManifest_HappyPath(t *testing.T) {
	manifest := `---
apiVersion: v1
kind: ServiceAccount
metadata:
  name: cilium
  namespace: kube-system
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: cilium-config
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: Role
metadata:
  name: cilium
  namespace: kube-system
---
apiVersion: rbac.authorization.k8s.io/v1
kind: RoleBinding
metadata:
  name: cilium
  namespace: kube-system
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
  namespace: kube-system
---
apiVersion: apps/v1
kind: Deployment
metadata:
  name: cilium-operator
  namespace: kube-system
`

	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	require.Len(t, refs, 10)

	// Spot-check ordering and key fields.
	assert.Equal(t, "ServiceAccount", refs[0].Kind)
	assert.Equal(t, "cilium", refs[0].Name)
	assert.Equal(t, "kube-system", refs[0].Namespace)

	assert.Equal(t, "ClusterRole", refs[4].Kind)
	assert.Empty(t, refs[4].Namespace, "cluster-scoped ClusterRole has no namespace")

	assert.Equal(t, "Namespace", refs[7].Kind)
	assert.Equal(t, "cilium-secrets", refs[7].Name)
	assert.Empty(t, refs[7].Namespace)

	assert.Equal(t, "DaemonSet", refs[8].Kind)
	assert.Equal(t, "Deployment", refs[9].Kind)
	assert.Equal(t, "cilium-operator", refs[9].Name)
}

// TestParseHelmManifest_EmptyInput returns no refs and no error.
func TestParseHelmManifest_EmptyInput(t *testing.T) {
	refs, err := parseHelmManifest("")
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// TestParseHelmManifest_OnlySeparators returns no refs when the input has
// only YAML document separators and whitespace.
func TestParseHelmManifest_OnlySeparators(t *testing.T) {
	manifest := `---
---

---
`
	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	assert.Empty(t, refs)
}

// TestParseHelmManifest_CommentOnlyDocSkipped verifies that a doc consisting
// only of comments is silently skipped.
func TestParseHelmManifest_CommentOnlyDocSkipped(t *testing.T) {
	manifest := `# top of file comment
# nothing here yet
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: real
  namespace: default
`
	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "ConfigMap", refs[0].Kind)
}

// TestParseHelmManifest_MalformedDocSkipped verifies that one bad doc does not
// abort parsing of the rest, and the function does not return an error.
func TestParseHelmManifest_MalformedDocSkipped(t *testing.T) {
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: good
  namespace: default
---
this: is: not: valid: yaml: : :
---
apiVersion: v1
kind: Service
metadata:
  name: also-good
  namespace: default
`
	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, "good", refs[0].Name)
	assert.Equal(t, "also-good", refs[1].Name)
}

// TestParseHelmManifest_ClusterScoped omits namespace for cluster-scoped kinds.
func TestParseHelmManifest_ClusterScoped(t *testing.T) {
	manifest := `apiVersion: rbac.authorization.k8s.io/v1
kind: ClusterRoleBinding
metadata:
  name: my-binding
`
	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "ClusterRoleBinding", refs[0].Kind)
	assert.Equal(t, "my-binding", refs[0].Name)
	assert.Empty(t, refs[0].Namespace)
}

// TestParseHelmManifest_CustomResource verifies that an arbitrary CRD-defined
// kind round-trips through the parser.
func TestParseHelmManifest_CustomResource(t *testing.T) {
	manifest := `apiVersion: example.com/v1
kind: Foo
metadata:
  name: bar
  namespace: ns1
`
	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "example.com/v1", refs[0].APIVersion)
	assert.Equal(t, "Foo", refs[0].Kind)
	assert.Equal(t, "bar", refs[0].Name)
	assert.Equal(t, "ns1", refs[0].Namespace)
}

// TestParseHelmManifest_SeparatorInsideStringValue ensures the YAML decoder is
// not splitting on `---` that lives inside a quoted string value, which would
// create spurious refs.
func TestParseHelmManifest_SeparatorInsideStringValue(t *testing.T) {
	manifest := `apiVersion: v1
kind: ConfigMap
metadata:
  name: with-yaml-in-data
  namespace: default
data:
  embedded.yaml: |
    apiVersion: v1
    kind: Pod
    metadata:
      name: not-a-real-resource
    ---
    apiVersion: v1
    kind: Pod
    metadata:
      name: also-not-real
`
	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	require.Len(t, refs, 1, "embedded YAML inside a string value must not be parsed as separate docs")
	assert.Equal(t, "with-yaml-in-data", refs[0].Name)
}

// TestParseHelmManifest_DocWithoutKindIsSkipped verifies a doc that has no
// kind field is silently dropped instead of producing a ghost ref.
func TestParseHelmManifest_DocWithoutKindIsSkipped(t *testing.T) {
	manifest := `metadata:
  name: orphan
---
apiVersion: v1
kind: ConfigMap
metadata:
  name: real
`
	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	require.Len(t, refs, 1)
	assert.Equal(t, "real", refs[0].Name)
}

// TestParseHelmManifest_SeparatorTrailing covers separator lines with trailing
// whitespace or a YAML comment, and also ensures lines beginning with "---"
// but followed by non-comment content (e.g. "---foo") are NOT treated as
// separators.
func TestParseHelmManifest_SeparatorTrailing(t *testing.T) {
	manifest := "--- \t\n" +
		"apiVersion: v1\n" +
		"kind: ConfigMap\n" +
		"metadata:\n" +
		"  name: one\n" +
		"--- # start next\n" +
		"apiVersion: v1\n" +
		"kind: ConfigMap\n" +
		"metadata:\n" +
		"  name: two\n"
	refs, err := parseHelmManifest(manifest)
	require.NoError(t, err)
	require.Len(t, refs, 2)
	assert.Equal(t, "one", refs[0].Name)
	assert.Equal(t, "two", refs[1].Name)
}

func TestIsYAMLDocSeparator(t *testing.T) {
	cases := []struct {
		line string
		want bool
	}{
		{"---", true},
		{"--- ", true},
		{"---\t", true},
		{"---\r", true},
		{"--- # comment", true},
		{"---#comment", false},
		{"----", false},
		{"---foo", false},
		{" ---", false},
		{"", false},
		{"apiVersion: v1", false},
	}
	for _, tc := range cases {
		assert.Equal(t, tc.want, isYAMLDocSeparator(tc.line), "line=%q", tc.line)
	}
}
