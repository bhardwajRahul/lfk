package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- renderTwoBoxes ---

func TestRenderTwoBoxes(t *testing.T) {
	t.Run("basic two boxes with arrow", func(t *testing.T) {
		left := []string{"Source"}
		right := []string{"Dest"}
		arrow := " --> "
		borderStyle := DimStyle
		result := renderTwoBoxes(left, right, arrow, borderStyle, 80)
		assert.Greater(t, len(result), 2) // top + content + bottom
		joined := strings.Join(result, "\n")
		assert.Contains(t, joined, "Source")
		assert.Contains(t, joined, "Dest")
		assert.Contains(t, joined, "-->")
	})

	t.Run("height equalized", func(t *testing.T) {
		left := []string{"a", "b", "c"}
		right := []string{"x"}
		arrow := " -> "
		result := renderTwoBoxes(left, right, arrow, DimStyle, 80)
		// Height = max(3, 1) + 2 borders = 5.
		assert.Equal(t, 5, len(result))
	})

	t.Run("narrow maxWidth caps box widths", func(t *testing.T) {
		left := []string{"a very long source label"}
		right := []string{"a very long destination label"}
		arrow := " -> "
		result := renderTwoBoxes(left, right, arrow, DimStyle, 40)
		assert.Greater(t, len(result), 0)
	})

	t.Run("empty content boxes", func(t *testing.T) {
		result := renderTwoBoxes(nil, nil, " -> ", DimStyle, 80)
		// Even empty boxes should render borders.
		assert.Greater(t, len(result), 0)
	})
}

// --- renderNetpolRuleDiagram ---

func TestRenderNetpolRuleDiagram(t *testing.T) {
	t.Run("pod peer with selector", func(t *testing.T) {
		rule := NetpolRuleEntry{
			Peers: []NetpolPeerEntry{
				{Type: "Pod", Selector: map[string]string{"app": "frontend"}},
			},
			Ports: []NetpolPortEntry{
				{Protocol: "TCP", Port: "8080"},
			},
		}
		lines := renderNetpolRuleDiagram(rule, "app=backend", true, 80, DimStyle, DimStyle, DimStyle, DimStyle, DimStyle)
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "app=frontend")
		assert.Contains(t, joined, "TCP/8080")
	})

	t.Run("all peer type", func(t *testing.T) {
		rule := NetpolRuleEntry{
			Peers: []NetpolPeerEntry{{Type: "All"}},
		}
		lines := renderNetpolRuleDiagram(rule, "(all pods)", true, 80, DimStyle, DimStyle, DimStyle, DimStyle, DimStyle)
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "All")
	})

	t.Run("namespace+pod peer", func(t *testing.T) {
		rule := NetpolRuleEntry{
			Peers: []NetpolPeerEntry{
				{Type: "Namespace+Pod", Namespace: "monitoring", Selector: map[string]string{"role": "scraper"}},
			},
		}
		lines := renderNetpolRuleDiagram(rule, "(all pods)", true, 80, DimStyle, DimStyle, DimStyle, DimStyle, DimStyle)
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "monitoring")
		assert.Contains(t, joined, "role=scraper")
	})

	t.Run("CIDR peer with except", func(t *testing.T) {
		rule := NetpolRuleEntry{
			Peers: []NetpolPeerEntry{
				{Type: "CIDR", CIDR: "10.0.0.0/8", Except: []string{"10.1.0.0/16"}},
			},
		}
		lines := renderNetpolRuleDiagram(rule, "(all pods)", false, 80, DimStyle, DimStyle, DimStyle, DimStyle, DimStyle)
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "10.0.0.0/8")
		assert.Contains(t, joined, "Except")
		assert.Contains(t, joined, "10.1.0.0/16")
	})

	t.Run("egress rule has target on left", func(t *testing.T) {
		rule := NetpolRuleEntry{
			Peers: []NetpolPeerEntry{{Type: "All"}},
		}
		lines := renderNetpolRuleDiagram(rule, "target-pods", false, 80, DimStyle, DimStyle, DimStyle, DimStyle, DimStyle)
		joined := strings.Join(lines, "\n")
		// Target should appear in the diagram (on left side for egress).
		assert.Contains(t, joined, "Target Pods")
	})

	t.Run("pod peer with empty selector shows all pods", func(t *testing.T) {
		rule := NetpolRuleEntry{
			Peers: []NetpolPeerEntry{
				{Type: "Pod", Selector: map[string]string{}},
			},
		}
		lines := renderNetpolRuleDiagram(rule, "(all)", true, 80, DimStyle, DimStyle, DimStyle, DimStyle, DimStyle)
		joined := strings.Join(lines, "\n")
		assert.Contains(t, joined, "(all pods)")
	})
}
