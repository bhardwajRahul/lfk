package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// The column toggle overlay's filter bar must follow the namespace
// overlay's layout: anchored right under the title (not at the
// bottom), and always present (placeholder when inactive). Without
// this, the filter row appears/disappears between renders ("disappears
// randomly") and adds an unaccounted row that overflows the overlay
// height ("resizes the window").

// firstLineContaining returns the index of the first line that contains
// substr, or -1 if none. Strips ANSI before matching.
func firstLineContaining(out, substr string) int {
	lines := strings.Split(stripANSI(out), "\n")
	for i, line := range lines {
		if strings.Contains(line, substr) {
			return i
		}
	}
	return -1
}

func TestColumnToggleOverlay_FilterBarAnchoredBelowTitle(t *testing.T) {
	entries := []ColumnToggleEntry{
		{Key: "Namespace", Visible: true},
		{Key: "Ready", Visible: true},
		{Key: "Status", Visible: true},
	}
	out := RenderColumnToggleOverlay(entries, 0, "", false, 50, 20)

	filterLine := firstLineContaining(out, "filter")
	itemLine := firstLineContaining(out, "Namespace")
	if !assert.GreaterOrEqual(t, filterLine, 0, "filter bar must render somewhere") {
		return
	}
	if !assert.GreaterOrEqual(t, itemLine, 0, "items must render somewhere") {
		return
	}
	// Filter bar must come BEFORE the items (anchored under title), not
	// after them ("appears at the bottom" was the bug).
	assert.Less(t, filterLine, itemLine,
		"filter bar must precede the items list, not sit at the bottom")
}

func TestColumnToggleOverlay_FilterBarAlwaysPresent(t *testing.T) {
	entries := []ColumnToggleEntry{{Key: "IP", Visible: true}}

	// No filter, not active — placeholder should still render so the
	// row count is stable across renders.
	out := RenderColumnToggleOverlay(entries, 0, "", false, 50, 20)
	assert.GreaterOrEqual(t, firstLineContaining(out, "filter"), 0,
		"filter row must show a placeholder when inactive so it never disappears")
}

func TestColumnToggleOverlay_FilterActiveShowsCursor(t *testing.T) {
	entries := []ColumnToggleEntry{{Key: "IP", Visible: true}}
	out := RenderColumnToggleOverlay(entries, 0, "ip", true, 50, 20)
	plain := stripANSI(out)
	assert.Contains(t, plain, "ip",
		"active filter must render the typed text")
}
