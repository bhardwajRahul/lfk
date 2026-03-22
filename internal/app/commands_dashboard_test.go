package app

import (
	"strings"
	"testing"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// stripANSI removes ANSI escape codes to allow plain-text assertions on
// rendered output. This covers the basic CSI sequences emitted by lipgloss.
func stripANSI(s string) string {
	var out strings.Builder
	i := 0
	for i < len(s) {
		if s[i] == '\x1b' {
			// Skip CSI sequence: ESC [ ... final byte.
			j := i + 1
			if j < len(s) && s[j] == '[' {
				j++
				for j < len(s) && s[j] >= 0x20 && s[j] <= 0x3F {
					j++
				}
				if j < len(s) {
					j++ // skip final byte
				}
			}
			i = j
			continue
		}
		out.WriteByte(s[i])
		i++
	}
	return out.String()
}

// --- renderBar ---

func TestRenderBar(t *testing.T) {
	tests := []struct {
		name         string
		used         int64
		total        int64
		width        int
		wantContains string
	}{
		{
			name:         "zero total shows N/A",
			used:         100,
			total:        0,
			width:        20,
			wantContains: "N/A",
		},
		{
			name:         "negative total shows N/A",
			used:         50,
			total:        -10,
			width:        20,
			wantContains: "N/A",
		},
		{
			name:         "0 percent usage",
			used:         0,
			total:        100,
			width:        20,
			wantContains: "0%",
		},
		{
			name:         "50 percent usage",
			used:         50,
			total:        100,
			width:        20,
			wantContains: "50%",
		},
		{
			name:         "100 percent usage",
			used:         100,
			total:        100,
			width:        20,
			wantContains: "100%",
		},
		{
			name:         "over 100 percent capped",
			used:         150,
			total:        100,
			width:        20,
			wantContains: "100%",
		},
		{
			name:         "75 percent boundary",
			used:         75,
			total:        100,
			width:        20,
			wantContains: "75%",
		},
		{
			name:         "90 percent boundary",
			used:         90,
			total:        100,
			width:        20,
			wantContains: "90%",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := renderBar(tt.used, tt.total, tt.width)
			stripped := stripANSI(result)
			assert.Contains(t, stripped, tt.wantContains)
		})
	}
}

func TestRenderBarStructure(t *testing.T) {
	result := renderBar(50, 100, 20)
	stripped := stripANSI(result)

	assert.True(t, strings.HasPrefix(stripped, "["), "bar should start with [")
	assert.Contains(t, stripped, "]", "bar should contain ]")
}

func TestRenderBarWidthZero(t *testing.T) {
	// Width 0 should not panic.
	result := renderBar(50, 100, 0)
	stripped := stripANSI(result)
	assert.Contains(t, stripped, "[")
	assert.Contains(t, stripped, "]")
}

func TestRenderBarFilledChars(t *testing.T) {
	result := renderBar(100, 100, 10)
	stripped := stripANSI(result)

	// Extract content between brackets.
	openIdx := strings.Index(stripped, "[")
	closeIdx := strings.Index(stripped, "]")
	inner := stripped[openIdx+1 : closeIdx]
	filledCount := strings.Count(inner, "\u2588")
	assert.Equal(t, 10, filledCount, "100%% usage should fill entire bar width")
}

func TestRenderBarEmptyChars(t *testing.T) {
	result := renderBar(0, 100, 10)
	stripped := stripANSI(result)

	openIdx := strings.Index(stripped, "[")
	closeIdx := strings.Index(stripped, "]")
	inner := stripped[openIdx+1 : closeIdx]
	emptyCount := strings.Count(inner, "\u2591")
	assert.Equal(t, 10, emptyCount, "0%% usage should have all empty blocks")
}

// --- renderStackedBar ---

func TestRenderStackedBar(t *testing.T) {
	t.Run("zero total shows empty bar", func(t *testing.T) {
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{5, lipgloss.NewStyle()},
		}
		result := renderStackedBar(segments, 0, 20)
		stripped := stripANSI(result)

		assert.True(t, strings.HasPrefix(stripped, "["))
		assert.True(t, strings.HasSuffix(stripped, "]"))
		inner := stripped[1 : len(stripped)-1]
		assert.Equal(t, 20, strings.Count(inner, "\u2591"), "zero total should produce all empty blocks")
	})

	t.Run("negative total shows empty bar", func(t *testing.T) {
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{5, lipgloss.NewStyle()},
		}
		result := renderStackedBar(segments, -10, 20)
		stripped := stripANSI(result)
		inner := stripped[1 : len(stripped)-1]
		assert.Equal(t, 20, strings.Count(inner, "\u2591"))
	})

	t.Run("single segment fills bar", func(t *testing.T) {
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{10, lipgloss.NewStyle()},
		}
		result := renderStackedBar(segments, 10, 20)
		stripped := stripANSI(result)

		assert.True(t, strings.HasPrefix(stripped, "["))
		assert.True(t, strings.HasSuffix(stripped, "]"))
		inner := stripped[1 : len(stripped)-1]
		filledCount := strings.Count(inner, "\u2588")
		assert.Equal(t, 20, filledCount, "single segment at 100%% should fill entire bar")
	})

	t.Run("two segments split evenly", func(t *testing.T) {
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{5, lipgloss.NewStyle()},
			{5, lipgloss.NewStyle()},
		}
		result := renderStackedBar(segments, 10, 20)
		stripped := stripANSI(result)

		inner := stripped[1 : len(stripped)-1]
		filledCount := strings.Count(inner, "\u2588")
		assert.Equal(t, 20, filledCount)
	})

	t.Run("three segments with remainder", func(t *testing.T) {
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{3, lipgloss.NewStyle()},
			{3, lipgloss.NewStyle()},
			{4, lipgloss.NewStyle()},
		}
		result := renderStackedBar(segments, 10, 20)
		stripped := stripANSI(result)

		inner := stripped[1 : len(stripped)-1]
		filledCount := strings.Count(inner, "\u2588")
		assert.Equal(t, 20, filledCount, "all segments together should fill the bar")
	})

	t.Run("segments exceeding total triggers overflow guard", func(t *testing.T) {
		// When non-last segments produce more chars than the width, the
		// used+chars > width guard kicks in.
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{10, lipgloss.NewStyle()},
			{10, lipgloss.NewStyle()},
			{10, lipgloss.NewStyle()},
		}
		// total=10, width=5: each segment would want 5 chars, but only 5 total.
		result := renderStackedBar(segments, 10, 5)
		stripped := stripANSI(result)
		inner := stripped[1 : len(stripped)-1]
		totalChars := strings.Count(inner, "\u2588") + strings.Count(inner, "\u2591")
		assert.Equal(t, 5, totalChars, "total characters should not exceed width")
	})

	t.Run("last segment negative chars guard", func(t *testing.T) {
		// When the first segments already fill the bar, the last segment
		// gets chars = width - used which could be negative before the guard.
		// Here: segment1 gets int(15/15*5) = 5 chars (fills bar),
		// segment2 (last) gets chars = 5 - 5 = 0, which is non-negative.
		// To trigger chars < 0 on the last segment, we need used > width,
		// but that's prevented by the prior guard. So instead test a
		// scenario where segment proportions cause rounding overflow.
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{7, lipgloss.NewStyle()},
			{7, lipgloss.NewStyle()},
			{1, lipgloss.NewStyle()},
		}
		// total=15, width=10: seg0 = int(7/15*10) = 4, seg1 = int(7/15*10) = 4, used=8
		// seg2 (last) = width - used = 10 - 8 = 2. All is fine.
		// This ensures no panics with multiple segment rounding.
		result := renderStackedBar(segments, 15, 10)
		stripped := stripANSI(result)
		inner := stripped[1 : len(stripped)-1]
		filledCount := strings.Count(inner, "\u2588")
		assert.Equal(t, 10, filledCount, "rounding should not leave gaps")
	})

	t.Run("negative count in non-last segment", func(t *testing.T) {
		// A negative count produces negative chars which triggers the chars < 0 guard.
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{-5, lipgloss.NewStyle()},
			{10, lipgloss.NewStyle()},
		}
		result := renderStackedBar(segments, 10, 10)
		stripped := stripANSI(result)
		// Should not panic and should produce a valid bar.
		assert.True(t, strings.HasPrefix(stripped, "["))
		assert.True(t, strings.HasSuffix(stripped, "]"))
	})

	t.Run("empty segments array", func(t *testing.T) {
		var segments []struct {
			count int
			style lipgloss.Style
		}
		result := renderStackedBar(segments, 10, 20)
		stripped := stripANSI(result)

		inner := stripped[1 : len(stripped)-1]
		emptyCount := strings.Count(inner, "\u2591")
		assert.Equal(t, 20, emptyCount, "no segments should produce all empty blocks")
	})

	t.Run("width zero", func(t *testing.T) {
		segments := []struct {
			count int
			style lipgloss.Style
		}{
			{5, lipgloss.NewStyle()},
		}
		result := renderStackedBar(segments, 10, 0)
		stripped := stripANSI(result)
		assert.Equal(t, "[]", stripped)
	})
}

// --- formatTimeAgo ---

func TestFormatTimeAgo(t *testing.T) {
	tests := []struct {
		name     string
		offset   time.Duration
		contains string
	}{
		{
			name:     "seconds ago",
			offset:   30 * time.Second,
			contains: "s ago",
		},
		{
			name:     "minutes ago",
			offset:   5 * time.Minute,
			contains: "m ago",
		},
		{
			name:     "hours ago",
			offset:   3 * time.Hour,
			contains: "h ago",
		},
		{
			name:     "days ago",
			offset:   48 * time.Hour,
			contains: "d ago",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			past := time.Now().Add(-tt.offset)
			result := formatTimeAgo(past)
			assert.Contains(t, result, tt.contains)
			assert.NotEmpty(t, result)
		})
	}
}
