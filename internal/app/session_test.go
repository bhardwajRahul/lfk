package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- sessionFilePath ---

func TestSessionFilePath(t *testing.T) {
	t.Run("uses XDG_STATE_HOME", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "/custom/state")
		path := sessionFilePath()
		assert.Equal(t, "/custom/state/lfk/session.yaml", path)
	})

	t.Run("falls back to home", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		path := sessionFilePath()
		assert.Contains(t, path, ".local/state/lfk/session.yaml")
	})
}

// --- migrateStateFile ---

func TestMigrateStateFile(t *testing.T) {
	t.Run("no legacy file returns nil", func(t *testing.T) {
		tmpDir := t.TempDir()
		newPath := filepath.Join(tmpDir, "new", "file.yaml")
		data := migrateStateFile("nonexistent.yaml", newPath)
		assert.Nil(t, data)
	})

	t.Run("migrates legacy file", func(t *testing.T) {
		// Create a fake home dir with legacy config.
		tmpDir := t.TempDir()
		t.Setenv("HOME", tmpDir)

		legacyDir := filepath.Join(tmpDir, ".config", "lfk")
		require.NoError(t, os.MkdirAll(legacyDir, 0o755))

		legacyContent := []byte("test: data\n")
		require.NoError(t, os.WriteFile(filepath.Join(legacyDir, "bookmarks.yaml"), legacyContent, 0o644))

		newDir := filepath.Join(tmpDir, "state", "lfk")
		newPath := filepath.Join(newDir, "bookmarks.yaml")

		data := migrateStateFile("bookmarks.yaml", newPath)
		assert.Equal(t, legacyContent, data)

		// Verify new file was created.
		newData, err := os.ReadFile(newPath)
		require.NoError(t, err)
		assert.Equal(t, legacyContent, newData)

		// Verify legacy file was removed.
		_, err = os.Stat(filepath.Join(legacyDir, "bookmarks.yaml"))
		assert.True(t, os.IsNotExist(err))
	})
}

// --- loadSession ---

func TestLoadSessionNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	session := loadSession()
	assert.Nil(t, session)
}
