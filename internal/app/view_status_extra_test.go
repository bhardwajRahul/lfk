package app

import (
	"testing"
	"time"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
	"github.com/janosmiko/lfk/internal/ui"
)

// --- statusBar: command bar active ---

func TestStatusBarCommandBarActive(t *testing.T) {
	m := Model{
		nav:              model.NavigationState{Level: model.LevelResources},
		commandBarActive: true,
		commandBarInput:  TextInput{Value: "get pods"},
		width:            120,
		height:           40,
		tabs:             []TabState{{}},
		selectedItems:    make(map[string]bool),
	}
	bar := m.statusBar()
	stripped := stripANSI(bar)
	assert.Contains(t, stripped, ":")
	assert.Contains(t, stripped, "get pods")
}

func TestStatusBarCommandBarWithSuggestions(t *testing.T) {
	m := Model{
		nav:                          model.NavigationState{Level: model.LevelResources},
		commandBarActive:             true,
		commandBarInput:              TextInput{Value: "get"},
		commandBarSuggestions:        []string{"get pods", "get deployments", "get services"},
		commandBarSelectedSuggestion: 1,
		width:                        120,
		height:                       40,
		tabs:                         []TabState{{}},
		selectedItems:                make(map[string]bool),
	}
	bar := m.statusBar()
	stripped := stripANSI(bar)
	assert.Contains(t, stripped, "get pods")
	assert.Contains(t, stripped, "get deployments")
}

// --- statusBar: filter active ---

func TestStatusBarFilterActive(t *testing.T) {
	m := Model{
		nav:           model.NavigationState{Level: model.LevelResources},
		filterActive:  true,
		filterInput:   TextInput{Value: "nginx"},
		width:         120,
		height:        40,
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
	}
	bar := m.statusBar()
	stripped := stripANSI(bar)
	assert.Contains(t, stripped, "filter")
	assert.Contains(t, stripped, "nginx")
}

// --- statusBar: search active ---

func TestStatusBarSearchActive(t *testing.T) {
	m := Model{
		nav:           model.NavigationState{Level: model.LevelResources},
		searchActive:  true,
		searchInput:   TextInput{Value: "redis"},
		width:         120,
		height:        40,
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
	}
	bar := m.statusBar()
	stripped := stripANSI(bar)
	assert.Contains(t, stripped, "search")
	assert.Contains(t, stripped, "redis")
}

// --- statusBar: status message ---

func TestStatusBarErrorMessage(t *testing.T) {
	m := Model{
		nav:              model.NavigationState{Level: model.LevelResources},
		middleItems:      []model.Item{{Name: "pod"}},
		statusMessage:    "Connection refused",
		statusMessageErr: true,
		statusMessageExp: time.Now().Add(5 * time.Second),
		width:            120,
		height:           40,
		tabs:             []TabState{{}},
		selectedItems:    make(map[string]bool),
	}
	bar := m.statusBar()
	stripped := stripANSI(bar)
	assert.Contains(t, stripped, "Connection refused")
}

func TestStatusBarOkMessage(t *testing.T) {
	m := Model{
		nav:              model.NavigationState{Level: model.LevelResources},
		middleItems:      []model.Item{{Name: "pod"}},
		statusMessage:    "Watch mode ON",
		statusMessageErr: false,
		statusMessageExp: time.Now().Add(5 * time.Second),
		width:            120,
		height:           40,
		tabs:             []TabState{{}},
		selectedItems:    make(map[string]bool),
	}
	bar := m.statusBar()
	stripped := stripANSI(bar)
	assert.Contains(t, stripped, "Watch mode ON")
}

// --- statusBar: active filter preset indicator ---

func TestStatusBarActiveFilterPreset(t *testing.T) {
	m := Model{
		nav:                model.NavigationState{Level: model.LevelResources},
		middleItems:        []model.Item{{Name: "pod"}},
		activeFilterPreset: &FilterPreset{Name: "Failing"},
		width:              120,
		height:             40,
		tabs:               []TabState{{}},
		selectedItems:      make(map[string]bool),
	}
	bar := m.statusBar()
	stripped := stripANSI(bar)
	assert.Contains(t, stripped, "filter: Failing")
}

// --- statusBar: dashboard hints ---

func TestStatusBarDashboardHints(t *testing.T) {
	m := Model{
		nav: model.NavigationState{
			Level: model.LevelResourceTypes,
		},
		middleItems: []model.Item{
			{Name: "Cluster Dashboard", Extra: "__overview__"},
		},
		width:         200,
		height:        40,
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
	}
	bar := m.statusBar()
	stripped := stripANSI(bar)
	// Dashboard hints are different from standard hints.
	assert.Contains(t, stripped, "scroll")
	assert.Contains(t, stripped, "namespace")
}

// --- statusBar: small width ---

func TestStatusBarSmallWidth(t *testing.T) {
	m := Model{
		nav:           model.NavigationState{Level: model.LevelResources},
		middleItems:   []model.Item{{Name: "pod"}},
		width:         15,
		height:        40,
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
	}
	bar := m.statusBar()
	assert.NotEmpty(t, bar)
}

// --- renderErrorLogOverlay ---

func TestRenderErrorLogOverlay(t *testing.T) {
	m := Model{
		width:  120,
		height: 40,
		errorLog: []ui.ErrorLogEntry{
			{Level: "ERR", Message: "test error", Time: time.Now()},
			{Level: "INF", Message: "test info", Time: time.Now()},
		},
	}
	bg := "background content\n"
	result := m.renderErrorLogOverlay(bg)
	assert.NotEmpty(t, result)
}

func TestRenderErrorLogOverlaySmall(t *testing.T) {
	m := Model{
		width:  20,
		height: 10,
	}
	bg := "bg\n"
	result := m.renderErrorLogOverlay(bg)
	assert.NotEmpty(t, result)
}
