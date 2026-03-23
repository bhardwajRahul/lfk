package k8s

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/client-go/tools/clientcmd"
)

// newTestClient creates a minimal Client backed by a temporary kubeconfig
// that has a single context pointing to a non-existent server.  This is
// sufficient for clientsetForContext to succeed (it only builds the HTTP
// client, it does not connect), so the switch-on-kind logic can be reached.
func newTestClient(t *testing.T) *Client {
	t.Helper()

	kubecfg := `
apiVersion: v1
kind: Config
clusters:
- cluster:
    server: https://127.0.0.1:1
  name: test-cluster
contexts:
- context:
    cluster: test-cluster
    namespace: default
    user: test-user
  name: test-context
current-context: test-context
users:
- name: test-user
  user: {}
`
	dir := t.TempDir()
	cfgPath := filepath.Join(dir, "kubeconfig")
	require.NoError(t, os.WriteFile(cfgPath, []byte(kubecfg), 0o600))

	loadingRules := &clientcmd.ClientConfigLoadingRules{
		Precedence: []string{cfgPath},
	}
	overrides := &clientcmd.ConfigOverrides{}
	cc := clientcmd.NewNonInteractiveDeferredLoadingClientConfig(loadingRules, overrides)

	rawConfig, err := cc.RawConfig()
	require.NoError(t, err)

	return &Client{
		rawConfig:    rawConfig,
		loadingRules: loadingRules,
	}
}

// --- ScaleResource ---

func TestScaleResource_UnsupportedKind(t *testing.T) {
	tests := []struct {
		name string
		kind string
	}{
		{"DaemonSet is unsupported for scaling", "DaemonSet"},
		{"Job is unsupported for scaling", "Job"},
		{"Pod is unsupported for scaling", "Pod"},
		{"empty kind is unsupported for scaling", ""},
	}

	c := newTestClient(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ScaleResource("test-context", "default", "my-resource", tt.kind, 3)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported kind for scaling")
			assert.Contains(t, err.Error(), tt.kind)
		})
	}
}

func TestScaleResource_SupportedKinds(t *testing.T) {
	// For supported kinds, clientsetForContext succeeds but the actual API
	// call to GetScale fails because there is no real server.  The key
	// assertion is that the error does NOT contain "unsupported kind for
	// scaling", proving the kind was dispatched to the correct branch.
	tests := []struct {
		name string
		kind string
	}{
		{"Deployment is supported for scaling", "Deployment"},
		{"StatefulSet is supported for scaling", "StatefulSet"},
		{"ReplicaSet is supported for scaling", "ReplicaSet"},
	}

	c := newTestClient(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.ScaleResource("test-context", "default", "my-resource", tt.kind, 3)
			// The call will fail (no server), but must NOT be an
			// "unsupported kind" error.
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "unsupported kind for scaling")
		})
	}
}

// --- RestartResource ---

func TestRestartResource_UnsupportedKind(t *testing.T) {
	tests := []struct {
		name string
		kind string
	}{
		{"ReplicaSet is unsupported for restart", "ReplicaSet"},
		{"Job is unsupported for restart", "Job"},
		{"Pod is unsupported for restart", "Pod"},
		{"empty kind is unsupported for restart", ""},
	}

	c := newTestClient(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.RestartResource("test-context", "default", "my-resource", tt.kind)
			require.Error(t, err)
			assert.Contains(t, err.Error(), "unsupported kind for restart")
			assert.Contains(t, err.Error(), tt.kind)
		})
	}
}

func TestRestartResource_SupportedKinds(t *testing.T) {
	// For supported kinds, the method should pass the kind switch and
	// attempt a real API patch.  The patch will fail (no server), but the
	// error must NOT mention "unsupported kind".
	tests := []struct {
		name string
		kind string
	}{
		{"Deployment is supported for restart", "Deployment"},
		{"StatefulSet is supported for restart", "StatefulSet"},
		{"DaemonSet is supported for restart", "DaemonSet"},
	}

	c := newTestClient(t)

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			err := c.RestartResource("test-context", "default", "my-resource", tt.kind)
			// The call will fail (no server), but must NOT be an
			// "unsupported kind" error.
			require.Error(t, err)
			assert.NotContains(t, err.Error(), "unsupported kind for restart")
		})
	}
}

// --- ScaleResource: invalid context ---

func TestScaleResource_InvalidContext(t *testing.T) {
	c := newTestClient(t)
	err := c.ScaleResource("nonexistent-context", "default", "my-deploy", "Deployment", 3)
	require.Error(t, err)
}

// --- RestartResource: invalid context ---

func TestRestartResource_InvalidContext(t *testing.T) {
	c := newTestClient(t)
	err := c.RestartResource("nonexistent-context", "default", "my-deploy", "Deployment")
	require.Error(t, err)
}
