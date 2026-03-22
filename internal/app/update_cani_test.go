package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- filterAllowedResources ---

func TestFilterAllowedResources(t *testing.T) {
	resources := []model.CanIResource{
		{
			Resource: "pods",
			Kind:     "Pod",
			Verbs:    map[string]bool{"get": true, "list": true, "delete": false},
		},
		{
			Resource: "secrets",
			Kind:     "Secret",
			Verbs:    map[string]bool{"get": false, "list": false},
		},
		{
			Resource: "deployments",
			Kind:     "Deployment",
			Verbs:    map[string]bool{"get": true, "create": false},
		},
	}

	filtered := filterAllowedResources(resources)
	assert.Len(t, filtered, 2)
	assert.Equal(t, "pods", filtered[0].Resource)
	assert.Equal(t, "deployments", filtered[1].Resource)
}

func TestFilterAllowedResourcesEmpty(t *testing.T) {
	var resources []model.CanIResource
	filtered := filterAllowedResources(resources)
	assert.Empty(t, filtered)
}

func TestFilterAllowedResourcesAllDenied(t *testing.T) {
	resources := []model.CanIResource{
		{
			Resource: "secrets",
			Verbs:    map[string]bool{"get": false, "list": false},
		},
	}
	filtered := filterAllowedResources(resources)
	assert.Empty(t, filtered)
}

func TestFilterAllowedResourcesEmptyVerbs(t *testing.T) {
	resources := []model.CanIResource{
		{Resource: "configmaps", Verbs: map[string]bool{}},
	}
	filtered := filterAllowedResources(resources)
	assert.Empty(t, filtered)
}

// --- countAllowedResources ---

func TestCountAllowedResources(t *testing.T) {
	resources := []model.CanIResource{
		{Resource: "pods", Verbs: map[string]bool{"get": true}},
		{Resource: "secrets", Verbs: map[string]bool{"get": false}},
		{Resource: "deployments", Verbs: map[string]bool{"list": true, "get": false}},
	}

	assert.Equal(t, 2, countAllowedResources(resources))
}

func TestCountAllowedResourcesNone(t *testing.T) {
	resources := []model.CanIResource{
		{Resource: "secrets", Verbs: map[string]bool{"get": false}},
	}
	assert.Equal(t, 0, countAllowedResources(resources))
}

func TestCountAllowedResourcesEmpty(t *testing.T) {
	assert.Equal(t, 0, countAllowedResources(nil))
}

// --- canIVisibleGroups ---

func TestCanIVisibleGroups(t *testing.T) {
	groups := []model.CanIGroup{
		{Name: "apps"},
		{Name: ""}, // core group
		{Name: "batch"},
		{Name: "networking.k8s.io"},
	}

	t.Run("no query returns all", func(t *testing.T) {
		m := Model{canIGroups: groups}
		indices := m.canIVisibleGroups()
		assert.Len(t, indices, 4)
	})

	t.Run("query filters by group name", func(t *testing.T) {
		m := Model{
			canIGroups:      groups,
			canISearchQuery: "app",
		}
		indices := m.canIVisibleGroups()
		assert.Len(t, indices, 1)
		assert.Equal(t, 0, indices[0])
	})

	t.Run("empty group name matches core", func(t *testing.T) {
		m := Model{
			canIGroups:      groups,
			canISearchQuery: "core",
		}
		indices := m.canIVisibleGroups()
		assert.Len(t, indices, 1)
		assert.Equal(t, 1, indices[0])
	})

	t.Run("case insensitive search", func(t *testing.T) {
		m := Model{
			canIGroups:      groups,
			canISearchQuery: "BATCH",
		}
		indices := m.canIVisibleGroups()
		assert.Len(t, indices, 1)
		assert.Equal(t, 2, indices[0])
	})

	t.Run("active search input overrides query", func(t *testing.T) {
		m := Model{
			canIGroups:       groups,
			canISearchQuery:  "apps", // would match "apps"
			canISearchActive: true,
			canISearchInput:  TextInput{Value: "batch"}, // overrides to "batch"
		}
		indices := m.canIVisibleGroups()
		assert.Len(t, indices, 1)
		assert.Equal(t, 2, indices[0])
	})

	t.Run("no match returns empty", func(t *testing.T) {
		m := Model{
			canIGroups:      groups,
			canISearchQuery: "nonexistent",
		}
		indices := m.canIVisibleGroups()
		assert.Empty(t, indices)
	})
}

// --- canIVisibleLines ---

func TestCanIVisibleLines(t *testing.T) {
	t.Run("normal terminal size", func(t *testing.T) {
		m := Model{height: 40}
		lines := m.canIVisibleLines()
		assert.Greater(t, lines, 0)
	})

	t.Run("small terminal", func(t *testing.T) {
		m := Model{height: 10}
		lines := m.canIVisibleLines()
		assert.Greater(t, lines, 0)
	})

	t.Run("very small terminal", func(t *testing.T) {
		m := Model{height: 5}
		lines := m.canIVisibleLines()
		assert.Greater(t, lines, 0)
	})
}
