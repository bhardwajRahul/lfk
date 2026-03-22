package app

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- commandHistory.add ---

func TestCommandHistoryAdd(t *testing.T) {
	h := &commandHistory{cursor: -1}

	h.add("get pods")
	assert.Equal(t, []string{"get pods"}, h.entries)

	h.add("get deployments")
	assert.Equal(t, []string{"get pods", "get deployments"}, h.entries)
}

func TestCommandHistoryAddIgnoresEmpty(t *testing.T) {
	h := &commandHistory{cursor: -1}

	h.add("")
	assert.Empty(t, h.entries)

	h.add("   ")
	assert.Empty(t, h.entries)
}

func TestCommandHistoryAddDeduplicates(t *testing.T) {
	h := &commandHistory{cursor: -1}

	h.add("get pods")
	h.add("get pods")
	assert.Len(t, h.entries, 1)

	h.add("get deployments")
	h.add("get pods") // different from last entry, so not deduplicated
	assert.Len(t, h.entries, 3)
}

func TestCommandHistoryAddTrimsToMax(t *testing.T) {
	h := &commandHistory{cursor: -1}
	for i := range maxHistoryEntries + 10 {
		h.add("cmd-" + string(rune('a'+i%26)) + string(rune('0'+i/26)))
	}
	assert.LessOrEqual(t, len(h.entries), maxHistoryEntries)
}

// --- commandHistory.up ---

func TestCommandHistoryUp(t *testing.T) {
	h := &commandHistory{cursor: -1}
	h.entries = []string{"first", "second", "third"}

	// First up: should save draft and return last entry.
	result := h.up("current draft")
	assert.Equal(t, "third", result)
	assert.Equal(t, "current draft", h.draft)
	assert.Equal(t, 2, h.cursor)

	// Second up: returns previous entry.
	result = h.up("")
	assert.Equal(t, "second", result)
	assert.Equal(t, 1, h.cursor)

	// Third up: returns first entry.
	result = h.up("")
	assert.Equal(t, "first", result)
	assert.Equal(t, 0, h.cursor)

	// Already at beginning: stays there.
	result = h.up("")
	assert.Equal(t, "first", result)
	assert.Equal(t, 0, h.cursor)
}

func TestCommandHistoryUpEmpty(t *testing.T) {
	h := &commandHistory{cursor: -1}

	result := h.up("my input")
	assert.Equal(t, "my input", result)
}

// --- commandHistory.down ---

func TestCommandHistoryDown(t *testing.T) {
	h := &commandHistory{cursor: -1}
	h.entries = []string{"first", "second"}

	// Navigate up first.
	h.up("draft text")
	h.up("") // at "first"
	assert.Equal(t, 0, h.cursor)

	// Down returns "second".
	result := h.down()
	assert.Equal(t, "second", result)
	assert.Equal(t, 1, h.cursor)

	// Down past end restores draft.
	result = h.down()
	assert.Equal(t, "draft text", result)
	assert.Equal(t, -1, h.cursor)
}

func TestCommandHistoryDownNotBrowsing(t *testing.T) {
	h := &commandHistory{cursor: -1, draft: "my draft"}

	result := h.down()
	assert.Equal(t, "my draft", result)
}

// --- commandHistory.reset ---

func TestCommandHistoryReset(t *testing.T) {
	h := &commandHistory{cursor: 2, draft: "something"}

	h.reset()
	assert.Equal(t, -1, h.cursor)
	assert.Empty(t, h.draft)
}

// --- commandHistory.save / loadCommandHistory ---

func TestCommandHistorySaveAndLoad(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	h := &commandHistory{cursor: -1}
	h.add("get pods")
	h.add("get deployments")
	h.add("logs my-pod")

	h.save()

	// Verify file exists.
	path := filepath.Join(tmpDir, "lfk", "history")
	_, err := os.Stat(path)
	require.NoError(t, err)

	// Load and verify.
	loaded := loadCommandHistory()
	assert.Equal(t, []string{"get pods", "get deployments", "logs my-pod"}, loaded.entries)
	assert.Equal(t, -1, loaded.cursor)
}

func TestLoadCommandHistoryNoFile(t *testing.T) {
	tmpDir := t.TempDir()
	t.Setenv("XDG_STATE_HOME", tmpDir)

	loaded := loadCommandHistory()
	assert.Empty(t, loaded.entries)
	assert.Equal(t, -1, loaded.cursor)
}

// --- historyFilePath ---

func TestHistoryFilePath(t *testing.T) {
	t.Run("uses XDG_STATE_HOME", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "/custom/state")
		path := historyFilePath()
		assert.Equal(t, "/custom/state/lfk/history", path)
	})

	t.Run("falls back to home", func(t *testing.T) {
		t.Setenv("XDG_STATE_HOME", "")
		path := historyFilePath()
		assert.Contains(t, path, ".local/state/lfk/history")
	})
}
