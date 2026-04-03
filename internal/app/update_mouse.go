package app

import (
	tea "github.com/charmbracelet/bubbletea"
	"github.com/janosmiko/lfk/internal/model"
	"github.com/janosmiko/lfk/internal/ui"
)

func (m Model) handleMouse(msg tea.MouseMsg) (tea.Model, tea.Cmd) {
	// Handle mouse scroll in log viewer mode.
	if m.mode == modeLogs {
		switch msg.Button {
		case tea.MouseButtonWheelUp:
			m.logFollow = false
			if m.logScroll > 0 {
				m.logScroll -= 3
				if m.logScroll < 0 {
					m.logScroll = 0
				}
			}
			cmd := m.maybeLoadMoreHistory()
			return m, cmd
		case tea.MouseButtonWheelDown:
			m.logFollow = false
			m.logScroll += 3
			m.clampLogScroll()
		}
		return m, nil
	}

	// Handle tab bar clicks in any mode.
	if len(m.tabs) > 1 && msg.Button == tea.MouseButtonLeft && msg.Action == tea.MouseActionPress && msg.Y == 1 {
		if tab := m.tabAtX(msg.X); tab >= 0 && tab != m.activeTab {
			return m.switchToTab(tab)
		}
		return m, nil
	}

	// Don't handle mouse in overlay/help/yaml modes.
	if m.overlay != overlayNone || m.mode != modeExplorer {
		return m, nil
	}

	switch msg.Button {
	case tea.MouseButtonWheelUp:
		return m.moveCursor(-3)
	case tea.MouseButtonWheelDown:
		return m.moveCursor(3)
	case tea.MouseButtonLeft:
		if msg.Action != tea.MouseActionPress {
			return m, nil
		}
		return m.handleMouseClick(msg.X, msg.Y)
	}
	return m, nil
}

func (m Model) handleMouseClick(x, y int) (tea.Model, tea.Cmd) {
	// Calculate column boundaries (must match viewExplorer).
	var leftEnd, middleEnd int
	if m.fullscreenMiddle || m.fullscreenDashboard {
		// Fullscreen: only middle column exists.
		leftEnd = 0
		middleEnd = m.width
	} else {
		usable := m.width - 6
		leftW := max(10, usable*12/100)
		middleW := max(10, usable*51/100)
		// Each column has border(2) + padding(2) = 4 extra chars width.
		leftEnd = leftW + 4
		middleEnd = leftEnd + middleW + 4
	}

	switch {
	case x < leftEnd:
		// Left column click: navigate parent.
		return m.navigateParent()
	case x < middleEnd:
		// Middle column click: select item.
		// y offset depends on whether column has a header line.
		// Title bar (1) + top border (1) = 2 base offset.
		baseOffset := 2 // title bar (1) + top border (1)
		if len(m.tabs) > 1 {
			baseOffset = 3 // title bar (1) + tab bar (1) + top border (1)
		}
		itemY := y - baseOffset
		switch m.nav.Level {
		case model.LevelResources, model.LevelOwned, model.LevelContainers:
			// Table view has a header row for column names.
			itemY-- // subtract table header row
			if itemY < 0 {
				// Header row click — sort by the clicked column.
				relX := x - 2 // border + padding
				if !m.fullscreenMiddle && !m.fullscreenDashboard {
					relX = x - leftEnd - 2
				}
				return m.handleHeaderClick(relX)
			}
			// Use line map built during rendering for accurate click targeting.
			if itemY < len(ui.ActiveMiddleLineMap) {
				targetIdx := ui.ActiveMiddleLineMap[itemY]
				if targetIdx >= 0 && targetIdx < len(m.visibleMiddleItems()) {
					m.setCursor(targetIdx)
					return m, m.loadPreview()
				}
			}
		default:
			// Column view has a header line (rendered by RenderColumn).
			itemY-- // subtract column header
			if itemY >= 0 && itemY < len(ui.ActiveMiddleLineMap) {
				targetIdx := ui.ActiveMiddleLineMap[itemY]
				if targetIdx >= 0 && targetIdx < len(m.visibleMiddleItems()) {
					m.setCursor(targetIdx)
					m.syncExpandedGroup()
					return m, m.loadPreview()
				}
			}
		}
		return m, nil
	default:
		// Right column click: navigate child.
		return m.navigateChild()
	}
}

// headerColumnWidths holds computed column widths for header click detection.
type headerColumnWidths struct {
	nsW, readyW, restartsW, ageW, statusW           int
	hasNs, hasReady, hasRestarts, hasAge, hasStatus bool
}

// detectHeaderColumns scans items to determine which columns are present and their widths.
func detectHeaderColumns(items []model.Item) headerColumnWidths {
	var h headerColumnWidths
	for _, item := range items {
		if item.Namespace != "" {
			h.hasNs = true
		}
		if item.Ready != "" {
			h.hasReady = true
		}
		if item.Restarts != "" {
			h.hasRestarts = true
		}
		if item.Age != "" {
			h.hasAge = true
		}
		if item.Status != "" {
			h.hasStatus = true
		}
	}

	if h.hasNs {
		h.nsW = len("NAMESPACE")
		for _, item := range items {
			if w := len(item.Namespace); w > h.nsW {
				h.nsW = w
			}
		}
		h.nsW = min(h.nsW+1, 30)
	}
	if h.hasReady {
		h.readyW = len("READY")
		for _, item := range items {
			if w := len(item.Ready); w > h.readyW {
				h.readyW = w
			}
		}
		h.readyW++
	}
	if h.hasRestarts {
		h.restartsW = len("RS") + 1
		for _, item := range items {
			if w := len(item.Restarts); w >= h.restartsW {
				h.restartsW = w + 1
			}
		}
	}
	if h.hasAge {
		h.ageW = len("AGE") + 1
		for _, item := range items {
			if w := len(item.Age); w >= h.ageW {
				h.ageW = w + 1
			}
		}
		h.ageW = min(h.ageW, 10)
	}
	if h.hasStatus {
		h.statusW = len("STATUS")
		for _, item := range items {
			if w := len(item.Status); w > h.statusW {
				h.statusW = w
			}
		}
		h.statusW = min(h.statusW+1, 20)
	}
	return h
}

// findSortableCol returns the index of name in ActiveSortableColumns, or -1.
func findSortableCol(name string) int {
	for i, c := range ui.ActiveSortableColumns {
		if c == name {
			return i
		}
	}
	return -1
}

// colRegion maps a screen position to a sortable column index.
type colRegion struct {
	end   int
	index int
}

// buildHeaderRegions builds the click regions for header columns.
func buildHeaderRegions(h headerColumnWidths, nameW int, extraCols []ui.ExtraColumnInfo) []colRegion {
	markerColW := 2
	var regions []colRegion

	pos := markerColW
	if h.hasNs {
		pos += h.nsW
		if idx := findSortableCol("Namespace"); idx >= 0 {
			regions = append(regions, colRegion{pos, idx})
		}
	}
	pos += nameW
	if idx := findSortableCol("Name"); idx >= 0 {
		regions = append(regions, colRegion{pos, idx})
	}
	if h.hasReady {
		pos += h.readyW
		if idx := findSortableCol("Ready"); idx >= 0 {
			regions = append(regions, colRegion{pos, idx})
		}
	}
	if h.hasRestarts {
		pos += h.restartsW
		if idx := findSortableCol("Restarts"); idx >= 0 {
			regions = append(regions, colRegion{pos, idx})
		}
	}
	if h.hasStatus {
		pos += h.statusW
		if idx := findSortableCol("Status"); idx >= 0 {
			regions = append(regions, colRegion{pos, idx})
		}
	}
	for _, ec := range extraCols {
		pos += ec.Width
		if idx := findSortableCol(ec.Key); idx >= 0 {
			regions = append(regions, colRegion{pos, idx})
		}
	}
	if h.hasAge {
		pos += h.ageW
		if idx := findSortableCol("Age"); idx >= 0 {
			regions = append(regions, colRegion{pos, idx})
		}
	}
	return regions
}

// handleHeaderClick sorts the table by the column that was clicked in the header row.
// relX is the click position relative to the start of the middle column content area.
func (m Model) handleHeaderClick(relX int) (tea.Model, tea.Cmd) {
	items := m.visibleMiddleItems()
	if len(items) == 0 || len(ui.ActiveSortableColumns) == 0 {
		return m, nil
	}

	usable := m.width - 6
	middleW := max(10, usable*51/100)
	if m.fullscreenMiddle {
		middleW = m.width - 2
	}
	width := middleW - 2
	markerColW := 2

	h := detectHeaderColumns(items)
	tableKind := items[0].Kind
	extraCols := ui.CollectExtraColumns(items, width, h.nsW+h.readyW+h.restartsW+h.ageW+h.statusW+markerColW, tableKind)

	extraTotalW := 0
	for _, ec := range extraCols {
		extraTotalW += ec.Width
	}
	nameW := max(width-h.nsW-h.readyW-h.restartsW-h.ageW-h.statusW-markerColW-extraTotalW, 10)

	regions := buildHeaderRegions(h, nameW, extraCols)

	// Find which region the click falls into.
	clickedIdx := -1
	for _, r := range regions {
		if relX < r.end {
			clickedIdx = r.index
			break
		}
	}
	if clickedIdx < 0 && len(regions) > 0 {
		clickedIdx = regions[len(regions)-1].index
	}
	if clickedIdx < 0 {
		return m, nil
	}

	clickedName := ui.ActiveSortableColumns[clickedIdx]
	if m.sortColumnName == clickedName {
		m.sortAscending = !m.sortAscending
	} else {
		m.sortColumnName = clickedName
		m.sortAscending = true
	}
	m.sortMiddleItems()
	m.clampCursor()
	m.setStatusMessage("Sort: "+m.sortModeName(), false)
	return m, tea.Batch(m.loadPreview(), scheduleStatusClear())
}

// tabAtX returns the tab index at the given X coordinate in the tab bar,
// or -1 if the click is not on any tab.
func (m *Model) tabAtX(x int) int {
	labels := m.tabLabels()
	// Tab bar: each tab label is padded with 1 char on each side (Padding(0,1)),
	// separated by " | " (3 chars). Tab bar starts at x=1 (bar left padding).
	pos := 1
	for i, label := range labels {
		tabW := len(label) + 2 // label + padding(0,1) on each side
		if x >= pos && x < pos+tabW {
			return i
		}
		pos += tabW + 3 // separator " | "
	}
	return -1
}

// switchToTab saves the current tab and loads the target tab.
func (m Model) switchToTab(tab int) (tea.Model, tea.Cmd) {
	m.saveCurrentTab()
	if cmd := m.loadTab(tab); cmd != nil {
		return m, cmd
	}
	if m.mode == modeExplorer {
		return m, m.loadPreview()
	}
	if m.mode == modeLogs && m.logCh != nil {
		return m, m.waitForLogLine()
	}
	if m.mode == modeExec && m.execPTY != nil {
		return m, m.scheduleExecTick()
	}
	return m, nil
}
