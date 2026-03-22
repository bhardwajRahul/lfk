package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/ui"
)

// --- filteredSchemeNames ---

func TestFilteredSchemeNames(t *testing.T) {
	entries := []ui.SchemeEntry{
		{Name: "Dark Themes", IsHeader: true},
		{Name: "catppuccin-mocha"},
		{Name: "dracula"},
		{Name: "gruvbox-dark"},
		{Name: "Light Themes", IsHeader: true},
		{Name: "catppuccin-latte"},
		{Name: "gruvbox-light"},
	}

	t.Run("no filter returns all non-header entries", func(t *testing.T) {
		m := Model{schemeEntries: entries}
		result := m.filteredSchemeNames()
		assert.Len(t, result, 5)
		assert.NotContains(t, result, "Dark Themes")
		assert.NotContains(t, result, "Light Themes")
	})

	t.Run("filter by prefix", func(t *testing.T) {
		m := Model{
			schemeEntries: entries,
			schemeFilter:  TextInput{Value: "catppuccin"},
		}
		result := m.filteredSchemeNames()
		assert.Len(t, result, 2)
		assert.Contains(t, result, "catppuccin-mocha")
		assert.Contains(t, result, "catppuccin-latte")
	})

	t.Run("filter by substring", func(t *testing.T) {
		m := Model{
			schemeEntries: entries,
			schemeFilter:  TextInput{Value: "gruvbox"},
		}
		result := m.filteredSchemeNames()
		assert.Len(t, result, 2)
	})

	t.Run("no match returns empty", func(t *testing.T) {
		m := Model{
			schemeEntries: entries,
			schemeFilter:  TextInput{Value: "nonexistent"},
		}
		result := m.filteredSchemeNames()
		assert.Empty(t, result)
	})

	t.Run("empty entries returns empty", func(t *testing.T) {
		m := Model{
			schemeFilter: TextInput{Value: "test"},
		}
		result := m.filteredSchemeNames()
		assert.Empty(t, result)
	})
}
