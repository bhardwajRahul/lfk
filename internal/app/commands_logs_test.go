package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- matchesContainerFilter ---

func TestMatchesContainerFilter(t *testing.T) {
	tests := []struct {
		name       string
		line       string
		containers []string
		expected   bool
	}{
		{
			name:       "matching container",
			line:       "[pod/my-pod/nginx] log line here",
			containers: []string{"nginx", "sidecar"},
			expected:   true,
		},
		{
			name:       "non-matching container",
			line:       "[pod/my-pod/nginx] log line here",
			containers: []string{"sidecar"},
			expected:   false,
		},
		{
			name:       "no prefix passes through",
			line:       "plain log line without prefix",
			containers: []string{"nginx"},
			expected:   true,
		},
		{
			name:       "empty line passes through",
			line:       "",
			containers: []string{"nginx"},
			expected:   true,
		},
		{
			name:       "bracket but no closing bracket passes through",
			line:       "[incomplete prefix without close",
			containers: []string{"nginx"},
			expected:   true,
		},
		{
			name:       "bracket with no slash passes through",
			line:       "[noslash] content",
			containers: []string{"noslash"},
			expected:   true,
		},
		{
			name:       "multiple containers all match",
			line:       "[pod/my-pod/sidecar] some log",
			containers: []string{"nginx", "sidecar"},
			expected:   true,
		},
		{
			name:       "empty container filter means none match",
			line:       "[pod/my-pod/nginx] log",
			containers: []string{},
			expected:   false,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, matchesContainerFilter(tt.line, tt.containers))
		})
	}
}

// --- sanitizeFilename ---

func TestSanitizeFilename(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{"simple name", "pod-name", "pod-name"},
		{"with slashes", "ns/pod/container", "ns_pod_container"},
		{"with backslash", "path\\to\\file", "path_to_file"},
		{"with colons", "host:port", "host_port"},
		{"with spaces", "my pod name", "my_pod_name"},
		{"mixed special chars", "ns/pod:8080 name", "ns_pod_8080_name"},
		{"empty string", "", ""},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, sanitizeFilename(tt.input))
		})
	}
}
