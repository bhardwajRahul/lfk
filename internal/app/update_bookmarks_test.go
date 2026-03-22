package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- filteredBookmarks ---

func TestFilteredBookmarks(t *testing.T) {
	bookmarks := []model.Bookmark{
		{Name: "prod > Deployments", Slot: "a"},
		{Name: "staging > Pods", Slot: "b"},
		{Name: "prod > Services", Slot: "c"},
	}

	t.Run("empty filter returns all", func(t *testing.T) {
		m := Model{
			bookmarks:      bookmarks,
			bookmarkFilter: TextInput{},
		}
		result := m.filteredBookmarks()
		assert.Len(t, result, 3)
	})

	t.Run("filter by context", func(t *testing.T) {
		m := Model{
			bookmarks:      bookmarks,
			bookmarkFilter: TextInput{Value: "prod"},
		}
		result := m.filteredBookmarks()
		assert.Len(t, result, 2)
		assert.Equal(t, "a", result[0].Slot)
		assert.Equal(t, "c", result[1].Slot)
	})

	t.Run("filter by resource type", func(t *testing.T) {
		m := Model{
			bookmarks:      bookmarks,
			bookmarkFilter: TextInput{Value: "pods"},
		}
		result := m.filteredBookmarks()
		assert.Len(t, result, 1)
		assert.Equal(t, "b", result[0].Slot)
	})

	t.Run("case insensitive filter", func(t *testing.T) {
		m := Model{
			bookmarks:      bookmarks,
			bookmarkFilter: TextInput{Value: "DEPLOYMENTS"},
		}
		result := m.filteredBookmarks()
		assert.Len(t, result, 1)
		assert.Equal(t, "a", result[0].Slot)
	})

	t.Run("no match returns empty", func(t *testing.T) {
		m := Model{
			bookmarks:      bookmarks,
			bookmarkFilter: TextInput{Value: "nonexistent"},
		}
		result := m.filteredBookmarks()
		assert.Empty(t, result)
	})

	t.Run("nil bookmarks returns nil", func(t *testing.T) {
		m := Model{
			bookmarkFilter: TextInput{Value: "prod"},
		}
		result := m.filteredBookmarks()
		assert.Empty(t, result)
	})
}

// --- contextInList ---

func TestContextInList(t *testing.T) {
	items := []model.Item{
		{Name: "cluster-a"},
		{Name: "cluster-b"},
		{Name: "cluster-c"},
	}

	assert.True(t, contextInList("cluster-b", items))
	assert.False(t, contextInList("nonexistent", items))
	assert.False(t, contextInList("cluster-a", nil))
	assert.False(t, contextInList("", items))
}

// --- applySessionNamespaces ---

func TestApplySessionNamespaces(t *testing.T) {
	t.Run("all namespaces mode", func(t *testing.T) {
		m := Model{namespace: "old"}
		applySessionNamespaces(&m, true, "", nil)
		assert.True(t, m.allNamespaces)
		assert.Nil(t, m.selectedNamespaces)
	})

	t.Run("single namespace", func(t *testing.T) {
		m := Model{}
		applySessionNamespaces(&m, false, "production", nil)
		assert.Equal(t, "production", m.namespace)
		assert.False(t, m.allNamespaces)
	})

	t.Run("multiple namespaces", func(t *testing.T) {
		m := Model{}
		applySessionNamespaces(&m, false, "ns-1", []string{"ns-1", "ns-2", "ns-3"})
		assert.Equal(t, "ns-1", m.namespace)
		assert.Len(t, m.selectedNamespaces, 3)
		assert.True(t, m.selectedNamespaces["ns-1"])
		assert.True(t, m.selectedNamespaces["ns-2"])
		assert.True(t, m.selectedNamespaces["ns-3"])
	})
}
