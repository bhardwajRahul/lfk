package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- portForwardStatePath ---

func TestPortForwardStatePath(t *testing.T) {
	t.Run("uses XDG_STATE_HOME", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "/custom/state")
		path := portForwardStatePath()
		assert.Equal(t, "/custom/state/lfk/portforwards.yaml", path)
	})

	t.Run("falls back to home", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		path := portForwardStatePath()
		assert.Contains(t, path, ".local/state/lfk/portforwards.yaml")
	})
}

// --- savePortForwardState / loadPortForwardState ---

func TestSaveAndLoadPortForwardState(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	state := &PortForwardStates{
		PortForwards: []PortForwardState{
			{
				ResourceKind: "Service",
				ResourceName: "my-svc",
				Namespace:    "default",
				Context:      "prod",
				LocalPort:    "8080",
				RemotePort:   "80",
			},
			{
				ResourceKind: "Pod",
				ResourceName: "my-pod",
				Namespace:    "dev",
				Context:      "dev-cluster",
				LocalPort:    "9090",
				RemotePort:   "9090",
			},
		},
	}

	err := savePortForwardState(state)
	require.NoError(t, err)

	// Verify file exists.
	path := filepath.Join(tmpDir, "lfk", "portforwards.yaml")
	_, err = os.Stat(path)
	require.NoError(t, err)

	// Load and verify.
	loaded := loadPortForwardState()
	require.Len(t, loaded.PortForwards, 2)
	assert.Equal(t, "my-svc", loaded.PortForwards[0].ResourceName)
	assert.Equal(t, "8080", loaded.PortForwards[0].LocalPort)
	assert.Equal(t, "my-pod", loaded.PortForwards[1].ResourceName)
	assert.Equal(t, "dev-cluster", loaded.PortForwards[1].Context)
}

func TestLoadPortForwardStateNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	loaded := loadPortForwardState()
	assert.NotNil(t, loaded)
	assert.Empty(t, loaded.PortForwards)
}

func TestSavePortForwardStateEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	err := savePortForwardState(&PortForwardStates{})
	require.NoError(t, err)

	loaded := loadPortForwardState()
	assert.Empty(t, loaded.PortForwards)
}
