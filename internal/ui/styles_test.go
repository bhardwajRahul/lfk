package ui

import (
	"fmt"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

// --- AgeStyle ---

func TestAgeStyle(t *testing.T) {
	// Helper to extract a comparable foreground color key from a style.
	fgKey := func(s lipgloss.Style) string {
		fg := s.GetForeground()
		r, g, b, a := fg.RGBA()
		return fmt.Sprintf("%d:%d:%d:%d", r, g, b, a)
	}

	dimFg := fgKey(DimStyle)
	cyanFg := fgKey(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorCyan)))
	greenFg := fgKey(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorSecondary)))
	borderFg := fgKey(lipgloss.NewStyle().Foreground(lipgloss.Color(ColorBorder)))

	tests := []struct {
		name       string
		age        string
		expectedFg string
		desc       string
	}{
		// Empty returns DimStyle.
		{"empty string", "", dimFg, "dim"},

		// Seconds: very new -> cyan.
		{"5 seconds", "5s", cyanFg, "cyan"},
		{"30 seconds", "30s", cyanFg, "cyan"},

		// Minutes: very new -> cyan.
		{"1 minute", "1m", cyanFg, "cyan"},
		{"59 minutes", "59m", cyanFg, "cyan"},

		// Hours < 24: recent -> green.
		{"1 hour", "1h", greenFg, "green"},
		{"12 hours", "12h", greenFg, "green"},
		{"23 hours", "23h", greenFg, "green"},

		// Hours >= 24: dim.
		{"24 hours", "24h", dimFg, "dim"},
		{"48 hours", "48h", dimFg, "dim"},

		// Days <= 7: dim.
		{"1 day", "1d", dimFg, "dim"},
		{"7 days", "7d", dimFg, "dim"},

		// Days > 7: extra dim (border color).
		{"8 days", "8d", borderFg, "border"},
		{"30 days", "30d", borderFg, "border"},
		{"365 days", "365d", borderFg, "border"},

		// Years: old -> border.
		{"1 year", "1y", borderFg, "border"},

		// Parse error returns dim.
		{"invalid number", "xm", dimFg, "dim"},
		{"no number", "m", dimFg, "dim"},

		// Unknown unit returns dim.
		{"unknown unit", "5x", dimFg, "dim"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			style := AgeStyle(tt.age)
			got := fgKey(style)
			assert.Equal(t, tt.expectedFg, got, "age=%q expected %s style", tt.age, tt.desc)
		})
	}
}
