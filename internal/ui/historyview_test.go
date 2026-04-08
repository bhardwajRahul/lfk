package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestRenderHelmHistoryOverlay(t *testing.T) {
	t.Run("empty revisions shows message", func(t *testing.T) {
		result := RenderHelmHistoryOverlay(nil, 0, 100, 40, false)
		assert.Contains(t, result, "No revisions found")
	})

	t.Run("loading shows loading placeholder not empty state", func(t *testing.T) {
		result := RenderHelmHistoryOverlay(nil, 0, 100, 40, true)
		assert.Contains(t, result, "Loading")
		assert.NotContains(t, result, "No revisions found")
	})

	t.Run("revisions rendered with columns", func(t *testing.T) {
		revisions := []HelmRevision{
			{
				Revision:    3,
				Status:      "deployed",
				Chart:       "nginx-1.2.3",
				AppVersion:  "1.21",
				Description: "Upgrade complete",
				Updated:     "2024-01-15 10:00:00",
			},
			{
				Revision:    2,
				Status:      "superseded",
				Chart:       "nginx-1.2.2",
				AppVersion:  "1.20",
				Description: "Install complete",
				Updated:     "2024-01-14 10:00:00",
			},
		}
		result := RenderHelmHistoryOverlay(revisions, 0, 120, 40, false)
		assert.Contains(t, result, "Helm Release History")
		assert.Contains(t, result, "REV")
		assert.Contains(t, result, "STATUS")
		assert.Contains(t, result, "CHART")
		assert.Contains(t, result, "APP VER")
		assert.Contains(t, result, "DESCRIPTION")
		assert.Contains(t, result, "UPDATED")
		assert.Contains(t, result, "nginx-1.2.3")
		assert.Contains(t, result, "Upgrade complete")
		assert.NotContains(t, result, "Rollback Helm")
	})

	t.Run("cursor renders revision at cursor position", func(t *testing.T) {
		revisions := []HelmRevision{
			{Revision: 2, Status: "deployed", Chart: "app-1.0"},
			{Revision: 1, Status: "superseded", Chart: "app-0.9"},
		}
		result := RenderHelmHistoryOverlay(revisions, 1, 120, 40, false)
		assert.Contains(t, result, "app-0.9")
	})
}
