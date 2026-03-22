package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/janosmiko/lfk/internal/model"
)

// --- loadBookmarks / saveBookmarks ---

func TestSaveAndLoadBookmarks(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	bookmarks := []model.Bookmark{
		{
			Name:         "prod-pods",
			Context:      "prod-cluster",
			Namespace:    "production",
			ResourceType: "pods",
		},
		{
			Name:         "dev-deployments",
			Context:      "dev-cluster",
			Namespace:    "development",
			ResourceType: "deployments",
		},
	}

	err := saveBookmarks(bookmarks)
	require.NoError(t, err)

	// Verify file was created.
	expectedPath := filepath.Join(tmpDir, "lfk", "bookmarks.yaml")
	_, err = os.Stat(expectedPath)
	require.NoError(t, err)

	// Load and verify.
	loaded := loadBookmarks()
	require.Len(t, loaded, 2)
	assert.Equal(t, "prod-pods", loaded[0].Name)
	assert.Equal(t, "prod-cluster", loaded[0].Context)
	assert.Equal(t, "production", loaded[0].Namespace)
	assert.Equal(t, "dev-deployments", loaded[1].Name)
}

func TestLoadBookmarksNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	loaded := loadBookmarks()
	assert.Nil(t, loaded)
}

func TestSaveBookmarksEmpty(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	err := saveBookmarks([]model.Bookmark{})
	require.NoError(t, err)

	loaded := loadBookmarks()
	assert.Empty(t, loaded)
}

func TestSaveBookmarksOverwrite(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	// Save initial bookmarks.
	err := saveBookmarks([]model.Bookmark{{Name: "first"}})
	require.NoError(t, err)

	// Overwrite with different bookmarks.
	err = saveBookmarks([]model.Bookmark{{Name: "second"}, {Name: "third"}})
	require.NoError(t, err)

	loaded := loadBookmarks()
	require.Len(t, loaded, 2)
	assert.Equal(t, "second", loaded[0].Name)
	assert.Equal(t, "third", loaded[1].Name)
}

// --- bookmarksFilePath ---

func TestBookmarksFilePath(t *testing.T) {
	t.Run("uses XDG_STATE_HOME when set", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "/custom/state")
		path := bookmarksFilePath()
		assert.Equal(t, "/custom/state/lfk/bookmarks.yaml", path)
	})

	t.Run("falls back to home directory", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		path := bookmarksFilePath()
		assert.Contains(t, path, ".local/state/lfk/bookmarks.yaml")
		assert.NotEmpty(t, path)
	})
}
