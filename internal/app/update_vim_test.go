package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- isWordBoundary ---

func TestIsWordBoundary(t *testing.T) {
	tests := []struct {
		r        rune
		expected bool
	}{
		{' ', true},
		{'\t', true},
		{'.', true},
		{':', true},
		{',', true},
		{';', true},
		{'/', true},
		{'-', true},
		{'_', true},
		{'"', true},
		{'\'', true},
		{'(', true},
		{')', true},
		{'[', true},
		{']', true},
		{'{', true},
		{'}', true},
		{'a', false},
		{'Z', false},
		{'0', false},
		{'@', false},
	}
	for _, tt := range tests {
		t.Run(string(tt.r), func(t *testing.T) {
			assert.Equal(t, tt.expected, isWordBoundary(tt.r))
		})
	}
}

// --- nextWordStart ---

func TestNextWordStart(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "simple words",
			line:     "hello world",
			col:      0,
			expected: 6,
		},
		{
			name:     "at space",
			line:     "hello world",
			col:      5,
			expected: 6,
		},
		{
			name:     "with punctuation",
			line:     "foo.bar baz",
			col:      0,
			expected: 4,
		},
		{
			name:     "already at last char",
			line:     "hello",
			col:      4,
			expected: 5,
		},
		{
			name:     "past end",
			line:     "hello",
			col:      10,
			expected: 5,
		},
		{
			name:     "empty line",
			line:     "",
			col:      0,
			expected: 0,
		},
		{
			name:     "single char",
			line:     "a",
			col:      0,
			expected: 1,
		},
		{
			name:     "multiple separators",
			line:     "a   b",
			col:      0,
			expected: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, nextWordStart(tt.line, tt.col))
		})
	}
}

// --- wordEnd ---

func TestWordEnd(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "end of first word",
			line:     "hello world",
			col:      0,
			expected: 4,
		},
		{
			name:     "at word end moves to next word end",
			line:     "hello world",
			col:      4,
			expected: 10,
		},
		{
			name:     "single character word",
			line:     "a b",
			col:      0,
			expected: 2,
		},
		{
			name:     "empty line",
			line:     "",
			col:      0,
			expected: 0,
		},
		{
			name:     "at last char",
			line:     "hello",
			col:      4,
			expected: 5,
		},
		{
			name:     "past end",
			line:     "hello",
			col:      10,
			expected: 5,
		},
		{
			name:     "with punctuation",
			line:     "foo.bar",
			col:      0,
			expected: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, wordEnd(tt.line, tt.col))
		})
	}
}

// --- prevWordStart ---

func TestPrevWordStart(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "back to start of second word",
			line:     "hello world",
			col:      10,
			expected: 6,
		},
		{
			name:     "at start of second word back to first",
			line:     "hello world",
			col:      6,
			expected: 0,
		},
		{
			name:     "at start returns -1",
			line:     "hello",
			col:      0,
			expected: -1,
		},
		{
			name:     "empty line",
			line:     "",
			col:      0,
			expected: -1,
		},
		{
			name:     "with punctuation",
			line:     "foo.bar",
			col:      4,
			expected: 0,
		},
		{
			name:     "past end",
			line:     "hello world",
			col:      50,
			expected: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, prevWordStart(tt.line, tt.col))
		})
	}
}

// --- nextWORDStart ---

func TestNextWORDStart(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "simple words",
			line:     "hello world",
			col:      0,
			expected: 6,
		},
		{
			name:     "ignores punctuation within WORD",
			line:     "foo.bar baz",
			col:      0,
			expected: 8,
		},
		{
			name:     "multiple spaces",
			line:     "a    b",
			col:      0,
			expected: 5,
		},
		{
			name:     "at last char",
			line:     "hello",
			col:      4,
			expected: 5,
		},
		{
			name:     "empty line",
			line:     "",
			col:      0,
			expected: 0,
		},
		{
			name:     "tab separator",
			line:     "foo\tbar",
			col:      0,
			expected: 4,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, nextWORDStart(tt.line, tt.col))
		})
	}
}

// --- prevWORDStart ---

func TestPrevWORDStart(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "back from end",
			line:     "hello world",
			col:      10,
			expected: 6,
		},
		{
			name:     "back with punctuation",
			line:     "foo.bar baz",
			col:      10,
			expected: 8,
		},
		{
			name:     "at start",
			line:     "hello",
			col:      0,
			expected: -1,
		},
		{
			name:     "empty",
			line:     "",
			col:      0,
			expected: -1,
		},
		{
			name:     "past end clamped",
			line:     "hello world",
			col:      50,
			expected: 6,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, prevWORDStart(tt.line, tt.col))
		})
	}
}

// --- WORDEnd ---

func TestWORDEnd(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		col      int
		expected int
	}{
		{
			name:     "end of first WORD",
			line:     "hello world",
			col:      0,
			expected: 4,
		},
		{
			name:     "WORD with punctuation",
			line:     "foo.bar baz",
			col:      0,
			expected: 6,
		},
		{
			name:     "at WORD end moves to next",
			line:     "hello world",
			col:      4,
			expected: 10,
		},
		{
			name:     "at last char",
			line:     "hello",
			col:      4,
			expected: 5,
		},
		{
			name:     "empty",
			line:     "",
			col:      0,
			expected: 0,
		},
		{
			name:     "tab delimiter",
			line:     "foo\tbar",
			col:      0,
			expected: 2,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, WORDEnd(tt.line, tt.col))
		})
	}
}

// --- firstNonWhitespace ---

func TestFirstNonWhitespace(t *testing.T) {
	tests := []struct {
		name     string
		line     string
		expected int
	}{
		{"no leading whitespace", "hello", 0},
		{"leading spaces", "   hello", 3},
		{"leading tabs", "\t\thello", 2},
		{"mixed leading whitespace", "  \thello", 3},
		{"all whitespace", "     ", 0},
		{"empty string", "", 0},
		{"single char", "a", 0},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, firstNonWhitespace(tt.line))
		})
	}
}

// --- countLines ---

func TestCountLines(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected int
	}{
		{"empty string", "", 0},
		{"no newlines", "hello", 0},
		{"one newline", "hello\nworld", 1},
		{"multiple newlines", "a\nb\nc\n", 3},
		{"only newlines", "\n\n\n", 3},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, countLines(tt.input))
		})
	}
}
