package app

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/lfk/internal/model"
	"github.com/janosmiko/lfk/internal/ui"
)

// openExplainBrowser determines the resource type from the current navigation
// context and launches kubectl explain for it.
func (m Model) openExplainBrowser() (tea.Model, tea.Cmd) {
	var resource, apiVersion string

	switch m.nav.Level {
	case model.LevelResourceTypes:
		// At resource types level: use the selected middle item.
		sel := m.selectedMiddleItem()
		if sel == nil {
			m.setStatusMessage("No resource type selected", true)
			return m, scheduleStatusClear()
		}
		// Skip virtual items (overview, monitoring, collapsed groups, etc.).
		if sel.Kind == "__collapsed_group__" || sel.Kind == "__overview__" ||
			sel.Kind == "__monitoring__" || sel.Extra == "__overview__" ||
			sel.Extra == "__monitoring__" {
			m.setStatusMessage("Cannot explain this item", true)
			return m, scheduleStatusClear()
		}

		// At LevelResourceTypes, Item.Extra holds the resource ref in
		// format "group/version/resource" (from ResourceTypeEntry.ResourceRef()).
		// We need to find the actual ResourceTypeEntry to build the kubectl explain specifier.
		crds := m.discoveredCRDs[m.nav.Context]
		rt, ok := model.FindResourceTypeIn(sel.Extra, crds)
		if ok {
			resource, apiVersion = buildExplainResourceFromType(rt)
		} else {
			// Fallback: use the kind name lowercased.
			if sel.Kind != "" {
				resource = strings.ToLower(sel.Kind) + "s"
			}
		}
		if resource == "" {
			m.setStatusMessage("Cannot determine resource type", true)
			return m, scheduleStatusClear()
		}

	case model.LevelResources, model.LevelOwned, model.LevelContainers:
		// Use the current resource type from navigation state.
		rt := m.nav.ResourceType
		resource, apiVersion = buildExplainResourceFromType(rt)
		if resource == "" {
			m.setStatusMessage("Cannot determine resource type", true)
			return m, scheduleStatusClear()
		}

	default:
		m.setStatusMessage("Select a resource type first", true)
		return m, scheduleStatusClear()
	}

	m.loading = true
	m.explainResource = resource
	m.explainAPIVersion = apiVersion
	m.setStatusMessage("Loading API structure...", false)
	return m, m.execKubectlExplain(resource, apiVersion, "")
}

// buildExplainResourceFromType returns the resource name and api-version flag value
// for kubectl explain. The resource is just the plural name (e.g., "deployments").
// The apiVersion is "group/version" (e.g., "apps/v1") for non-core resources, empty for core.
func buildExplainResourceFromType(rt model.ResourceTypeEntry) (resource, apiVersion string) {
	if rt.Resource == "" {
		return "", ""
	}
	if rt.APIGroup != "" && rt.APIVersion != "" {
		return rt.Resource, rt.APIGroup + "/" + rt.APIVersion
	}
	return rt.Resource, ""
}

// handleExplainKey handles keyboard input in the explain view mode.
func (m Model) handleExplainKey(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	fieldCount := len(m.explainFields)
	visibleLines := m.height - 6
	if visibleLines < 3 {
		visibleLines = 3
	}

	switch msg.String() {
	case "?":
		m.mode = modeHelp
		m.helpScroll = 0
		m.helpFilter.Clear()
		m.helpSearchActive = false
		m.helpContextMode = "API Explorer"
		return m, nil

	case "q", "esc":
		m.mode = modeExplorer
		m.explainFields = nil
		m.explainDesc = ""
		m.explainPath = ""
		m.explainResource = ""
		m.explainAPIVersion = ""
		m.explainTitle = ""
		m.explainCursor = 0
		m.explainScroll = 0
		return m, nil

	case "j", "down":
		if m.explainCursor < fieldCount-1 {
			m.explainCursor++
			// Scroll down if cursor goes below visible area.
			if m.explainCursor >= m.explainScroll+visibleLines {
				m.explainScroll = m.explainCursor - visibleLines + 1
			}
		}
		return m, nil

	case "k", "up":
		if m.explainCursor > 0 {
			m.explainCursor--
			// Scroll up if cursor goes above visible area.
			if m.explainCursor < m.explainScroll {
				m.explainScroll = m.explainCursor
			}
		}
		return m, nil

	case "g":
		if m.pendingG {
			m.pendingG = false
			m.explainCursor = 0
			m.explainScroll = 0
			return m, nil
		}
		m.pendingG = true
		return m, nil

	case "G":
		if fieldCount > 0 {
			m.explainCursor = fieldCount - 1
			maxScroll := fieldCount - visibleLines
			if maxScroll < 0 {
				maxScroll = 0
			}
			m.explainScroll = maxScroll
		}
		return m, nil

	case "ctrl+d":
		halfPage := visibleLines / 2
		m.explainCursor += halfPage
		if m.explainCursor >= fieldCount {
			m.explainCursor = fieldCount - 1
		}
		if m.explainCursor < 0 {
			m.explainCursor = 0
		}
		m.explainScroll += halfPage
		maxScroll := fieldCount - visibleLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.explainScroll > maxScroll {
			m.explainScroll = maxScroll
		}
		return m, nil

	case "ctrl+u":
		halfPage := visibleLines / 2
		m.explainCursor -= halfPage
		if m.explainCursor < 0 {
			m.explainCursor = 0
		}
		m.explainScroll -= halfPage
		if m.explainScroll < 0 {
			m.explainScroll = 0
		}
		return m, nil

	case "ctrl+f":
		m.explainCursor += visibleLines
		if m.explainCursor >= fieldCount {
			m.explainCursor = fieldCount - 1
		}
		if m.explainCursor < 0 {
			m.explainCursor = 0
		}
		m.explainScroll += visibleLines
		maxScroll := fieldCount - visibleLines
		if maxScroll < 0 {
			maxScroll = 0
		}
		if m.explainScroll > maxScroll {
			m.explainScroll = maxScroll
		}
		return m, nil

	case "ctrl+b":
		m.explainCursor -= visibleLines
		if m.explainCursor < 0 {
			m.explainCursor = 0
		}
		m.explainScroll -= visibleLines
		if m.explainScroll < 0 {
			m.explainScroll = 0
		}
		return m, nil

	case "l", "right", "enter":
		// Drill into the selected field if it has an object/array type.
		if m.explainCursor >= 0 && m.explainCursor < fieldCount {
			f := m.explainFields[m.explainCursor]
			if ui.IsDrillableType(f.Type) {
				m.loading = true
				m.setStatusMessage("Loading field structure...", false)
				return m, m.execKubectlExplain(m.explainResource, m.explainAPIVersion, f.Path)
			}
			m.setStatusMessage("This field is a primitive type and cannot be drilled into", true)
			return m, scheduleStatusClear()
		}
		return m, nil

	case "h", "left", "backspace":
		// Go back one level in the path.
		if m.explainPath == "" {
			// Already at root: exit explain view.
			m.mode = modeExplorer
			m.explainFields = nil
			m.explainDesc = ""
			m.explainPath = ""
			m.explainResource = ""
			m.explainTitle = ""
			m.explainCursor = 0
			m.explainScroll = 0
			return m, nil
		}
		// Trim the last path segment.
		newPath := m.explainPath
		if idx := strings.LastIndex(newPath, "."); idx >= 0 {
			newPath = newPath[:idx]
		} else {
			newPath = ""
		}
		m.loading = true
		m.setStatusMessage("Loading parent structure...", false)
		return m, m.execKubectlExplain(m.explainResource, m.explainAPIVersion, newPath)
	}

	return m, nil
}
