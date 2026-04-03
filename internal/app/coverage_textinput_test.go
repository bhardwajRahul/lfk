package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// =====================================================================
// textinput.go: all TextInput methods (all 0%)
// =====================================================================

func TestCovTextInputInsert(t *testing.T) {
	ti := TextInput{}
	ti.Insert("hello")
	assert.Equal(t, "hello", ti.Value)
	assert.Equal(t, 5, ti.Cursor)

	ti.Insert(" world")
	assert.Equal(t, "hello world", ti.Value)
	assert.Equal(t, 11, ti.Cursor)
}

func TestCovTextInputInsertMiddle(t *testing.T) {
	ti := TextInput{Value: "helo", Cursor: 2}
	ti.Insert("l")
	assert.Equal(t, "hello", ti.Value)
	assert.Equal(t, 3, ti.Cursor)
}

func TestCovTextInputBackspace(t *testing.T) {
	ti := TextInput{Value: "hello", Cursor: 5}
	ti.Backspace()
	assert.Equal(t, "hell", ti.Value)
	assert.Equal(t, 4, ti.Cursor)

	// At start: no-op.
	ti.Cursor = 0
	ti.Backspace()
	assert.Equal(t, "hell", ti.Value)
	assert.Equal(t, 0, ti.Cursor)
}

func TestCovTextInputDeleteWord(t *testing.T) {
	ti := TextInput{Value: "hello world", Cursor: 11}
	ti.DeleteWord()
	assert.Equal(t, "hello ", ti.Value)
	assert.Equal(t, 6, ti.Cursor)

	ti.DeleteWord()
	assert.Equal(t, "", ti.Value)
	assert.Equal(t, 0, ti.Cursor)

	// At cursor 0: no-op.
	ti.DeleteWord()
	assert.Equal(t, "", ti.Value)
}

func TestCovTextInputDeleteWordWithSpaces(t *testing.T) {
	ti := TextInput{Value: "hello   world  ", Cursor: 15}
	ti.DeleteWord()
	assert.Equal(t, "hello   ", ti.Value)
}

func TestCovTextInputHome(t *testing.T) {
	ti := TextInput{Value: "hello", Cursor: 3}
	ti.Home()
	assert.Equal(t, 0, ti.Cursor)
}

func TestCovTextInputEnd(t *testing.T) {
	ti := TextInput{Value: "hello", Cursor: 0}
	ti.End()
	assert.Equal(t, 5, ti.Cursor)
}

func TestCovTextInputLeft(t *testing.T) {
	ti := TextInput{Value: "hello", Cursor: 3}
	ti.Left()
	assert.Equal(t, 2, ti.Cursor)

	ti.Cursor = 0
	ti.Left()
	assert.Equal(t, 0, ti.Cursor)
}

func TestCovTextInputRight(t *testing.T) {
	ti := TextInput{Value: "hello", Cursor: 3}
	ti.Right()
	assert.Equal(t, 4, ti.Cursor)

	ti.Cursor = 5
	ti.Right()
	assert.Equal(t, 5, ti.Cursor)
}

func TestCovTextInputSet(t *testing.T) {
	ti := TextInput{}
	ti.Set("new value")
	assert.Equal(t, "new value", ti.Value)
	assert.Equal(t, 9, ti.Cursor)
}

func TestCovTextInputClear(t *testing.T) {
	ti := TextInput{Value: "hello", Cursor: 3}
	ti.Clear()
	assert.Empty(t, ti.Value)
	assert.Equal(t, 0, ti.Cursor)
}

func TestCovTextInputString(t *testing.T) {
	ti := TextInput{Value: "hello"}
	assert.Equal(t, "hello", ti.String())
}

func TestCovTextInputCursorLeft(t *testing.T) {
	ti := TextInput{Value: "hello", Cursor: 3}
	assert.Equal(t, "hel", ti.CursorLeft())
}

func TestCovTextInputCursorRight(t *testing.T) {
	ti := TextInput{Value: "hello", Cursor: 3}
	assert.Equal(t, "lo", ti.CursorRight())
}

// =====================================================================
// filter_input.go: handleFilterKey + stringFilterInput
// =====================================================================

func TestCovHandleFilterKeyAllActions(t *testing.T) {
	tests := []struct {
		name   string
		key    string
		expect filterAction
	}{
		{"escape", "esc", filterEscape},
		{"enter", "enter", filterAccept},
		{"ctrl+c", "ctrl+c", filterClose},
		{"backspace", "backspace", filterContinue},
		{"ctrl+w", "ctrl+w", filterContinue},
		{"ctrl+a home", "ctrl+a", filterNavigate},
		{"ctrl+e end", "ctrl+e", filterNavigate},
		{"left", "left", filterNavigate},
		{"right", "right", filterNavigate},
		{"printable char", "a", filterContinue},
		{"non-printable", "f1", filterIgnored},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ti := &TextInput{Value: "hello", Cursor: 3}
			result := handleFilterKey(ti, tt.key)
			assert.Equal(t, tt.expect, result)
		})
	}
}

func TestCovStringFilterInput(t *testing.T) {
	s := "hello"
	fi := &stringFilterInput{ptr: &s}

	fi.Insert("!")
	assert.Equal(t, "hello!", s)

	fi.Backspace()
	assert.Equal(t, "hello", s)

	fi.DeleteWord()
	assert.Equal(t, "", s)

	// DeleteWord on empty: no-op.
	fi.DeleteWord()
	assert.Equal(t, "", s)

	// Backspace on empty: no-op.
	fi.Backspace()
	assert.Equal(t, "", s)

	// Clear.
	s = "test"
	fi.Clear()
	assert.Equal(t, "", s)

	// No-op cursor methods.
	fi.Home()
	fi.End()
	fi.Left()
	fi.Right()
}

func TestCovStringFilterInputDeleteWordWithSpaces(t *testing.T) {
	s := "hello world  "
	fi := &stringFilterInput{ptr: &s}
	fi.DeleteWord()
	assert.Equal(t, "hello ", s)
}

// =====================================================================
// overlay_nav.go: clampOverlayCursor
// =====================================================================

func TestCovClampOverlayCursorExhaustive(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		delta    int
		maxIdx   int
		expected int
	}{
		{"move down", 0, 1, 5, 1},
		{"move up", 3, -1, 5, 2},
		{"clamp at max", 5, 1, 5, 5},
		{"clamp at zero", 0, -1, 5, 0},
		{"empty list", 0, 1, -1, 0},
		{"big jump", 0, 100, 10, 10},
		{"big negative", 5, -100, 10, 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, clampOverlayCursor(tt.cursor, tt.delta, tt.maxIdx))
		})
	}
}
