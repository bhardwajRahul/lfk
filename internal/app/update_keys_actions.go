package app

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/lfk/internal/model"
	"github.com/janosmiko/lfk/internal/ui"
)

// handleExplorerActionKey handles key bindings for explorer-mode actions
// such as namespace toggling, scrolling/paging, tab management, editors,
// resource actions, and configurable direct-action keybindings.
// Returns (model, cmd, handled) where handled indicates whether the key
// was consumed. If not handled, the caller should fall through.
func (m Model) handleExplorerActionKey(msg tea.KeyMsg) (tea.Model, tea.Cmd, bool) {
	switch msg.String() {
	case "A":
		m.allNamespaces = !m.allNamespaces
		if m.allNamespaces {
			m.selectedNamespaces = nil
			m.setStatusMessage("All namespaces mode ON", false)
		} else {
			m.setStatusMessage("All namespaces mode OFF (ns: "+m.namespace+")", false)
		}
		m.cancelAndReset()
		m.requestGen++
		return m, tea.Batch(m.refreshCurrentLevel(), scheduleStatusClear()), true

	case "Q":
		m.loading = true
		m.setStatusMessage("Loading quota data...", false)
		return m, m.loadQuotas(), true

	case "ctrl+d":
		if m.fullscreenDashboard {
			halfPage := (m.height - 4) / 2
			m.previewScroll += halfPage
			m.clampPreviewScroll()
			return m, nil, true
		}
		visible := m.visibleMiddleItems()
		halfPage := (m.height - 4) / 2
		c := m.cursor() + halfPage
		if c >= len(visible) {
			c = len(visible) - 1
		}
		if c < 0 {
			c = 0
		}
		m.setCursor(c)
		m.syncExpandedGroup()
		m.rightItems = nil
		m.previewYAML = ""
		m.previewScroll = 0
		m.loading = true
		return m, m.loadPreview(), true

	case "ctrl+u":
		if m.fullscreenDashboard {
			halfPage := (m.height - 4) / 2
			m.previewScroll -= halfPage
			if m.previewScroll < 0 {
				m.previewScroll = 0
			}
			return m, nil, true
		}
		visible := m.visibleMiddleItems()
		halfPage := (m.height - 4) / 2
		c := m.cursor() - halfPage
		if c < 0 {
			c = 0
		}
		if c >= len(visible) {
			c = len(visible) - 1
		}
		m.setCursor(c)
		m.syncExpandedGroup()
		m.rightItems = nil
		m.previewYAML = ""
		m.previewScroll = 0
		m.loading = true
		return m, m.loadPreview(), true

	case "ctrl+f":
		if m.fullscreenDashboard {
			fullPage := m.height - 4
			m.previewScroll += fullPage
			m.clampPreviewScroll()
			return m, nil, true
		}
		visible := m.visibleMiddleItems()
		fullPage := m.height - 4
		c := m.cursor() + fullPage
		if c >= len(visible) {
			c = len(visible) - 1
		}
		if c < 0 {
			c = 0
		}
		m.setCursor(c)
		m.syncExpandedGroup()
		m.rightItems = nil
		m.previewYAML = ""
		m.previewScroll = 0
		m.loading = true
		return m, m.loadPreview(), true

	case "ctrl+b":
		if m.fullscreenDashboard {
			fullPage := m.height - 4
			m.previewScroll -= fullPage
			if m.previewScroll < 0 {
				m.previewScroll = 0
			}
			return m, nil, true
		}
		visible := m.visibleMiddleItems()
		fullPage := m.height - 4
		c := m.cursor() - fullPage
		if c < 0 {
			c = 0
		}
		if c >= len(visible) {
			c = len(visible) - 1
		}
		m.setCursor(c)
		m.syncExpandedGroup()
		m.rightItems = nil
		m.previewYAML = ""
		m.previewScroll = 0
		m.loading = true
		return m, m.loadPreview(), true

	case "0":
		// Jump to clusters level.
		for m.nav.Level > model.LevelClusters {
			ret, _ := m.navigateParent()
			m = ret.(Model)
		}
		return m, m.loadPreview(), true

	case "1":
		// Jump to resource types level.
		if m.nav.Level < model.LevelResourceTypes {
			return m, nil, true // can't jump forward
		}
		for m.nav.Level > model.LevelResourceTypes {
			ret, _ := m.navigateParent()
			m = ret.(Model)
		}
		return m, m.loadPreview(), true

	case "2":
		// Jump to resources level.
		if m.nav.Level < model.LevelResources {
			return m, nil, true
		}
		for m.nav.Level > model.LevelResources {
			ret, _ := m.navigateParent()
			m = ret.(Model)
		}
		return m, m.loadPreview(), true

	case "W":
		if m.nav.Level == model.LevelResources && m.nav.ResourceType.Kind == "Event" {
			m.warningEventsOnly = !m.warningEventsOnly
			// Re-filter the current items.
			if cached, ok := m.itemCache[m.navKey()]; ok {
				if m.warningEventsOnly {
					var filtered []model.Item
					for _, item := range cached {
						if item.Status == "Warning" {
							filtered = append(filtered, item)
						}
					}
					m.middleItems = filtered
				} else {
					m.middleItems = cached
				}
				m.clampCursor()
			}
			if m.warningEventsOnly {
				m.setStatusMessage("Showing warnings only", false)
			} else {
				m.setStatusMessage("Showing all events", false)
			}
			return m, scheduleStatusClear(), true
		}
		// Save resource YAML to file (same as 'S').
		if m.nav.Level == model.LevelResources || m.nav.Level == model.LevelOwned || m.nav.Level == model.LevelContainers {
			sel := m.selectedMiddleItem()
			if sel != nil {
				m.setStatusMessage("Exporting...", false)
				return m, m.exportResourceToFile(), true
			}
		}
		return m, nil, true

	case "J":
		// Scroll preview pane down, clamped to content length.
		m.previewScroll++
		m.clampPreviewScroll()
		return m, nil, true

	case "K":
		// Scroll preview pane up.
		if m.previewScroll > 0 {
			m.previewScroll--
		}
		return m, nil, true

	case "o":
		// Navigate to owner/controller.
		sel := m.selectedMiddleItem()
		if sel == nil {
			return m, nil, true
		}
		type ownerRef struct {
			kind, name, apiVersion string
		}
		var owners []ownerRef
		for _, kv := range sel.Columns {
			if strings.HasPrefix(kv.Key, "owner:") {
				parts := strings.SplitN(kv.Value, "||", 3)
				if len(parts) == 3 {
					owners = append(owners, ownerRef{
						apiVersion: parts[0],
						kind:       parts[1],
						name:       parts[2],
					})
				}
			}
		}
		if len(owners) == 0 {
			m.setStatusMessage("No owner references found", true)
			return m, scheduleStatusClear(), true
		}
		// Use first owner (most resources have exactly one).
		owner := owners[0]
		ret, cmd := m.navigateToOwner(owner.kind, owner.name)
		return ret, cmd, true

	case ",":
		m.sortBy = (m.sortBy + 1) % 3
		m.sortMiddleItems()
		m.clampCursor()
		m.setStatusMessage("Sort: "+m.sortModeName(), false)
		return m, tea.Batch(m.loadPreview(), scheduleStatusClear()), true

	case "ctrl+o":
		// Open ingress host in browser (works when viewing an Ingress resource).
		kind := m.selectedResourceKind()
		if kind != "Ingress" {
			m.setStatusMessage("Open in browser is only available for Ingress resources", true)
			return m, scheduleStatusClear(), true
		}
		sel := m.selectedMiddleItem()
		if sel == nil {
			return m, nil, true
		}
		m.actionCtx = m.buildActionCtx(sel, kind)
		ret, cmd := m.executeAction("Open in Browser")
		return ret, cmd, true

	case "c", "y":
		// Copy resource name to clipboard.
		sel := m.selectedMiddleItem()
		if sel != nil {
			m.setStatusMessage("Copied: "+sel.Name, false)
			return m, tea.Batch(copyToSystemClipboard(sel.Name), scheduleStatusClear()), true
		}
		return m, nil, true

	case "C", "ctrl+y":
		// Copy resource YAML to clipboard.
		sel := m.selectedMiddleItem()
		if sel != nil {
			return m, m.copyYAMLToClipboard(), true
		}
		return m, nil, true

	case "ctrl+p":
		return m, m.applyFromClipboard(), true

	case "t":
		// Create new tab (clone current state, max 9).
		if len(m.tabs) >= 9 {
			m.setStatusMessage("Max 9 tabs", true)
			return m, scheduleStatusClear(), true
		}
		m.saveCurrentTab()
		insertAt := m.activeTab + 1
		newTab := m.cloneCurrentTab()
		m.tabs = append(m.tabs[:insertAt], append([]TabState{newTab}, m.tabs[insertAt:]...)...)
		m.activeTab = insertAt
		m.setStatusMessage(fmt.Sprintf("Tab %d created", m.activeTab+1), false)
		return m, scheduleStatusClear(), true

	case "]":
		// Next tab.
		if len(m.tabs) <= 1 {
			return m, nil, true
		}
		m.saveCurrentTab()
		next := (m.activeTab + 1) % len(m.tabs)
		if cmd := m.loadTab(next); cmd != nil {
			return m, cmd, true
		}
		if m.mode == modeExec && m.execPTY != nil {
			return m, m.scheduleExecTick(), true
		}
		return m, m.loadPreview(), true

	case "[":
		// Previous tab.
		if len(m.tabs) <= 1 {
			return m, nil, true
		}
		m.saveCurrentTab()
		prev := (m.activeTab - 1 + len(m.tabs)) % len(m.tabs)
		if cmd := m.loadTab(prev); cmd != nil {
			return m, cmd, true
		}
		if m.mode == modeExec && m.execPTY != nil {
			return m, m.scheduleExecTick(), true
		}
		return m, m.loadPreview(), true

	case "a":
		// Open template creation overlay.
		m.templateItems = model.BuiltinTemplates()
		m.templateCursor = 0
		m.overlay = overlayTemplates
		return m, nil, true

	case "e":
		// Open secret editor when a Secret resource is selected.
		if m.nav.Level == model.LevelResources && m.nav.ResourceType.Kind == "Secret" {
			sel := m.selectedMiddleItem()
			if sel != nil {
				return m, m.loadSecretData(), true
			}
		}
		// Open configmap editor when a ConfigMap resource is selected.
		if m.nav.Level == model.LevelResources && m.nav.ResourceType.Kind == "ConfigMap" {
			sel := m.selectedMiddleItem()
			if sel != nil {
				return m, m.loadConfigMapData(), true
			}
		}
		return m, nil, true

	case "I":
		// Open API explain browser (resource structure).
		ret, cmd := m.openExplainBrowser()
		return ret, cmd, true

	case "U":
		// Open RBAC permissions browser (can-i).
		ret, cmd := m.openCanIBrowser()
		return ret, cmd, true

	case "i":
		// Open label/annotation editor for any resource (not port forwards).
		if m.nav.Level == model.LevelResources && m.nav.ResourceType.Kind != "__port_forwards__" {
			sel := m.selectedMiddleItem()
			if sel != nil {
				m.labelResourceType = m.nav.ResourceType
				return m, m.loadLabelData(), true
			}
		} else if m.nav.Level == model.LevelOwned {
			sel := m.selectedMiddleItem()
			if sel != nil {
				rt, ok := m.resolveOwnedResourceType(sel)
				if ok {
					m.labelResourceType = rt
					return m, m.loadLabelData(), true
				}
			}
		}
		return m, nil, true

	case ".":
		// Quick filter presets: toggle or open overlay.
		if m.nav.Level < model.LevelResources {
			m.setStatusMessage("Quick filters are only available at resource level", true)
			return m, scheduleStatusClear(), true
		}
		if m.activeFilterPreset != nil {
			// Clear the active filter preset and restore the full list.
			name := m.activeFilterPreset.Name
			m.activeFilterPreset = nil
			m.middleItems = m.unfilteredMiddleItems
			m.unfilteredMiddleItems = nil
			m.setCursor(0)
			m.clampCursor()
			m.setStatusMessage("Filter cleared: "+name, false)
			return m, tea.Batch(scheduleStatusClear(), m.loadPreview()), true
		}
		// Open the filter preset overlay.
		m.filterPresets = builtinFilterPresets(m.nav.ResourceType.Kind)
		m.overlayCursor = 0
		m.overlay = overlayFilterPreset
		return m, nil, true

	case "S":
		// Export resource YAML to file.
		if m.nav.Level == model.LevelResources || m.nav.Level == model.LevelOwned || m.nav.Level == model.LevelContainers {
			sel := m.selectedMiddleItem()
			if sel != nil {
				m.setStatusMessage("Exporting...", false)
				return m, m.exportResourceToFile(), true
			}
		}
		return m, nil, true

	case "d":
		// Diff two selected resources side by side.
		if m.nav.Level < model.LevelResources {
			m.setStatusMessage("Diff is only available at resource level", true)
			return m, scheduleStatusClear(), true
		}
		selected := m.selectedItemsList()
		if len(selected) != 2 {
			m.setStatusMessage("Select exactly 2 resources to diff (use Space to select)", true)
			return m, scheduleStatusClear(), true
		}
		m.loading = true
		m.setStatusMessage("Loading diff...", false)
		return m, m.loadDiff(m.nav.ResourceType, selected[0], selected[1]), true

	case "!":
		// Open the error log overlay.
		m.overlayErrorLog = true
		m.errorLogScroll = 0
		return m, nil, true

	case "@":
		// Navigate to the Monitoring dashboard item.
		if m.nav.Level < model.LevelResourceTypes {
			m.setStatusMessage("Select a cluster first", true)
			return m, scheduleStatusClear(), true
		}
		// Find the Monitoring item in the middle column and select it.
		for i, item := range m.middleItems {
			if item.Extra == "__monitoring__" {
				m.setCursor(i)
				m.clampCursor()
				return m, m.loadPreview(), true
			}
		}
		return m, nil, true
	}

	// Configurable direct action keybindings.
	key := msg.String()
	kb := ui.ActiveKeybindings
	if key == kb.Logs {
		ret, cmd := m.directActionLogs()
		return ret, cmd, true
	}
	if key == kb.Refresh {
		ret, cmd := m.directActionRefresh()
		return ret, cmd, true
	}
	if key == kb.Restart {
		ret, cmd := m.directActionRestart()
		return ret, cmd, true
	}
	if key == kb.Exec {
		ret, cmd := m.directActionExec()
		return ret, cmd, true
	}
	if key == kb.Edit {
		ret, cmd := m.directActionEdit()
		return ret, cmd, true
	}
	if key == kb.Describe {
		ret, cmd := m.directActionDescribe()
		return ret, cmd, true
	}
	if key == kb.Delete {
		ret, cmd := m.directActionDelete()
		return ret, cmd, true
	}
	if key == kb.ForceDelete {
		ret, cmd := m.directActionForceDelete()
		return ret, cmd, true
	}
	if key == kb.Scale {
		ret, cmd := m.directActionScale()
		return ret, cmd, true
	}

	return m, nil, false
}
