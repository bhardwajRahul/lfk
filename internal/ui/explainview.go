package ui

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/janosmiko/lfk/internal/model"
)

// RenderExplainView renders the API explain browser view with a two-pane layout:
// fields list on the left, description on the right.
func RenderExplainView(fields []model.ExplainField, cursor, scroll int, resourceDesc, title, path string, width, height int) string {
	// Title / breadcrumb.
	titleText := TitleStyle.Render("Explain: " + title)

	// Hint bar.
	hints := []struct{ key, desc string }{
		{"j/k", "navigate"},
		{"l/Enter", "drill in"},
		{"h/Backspace", "back"},
		{"g/G", "top/bottom"},
		{"ctrl+d/u", "half page"},
		{"q/esc", "close"},
	}
	hintParts := make([]string, 0, len(hints))
	for _, h := range hints {
		hintParts = append(hintParts, HelpKeyStyle.Render(h.key)+DimStyle.Render(": "+h.desc))
	}
	hint := StatusBarBgStyle.Width(width).Render(strings.Join(hintParts, DimStyle.Render(" | ")))

	// Calculate available content height: subtract title (1), hint (1), borders (2).
	contentHeight := height - 4
	if contentHeight < 5 {
		contentHeight = 5
	}

	// Split width: 45% fields, 55% description.
	innerWidth := width - 4 // account for outer border
	if innerWidth < 20 {
		innerWidth = 20
	}
	leftWidth := innerWidth * 45 / 100
	rightWidth := innerWidth - leftWidth - 3 // 3 for the separator
	if leftWidth < 15 {
		leftWidth = 15
	}
	if rightWidth < 15 {
		rightWidth = 15
	}

	// --- Left pane: field list ---
	fieldLines := renderFieldList(fields, cursor, scroll, leftWidth, contentHeight-2)

	// --- Right pane: description ---
	descLines := renderFieldDescription(fields, cursor, resourceDesc, rightWidth, contentHeight-2)

	// Build left pane with header.
	leftHeader := HeaderStyle.Render("Fields")
	leftContent := leftHeader + "\n" + strings.Join(fieldLines, "\n")
	leftPane := lipgloss.NewStyle().
		Width(leftWidth).
		Height(contentHeight).
		Render(leftContent)

	// Build right pane with header.
	rightHeader := HeaderStyle.Render("Description")
	rightContent := rightHeader + "\n" + strings.Join(descLines, "\n")
	rightPane := lipgloss.NewStyle().
		Width(rightWidth).
		Height(contentHeight).
		Render(rightContent)

	// Separator.
	sep := DimStyle.Render(strings.Repeat("\u2502\n", contentHeight))
	sep = lipgloss.NewStyle().Height(contentHeight).Render(
		strings.TrimRight(sep, "\n"),
	)

	body := lipgloss.JoinHorizontal(lipgloss.Top, leftPane, " "+sep+" ", rightPane)

	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ColorPrimary)).
		Padding(0, 1).
		Width(width - 2).
		Height(contentHeight)
	borderedBody := borderStyle.Render(body)

	return lipgloss.JoinVertical(lipgloss.Left, titleText, borderedBody, hint)
}

// renderFieldList renders the scrollable field list for the left pane.
func renderFieldList(fields []model.ExplainField, cursor, scroll, width, maxLines int) []string {
	if len(fields) == 0 {
		lines := make([]string, maxLines)
		lines[0] = DimStyle.Render("No fields found")
		for i := 1; i < maxLines; i++ {
			lines[i] = ""
		}
		return lines
	}

	// Clamp scroll.
	maxScroll := len(fields) - maxLines
	if maxScroll < 0 {
		maxScroll = 0
	}
	if scroll > maxScroll {
		scroll = maxScroll
	}
	if scroll < 0 {
		scroll = 0
	}

	lines := make([]string, 0, maxLines)
	end := scroll + maxLines
	if end > len(fields) {
		end = len(fields)
	}

	// Calculate the maximum name width for alignment.
	nameWidth := 0
	for _, f := range fields {
		if len(f.Name) > nameWidth {
			nameWidth = len(f.Name)
		}
	}
	if nameWidth > width/2 {
		nameWidth = width / 2
	}

	for i := scroll; i < end; i++ {
		f := fields[i]
		name := f.Name
		if len(name) > nameWidth {
			name = name[:nameWidth]
		}

		// Format: "> name     <type>" or "  name     <type>"
		prefix := "  "
		if i == cursor {
			prefix = "> "
		}

		typeStr := f.Type
		maxTypeLen := width - nameWidth - 5 // prefix + padding
		if maxTypeLen > 0 && len(typeStr) > maxTypeLen {
			typeStr = typeStr[:maxTypeLen]
		}

		line := fmt.Sprintf("%s%-*s  %s", prefix, nameWidth, name, typeStr)

		// Truncate to width.
		if len(line) > width {
			line = line[:width]
		}

		if i == cursor {
			lines = append(lines, OverlaySelectedStyle.Render(line))
		} else {
			namePart := NormalStyle.Render(fmt.Sprintf("%s%-*s", prefix, nameWidth, name))
			typePart := DimStyle.Render("  " + typeStr)
			lines = append(lines, namePart+typePart)
		}
	}

	// Pad remaining lines.
	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	return lines
}

// renderFieldDescription renders the description panel for the selected field.
func renderFieldDescription(fields []model.ExplainField, cursor int, resourceDesc string, width, maxLines int) []string {
	lines := make([]string, 0, maxLines)

	if len(fields) == 0 {
		// Show resource description when no fields.
		if resourceDesc != "" {
			wrapped := wrapText(resourceDesc, width)
			for _, line := range wrapped {
				lines = append(lines, NormalStyle.Render(line))
			}
		} else {
			lines = append(lines, DimStyle.Render("No description available"))
		}
		for len(lines) < maxLines {
			lines = append(lines, "")
		}
		return lines
	}

	if cursor < 0 || cursor >= len(fields) {
		for range maxLines {
			lines = append(lines, "")
		}
		return lines
	}

	f := fields[cursor]

	// Field name and type header.
	lines = append(lines, HeaderStyle.Render(f.Name))
	if f.Type != "" {
		lines = append(lines, DimStyle.Render("TYPE: "+f.Type))
	}
	lines = append(lines, "")

	// Field description.
	if f.Description != "" {
		wrapped := wrapText(f.Description, width)
		for _, w := range wrapped {
			lines = append(lines, NormalStyle.Render(w))
		}
	} else {
		lines = append(lines, DimStyle.Render("No description available"))
	}

	// If the field has an Object or array type, show drill-in hint.
	if IsDrillableType(f.Type) {
		lines = append(lines, "")
		lines = append(lines, HelpKeyStyle.Render("Press l or Enter to drill into this field"))
	}

	// Pad remaining lines.
	for len(lines) < maxLines {
		lines = append(lines, "")
	}

	// Truncate if too many lines.
	if len(lines) > maxLines {
		lines = lines[:maxLines]
	}

	return lines
}

// IsDrillableType returns true if the type indicates the field can be drilled into.
func IsDrillableType(typ string) bool {
	if typ == "" {
		return false
	}
	lower := strings.ToLower(typ)
	// Object types: <Object>, <ObjectMeta>, <PodSpec>, etc.
	// Array of objects: <[]Object>, <[]Container>, etc.
	// Map types: <map[string]string>, etc.
	if strings.Contains(lower, "object") {
		return true
	}
	if strings.Contains(lower, "[]") {
		return true
	}
	if strings.Contains(lower, "map[") {
		return true
	}
	// Types that are likely objects (capitalized and not primitive).
	inner := strings.Trim(typ, "<>[]")
	if len(inner) > 0 && inner[0] >= 'A' && inner[0] <= 'Z' {
		// Capitalized types are usually objects (e.g., <PodSpec>, <Container>).
		// Exclude known primitives.
		switch inner {
		case "string", "integer", "boolean", "number", "int32", "int64",
			"Time", "Duration", "Quantity":
			return false
		}
		return true
	}
	return false
}

// wrapText wraps a text string to the given width, breaking on word boundaries.
func wrapText(text string, width int) []string {
	if width <= 0 {
		return []string{text}
	}

	words := strings.Fields(text)
	if len(words) == 0 {
		return nil
	}

	var lines []string
	currentLine := words[0]

	for _, word := range words[1:] {
		if len(currentLine)+1+len(word) <= width {
			currentLine += " " + word
		} else {
			lines = append(lines, currentLine)
			currentLine = word
		}
	}
	if currentLine != "" {
		lines = append(lines, currentLine)
	}

	return lines
}
