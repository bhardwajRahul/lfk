package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- wrapText ---

func TestWrapText(t *testing.T) {
	tests := []struct {
		name     string
		text     string
		width    int
		expected []string
	}{
		{
			name:     "zero width returns whole text",
			text:     "hello world",
			width:    0,
			expected: []string{"hello world"},
		},
		{
			name:     "negative width returns whole text",
			text:     "hello world",
			width:    -1,
			expected: []string{"hello world"},
		},
		{
			name:     "empty text returns nil",
			text:     "",
			width:    10,
			expected: nil,
		},
		{
			name:     "whitespace only returns nil",
			text:     "   \t  ",
			width:    10,
			expected: nil,
		},
		{
			name:     "single word fits",
			text:     "hello",
			width:    10,
			expected: []string{"hello"},
		},
		{
			name:     "words fit on one line",
			text:     "hello world",
			width:    20,
			expected: []string{"hello world"},
		},
		{
			name:     "words wrap to two lines",
			text:     "hello world foo",
			width:    11,
			expected: []string{"hello world", "foo"},
		},
		{
			name:     "each word on own line",
			text:     "a bb ccc",
			width:    3,
			expected: []string{"a", "bb", "ccc"},
		},
		{
			name:     "word longer than width stays on its own line",
			text:     "a verylongword b",
			width:    5,
			expected: []string{"a", "verylongword", "b"},
		},
		{
			name:     "multiple spaces collapsed",
			text:     "hello   world   foo",
			width:    20,
			expected: []string{"hello world foo"},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, wrapText(tt.text, tt.width))
		})
	}
}

// --- IsDrillableType ---

func TestIsDrillableType(t *testing.T) {
	tests := []struct {
		name     string
		typ      string
		expected bool
	}{
		// Empty type.
		{"empty string", "", false},

		// Object types.
		{"Object", "<Object>", true},
		{"ObjectMeta", "<ObjectMeta>", true},
		{"object lowercase", "<object>", true},

		// Array types.
		{"array of objects", "<[]Container>", true},
		{"array of strings", "<[]string>", true},
		{"empty array", "<[]>", true},

		// Map types.
		{"map string string", "<map[string]string>", true},
		{"map string int", "<map[string]int>", true},

		// Capitalized types (likely objects).
		{"PodSpec", "<PodSpec>", true},
		{"Container", "<Container>", true},
		{"ServicePort", "<ServicePort>", true},

		// Known primitives return false.
		{"string primitive", "<string>", false},
		{"integer primitive", "<integer>", false},
		{"boolean primitive", "<boolean>", false},
		{"number primitive", "<number>", false},
		{"int32 primitive", "<int32>", false},
		{"int64 primitive", "<int64>", false},
		{"Time primitive", "<Time>", false},
		{"Duration primitive", "<Duration>", false},
		{"Quantity primitive", "<Quantity>", false},

		// Lowercase non-special types.
		{"lowercase custom", "<custom>", false},
		{"lowercase field", "<status>", false},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, IsDrillableType(tt.typ))
		})
	}
}
