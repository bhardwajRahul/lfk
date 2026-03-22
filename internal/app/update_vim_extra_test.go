package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- nextWordStart edge cases ---

func TestNextWordStartUnicode(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "unicode chars",
			line:     "hello wörld",
			col:      0,
			expected: 6,
		},
		{
			name:     "consecutive punctuation",
			line:     "a...b",
			col:      0,
			expected: 4,
		},
		{
			name:     "starts with punctuation",
			line:     ".hello",
			col:      0,
			expected: 1,
		},
		{
			name:     "trailing whitespace",
			line:     "hello   ",
			col:      0,
			expected: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, nextWordStart(tt.line, tt.col))
		})
	}
}

// --- wordEnd edge cases ---

func TestWordEndUnicode(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "unicode chars",
			line:     "wörld bär",
			col:      0,
			expected: 4,
		},
		{
			name:     "starting at boundary",
			line:     " hello",
			col:      0,
			expected: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, wordEnd(tt.line, tt.col))
		})
	}
}

// --- prevWordStart edge cases ---

func TestPrevWordStartUnicode(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "multiple separators",
			line:     "hello...world",
			col:      8,
			expected: 0,
		},
		{
			name:     "col equals 1",
			line:     "ab",
			col:      1,
			expected: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, prevWordStart(tt.line, tt.col))
		})
	}
}

// --- nextWORDStart edge cases ---

func TestNextWORDStartEdge(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "WORD with dots",
			line:     "foo.bar.baz qux",
			col:      0,
			expected: 12,
		},
		{
			name:     "only whitespace after",
			line:     "hello   ",
			col:      0,
			expected: 8,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, nextWORDStart(tt.line, tt.col))
		})
	}
}

// --- prevWORDStart edge cases ---

func TestPrevWORDStartEdge(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "at position 1",
			line:     "ab",
			col:      1,
			expected: 0,
		},
		{
			name:     "WORD with punctuation",
			line:     "foo.bar baz",
			col:      8,
			expected: 0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, prevWORDStart(tt.line, tt.col))
		})
	}
}

// --- WORDEnd edge cases ---

func TestWORDEndEdge(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "only whitespace after position",
			line:     "a   ",
			col:      0,
			expected: 4,
		},
		{
			name:     "single char at end",
			line:     "hello a",
			col:      5,
			expected: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, WORDEnd(tt.line, tt.col))
		})
	}
}
