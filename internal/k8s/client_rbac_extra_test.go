package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- computeQuotaPercent: additional edge cases ---

func TestComputeQuotaPercent_Extra(t *testing.T) {
	tests := []struct {
		name    string
		resName string
		hard    string
		used    string
		want    float64
	}{
		{"zero usage", "pods", "10", "0", 0},
		{"50 percent", "pods", "10", "5", 50},
		{"100 percent", "cpu", "1", "1", 100},
		{"over 100 percent capped", "cpu", "1", "2", 100},
		{"millicpu values", "limits.cpu", "2000m", "1000m", 50},
		{"memory values", "memory", "1Gi", "512Mi", 50},
		{"storage quantities", "requests.storage", "100Gi", "50Gi", 50},
		{"zero hard zero used", "pods", "0", "0", 0},
		{"zero hard nonzero used", "pods", "0", "5", 0},
		{"plain integer values", "pods", "20", "4", 20},
		{"invalid hard returns zero", "test", "notanumber-xyz", "5", 0},
		{"invalid used returns zero", "test", "10", "notanumber-xyz", 0},
		{"both invalid return zero", "test", "abc", "def", 0},
		{"fractional cpu", "cpu", "4", "1.5", 37.5},
		{"large storage", "ephemeral-storage", "500Gi", "250Gi", 50},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := computeQuotaPercent(tt.resName, tt.hard, tt.used)
			assert.InDelta(t, tt.want, got, 0.5)
		})
	}
}

// --- ruleKey: additional patterns ---

func TestRuleKey_Additional(t *testing.T) {
	t.Run("identical rules produce identical keys", func(t *testing.T) {
		r1 := AccessRule{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{"apps"},
			Resources: []string{"deployments"},
		}
		r2 := AccessRule{
			Verbs:     []string{"get", "list"},
			APIGroups: []string{"apps"},
			Resources: []string{"deployments"},
		}
		assert.Equal(t, ruleKey(r1), ruleKey(r2))
	})

	t.Run("different verb order produces different keys", func(t *testing.T) {
		r1 := AccessRule{Verbs: []string{"get", "list"}}
		r2 := AccessRule{Verbs: []string{"list", "get"}}
		assert.NotEqual(t, ruleKey(r1), ruleKey(r2))
	})

	t.Run("nil slices produce same key as empty slices", func(t *testing.T) {
		r1 := AccessRule{}
		r2 := AccessRule{
			Verbs:         []string{},
			APIGroups:     []string{},
			Resources:     []string{},
			ResourceNames: []string{},
		}
		assert.Equal(t, ruleKey(r1), ruleKey(r2))
	})
}
