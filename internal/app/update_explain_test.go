package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- explainJumpToMatch ---

func TestExplainJumpToMatch(t *testing.T) {
	fields := []model.ExplainField{
		{Name: "apiVersion"},
		{Name: "kind"},
		{Name: "metadata"},
		{Name: "spec"},
		{Name: "status"},
	}

	t.Run("forward finds match", func(t *testing.T) {
		m := Model{
			height:        30,
			explainFields: fields,
			explainCursor: 0,
		}
		found := m.explainJumpToMatch("spec", 1, true)
		assert.True(t, found)
		assert.Equal(t, 3, m.explainCursor)
	})

	t.Run("forward wraps around", func(t *testing.T) {
		m := Model{
			height:        30,
			explainFields: fields,
			explainCursor: 4,
		}
		found := m.explainJumpToMatch("api", 4, true)
		assert.True(t, found)
		assert.Equal(t, 0, m.explainCursor)
	})

	t.Run("backward finds match", func(t *testing.T) {
		m := Model{
			height:        30,
			explainFields: fields,
			explainCursor: 4,
		}
		found := m.explainJumpToMatch("kind", 3, false)
		assert.True(t, found)
		assert.Equal(t, 1, m.explainCursor)
	})

	t.Run("backward wraps around", func(t *testing.T) {
		m := Model{
			height:        30,
			explainFields: fields,
			explainCursor: 0,
		}
		found := m.explainJumpToMatch("status", 0, false)
		assert.True(t, found)
		assert.Equal(t, 4, m.explainCursor)
	})

	t.Run("empty query returns false", func(t *testing.T) {
		m := Model{
			height:        30,
			explainFields: fields,
		}
		found := m.explainJumpToMatch("", 0, true)
		assert.False(t, found)
	})

	t.Run("no fields returns false", func(t *testing.T) {
		m := Model{height: 30}
		found := m.explainJumpToMatch("test", 0, true)
		assert.False(t, found)
	})

	t.Run("no match returns false", func(t *testing.T) {
		m := Model{
			height:        30,
			explainFields: fields,
		}
		found := m.explainJumpToMatch("nonexistent", 0, true)
		assert.False(t, found)
	})

	t.Run("case insensitive matching", func(t *testing.T) {
		m := Model{
			height:        30,
			explainFields: fields,
		}
		found := m.explainJumpToMatch("SPEC", 0, true)
		assert.True(t, found)
		assert.Equal(t, 3, m.explainCursor)
	})

	t.Run("adjusts scroll when cursor above viewport", func(t *testing.T) {
		m := Model{
			height:        30,
			explainFields: fields,
			explainScroll: 4,
		}
		found := m.explainJumpToMatch("api", 4, true)
		assert.True(t, found)
		assert.Equal(t, 0, m.explainScroll)
	})
}
