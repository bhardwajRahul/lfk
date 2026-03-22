package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- maskYAMLIfSecret ---

func TestMaskYAMLIfSecretRightColumn(t *testing.T) {
	t.Run("non-secret returns unchanged", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "ConfigMap"},
			},
			showSecretValues: false,
		}
		yaml := "data:\n  key: value"
		assert.Equal(t, yaml, m.maskYAMLIfSecret(yaml))
	})

	t.Run("secret with show values returns unchanged", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "Secret"},
			},
			showSecretValues: true,
		}
		yaml := "data:\n  key: c2VjcmV0"
		assert.Equal(t, yaml, m.maskYAMLIfSecret(yaml))
	})

	t.Run("secret with hidden values masks content", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "Secret"},
			},
			showSecretValues: false,
		}
		yaml := "data:\n  key: c2VjcmV0"
		result := m.maskYAMLIfSecret(yaml)
		assert.NotEqual(t, yaml, result)
	})
}

// --- renderFallbackYAML ---

func TestRenderFallbackYAML(t *testing.T) {
	t.Run("no YAML shows placeholder", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "ConfigMap"},
			},
		}
		result := m.renderFallbackYAML(80, 20)
		assert.Contains(t, result, "No preview")
	})

	t.Run("previewYAML used first", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "ConfigMap"},
			},
			previewYAML: "apiVersion: v1\nkind: ConfigMap\n",
			yamlContent: "fallback: content\n",
		}
		result := m.renderFallbackYAML(80, 20)
		assert.Contains(t, result, "apiVersion")
	})

	t.Run("yamlContent used as fallback", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "ConfigMap"},
			},
			yamlContent: "fallback: content\n",
		}
		result := m.renderFallbackYAML(80, 20)
		assert.Contains(t, result, "fallback")
	})
}
