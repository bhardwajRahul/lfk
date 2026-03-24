package model

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"gopkg.in/yaml.v3"
)

func TestBuiltinTemplatesCount(t *testing.T) {
	templates := BuiltinTemplates()
	// 8 Workloads + 3 Networking + 4 Config + 3 Storage + 5 Access Control
	// + 1 Monitoring + 1 Cluster + 1 Custom = 26
	assert.Len(t, templates, 26)
}

func TestBuiltinTemplatesAllFieldsNonEmpty(t *testing.T) {
	for _, tmpl := range BuiltinTemplates() {
		t.Run(tmpl.Name, func(t *testing.T) {
			assert.NotEmpty(t, tmpl.Name, "template Name must not be empty")
			assert.NotEmpty(t, tmpl.Description, "template Description must not be empty")
			assert.NotEmpty(t, tmpl.Category, "template Category must not be empty")
			assert.NotEmpty(t, tmpl.YAML, "template YAML must not be empty")
		})
	}
}

func TestBuiltinTemplatesExpectedNames(t *testing.T) {
	templates := BuiltinTemplates()
	names := make([]string, 0, len(templates))
	for _, tmpl := range templates {
		names = append(names, tmpl.Name)
	}

	expected := []string{
		"Pod",
		"Deployment",
		"ReplicaSet",
		"StatefulSet",
		"DaemonSet",
		"Job",
		"CronJob",
		"HorizontalPodAutoscaler",
		"Service",
		"Ingress",
		"NetworkPolicy",
		"ConfigMap",
		"Secret",
		"ResourceQuota",
		"LimitRange",
		"PersistentVolumeClaim",
		"PersistentVolume",
		"StorageClass",
		"ServiceAccount",
		"Role",
		"RoleBinding",
		"ClusterRole",
		"ClusterRoleBinding",
		"ServiceMonitor",
		"Namespace",
		"Custom Resource",
	}

	assert.Equal(t, expected, names, "templates must appear in the expected order")
}

func TestBuiltinTemplatesCustomResourceIsLast(t *testing.T) {
	templates := BuiltinTemplates()
	require.NotEmpty(t, templates)
	last := templates[len(templates)-1]
	assert.Equal(t, "Custom Resource", last.Name)
	assert.Equal(t, "Custom", last.Category)
}

func TestBuiltinTemplatesYAMLIsValid(t *testing.T) {
	for _, tmpl := range BuiltinTemplates() {
		t.Run(tmpl.Name, func(t *testing.T) {
			var parsed map[string]any
			err := yaml.Unmarshal([]byte(tmpl.YAML), &parsed)
			require.NoError(t, err, "YAML for template %q must be valid", tmpl.Name)
			assert.NotEmpty(t, parsed, "parsed YAML for template %q must not be empty", tmpl.Name)
		})
	}
}

func TestBuiltinTemplatesCategoryOrder(t *testing.T) {
	templates := BuiltinTemplates()

	// Build ordered list of categories as they appear (preserving first occurrence order).
	seen := make(map[string]bool)
	var categories []string
	for _, tmpl := range templates {
		if !seen[tmpl.Category] {
			seen[tmpl.Category] = true
			categories = append(categories, tmpl.Category)
		}
	}

	expected := []string{
		"Workloads",
		"Networking",
		"Config",
		"Storage",
		"Access Control",
		"Monitoring",
		"Cluster",
		"Custom",
	}

	assert.Equal(t, expected, categories, "categories must appear in the expected order")
}

func TestBuiltinTemplatesClusterScopedNoNamespace(t *testing.T) {
	// Cluster-scoped resources must not have namespace in their YAML.
	clusterScoped := map[string]bool{
		"PersistentVolume":   true,
		"StorageClass":       true,
		"ClusterRole":        true,
		"ClusterRoleBinding": true,
		"Namespace":          true,
	}

	for _, tmpl := range BuiltinTemplates() {
		if !clusterScoped[tmpl.Name] {
			continue
		}
		t.Run(tmpl.Name, func(t *testing.T) {
			var parsed map[string]any
			err := yaml.Unmarshal([]byte(tmpl.YAML), &parsed)
			require.NoError(t, err)

			metadata, ok := parsed["metadata"].(map[string]any)
			if ok {
				_, hasNS := metadata["namespace"]
				assert.False(t, hasNS,
					"cluster-scoped resource %q must not have namespace in metadata", tmpl.Name)
			}
		})
	}
}

func TestBuiltinTemplatesNamespacedHaveNamespace(t *testing.T) {
	// Cluster-scoped resources that should NOT have namespace.
	clusterScoped := map[string]bool{
		"PersistentVolume":   true,
		"StorageClass":       true,
		"ClusterRole":        true,
		"ClusterRoleBinding": true,
		"Namespace":          true,
	}
	// Custom Resource is a special case - it has NAMESPACE as a generic placeholder.
	skip := map[string]bool{
		"Custom Resource": true,
	}

	for _, tmpl := range BuiltinTemplates() {
		if clusterScoped[tmpl.Name] || skip[tmpl.Name] {
			continue
		}
		t.Run(tmpl.Name, func(t *testing.T) {
			var parsed map[string]any
			err := yaml.Unmarshal([]byte(tmpl.YAML), &parsed)
			require.NoError(t, err)

			metadata, ok := parsed["metadata"].(map[string]any)
			require.True(t, ok, "template %q must have metadata", tmpl.Name)

			ns, hasNS := metadata["namespace"]
			assert.True(t, hasNS,
				"namespaced resource %q must have namespace in metadata", tmpl.Name)
			assert.Equal(t, "NAMESPACE", ns,
				"namespaced resource %q must use NAMESPACE placeholder", tmpl.Name)
		})
	}
}
