package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- yamlTitle ---

func TestYamlTitle(t *testing.T) {
	tests := []struct {
		name     string
		model    Model
		expected string
	}{
		{
			name: "LevelResources with selected item",
			model: Model{
				nav: model.NavigationState{
					Level: model.LevelResources,
				},
				namespace: "default",
				middleItems: []model.Item{
					{Name: "my-pod"},
				},
			},
			expected: "YAML: default/my-pod",
		},
		{
			name: "LevelOwned with selected item",
			model: Model{
				nav: model.NavigationState{
					Level: model.LevelOwned,
				},
				namespace: "prod",
				middleItems: []model.Item{
					{Name: "my-deploy-pod-abc"},
				},
			},
			expected: "YAML: prod/my-deploy-pod-abc",
		},
		{
			name: "LevelContainers uses OwnedName",
			model: Model{
				nav: model.NavigationState{
					Level:     model.LevelContainers,
					OwnedName: "my-pod-xyz",
				},
				namespace: "staging",
			},
			expected: "YAML: staging/my-pod-xyz",
		},
		{
			name: "LevelResources with no items",
			model: Model{
				nav: model.NavigationState{
					Level: model.LevelResources,
				},
				namespace: "default",
			},
			expected: "YAML",
		},
		{
			name: "LevelClusters returns generic",
			model: Model{
				nav: model.NavigationState{
					Level: model.LevelClusters,
				},
			},
			expected: "YAML",
		},
		{
			name: "LevelResourceTypes returns generic",
			model: Model{
				nav: model.NavigationState{
					Level: model.LevelResourceTypes,
				},
			},
			expected: "YAML",
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.model.yamlTitle())
		})
	}
}

// --- yamlCursorCol ---

func TestYamlCursorCol(t *testing.T) {
	m := Model{yamlVisualCurCol: 15}
	assert.Equal(t, 15, m.yamlCursorCol())

	m.yamlVisualCurCol = 0
	assert.Equal(t, 0, m.yamlCursorCol())
}

// --- maskYAMLIfSecret ---

func TestMaskYAMLIfSecret(t *testing.T) {
	t.Run("non-secret resource returns yaml unchanged", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "ConfigMap"},
			},
		}
		yaml := "apiVersion: v1\nkind: ConfigMap\ndata:\n  key: value"
		assert.Equal(t, yaml, m.maskYAMLIfSecret(yaml))
	})

	t.Run("secret with showSecretValues true returns yaml unchanged", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "Secret"},
			},
			showSecretValues: true,
		}
		yaml := "apiVersion: v1\nkind: Secret\ndata:\n  password: c2VjcmV0"
		assert.Equal(t, yaml, m.maskYAMLIfSecret(yaml))
	})

	t.Run("secret with showSecretValues false returns masked yaml", func(t *testing.T) {
		m := Model{
			nav: model.NavigationState{
				ResourceType: model.ResourceTypeEntry{Kind: "Secret"},
			},
			showSecretValues: false,
		}
		yaml := "apiVersion: v1\nkind: Secret\ndata:\n  password: c2VjcmV0"
		result := m.maskYAMLIfSecret(yaml)
		// The result should be different from the input (masked).
		assert.NotEqual(t, yaml, result)
	})
}
