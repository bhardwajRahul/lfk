package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- findNextLogMatch ---

func TestFindNextLogMatch(t *testing.T) {
	t.Run("forward finds next match", func(t *testing.T) {
		m := Model{
			height:         30,
			width:          80,
			tabs:           []TabState{{}},
			logLines:       []string{"info: start", "error: failed", "info: ok", "error: timeout"},
			logSearchQuery: "error",
			logCursor:      0,
		}
		m.findNextLogMatch(true)
		assert.Equal(t, 1, m.logCursor)
	})

	t.Run("forward wraps around", func(t *testing.T) {
		m := Model{
			height:         30,
			width:          80,
			tabs:           []TabState{{}},
			logLines:       []string{"error: first", "info: ok", "info: ok2"},
			logSearchQuery: "error",
			logCursor:      2,
		}
		m.findNextLogMatch(true)
		assert.Equal(t, 0, m.logCursor)
	})

	t.Run("backward finds previous match", func(t *testing.T) {
		m := Model{
			height:         30,
			width:          80,
			tabs:           []TabState{{}},
			logLines:       []string{"error: first", "info: ok", "error: second", "info: ok2"},
			logSearchQuery: "error",
			logCursor:      3,
		}
		m.findNextLogMatch(false)
		assert.Equal(t, 2, m.logCursor)
	})

	t.Run("backward wraps around", func(t *testing.T) {
		m := Model{
			height:         30,
			width:          80,
			tabs:           []TabState{{}},
			logLines:       []string{"info: ok", "info: ok2", "error: last"},
			logSearchQuery: "error",
			logCursor:      0,
		}
		m.findNextLogMatch(false)
		assert.Equal(t, 2, m.logCursor)
	})

	t.Run("empty query does nothing", func(t *testing.T) {
		m := Model{
			height:         30,
			width:          80,
			tabs:           []TabState{{}},
			logLines:       []string{"error: test"},
			logSearchQuery: "",
			logCursor:      0,
		}
		m.findNextLogMatch(true)
		assert.Equal(t, 0, m.logCursor)
	})

	t.Run("no match keeps cursor", func(t *testing.T) {
		m := Model{
			height:         30,
			width:          80,
			tabs:           []TabState{{}},
			logLines:       []string{"info: ok", "debug: test"},
			logSearchQuery: "error",
			logCursor:      0,
		}
		m.findNextLogMatch(true)
		assert.Equal(t, 0, m.logCursor)
	})

	t.Run("case insensitive search", func(t *testing.T) {
		m := Model{
			height:         30,
			width:          80,
			tabs:           []TabState{{}},
			logLines:       []string{"info: ok", "ERROR: FAILED"},
			logSearchQuery: "error",
			logCursor:      0,
		}
		m.findNextLogMatch(true)
		assert.Equal(t, 1, m.logCursor)
	})

	t.Run("disables log follow on match", func(t *testing.T) {
		m := Model{
			height:         30,
			width:          80,
			tabs:           []TabState{{}},
			logLines:       []string{"info: ok", "error: test"},
			logSearchQuery: "error",
			logCursor:      0,
			logFollow:      true,
		}
		m.findNextLogMatch(true)
		assert.Equal(t, 1, m.logCursor)
		assert.False(t, m.logFollow)
	})
}
