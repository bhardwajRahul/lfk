package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/janosmiko/lfk/internal/model"
)

func tableRendererTestItems() []model.Item {
	return []model.Item{
		{Name: "pod-a", Namespace: "ns1", Kind: "Pod", Status: "Running", Age: "1d"},
		{Name: "pod-b", Namespace: "ns1", Kind: "Pod", Status: "Running", Age: "2d"},
		{Name: "pod-c", Namespace: "ns2", Kind: "Pod", Status: "Pending", Age: "1h"},
		{Name: "pod-d", Namespace: "ns2", Kind: "Pod", Status: "Running", Age: "5m"},
	}
}

func TestTableRendererPopulatesRowCache(t *testing.T) {
	r := NewTableRenderer()
	items := tableRendererTestItems()

	out := r.Render("NAME", items, 0, 80, 20, false, "", "", 0, 0)
	require.NotEmpty(t, out)

	assert.NotContains(t, r.rows, 0, "cursor row must not be cached")
	assert.Contains(t, r.rows, 1)
	assert.Contains(t, r.rows, 2)
	assert.Contains(t, r.rows, 3)
}

func TestTableRendererCacheSurvivesCursorMove(t *testing.T) {
	r := NewTableRenderer()
	items := tableRendererTestItems()

	_ = r.Render("NAME", items, 0, 80, 20, false, "", "", 0, 0)
	rowAt2 := r.rows[2]
	require.NotEmpty(t, rowAt2)

	_ = r.Render("NAME", items, 1, 80, 20, false, "", "", 0, 0)

	assert.Equal(t, rowAt2, r.rows[2])
}

func TestTableRendererInvalidatesOnMiddleRev(t *testing.T) {
	r := NewTableRenderer()
	items := tableRendererTestItems()

	_ = r.Render("NAME", items, 0, 80, 20, false, "", "", 0, 0)
	require.NotEmpty(t, r.rows)

	_ = r.Render("NAME", items, 0, 80, 20, false, "", "", 1, 0)

	assert.Equal(t, uint64(1), r.fp.middleRev)
}

func TestTableRendererInvalidatesOnWidthChange(t *testing.T) {
	r := NewTableRenderer()
	items := tableRendererTestItems()

	_ = r.Render("NAME", items, 0, 80, 20, false, "", "", 0, 0)
	prevRow := r.rows[1]

	_ = r.Render("NAME", items, 0, 100, 20, false, "", "", 0, 0)
	newRow := r.rows[1]

	assert.NotEqual(t, prevRow, newRow)
}
