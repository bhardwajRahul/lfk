package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- itemIndexFromDisplayLine ---

func TestItemIndexFromDisplayLine(t *testing.T) {
	t.Run("single category with items", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{
				{Name: "pod-a"},
				{Name: "pod-b"},
				{Name: "pod-c"},
			},
		}
		// No categories, so display lines map 1:1 to items.
		assert.Equal(t, 0, m.itemIndexFromDisplayLine(0))
		assert.Equal(t, 1, m.itemIndexFromDisplayLine(1))
		assert.Equal(t, 2, m.itemIndexFromDisplayLine(2))
	})

	t.Run("display line out of range returns -1", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
			middleItems: []model.Item{
				{Name: "pod-a"},
			},
		}
		assert.Equal(t, -1, m.itemIndexFromDisplayLine(100))
	})

	t.Run("empty items returns -1", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResources},
		}
		assert.Equal(t, -1, m.itemIndexFromDisplayLine(0))
	})

	t.Run("items with categories include headers and separators", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{Level: model.LevelResourceTypes},
			middleItems: []model.Item{
				{Name: "Pods", Category: "Workloads"},
				{Name: "Deployments", Category: "Workloads"},
				{Name: "Services", Category: "Networking"},
			},
			allGroupsExpanded: true,
		}
		// Display lines with categories:
		// 0: "Workloads" header
		// 1: Pods
		// 2: Deployments
		// 3: separator
		// 4: "Networking" header
		// 5: Services
		idx := m.itemIndexFromDisplayLine(1) // Pods
		assert.Equal(t, 0, idx)
	})
}
