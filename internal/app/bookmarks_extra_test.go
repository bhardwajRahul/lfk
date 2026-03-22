package app

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- removeBookmark ---

func TestRemoveBookmarkEdgeCases(t *testing.T) {
	bookmarks := []model.Bookmark{
		{Slot: "a", Name: "first"},
		{Slot: "b", Name: "second"},
		{Slot: "c", Name: "third"},
	}

	t.Run("removes middle element", func(t *testing.T) {
		bm := make([]model.Bookmark, len(bookmarks))
		copy(bm, bookmarks)
		result := removeBookmark(bm, 1)
		assert.Len(t, result, 2)
		assert.Equal(t, "a", result[0].Slot)
		assert.Equal(t, "c", result[1].Slot)
	})

	t.Run("removes first element", func(t *testing.T) {
		bm := []model.Bookmark{
			{Slot: "a", Name: "first"},
			{Slot: "b", Name: "second"},
		}
		result := removeBookmark(bm, 0)
		assert.Len(t, result, 1)
		assert.Equal(t, "b", result[0].Slot)
	})

	t.Run("removes last element", func(t *testing.T) {
		bm := []model.Bookmark{
			{Slot: "a", Name: "first"},
			{Slot: "b", Name: "second"},
		}
		result := removeBookmark(bm, 1)
		assert.Len(t, result, 1)
		assert.Equal(t, "a", result[0].Slot)
	})

	t.Run("negative index returns unchanged", func(t *testing.T) {
		result := removeBookmark(bookmarks, -1)
		assert.Len(t, result, 3)
	})

	t.Run("out of range index returns unchanged", func(t *testing.T) {
		result := removeBookmark(bookmarks, 100)
		assert.Len(t, result, 3)
	})

	t.Run("empty slice returns empty", func(t *testing.T) {
		result := removeBookmark(nil, 0)
		assert.Nil(t, result)
	})
}

// --- bookmarkDeleteCurrent ---

func TestBookmarkDeleteCurrent(t *testing.T) {
	t.Run("deletes bookmark at cursor", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_STATE_HOME", tmpDir)

		m := Model{
			bookmarks: []model.Bookmark{
				{Slot: "a", Name: "first"},
				{Slot: "b", Name: "second"},
				{Slot: "c", Name: "third"},
			},
			overlayCursor:  1,
			bookmarkFilter: TextInput{},
		}
		cmd := m.bookmarkDeleteCurrent()
		assert.NotNil(t, cmd)
		assert.Len(t, m.bookmarks, 2)
		assert.Equal(t, "a", m.bookmarks[0].Slot)
		assert.Equal(t, "c", m.bookmarks[1].Slot)
	})

	t.Run("empty bookmarks returns nil", func(t *testing.T) {
		m := Model{
			bookmarks:     nil,
			overlayCursor: 0,
		}
		cmd := m.bookmarkDeleteCurrent()
		assert.Nil(t, cmd)
	})

	t.Run("cursor out of range returns nil", func(t *testing.T) {
		m := Model{
			bookmarks: []model.Bookmark{
				{Slot: "a", Name: "first"},
			},
			overlayCursor: 5,
		}
		cmd := m.bookmarkDeleteCurrent()
		assert.Nil(t, cmd)
	})

	t.Run("deleting last bookmark closes overlay", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_STATE_HOME", tmpDir)

		m := Model{
			bookmarks: []model.Bookmark{
				{Slot: "a", Name: "only"},
			},
			overlayCursor:  0,
			bookmarkFilter: TextInput{},
			overlay:        overlayBookmarks,
		}
		cmd := m.bookmarkDeleteCurrent()
		assert.NotNil(t, cmd)
		assert.Empty(t, m.bookmarks)
		assert.Equal(t, overlayNone, m.overlay)
	})

	t.Run("adjusts cursor when at end", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_STATE_HOME", tmpDir)

		m := Model{
			bookmarks: []model.Bookmark{
				{Slot: "a", Name: "first"},
				{Slot: "b", Name: "second"},
			},
			overlayCursor:  1,
			bookmarkFilter: TextInput{},
		}
		m.bookmarkDeleteCurrent()
		assert.Equal(t, 0, m.overlayCursor)
	})
}

// --- bookmarkDeleteAll ---

func TestBookmarkDeleteAll(t *testing.T) {
	t.Run("deletes all bookmarks without filter", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_STATE_HOME", tmpDir)

		m := Model{
			bookmarks: []model.Bookmark{
				{Slot: "a", Name: "first"},
				{Slot: "b", Name: "second"},
			},
			bookmarkFilter: TextInput{},
			overlay:        overlayBookmarks,
		}
		cmd := m.bookmarkDeleteAll()
		assert.NotNil(t, cmd)
		assert.Nil(t, m.bookmarks)
		assert.Equal(t, overlayNone, m.overlay)
	})

	t.Run("deletes only filtered bookmarks", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_STATE_HOME", tmpDir)

		m := Model{
			bookmarks: []model.Bookmark{
				{Slot: "a", Name: "prod > Deployments"},
				{Slot: "b", Name: "staging > Pods"},
				{Slot: "c", Name: "prod > Services"},
			},
			bookmarkFilter: TextInput{Value: "prod"},
			overlay:        overlayBookmarks,
		}
		cmd := m.bookmarkDeleteAll()
		assert.NotNil(t, cmd)
		assert.Len(t, m.bookmarks, 1)
		assert.Equal(t, "b", m.bookmarks[0].Slot)
	})

	t.Run("empty bookmarks returns nil", func(t *testing.T) {
		m := Model{
			bookmarks:      nil,
			bookmarkFilter: TextInput{},
		}
		cmd := m.bookmarkDeleteAll()
		assert.Nil(t, cmd)
	})

	t.Run("no filtered match returns nil", func(t *testing.T) {
		m := Model{
			bookmarks: []model.Bookmark{
				{Slot: "a", Name: "first"},
			},
			bookmarkFilter: TextInput{Value: "nonexistent"},
		}
		cmd := m.bookmarkDeleteAll()
		assert.Nil(t, cmd)
	})

	t.Run("resets cursor", func(t *testing.T) {
		tmpDir := t.TempDir()
		t.Setenv("XDG_STATE_HOME", tmpDir)

		m := Model{
			bookmarks: []model.Bookmark{
				{Slot: "a", Name: "first"},
				{Slot: "b", Name: "second"},
			},
			bookmarkFilter: TextInput{},
			overlayCursor:  1,
		}
		m.bookmarkDeleteAll()
		assert.Equal(t, 0, m.overlayCursor)
	})
}

// --- loadBookmarks invalid YAML ---

func TestLoadBookmarksInvalidYAML(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	dir := tmpDir + "/lfk"
	_ = os.MkdirAll(dir, 0o755)
	_ = os.WriteFile(dir+"/bookmarks.yaml", []byte("{{invalid"), 0o644)

	bookmarks := loadBookmarks()
	assert.Nil(t, bookmarks)
}
