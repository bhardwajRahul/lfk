package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestClassifyInput_Shell(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "bang_with_space", input: "! grep error"},
		{name: "bang_no_space", input: "!ls"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, cmdShell, classifyInput(tt.input))
		})
	}
}

func TestClassifyInput_Builtin(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "ns_with_arg", input: "ns production"},
		{name: "namespace_with_arg", input: "namespace kube-system"},
		{name: "ctx_with_arg", input: "ctx my-cluster"},
		{name: "context_with_arg", input: "context my-cluster"},
		{name: "set_with_arg", input: "set wrap"},
		{name: "sort_with_arg", input: "sort Name"},
		{name: "export_with_arg", input: "export yaml"},
		{name: "q_alone", input: "q"},
		{name: "q_bang", input: "q!"},
		{name: "quit_alone", input: "quit"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, cmdBuiltin, classifyInput(tt.input))
		})
	}
}

func TestClassifyInput_Kubectl(t *testing.T) {
	tests := []struct {
		name  string
		input string
		want  commandType
	}{
		{name: "kubectl_get_pods", input: "kubectl get pods", want: cmdKubectl},
		{name: "k_get_pods", input: "k get pods", want: cmdKubectl},
		{name: "get_pods", input: "get pods", want: cmdUnknown},
		{name: "describe_pod", input: "describe pod nginx", want: cmdUnknown},
		{name: "logs_nginx", input: "logs nginx", want: cmdUnknown},
		{name: "delete_pod", input: "delete pod nginx", want: cmdUnknown},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.want, classifyInput(tt.input))
		})
	}
}

func TestClassifyInput_ResourceJump(t *testing.T) {
	tests := []struct {
		name  string
		input string
	}{
		{name: "pods", input: "pods"},
		{name: "deployments", input: "deployments"},
		{name: "services", input: "services"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, cmdResourceJump, classifyInput(tt.input))
		})
	}
}

func TestClassifyInput_Empty(t *testing.T) {
	assert.Equal(t, cmdUnknown, classifyInput(""))
}

func TestClassifyInput_Partial(t *testing.T) {
	// "g" is not a complete match for any known command or resource.
	assert.Equal(t, cmdUnknown, classifyInput("g"))
}

func TestParseTokens(t *testing.T) {
	tests := []struct {
		name   string
		input  string
		cursor int
		want   []token
	}{
		{
			name:   "two_words",
			input:  "get pods",
			cursor: 8,
			want:   []token{{text: "get", start: 0, end: 3}, {text: "pods", start: 4, end: 8}},
		},
		{
			name:   "four_words",
			input:  "get pods -n kube-system",
			cursor: 23,
			want: []token{
				{text: "get", start: 0, end: 3},
				{text: "pods", start: 4, end: 8},
				{text: "-n", start: 9, end: 11},
				{text: "kube-system", start: 12, end: 23},
			},
		},
		{
			name:   "trailing_space",
			input:  "get pods ",
			cursor: 9,
			want:   []token{{text: "get", start: 0, end: 3}, {text: "pods", start: 4, end: 8}, {text: "", start: 9, end: 9}},
		},
		{
			name:   "empty",
			input:  "",
			cursor: 0,
			want:   []token{{text: "", start: 0, end: 0}},
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := parseTokens(tt.input, tt.cursor)
			assert.Equal(t, tt.want, got)
		})
	}
}
