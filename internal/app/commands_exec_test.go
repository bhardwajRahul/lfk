package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- randomSuffix ---

func TestRandomSuffix(t *testing.T) {
	t.Run("returns correct length", func(t *testing.T) {
		s := randomSuffix(5)
		assert.Len(t, s, 5)
	})

	t.Run("zero length returns empty", func(t *testing.T) {
		s := randomSuffix(0)
		assert.Empty(t, s)
	})

	t.Run("contains only valid characters", func(t *testing.T) {
		const validChars = "abcdefghijklmnopqrstuvwxyz0123456789"
		s := randomSuffix(100)
		for _, c := range s {
			assert.Contains(t, validChars, string(c))
		}
	})

	t.Run("different calls produce different results", func(t *testing.T) {
		// With 36^10 possible values, collision probability is negligible.
		s1 := randomSuffix(10)
		s2 := randomSuffix(10)
		assert.NotEqual(t, s1, s2)
	})
}
