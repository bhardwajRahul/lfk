package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- cursor / setCursor ---

func TestCursorGetSet(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResources},
	}
	assert.Equal(t, 0, m.cursor())

	m.setCursor(5)
	assert.Equal(t, 5, m.cursor())

	// Setting cursor for a different level should not affect the current level.
	m.nav.Level = model.LevelOwned
	assert.Equal(t, 0, m.cursor())
	m.setCursor(3)
	assert.Equal(t, 3, m.cursor())

	// Switch back and verify previous level is preserved.
	m.nav.Level = model.LevelResources
	assert.Equal(t, 5, m.cursor())
}

// --- clampCursor ---

func TestClampCursor(t *testing.T) {
	tests := []struct {
		name     string
		cursor   int
		items    []model.Item
		expected int
	}{
		{
			name:     "cursor within bounds",
			cursor:   1,
			items:    []model.Item{{Name: "a"}, {Name: "b"}, {Name: "c"}},
			expected: 1,
		},
		{
			name:     "cursor past end clamped",
			cursor:   10,
			items:    []model.Item{{Name: "a"}, {Name: "b"}},
			expected: 1,
		},
		{
			name:     "negative cursor clamped to 0",
			cursor:   -5,
			items:    []model.Item{{Name: "a"}},
			expected: 0,
		},
		{
			name:     "empty items cursor stays 0",
			cursor:   5,
			items:    nil,
			expected: 5,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{
				nav:         model.NavigationState{Level: model.LevelResources},
				middleItems: tt.items,
			}
			m.setCursor(tt.cursor)
			m.clampCursor()
			assert.Equal(t, tt.expected, m.cursor())
		})
	}
}

// --- cursorItemKey ---

func TestCursorItemKey(t *testing.T) {
	t.Run("returns item fields at cursor", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{
				{Name: "pod-a", Namespace: "ns-1", Extra: "extra-a"},
				{Name: "pod-b", Namespace: "ns-2", Extra: "extra-b"},
			},
		}
		m.setCursor(1)
		name, ns, extra := m.cursorItemKey()
		assert.Equal(t, "pod-b", name)
		assert.Equal(t, "ns-2", ns)
		assert.Equal(t, "extra-b", extra)
	})

	t.Run("returns empty strings when no items", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
		}
		name, ns, extra := m.cursorItemKey()
		assert.Empty(t, name)
		assert.Empty(t, ns)
		assert.Empty(t, extra)
	})

	t.Run("returns empty strings when cursor out of range", func(t *testing.T) {
		m := Model{
			nav:         model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{{Name: "only"}},
		}
		m.setCursor(5)
		name, ns, extra := m.cursorItemKey()
		assert.Empty(t, name)
		assert.Empty(t, ns)
		assert.Empty(t, extra)
	})
}

// --- restoreCursorToItem ---

func TestRestoreCursorToItem(t *testing.T) {
	t.Run("restores to matching item", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{
				{Name: "pod-a", Namespace: "ns-1"},
				{Name: "pod-b", Namespace: "ns-2"},
				{Name: "pod-c", Namespace: "ns-3"},
			},
		}
		m.setCursor(0)
		m.restoreCursorToItem("pod-c", "ns-3", "")
		assert.Equal(t, 2, m.cursor())
	})

	t.Run("clamps when item gone", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{
				{Name: "pod-a"},
			},
		}
		m.setCursor(10)
		m.restoreCursorToItem("gone", "", "")
		assert.Equal(t, 0, m.cursor())
	})

	t.Run("empty name clamps", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{
				{Name: "pod-a"},
			},
		}
		m.setCursor(5)
		m.restoreCursorToItem("", "", "")
		assert.Equal(t, 0, m.cursor())
	})
}

// --- navKey ---

func TestNavKey(t *testing.T) {
	tests := []struct {
		name     string
		nav      model.NavigationState
		expected string
	}{
		{
			name:     "context only",
			nav:      model.NavigationState{Context: "prod"},
			expected: "prod",
		},
		{
			name: "context with resource type",
			nav: model.NavigationState{
				Context:      "prod",
				ResourceType: model.ResourceTypeEntry{Resource: "deployments"},
			},
			expected: "prod/deployments",
		},
		{
			name: "full path",
			nav: model.NavigationState{
				Context:      "prod",
				ResourceType: model.ResourceTypeEntry{Resource: "deployments"},
				ResourceName: "my-deploy",
				OwnedName:    "my-pod",
			},
			expected: "prod/deployments/my-deploy/my-pod",
		},
		{
			name:     "empty state",
			nav:      model.NavigationState{},
			expected: "",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			m := Model{nav: tt.nav}
			assert.Equal(t, tt.expected, m.navKey())
		})
	}
}

// --- saveCursor / restoreCursor ---

func TestSaveCursorAndRestoreCursor(t *testing.T) {
	m := Model{
		nav: model.NavigationState{
			Level:   model.LevelResources,
			Context: "prod",
			ResourceType: model.ResourceTypeEntry{
				Resource: "pods",
			},
		},
		cursorMemory: make(map[string]int),
		middleItems: []model.Item{
			{Name: "a"}, {Name: "b"}, {Name: "c"},
		},
	}

	m.setCursor(2)
	m.saveCursor()

	// Move cursor away.
	m.setCursor(0)
	assert.Equal(t, 0, m.cursor())

	// Restore should bring it back.
	m.restoreCursor()
	assert.Equal(t, 2, m.cursor())
}

func TestRestoreCursorNoSavedPosition(t *testing.T) {
	m := Model{
		nav: model.NavigationState{
			Level:   model.LevelResources,
			Context: "dev",
		},
		cursorMemory: make(map[string]int),
		middleItems:  []model.Item{{Name: "a"}},
	}
	m.setCursor(5)

	// No saved position for this nav key, should reset to 0.
	m.restoreCursor()
	assert.Equal(t, 0, m.cursor())
}

// --- selectedMiddleItem ---

func TestSelectedMiddleItem(t *testing.T) {
	t.Run("returns correct item", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{
				{Name: "pod-a", Kind: "Pod"},
				{Name: "pod-b", Kind: "Pod"},
			},
		}
		m.setCursor(1)
		item := m.selectedMiddleItem()
		assert.NotNil(t, item)
		assert.Equal(t, "pod-b", item.Name)
	})

	t.Run("returns nil when empty", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
		}
		assert.Nil(t, m.selectedMiddleItem())
	})

	t.Run("returns nil when cursor out of range", func(t *testing.T) {
		m := Model{
			nav:         model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{{Name: "only"}},
		}
		m.setCursor(10)
		assert.Nil(t, m.selectedMiddleItem())
	})
}

// --- isSelected / toggleSelection / clearSelection / hasSelection ---

func TestSelectionOperations(t *testing.T) {
	m := Model{
		selectedItems: make(map[string]bool),
	}
	podA := model.Item{Name: "pod-a", Namespace: "ns-1"}
	podB := model.Item{Name: "pod-b", Namespace: "ns-2"}

	assert.False(t, m.isSelected(podA))
	assert.False(t, m.hasSelection())

	m.toggleSelection(podA)
	assert.True(t, m.isSelected(podA))
	assert.False(t, m.isSelected(podB))
	assert.True(t, m.hasSelection())

	m.toggleSelection(podB)
	assert.True(t, m.isSelected(podA))
	assert.True(t, m.isSelected(podB))

	// Toggle off.
	m.toggleSelection(podA)
	assert.False(t, m.isSelected(podA))
	assert.True(t, m.isSelected(podB))

	m.clearSelection()
	assert.False(t, m.isSelected(podB))
	assert.False(t, m.hasSelection())
	assert.Equal(t, -1, m.selectionAnchor)
}

// --- selectedItemsList ---

func TestSelectedItemsList(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResources},
		middleItems: []model.Item{
			{Name: "pod-a", Namespace: "ns-1"},
			{Name: "pod-b", Namespace: "ns-2"},
			{Name: "pod-c", Namespace: "ns-3"},
		},
		selectedItems: make(map[string]bool),
	}

	m.toggleSelection(m.middleItems[0])
	m.toggleSelection(m.middleItems[2])

	selected := m.selectedItemsList()
	assert.Len(t, selected, 2)
	assert.Equal(t, "pod-a", selected[0].Name)
	assert.Equal(t, "pod-c", selected[1].Name)
}

// --- visibleMiddleItems ---

func TestVisibleMiddleItemsNoFilter(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResources},
		middleItems: []model.Item{
			{Name: "pod-a"},
			{Name: "pod-b"},
		},
	}
	visible := m.visibleMiddleItems()
	assert.Len(t, visible, 2)
}

func TestVisibleMiddleItemsWithFilter(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResources},
		middleItems: []model.Item{
			{Name: "nginx-pod"},
			{Name: "redis-pod"},
			{Name: "nginx-deployment"},
		},
		filterText: "nginx",
	}
	visible := m.visibleMiddleItems()
	assert.Len(t, visible, 2)
	assert.Equal(t, "nginx-pod", visible[0].Name)
	assert.Equal(t, "nginx-deployment", visible[1].Name)
}

func TestVisibleMiddleItemsFilterByNamespace(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResources},
		middleItems: []model.Item{
			{Name: "pod-a", Namespace: "production"},
			{Name: "pod-b", Namespace: "staging"},
		},
		filterText: "production",
	}
	visible := m.visibleMiddleItems()
	assert.Len(t, visible, 1)
	assert.Equal(t, "pod-a", visible[0].Name)
}

func TestVisibleMiddleItemsCollapsedGroups(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResourceTypes},
		middleItems: []model.Item{
			{Name: "Pods", Category: "Workloads"},
			{Name: "Deployments", Category: "Workloads"},
			{Name: "Services", Category: "Networking"},
			{Name: "Ingresses", Category: "Networking"},
		},
		expandedGroup:     "Workloads",
		allGroupsExpanded: false,
	}
	visible := m.visibleMiddleItems()
	// Workloads should be expanded (2 items), Networking should be collapsed (1 header).
	assert.Len(t, visible, 3)
	assert.Equal(t, "Pods", visible[0].Name)
	assert.Equal(t, "Deployments", visible[1].Name)
	assert.Equal(t, "Networking", visible[2].Name)
	assert.Equal(t, "__collapsed_group__", visible[2].Kind)
}

func TestVisibleMiddleItemsAllGroupsExpanded(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResourceTypes},
		middleItems: []model.Item{
			{Name: "Pods", Category: "Workloads"},
			{Name: "Services", Category: "Networking"},
		},
		allGroupsExpanded: true,
	}
	visible := m.visibleMiddleItems()
	assert.Len(t, visible, 2)
}

func TestVisibleMiddleItemsCategoryFilterIncludesAll(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResources},
		middleItems: []model.Item{
			{Name: "Pods", Category: "Workloads"},
			{Name: "Deployments", Category: "Workloads"},
			{Name: "Services", Category: "Networking"},
		},
		filterText: "Workloads",
	}
	visible := m.visibleMiddleItems()
	// Category match should include all items in the "Workloads" category.
	assert.Len(t, visible, 2)
}

// --- categoryCounts ---

func TestCategoryCounts(t *testing.T) {
	m := Model{
		middleItems: []model.Item{
			{Name: "a", Category: "Workloads"},
			{Name: "b", Category: "Workloads"},
			{Name: "c", Category: "Networking"},
			{Name: "d"},
		},
	}
	counts := m.categoryCounts()
	assert.Equal(t, 2, counts["Workloads"])
	assert.Equal(t, 1, counts["Networking"])
	assert.Equal(t, 0, counts[""])
}

// --- parentIndex ---

func TestParentIndex(t *testing.T) {
	t.Run("LevelResourceTypes returns context match", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				Level:   model.LevelResourceTypes,
				Context: "prod-cluster",
			},
			leftItems: []model.Item{
				{Name: "dev-cluster"},
				{Name: "prod-cluster"},
			},
		}
		assert.Equal(t, 1, m.parentIndex())
	})

	t.Run("LevelResources returns resource type match", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				Level:        model.LevelResources,
				ResourceType: model.ResourceTypeEntry{DisplayName: "Deployments"},
			},
			leftItems: []model.Item{
				{Name: "Pods"},
				{Name: "Deployments"},
				{Name: "Services"},
			},
		}
		assert.Equal(t, 1, m.parentIndex())
	})

	t.Run("LevelOwned returns resource name match", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				Level:        model.LevelOwned,
				ResourceName: "my-deploy",
			},
			leftItems: []model.Item{
				{Name: "other-deploy"},
				{Name: "my-deploy"},
			},
		}
		assert.Equal(t, 1, m.parentIndex())
	})

	t.Run("LevelContainers returns owned name match", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				Level:     model.LevelContainers,
				OwnedName: "my-pod",
			},
			leftItems: []model.Item{
				{Name: "my-pod"},
			},
		}
		assert.Equal(t, 0, m.parentIndex())
	})

	t.Run("LevelClusters returns -1", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelClusters},
		}
		assert.Equal(t, -1, m.parentIndex())
	})

	t.Run("no match returns -1", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				Level:   model.LevelResourceTypes,
				Context: "nonexistent",
			},
			leftItems: []model.Item{{Name: "prod"}},
		}
		assert.Equal(t, -1, m.parentIndex())
	})
}

// --- carryOverMetricsColumns ---

func TestCarryOverMetricsColumns(t *testing.T) {
	t.Run("carries over CPU and MEM from old items", func(t *testing.T) {
		m := Model{
			middleItems: []model.Item{
				{
					Name: "pod-a", Namespace: "ns-1",
					Columns: []model.KeyValue{
						{Key: "CPU", Value: "100m"},
						{Key: "MEM", Value: "256Mi"},
						{Key: "CPU/R", Value: "200m"},
						{Key: "Other", Value: "val"},
					},
				},
			},
		}
		newItems := []model.Item{
			{
				Name: "pod-a", Namespace: "ns-1",
				Columns: []model.KeyValue{
					{Key: "Status", Value: "Running"},
				},
			},
		}
		m.carryOverMetricsColumns(newItems)
		// Should have metrics columns prepended, plus the non-metrics "Status".
		assert.Len(t, newItems[0].Columns, 4)
		assert.Equal(t, "CPU", newItems[0].Columns[0].Key)
		assert.Equal(t, "MEM", newItems[0].Columns[1].Key)
		assert.Equal(t, "CPU/R", newItems[0].Columns[2].Key)
		assert.Equal(t, "Status", newItems[0].Columns[3].Key)
	})

	t.Run("no carryover when old items have no real usage", func(t *testing.T) {
		m := Model{
			middleItems: []model.Item{
				{
					Name: "pod-a", Namespace: "ns-1",
					Columns: []model.KeyValue{
						{Key: "CPU", Value: "0"},
						{Key: "MEM", Value: "0"},
					},
				},
			},
		}
		newItems := []model.Item{
			{
				Name: "pod-a", Namespace: "ns-1",
				Columns: []model.KeyValue{
					{Key: "Status", Value: "Running"},
				},
			},
		}
		m.carryOverMetricsColumns(newItems)
		assert.Len(t, newItems[0].Columns, 1)
		assert.Equal(t, "Status", newItems[0].Columns[0].Key)
	})

	t.Run("no carryover for unmatched items", func(t *testing.T) {
		m := Model{
			middleItems: []model.Item{
				{
					Name: "pod-x", Namespace: "ns-1",
					Columns: []model.KeyValue{
						{Key: "CPU", Value: "100m"},
						{Key: "MEM", Value: "256Mi"},
					},
				},
			},
		}
		newItems := []model.Item{
			{
				Name: "pod-a", Namespace: "ns-1",
				Columns: []model.KeyValue{
					{Key: "Status", Value: "Running"},
				},
			},
		}
		m.carryOverMetricsColumns(newItems)
		assert.Len(t, newItems[0].Columns, 1)
	})

	t.Run("empty old items does nothing", func(t *testing.T) {
		m := Model{}
		newItems := []model.Item{
			{Name: "pod-a", Columns: []model.KeyValue{{Key: "Status", Value: "Running"}}},
		}
		m.carryOverMetricsColumns(newItems)
		assert.Len(t, newItems[0].Columns, 1)
	})
}

// --- filteredOverlayItems ---

func TestFilteredOverlayItems(t *testing.T) {
	m := Model{
		overlayItems: []model.Item{
			{Name: "default"},
			{Name: "kube-system"},
			{Name: "production"},
		},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		m.overlayFilter = TextInput{}
		result := m.filteredOverlayItems()
		assert.Len(t, result, 3)
	})

	t.Run("filter matches subset", func(t *testing.T) {
		m.overlayFilter = TextInput{Value: "kube"}
		result := m.filteredOverlayItems()
		assert.Len(t, result, 1)
		assert.Equal(t, "kube-system", result[0].Name)
	})

	t.Run("case insensitive filter", func(t *testing.T) {
		m.overlayFilter = TextInput{Value: "DEFAULT"}
		result := m.filteredOverlayItems()
		assert.Len(t, result, 1)
		assert.Equal(t, "default", result[0].Name)
	})

	t.Run("no match returns empty", func(t *testing.T) {
		m.overlayFilter = TextInput{Value: "nonexistent"}
		result := m.filteredOverlayItems()
		assert.Empty(t, result)
	})
}

// --- filteredExplainRecursiveResults ---

func TestFilteredExplainRecursiveResults(t *testing.T) {
	m := Model{
		explainRecursiveResults: []model.ExplainField{
			{Name: "spec", Path: "spec"},
			{Name: "metadata", Path: "metadata"},
			{Name: "status", Path: "status"},
		},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		m.explainRecursiveFilter = TextInput{}
		result := m.filteredExplainRecursiveResults()
		assert.Len(t, result, 3)
	})

	t.Run("filter by name", func(t *testing.T) {
		m.explainRecursiveFilter = TextInput{Value: "spec"}
		result := m.filteredExplainRecursiveResults()
		assert.Len(t, result, 1)
		assert.Equal(t, "spec", result[0].Name)
	})

	t.Run("filter by path", func(t *testing.T) {
		m.explainRecursiveFilter = TextInput{Value: "meta"}
		result := m.filteredExplainRecursiveResults()
		assert.Len(t, result, 1)
		assert.Equal(t, "metadata", result[0].Name)
	})
}

// --- filteredLogPodItems ---

func TestFilteredLogPodItems(t *testing.T) {
	m := Model{
		overlayItems: []model.Item{
			{Name: "nginx-pod-1"},
			{Name: "redis-pod-1"},
			{Name: "nginx-pod-2"},
		},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		m.logPodFilterText = ""
		result := m.filteredLogPodItems()
		assert.Len(t, result, 3)
	})

	t.Run("filter matches subset", func(t *testing.T) {
		m.logPodFilterText = "nginx"
		result := m.filteredLogPodItems()
		assert.Len(t, result, 2)
	})
}

// --- clampAllCursors ---

func TestClampAllCursors(t *testing.T) {
	m := Model{
		nav: model.NavigationState{Level: model.LevelResources},
		middleItems: []model.Item{
			{Name: "a"}, {Name: "b"},
		},
	}
	m.setCursor(10)
	m.clampAllCursors()
	assert.Equal(t, 1, m.cursor())
}

// --- syncExpandedGroup ---

func TestSyncExpandedGroup(t *testing.T) {
	t.Run("updates expanded group when cursor moves to different category", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResourceTypes},
			middleItems: []model.Item{
				{Name: "Pods", Category: "Workloads"},
				{Name: "Deployments", Category: "Workloads"},
				{Name: "Services", Category: "Networking"},
				{Name: "Ingresses", Category: "Networking"},
			},
			expandedGroup:     "Workloads",
			allGroupsExpanded: false,
		}
		// Move cursor to a collapsed group header
		// visibleMiddleItems with expandedGroup="Workloads" shows:
		// 0: Pods(Workloads), 1: Deployments(Workloads), 2: Networking(collapsed header)
		m.setCursor(2)
		m.syncExpandedGroup()
		assert.Equal(t, "Networking", m.expandedGroup)
	})

	t.Run("no-op when allGroupsExpanded", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResourceTypes},
			middleItems: []model.Item{
				{Name: "Pods", Category: "Workloads"},
			},
			expandedGroup:     "Workloads",
			allGroupsExpanded: true,
		}
		m.syncExpandedGroup()
		assert.Equal(t, "Workloads", m.expandedGroup)
	})

	t.Run("no-op when not at LevelResourceTypes", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{
				{Name: "Pods", Category: "Workloads"},
			},
			expandedGroup: "Workloads",
		}
		m.syncExpandedGroup()
		assert.Equal(t, "Workloads", m.expandedGroup)
	})
}

// --- filteredLogContainerItems ---

func TestFilteredLogContainerItems(t *testing.T) {
	m := Model{
		overlayItems: []model.Item{
			{Name: "All Containers", Status: "all"},
			{Name: "nginx"},
			{Name: "sidecar"},
		},
	}

	t.Run("no filter returns all", func(t *testing.T) {
		m.logContainerFilterText = ""
		result := m.filteredLogContainerItems()
		assert.Len(t, result, 3)
	})

	t.Run("filter matches plus always includes All Containers", func(t *testing.T) {
		m.logContainerFilterText = "nginx"
		result := m.filteredLogContainerItems()
		assert.Len(t, result, 2)
		assert.Equal(t, "All Containers", result[0].Name)
		assert.Equal(t, "nginx", result[1].Name)
	})
}
