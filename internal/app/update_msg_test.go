package app

import (
	"sync"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- Update: tea.WindowSizeMsg ---

func TestUpdateWindowSizeMsg(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResources},
		middleItems: []model.Item{
			{Name: "pod-1"},
			{Name: "pod-2"},
		},
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
		cursorMemory:  make(map[string]int),
		itemCache:     make(map[string][]model.Item),
		execMu:        &sync.Mutex{},
		requestGen:    0,
	}

	msg := tea.WindowSizeMsg{Width: 120, Height: 40}
	result, cmd := m.Update(msg)
	mdl := result.(Model)
	assert.Equal(t, 120, mdl.width)
	assert.Equal(t, 40, mdl.height)
	assert.Nil(t, cmd)
}

func TestUpdateWindowSizeMsgSmall(t *testing.T) {
	m := Model{
		nav:           model.NavigationState{Level: model.LevelResources},
		middleItems:   []model.Item{{Name: "pod-1"}},
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
		cursorMemory:  make(map[string]int),
		itemCache:     make(map[string][]model.Item),
		execMu:        &sync.Mutex{},
	}
	m.setCursor(0)

	msg := tea.WindowSizeMsg{Width: 20, Height: 10}
	result, _ := m.Update(msg)
	mdl := result.(Model)
	assert.Equal(t, 20, mdl.width)
	assert.Equal(t, 10, mdl.height)
}

// --- Update: statusMessageExpiredMsg ---

func TestUpdateStatusMessageExpired(t *testing.T) {
	m := Model{
		nav:              model.NavigationState{Level: model.LevelResources},
		tabs:             []TabState{{}},
		selectedItems:    make(map[string]bool),
		cursorMemory:     make(map[string]int),
		itemCache:        make(map[string][]model.Item),
		statusMessage:    "some message",
		statusMessageErr: false,
		width:            80,
		height:           40,
		execMu:           &sync.Mutex{},
	}

	result, _ := m.Update(statusMessageExpiredMsg{})
	mdl := result.(Model)
	assert.Empty(t, mdl.statusMessage)
}

// --- Update: resourceTypesMsg ---

func TestUpdateResourceTypesMsg(t *testing.T) {
	m := Model{
		nav: model.NavigationState{
			Level:   model.LevelResourceTypes,
			Context: "test",
		},
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
		cursorMemory:  make(map[string]int),
		itemCache:     make(map[string][]model.Item),
		loading:       true,
		width:         80,
		height:        40,
		execMu:        &sync.Mutex{},
	}

	items := []model.Item{
		{Name: "Pods", Category: "Workloads"},
		{Name: "Deployments", Category: "Workloads"},
	}
	result, _ := m.Update(resourceTypesMsg{items: items})
	mdl := result.(Model)
	assert.False(t, mdl.loading)
	assert.Len(t, mdl.middleItems, 2)
}

func TestUpdateResourceTypesMsgAtClusterLevel(t *testing.T) {
	m := Model{
		nav: model.NavigationState{
			Level: model.LevelClusters,
		},
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
		cursorMemory:  make(map[string]int),
		itemCache:     make(map[string][]model.Item),
		loading:       true,
		width:         80,
		height:        40,
		execMu:        &sync.Mutex{},
	}

	items := []model.Item{
		{Name: "Pods"},
		{Name: "Services"},
	}
	result, cmd := m.Update(resourceTypesMsg{items: items})
	mdl := result.(Model)
	assert.False(t, mdl.loading)
	assert.Len(t, mdl.rightItems, 2) // goes to rightItems at cluster level
	assert.Nil(t, cmd)
}

// --- Update: startupTipMsg ---

func TestUpdateStartupTipMsg(t *testing.T) {
	m := Model{
		nav:           model.NavigationState{Level: model.LevelResources},
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
		cursorMemory:  make(map[string]int),
		itemCache:     make(map[string][]model.Item),
		width:         80,
		height:        40,
		execMu:        &sync.Mutex{},
	}

	result, cmd := m.Update(startupTipMsg{tip: "Press ? for help"})
	mdl := result.(Model)
	assert.Contains(t, mdl.statusMessage, "Press ? for help")
	assert.NotNil(t, cmd) // scheduleStatusClear
}

// --- Update: actionResultMsg ---

func TestUpdateActionResultSuccess(t *testing.T) {
	m := Model{
		nav:           model.NavigationState{Level: model.LevelResources},
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
		cursorMemory:  make(map[string]int),
		itemCache:     make(map[string][]model.Item),
		width:         80,
		height:        40,
		execMu:        &sync.Mutex{},
	}

	result, cmd := m.Update(actionResultMsg{message: "Resource deleted"})
	mdl := result.(Model)
	assert.Equal(t, "Resource deleted", mdl.statusMessage)
	assert.NotNil(t, cmd)
}

func TestUpdateActionResultError(t *testing.T) {
	m := Model{
		nav:           model.NavigationState{Level: model.LevelResources},
		tabs:          []TabState{{}},
		selectedItems: make(map[string]bool),
		cursorMemory:  make(map[string]int),
		itemCache:     make(map[string][]model.Item),
		width:         80,
		height:        40,
		execMu:        &sync.Mutex{},
	}

	result, _ := m.Update(actionResultMsg{err: assert.AnError})
	mdl := result.(Model)
	assert.True(t, mdl.statusMessageErr)
}
