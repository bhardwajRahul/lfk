package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- saveSession + loadSession round-trip ---

func TestSaveAndLoadSession(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	original := SessionState{
		Context:   "prod-cluster",
		Namespace: "default",
		Tabs: []SessionTab{
			{Context: "prod-cluster", Namespace: "default", ResourceType: "apps/v1/deployments"},
			{Context: "staging-cluster", AllNamespaces: true},
		},
		ActiveTab: 0,
	}

	err := saveSession(original)
	require.NoError(t, err)

	loaded := loadSession()
	require.NotNil(t, loaded)
	assert.Equal(t, "prod-cluster", loaded.Context)
	assert.Equal(t, "default", loaded.Namespace)
	assert.Len(t, loaded.Tabs, 2)
	assert.Equal(t, "prod-cluster", loaded.Tabs[0].Context)
	assert.Equal(t, "apps/v1/deployments", loaded.Tabs[0].ResourceType)
	assert.True(t, loaded.Tabs[1].AllNamespaces)
}

func TestSaveSessionCreatesDirectory(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", filepath.Join(tmpDir, "nested", "dir"))

	err := saveSession(SessionState{Context: "test"})
	require.NoError(t, err)

	path := filepath.Join(tmpDir, "nested", "dir", "lfk", "session.yaml")
	_, err = os.Stat(path)
	assert.NoError(t, err)
}

func TestLoadSessionEmptyContextReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	// Save session with empty context.
	err := saveSession(SessionState{Context: "", Namespace: "default"})
	require.NoError(t, err)

	loaded := loadSession()
	assert.Nil(t, loaded)
}

func TestLoadSessionInvalidYAMLReturnsNil(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	dir := filepath.Join(tmpDir, "lfk")
	require.NoError(t, os.MkdirAll(dir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "session.yaml"), []byte("{{invalid yaml"), 0o644))

	loaded := loadSession()
	assert.Nil(t, loaded)
}

func TestLoadSessionWithSelectedNamespaces(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	original := SessionState{
		Context: "test-cluster",
		Tabs: []SessionTab{
			{
				Context:            "test-cluster",
				Namespace:          "ns-1",
				SelectedNamespaces: []string{"ns-1", "ns-2", "ns-3"},
			},
		},
	}

	require.NoError(t, saveSession(original))

	loaded := loadSession()
	require.NotNil(t, loaded)
	assert.Len(t, loaded.Tabs[0].SelectedNamespaces, 3)
}

func TestSessionFilePathWithXDG(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "/custom/state/dir")
	assert.Equal(t, "/custom/state/dir/lfk/session.yaml", sessionFilePath())
}

func TestSessionFilePathDefault(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", "")
	path := sessionFilePath()
	assert.Contains(t, path, ".local/state/lfk/session.yaml")
	assert.NotEmpty(t, path)
}

// --- SessionTab/SessionState struct ---

func TestSessionTabResourceNamePersisted(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	original := SessionState{
		Context: "test",
		Tabs: []SessionTab{
			{
				Context:      "test",
				Namespace:    "default",
				ResourceType: "apps/v1/deployments",
				ResourceName: "my-nginx",
			},
		},
	}

	require.NoError(t, saveSession(original))

	loaded := loadSession()
	require.NotNil(t, loaded)
	assert.Equal(t, "my-nginx", loaded.Tabs[0].ResourceName)
}
