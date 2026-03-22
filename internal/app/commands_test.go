package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- shellQuote ---

func TestShellQuote(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "simple word",
			input:    "hello",
			expected: "'hello'",
		},
		{
			name:     "empty string",
			input:    "",
			expected: "''",
		},
		{
			name:     "string with spaces",
			input:    "hello world",
			expected: "'hello world'",
		},
		{
			name:     "string with single quote",
			input:    "it's",
			expected: "'it'\"'\"'s'",
		},
		{
			name:     "string with multiple single quotes",
			input:    "it's a 'test'",
			expected: "'it'\"'\"'s a '\"'\"'test'\"'\"''",
		},
		{
			name:     "string with double quotes",
			input:    `say "hello"`,
			expected: `'say "hello"'`,
		},
		{
			name:     "string with special characters",
			input:    "a$b&c|d;e",
			expected: "'a$b&c|d;e'",
		},
		{
			name:     "string with newline",
			input:    "line1\nline2",
			expected: "'line1\nline2'",
		},
		{
			name:     "string with backslash",
			input:    `path\to\file`,
			expected: `'path\to\file'`,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := shellQuote(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- isKubectlCommand ---

func TestIsKubectlCommand(t *testing.T) {
	tests := []struct {
		name     string
		input    string
		expected bool
	}{
		// Explicit "kubectl" prefix cases.
		{
			name:     "starts with kubectl space",
			input:    "kubectl get pods",
			expected: true,
		},
		{
			name:     "just kubectl",
			input:    "kubectl",
			expected: true,
		},
		{
			name:     "kubectl with leading whitespace",
			input:    "  kubectl get pods",
			expected: true,
		},
		// Known subcommand cases.
		{
			name:     "get subcommand",
			input:    "get pods -n kube-system",
			expected: true,
		},
		{
			name:     "describe subcommand",
			input:    "describe pod my-pod",
			expected: true,
		},
		{
			name:     "logs subcommand",
			input:    "logs my-pod -f",
			expected: true,
		},
		{
			name:     "exec subcommand",
			input:    "exec -it my-pod -- bash",
			expected: true,
		},
		{
			name:     "delete subcommand",
			input:    "delete pod my-pod",
			expected: true,
		},
		{
			name:     "apply subcommand",
			input:    "apply -f deployment.yaml",
			expected: true,
		},
		{
			name:     "create subcommand",
			input:    "create namespace test",
			expected: true,
		},
		{
			name:     "edit subcommand",
			input:    "edit deployment my-deploy",
			expected: true,
		},
		{
			name:     "patch subcommand",
			input:    "patch svc my-svc -p {}",
			expected: true,
		},
		{
			name:     "scale subcommand",
			input:    "scale deployment my-deploy --replicas=3",
			expected: true,
		},
		{
			name:     "rollout subcommand",
			input:    "rollout restart deployment my-deploy",
			expected: true,
		},
		{
			name:     "top subcommand",
			input:    "top pods",
			expected: true,
		},
		{
			name:     "label subcommand",
			input:    "label pod my-pod env=prod",
			expected: true,
		},
		{
			name:     "annotate subcommand",
			input:    "annotate pod my-pod note=test",
			expected: true,
		},
		{
			name:     "port-forward subcommand",
			input:    "port-forward svc/my-svc 8080:80",
			expected: true,
		},
		{
			name:     "cp subcommand",
			input:    "cp /tmp/foo my-pod:/tmp/bar",
			expected: true,
		},
		{
			name:     "cordon subcommand",
			input:    "cordon my-node",
			expected: true,
		},
		{
			name:     "uncordon subcommand",
			input:    "uncordon my-node",
			expected: true,
		},
		{
			name:     "drain subcommand",
			input:    "drain my-node",
			expected: true,
		},
		{
			name:     "taint subcommand",
			input:    "taint node my-node key=val:NoSchedule",
			expected: true,
		},
		{
			name:     "config subcommand",
			input:    "config view",
			expected: true,
		},
		{
			name:     "auth subcommand",
			input:    "auth can-i get pods",
			expected: true,
		},
		{
			name:     "api-resources subcommand",
			input:    "api-resources",
			expected: true,
		},
		{
			name:     "explain subcommand",
			input:    "explain pod.spec",
			expected: true,
		},
		{
			name:     "diff subcommand",
			input:    "diff -f deployment.yaml",
			expected: true,
		},
		// Non-kubectl cases.
		{
			name:     "shell command",
			input:    "echo hello",
			expected: false,
		},
		{
			name:     "arbitrary command",
			input:    "ls -la /tmp",
			expected: false,
		},
		{
			name:     "curl command",
			input:    "curl http://example.com",
			expected: false,
		},
		{
			name:     "helm command",
			input:    "helm list",
			expected: false,
		},
		{
			name:     "subcommand case insensitive",
			input:    "GET pods",
			expected: true,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result := isKubectlCommand(tt.input)
			assert.Equal(t, tt.expected, result)
		})
	}
}

// --- startupTips ---

func TestStartupTipsNotEmpty(t *testing.T) {
	assert.NotEmpty(t, startupTips, "startupTips should not be empty")
}

func TestStartupTipsAllNonEmpty(t *testing.T) {
	for i, tip := range startupTips {
		assert.NotEmpty(t, tip, "tip at index %d should not be empty", i)
	}
}
