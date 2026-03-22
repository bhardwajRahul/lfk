package app

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
)

// --- keyToBytes ---

func TestKeyToBytes(t *testing.T) {
	tests := []struct {
		name     string
		msg      tea.KeyMsg
		expected []byte
	}{
		{
			name:     "runes",
			msg:      tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}},
			expected: []byte("a"),
		},
		{
			name:     "enter",
			msg:      tea.KeyMsg{Type: tea.KeyEnter},
			expected: []byte{'\r'},
		},
		{
			name:     "tab",
			msg:      tea.KeyMsg{Type: tea.KeyTab},
			expected: []byte{'\t'},
		},
		{
			name:     "backspace",
			msg:      tea.KeyMsg{Type: tea.KeyBackspace},
			expected: []byte{'\x7f'},
		},
		{
			name:     "delete",
			msg:      tea.KeyMsg{Type: tea.KeyDelete},
			expected: []byte{'\x1b', '[', '3', '~'},
		},
		{
			name:     "space",
			msg:      tea.KeyMsg{Type: tea.KeySpace},
			expected: []byte{' '},
		},
		{
			name:     "escape",
			msg:      tea.KeyMsg{Type: tea.KeyEscape},
			expected: []byte{'\x1b'},
		},
		{
			name:     "up arrow",
			msg:      tea.KeyMsg{Type: tea.KeyUp},
			expected: []byte{'\x1b', '[', 'A'},
		},
		{
			name:     "down arrow",
			msg:      tea.KeyMsg{Type: tea.KeyDown},
			expected: []byte{'\x1b', '[', 'B'},
		},
		{
			name:     "right arrow",
			msg:      tea.KeyMsg{Type: tea.KeyRight},
			expected: []byte{'\x1b', '[', 'C'},
		},
		{
			name:     "left arrow",
			msg:      tea.KeyMsg{Type: tea.KeyLeft},
			expected: []byte{'\x1b', '[', 'D'},
		},
		{
			name:     "home",
			msg:      tea.KeyMsg{Type: tea.KeyHome},
			expected: []byte{'\x1b', '[', 'H'},
		},
		{
			name:     "end",
			msg:      tea.KeyMsg{Type: tea.KeyEnd},
			expected: []byte{'\x1b', '[', 'F'},
		},
		{
			name:     "page up",
			msg:      tea.KeyMsg{Type: tea.KeyPgUp},
			expected: []byte{'\x1b', '[', '5', '~'},
		},
		{
			name:     "page down",
			msg:      tea.KeyMsg{Type: tea.KeyPgDown},
			expected: []byte{'\x1b', '[', '6', '~'},
		},
		{
			name:     "ctrl+c",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlC},
			expected: []byte{'\x03'},
		},
		{
			name:     "ctrl+d",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlD},
			expected: []byte{'\x04'},
		},
		{
			name:     "ctrl+z",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlZ},
			expected: []byte{'\x1a'},
		},
		{
			name:     "ctrl+a",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlA},
			expected: []byte{'\x01'},
		},
		{
			name:     "ctrl+l",
			msg:      tea.KeyMsg{Type: tea.KeyCtrlL},
			expected: []byte{'\x0c'},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := keyToBytes(tt.msg)
			assert.Equal(t, tt.expected, result)
		})
	}
}
