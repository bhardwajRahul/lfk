package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- ruleKey ---

func TestRuleKey(t *testing.T) {
	tests := []struct {
		name string
		rule AccessRule
		want string
	}{
		{
			name: "full rule with all fields",
			rule: AccessRule{
				Verbs:         []string{"get", "list"},
				APIGroups:     []string{"", "apps"},
				Resources:     []string{"pods", "deployments"},
				ResourceNames: []string{"my-pod"},
			},
			want: "get,list|,apps|pods,deployments|my-pod",
		},
		{
			name: "rule with empty slices",
			rule: AccessRule{
				Verbs:         []string{},
				APIGroups:     []string{},
				Resources:     []string{},
				ResourceNames: []string{},
			},
			want: "|||",
		},
		{
			name: "wildcard rule",
			rule: AccessRule{
				Verbs:     []string{"*"},
				APIGroups: []string{"*"},
				Resources: []string{"*"},
			},
			want: "*|*|*|",
		},
		{
			name: "single verb single resource",
			rule: AccessRule{
				Verbs:     []string{"get"},
				APIGroups: []string{""},
				Resources: []string{"secrets"},
			},
			want: "get||secrets|",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := ruleKey(tt.rule)
			assert.Equal(t, tt.want, got)
		})
	}
}

// --- ruleKey determinism ---

func TestRuleKey_Deterministic(t *testing.T) {
	rule := AccessRule{
		Verbs:         []string{"get", "list", "watch"},
		APIGroups:     []string{"apps"},
		Resources:     []string{"deployments"},
		ResourceNames: []string{"my-deploy"},
	}

	first := ruleKey(rule)
	for range 100 {
		assert.Equal(t, first, ruleKey(rule),
			"ruleKey should be deterministic for the same input")
	}
}

// --- ruleKey uniqueness ---

func TestRuleKey_Uniqueness(t *testing.T) {
	rule1 := AccessRule{
		Verbs:     []string{"get"},
		APIGroups: []string{""},
		Resources: []string{"pods"},
	}
	rule2 := AccessRule{
		Verbs:     []string{"get"},
		APIGroups: []string{""},
		Resources: []string{"services"},
	}

	assert.NotEqual(t, ruleKey(rule1), ruleKey(rule2),
		"different rules should produce different keys")
}

// --- computeQuotaPercent edge cases ---

func TestComputeQuotaPercent_EdgeCases(t *testing.T) {
	tests := []struct {
		name    string
		resName string
		hard    string
		used    string
		want    float64
	}{
		{"over 100 percent is capped", "cpu", "1", "2", 100},
		{"storage quantities", "requests.storage", "100Gi", "50Gi", 50},
		{"zero hard zero used", "pods", "0", "0", 0},
		{"millicpu", "limits.cpu", "2000m", "1000m", 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeQuotaPercent(tt.resName, tt.hard, tt.used)
			assert.InDelta(t, tt.want, got, 0.5)
		})
	}
}
