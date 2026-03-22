package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- highlightSearchMatches ---

func TestHighlightSearchMatches(t *testing.T) {
	tests := []struct {
		name       string
		lines      []string
		query      string
		wantSubstr [][]string // per-line expected substrings
		wantMatch  []bool     // whether each line should be modified
	}{
		{
			name:      "no match leaves lines unchanged",
			lines:     []string{"hello world", "foo bar"},
			query:     "xyz",
			wantMatch: []bool{false, false},
		},
		{
			name:       "case-insensitive match highlights query",
			lines:      []string{"Hello World", "nothing here"},
			query:      "hello",
			wantSubstr: [][]string{{"Hello"}, nil},
			wantMatch:  []bool{true, false},
		},
		{
			name:       "multiple matches in one line",
			lines:      []string{"foo bar foo baz"},
			query:      "foo",
			wantSubstr: [][]string{{"foo"}},
			wantMatch:  []bool{true},
		},
		{
			name:       "match in all lines",
			lines:      []string{"error here", "error there"},
			query:      "error",
			wantSubstr: [][]string{{"error"}, {"error"}},
			wantMatch:  []bool{true, true},
		},
		{
			name:      "empty lines are unchanged",
			lines:     []string{"", "test"},
			query:     "test",
			wantMatch: []bool{false, true},
		},
		{
			name:       "preserves text around match",
			lines:      []string{"prefix ERROR suffix"},
			query:      "error",
			wantSubstr: [][]string{{"prefix", "ERROR", "suffix"}},
			wantMatch:  []bool{true},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := highlightSearchMatches(tt.lines, tt.query)
			assert.Equal(t, len(tt.lines), len(result), "result length should match input length")
			for i, matched := range tt.wantMatch {
				if !matched {
					assert.Equal(t, tt.lines[i], result[i], "unmatched line %d should be unchanged", i)
				} else {
					// The result should contain the original text plus highlighting.
					if tt.wantSubstr != nil && tt.wantSubstr[i] != nil {
						for _, sub := range tt.wantSubstr[i] {
							assert.Contains(t, result[i], sub, "line %d should contain %q", i, sub)
						}
					}
				}
			}
		})
	}
}

// --- renderPlainLines ---

func TestRenderPlainLines(t *testing.T) {
	tests := []struct {
		name         string
		lines        []string
		scroll       int
		height       int
		width        int
		lineNumbers  bool
		lineNumWidth int
		cursor       int
		wantCount    int
		wantSubstr   []string
	}{
		{
			name:        "basic rendering",
			lines:       []string{"line 1", "line 2", "line 3"},
			scroll:      0,
			height:      3,
			width:       80,
			lineNumbers: false,
			cursor:      -1,
			wantCount:   3,
			wantSubstr:  []string{"line 1", "line 2", "line 3"},
		},
		{
			name:        "scroll skips initial lines",
			lines:       []string{"line 1", "line 2", "line 3", "line 4"},
			scroll:      2,
			height:      2,
			width:       80,
			lineNumbers: false,
			cursor:      -1,
			wantCount:   2,
			wantSubstr:  []string{"line 3", "line 4"},
		},
		{
			name:        "height limits output",
			lines:       []string{"a", "b", "c", "d", "e"},
			scroll:      0,
			height:      3,
			width:       80,
			lineNumbers: false,
			cursor:      -1,
			wantCount:   3,
		},
		{
			name:         "line numbers shown",
			lines:        []string{"line 1", "line 2"},
			scroll:       0,
			height:       2,
			width:        80,
			lineNumbers:  true,
			lineNumWidth: 3,
			cursor:       -1,
			wantCount:    2,
			wantSubstr:   []string{"1", "2"},
		},
		{
			name:        "cursor line gets indicator",
			lines:       []string{"line 1", "line 2", "line 3"},
			scroll:      0,
			height:      3,
			width:       80,
			lineNumbers: false,
			cursor:      1,
			wantCount:   3,
		},
		{
			name:        "empty lines list",
			lines:       []string{},
			scroll:      0,
			height:      5,
			width:       80,
			lineNumbers: false,
			cursor:      -1,
			wantCount:   0,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderPlainLines(tt.lines, tt.scroll, tt.height, tt.width,
				tt.lineNumbers, tt.lineNumWidth, tt.cursor, -1, -1, -1, 0, 0, 0)
			assert.Equal(t, tt.wantCount, len(result), "rendered line count")
			for _, sub := range tt.wantSubstr {
				found := false
				for _, line := range result {
					if strings.Contains(line, sub) {
						found = true
						break
					}
				}
				assert.True(t, found, "rendered output should contain %q", sub)
			}
		})
	}

	t.Run("cursor line has cursor indicator glyph", func(t *testing.T) {
		lines := []string{"hello", "world"}
		result := renderPlainLines(lines, 0, 2, 80, false, 0, 0, -1, -1, -1, 0, 0, 0)
		// Cursor line should contain the bar cursor indicator.
		assert.Contains(t, result[0], "\u258e", "cursor line should have indicator glyph")
		// Non-cursor line should start with a space.
		assert.True(t, strings.HasPrefix(result[1], " "), "non-cursor line should start with space")
	})

	t.Run("scroll beyond end produces empty result", func(t *testing.T) {
		lines := []string{"a", "b"}
		result := renderPlainLines(lines, 5, 3, 80, false, 0, -1, -1, -1, -1, 0, 0, 0)
		assert.Empty(t, result)
	})
}
