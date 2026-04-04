package ui

import (
	"fmt"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// buildTestSuggestions creates n Suggestion values with sequential text labels.
func buildTestSuggestions(n int, category string) []Suggestion {
	suggestions := make([]Suggestion, n)
	for i := range n {
		suggestions[i] = Suggestion{Text: fmt.Sprintf("item%d", i), Category: category}
	}

	return suggestions
}

func TestRenderCommandDropdown_Empty(t *testing.T) {
	result := RenderCommandDropdown(nil, 0, 10, 60)
	assert.Empty(t, result)
}

func TestRenderCommandDropdown_SingleItem(t *testing.T) {
	suggestions := []Suggestion{{Text: "pods", Category: "resource"}}
	result := RenderCommandDropdown(suggestions, 0, 10, 60)
	assert.Contains(t, result, "pods")
	assert.Contains(t, result, "resource")
}

func TestRenderCommandDropdown_MultipleItems(t *testing.T) {
	suggestions := []Suggestion{
		{Text: "pods", Category: "resource"},
		{Text: "deployments", Category: "resource"},
		{Text: "services", Category: "resource"},
	}
	result := RenderCommandDropdown(suggestions, 1, 10, 60)
	assert.Contains(t, result, "pods")
	assert.Contains(t, result, "deployments")
	assert.Contains(t, result, "services")
}

func TestRenderCommandDropdown_MaxHeightClamp(t *testing.T) {
	suggestions := buildTestSuggestions(20, "test")
	result := RenderCommandDropdown(suggestions, 0, 5, 60)
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	assert.LessOrEqual(t, len(lines), 5)
}

func TestRenderCommandDropdown_ScrollKeepsSelectedVisible(t *testing.T) {
	suggestions := buildTestSuggestions(20, "test")
	// Selected item 15 should still be visible in a max-5 dropdown.
	result := RenderCommandDropdown(suggestions, 15, 5, 60)
	assert.Contains(t, result, "item15")
}

func TestRenderCommandDropdown_SelectedAtZero(t *testing.T) {
	suggestions := []Suggestion{
		{Text: "pods", Category: "resource"},
		{Text: "deployments", Category: "resource"},
	}
	result := RenderCommandDropdown(suggestions, 0, 10, 60)
	// The selected item should still be present.
	assert.Contains(t, result, "pods")
}

func TestRenderCommandDropdown_SelectedOutOfBounds(t *testing.T) {
	suggestions := []Suggestion{
		{Text: "pods", Category: "resource"},
	}
	// selected index beyond range should be clamped, not panic.
	result := RenderCommandDropdown(suggestions, 5, 10, 60)
	assert.Contains(t, result, "pods")
}

func TestRenderCommandDropdown_NegativeSelected(t *testing.T) {
	suggestions := []Suggestion{
		{Text: "pods", Category: "resource"},
	}
	// negative selected should be clamped to 0.
	result := RenderCommandDropdown(suggestions, -1, 10, 60)
	assert.Contains(t, result, "pods")
}

func TestRenderCommandDropdown_WidthPadding(t *testing.T) {
	suggestions := []Suggestion{{Text: "x", Category: "c"}}
	result := RenderCommandDropdown(suggestions, 0, 10, 40)
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	// Each rendered line should exist (non-empty).
	assert.NotEmpty(t, lines)
	assert.Greater(t, len(lines[0]), 0)
}

func TestRenderCommandDropdown_ScrollMiddleSelected(t *testing.T) {
	suggestions := buildTestSuggestions(20, "test")
	// Selected item 10 with maxHeight 5 should center item10 in the viewport.
	result := RenderCommandDropdown(suggestions, 10, 5, 60)
	assert.Contains(t, result, "item10")
	lines := strings.Split(strings.TrimRight(result, "\n"), "\n")
	assert.LessOrEqual(t, len(lines), 5)
}
