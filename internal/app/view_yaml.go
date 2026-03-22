package app

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	"github.com/janosmiko/lfk/internal/model"
	"github.com/janosmiko/lfk/internal/ui"
)

func (m Model) viewYAML() string {
	yamlTitleText := m.yamlTitle()
	if m.yamlVisualMode {
		switch m.yamlVisualType {
		case 'v':
			yamlTitleText += " [VISUAL]"
		case 'B':
			yamlTitleText += " [VISUAL BLOCK]"
		default:
			yamlTitleText += " [VISUAL LINE]"
		}
	}
	title := ui.TitleStyle.Width(m.width).MaxWidth(m.width).MaxHeight(1).Render(yamlTitleText)
	var yamlHints []struct{ key, desc string }
	if m.yamlVisualMode {
		yamlHints = []struct{ key, desc string }{
			{"j/k", "extend selection"},
			{"y", "copy selected"},
			{"v/V/ctrl+v", "switch mode"},
			{"esc", "cancel"},
		}
	} else {
		yamlHints = []struct{ key, desc string }{
			{"j/k", "scroll"},
			{"g/G", "top/bottom"},
			{"ctrl+d/u", "half page"},
			{"ctrl+f/b", "page"},
			{"/", "search"},
			{"v/V/ctrl+v", "visual select"},
			{"tab/z", "fold"},
			{"ctrl+e", "edit"},
			{"q/esc", "back"},
		}
	}
	yamlHintParts := make([]string, 0, len(yamlHints))
	for _, h := range yamlHints {
		yamlHintParts = append(yamlHintParts, ui.HelpKeyStyle.Render(h.key)+ui.DimStyle.Render(": "+h.desc))
	}
	hint := ui.StatusBarBgStyle.Width(m.width).MaxWidth(m.width).MaxHeight(1).Render(strings.Join(yamlHintParts, ui.DimStyle.Render(" \u2502 ")))

	// If search is active, show search bar instead of hints.
	if m.yamlSearchMode {
		searchBar := ui.HelpKeyStyle.Render("/") + ui.NormalStyle.Render(m.yamlSearchText.CursorLeft()) + ui.DimStyle.Render("\u2588") + ui.NormalStyle.Render(m.yamlSearchText.CursorRight())
		hint = ui.StatusBarBgStyle.Width(m.width).MaxWidth(m.width).MaxHeight(1).Render(searchBar)
	} else if m.yamlSearchText.Value != "" {
		matchInfo := fmt.Sprintf(" [%d/%d]", m.yamlMatchIdx+1, len(m.yamlMatchLines))
		if len(m.yamlMatchLines) == 0 {
			matchInfo = " [no matches]"
		}
		searchBar := ui.HelpKeyStyle.Render("/") + ui.NormalStyle.Render(m.yamlSearchText.Value) + ui.DimStyle.Render(matchInfo)
		hint = ui.StatusBarBgStyle.Width(m.width).MaxWidth(m.width).MaxHeight(1).Render(searchBar)
	}

	maxLines := m.height - 4
	if maxLines < 3 {
		maxLines = 3
	}

	// Build visible lines with fold indicators, respecting collapsed sections.
	// Mask secret data values when secret display is toggled off.
	yamlForDisplay := m.maskYAMLIfSecret(m.yamlContent)
	visLines, mapping := buildVisibleLines(yamlForDisplay, m.yamlSections, m.yamlCollapsed)

	yamlScroll := m.yamlScroll
	if yamlScroll >= len(visLines) {
		yamlScroll = len(visLines) - 1
	}
	if yamlScroll < 0 {
		yamlScroll = 0
	}
	viewport := visLines[yamlScroll:]
	if len(viewport) > maxLines {
		viewport = viewport[:maxLines]
	}

	// Compute line number gutter width.
	totalOrigLines := len(strings.Split(m.yamlContent, "\n"))
	gutterWidth := len(fmt.Sprintf("%d", totalOrigLines))
	if gutterWidth < 2 {
		gutterWidth = 2
	}

	// Build a set of original matching lines for search highlight.
	matchSet := make(map[int]bool)
	for _, ml := range m.yamlMatchLines {
		matchSet[ml] = true
	}
	currentMatchLine := -1
	if len(m.yamlMatchLines) > 0 && m.yamlMatchIdx >= 0 && m.yamlMatchIdx < len(m.yamlMatchLines) {
		currentMatchLine = m.yamlMatchLines[m.yamlMatchIdx]
	}

	// Clamp yamlCursor to valid range.
	if m.yamlCursor < 0 {
		m.yamlCursor = 0
	}
	if m.yamlCursor >= len(visLines) {
		m.yamlCursor = len(visLines) - 1
	}
	if m.yamlCursor < 0 {
		m.yamlCursor = 0
	}

	// Compute visual selection range (if active).
	selStart, selEnd := -1, -1
	if m.yamlVisualMode {
		selStart = min(m.yamlVisualStart, m.yamlCursor)
		selEnd = max(m.yamlVisualStart, m.yamlCursor)
	}

	// Compute column range for char/block visual modes.
	// For char mode: anchorCol on anchor line, cursorCol on cursor line.
	// For block mode: rectangular column range on every selected line.
	visualColStart, visualColEnd := 0, 0
	if m.yamlVisualMode && (m.yamlVisualType == 'v' || m.yamlVisualType == 'B') {
		visualColStart = min(m.yamlVisualCol, m.yamlCursorCol())
		visualColEnd = max(m.yamlVisualCol, m.yamlCursorCol())
	}

	// Apply YAML highlighting to visible lines, with search highlights and cursor.
	highlightedLines := make([]string, 0, len(viewport))
	for i, line := range viewport {
		visIdx := yamlScroll + i
		origLine := -1
		if visIdx < len(mapping) {
			origLine = mapping[visIdx]
		}
		// Separate fold prefix from content for column-accurate selection/cursor.
		// Use rune-based slicing because fold indicators are multi-byte UTF-8.
		foldPrefix := ""
		contentLine := line
		lineRunes := []rune(line)
		if len(lineRunes) > yamlFoldPrefixLen {
			foldPrefix = string(lineRunes[:yamlFoldPrefixLen])
			contentLine = string(lineRunes[yamlFoldPrefixLen:])
		}
		highlighted := ui.HighlightYAMLLine(contentLine)
		if m.yamlSearchText.Value != "" && origLine >= 0 && matchSet[origLine] {
			if origLine == currentMatchLine {
				highlighted = ui.HighlightSearchInLine(contentLine, m.yamlSearchText.Value, true)
			} else {
				highlighted = ui.HighlightSearchInLine(contentLine, m.yamlSearchText.Value, false)
			}
		}
		// Visual selection highlight: override with selected style.
		isSelected := m.yamlVisualMode && visIdx >= selStart && visIdx <= selEnd
		if isSelected {
			adjAnchorCol := m.yamlVisualCol - yamlFoldPrefixLen
			adjCursorCol := m.yamlCursorCol() - yamlFoldPrefixLen
			adjColStart := visualColStart - yamlFoldPrefixLen
			adjColEnd := visualColEnd - yamlFoldPrefixLen
			highlighted = ui.RenderVisualSelection(contentLine, m.yamlVisualType, visIdx, selStart, selEnd, m.yamlVisualStart, adjAnchorCol, adjCursorCol, adjColStart, adjColEnd)
		}
		// Line number gutter
		lineNumStr := strings.Repeat(" ", gutterWidth+1)
		if origLine >= 0 {
			lineNumStr = fmt.Sprintf("%*d ", gutterWidth, origLine+1)
		}
		// Cursor indicator + line number + content
		if visIdx == m.yamlCursor {
			if m.yamlVisualMode {
				// In visual mode, don't overlay block cursor on top of visual selection styling.
				highlighted = ui.YamlCursorIndicatorStyle.Render("\u258e") + ui.DimStyle.Render(lineNumStr) + foldPrefix + highlighted
			} else {
				highlighted = ui.YamlCursorIndicatorStyle.Render("\u258e") + ui.DimStyle.Render(lineNumStr) + foldPrefix + ui.RenderCursorAtCol(highlighted, contentLine, m.yamlVisualCurCol-yamlFoldPrefixLen)
			}
		} else if isSelected {
			highlighted = ui.YamlCursorIndicatorStyle.Render(" ") + ui.DimStyle.Render(lineNumStr) + foldPrefix + highlighted
		} else {
			highlighted = " " + ui.DimStyle.Render(lineNumStr) + foldPrefix + highlighted
		}
		highlightedLines = append(highlightedLines, highlighted)
	}

	// Pad to fill available height so the hint bar stays at the bottom.
	for len(highlightedLines) < maxLines {
		highlightedLines = append(highlightedLines, "")
	}

	bodyContent := strings.Join(highlightedLines, "\n")
	borderStyle := lipgloss.NewStyle().
		Border(lipgloss.RoundedBorder()).
		BorderForeground(lipgloss.Color(ui.ColorPrimary)).
		Padding(0, 1).
		Width(m.width - 2).
		Height(maxLines).
		MaxHeight(maxLines + 2)
	body := borderStyle.Render(bodyContent)

	return lipgloss.JoinVertical(lipgloss.Left, title, body, hint)
}

func (m Model) yamlTitle() string {
	switch m.nav.Level {
	case model.LevelResources:
		sel := m.selectedMiddleItem()
		if sel != nil {
			return fmt.Sprintf("YAML: %s/%s", m.namespace, sel.Name)
		}
	case model.LevelOwned:
		sel := m.selectedMiddleItem()
		if sel != nil {
			return fmt.Sprintf("YAML: %s/%s", m.namespace, sel.Name)
		}
	case model.LevelContainers:
		return fmt.Sprintf("YAML: %s/%s", m.namespace, m.nav.OwnedName)
	}
	return "YAML"
}

// yamlCursorCol returns the current cursor column position within the YAML line.
func (m Model) yamlCursorCol() int {
	return m.yamlVisualCurCol
}
