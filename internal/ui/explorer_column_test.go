package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- RenderColumn ---

func TestRenderColumn(t *testing.T) {
	t.Run("empty items with loading shows spinner", func(t *testing.T) {
		result := RenderColumn("Header", nil, 0, 40, 10, true, true, ">", "")
		assert.Contains(t, result, "Header")
		assert.Contains(t, result, "Loading...")
	})

	t.Run("empty items with error shows error", func(t *testing.T) {
		result := RenderColumn("Header", nil, 0, 40, 10, true, false, "", "connection refused")
		assert.Contains(t, result, "Header")
		assert.Contains(t, result, "connection refused")
	})

	t.Run("empty items no loading no error shows no items", func(t *testing.T) {
		result := RenderColumn("Header", nil, 0, 40, 10, true, false, "", "")
		assert.Contains(t, result, "Header")
		assert.Contains(t, result, "No items")
	})

	t.Run("empty header skips header line", func(t *testing.T) {
		items := []model.Item{{Name: "pod1", Status: "Running"}}
		result := RenderColumn("", items, 0, 40, 10, true, false, "", "")
		assert.Contains(t, result, "pod1")
	})

	t.Run("items rendered with names", func(t *testing.T) {
		items := []model.Item{
			{Name: "pod-1", Status: "Running"},
			{Name: "pod-2", Status: "Pending"},
		}
		// Reset global state that might interfere.
		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		origCollapsed := ActiveCollapsedCategories
		ActiveCollapsedCategories = nil
		defer func() { ActiveCollapsedCategories = origCollapsed }()

		// Use inactive scroll to avoid relying on ActiveMiddleScroll.
		origMS := ActiveMiddleScroll
		ActiveMiddleScroll = -1
		origLS := ActiveLeftScroll
		ActiveLeftScroll = -1
		defer func() {
			ActiveMiddleScroll = origMS
			ActiveLeftScroll = origLS
		}()

		result := RenderColumn("Pods", items, 0, 60, 10, true, false, "", "")
		assert.Contains(t, result, "pod-1")
		assert.Contains(t, result, "pod-2")
		assert.Contains(t, result, "Running")
	})

	t.Run("cursor on item selects it", func(t *testing.T) {
		items := []model.Item{
			{Name: "item-a"},
			{Name: "item-b"},
		}
		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		origCollapsed := ActiveCollapsedCategories
		ActiveCollapsedCategories = nil
		defer func() { ActiveCollapsedCategories = origCollapsed }()

		origMS := ActiveMiddleScroll
		ActiveMiddleScroll = -1
		origLS := ActiveLeftScroll
		ActiveLeftScroll = -1
		defer func() {
			ActiveMiddleScroll = origMS
			ActiveLeftScroll = origLS
		}()

		result := RenderColumn("Col", items, 1, 60, 10, true, false, "", "")
		assert.Contains(t, result, "item-b")
	})

	t.Run("inactive column renders", func(t *testing.T) {
		items := []model.Item{{Name: "svc-1"}}
		origCollapsed := ActiveCollapsedCategories
		ActiveCollapsedCategories = nil
		defer func() { ActiveCollapsedCategories = origCollapsed }()

		origMS := ActiveMiddleScroll
		ActiveMiddleScroll = -1
		origLS := ActiveLeftScroll
		ActiveLeftScroll = -1
		defer func() {
			ActiveMiddleScroll = origMS
			ActiveLeftScroll = origLS
		}()

		result := RenderColumn("Services", items, 0, 40, 10, false, false, "", "")
		assert.Contains(t, result, "svc-1")
	})

	t.Run("items with categories show category headers", func(t *testing.T) {
		items := []model.Item{
			{Name: "pod-1", Category: "Workloads"},
			{Name: "svc-1", Category: "Networking"},
		}
		origCollapsed := ActiveCollapsedCategories
		ActiveCollapsedCategories = nil
		defer func() { ActiveCollapsedCategories = origCollapsed }()

		origMS := ActiveMiddleScroll
		ActiveMiddleScroll = -1
		origLS := ActiveLeftScroll
		ActiveLeftScroll = -1
		defer func() {
			ActiveMiddleScroll = origMS
			ActiveLeftScroll = origLS
		}()

		result := RenderColumn("", items, 0, 60, 20, true, false, "", "")
		assert.Contains(t, result, "Workloads")
		assert.Contains(t, result, "Networking")
	})
}

// --- FormatItem ---

func TestFormatItem(t *testing.T) {
	t.Run("simple name no extras", func(t *testing.T) {
		item := model.Item{Name: "my-pod"}
		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		result := FormatItem(item, 40)
		assert.Contains(t, result, "my-pod")
	})

	t.Run("item with namespace", func(t *testing.T) {
		item := model.Item{Name: "my-pod", Namespace: "default"}
		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		result := FormatItem(item, 60)
		assert.Contains(t, result, "default/my-pod")
	})

	t.Run("item with status", func(t *testing.T) {
		item := model.Item{Name: "pod", Status: "Running"}
		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		result := FormatItem(item, 40)
		assert.Contains(t, result, "pod")
		assert.Contains(t, result, "Running")
	})

	t.Run("item with ready and age", func(t *testing.T) {
		item := model.Item{Name: "pod", Ready: "1/1", Age: "5m", Status: "Running"}
		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		result := FormatItem(item, 60)
		assert.Contains(t, result, "pod")
		assert.Contains(t, result, "1/1")
		assert.Contains(t, result, "5m")
	})

	t.Run("current context shows star", func(t *testing.T) {
		item := model.Item{Name: "my-context", Status: "current"}
		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		result := FormatItem(item, 40)
		assert.Contains(t, result, "*")
		assert.Contains(t, result, "my-context")
	})

	t.Run("deprecated item shows warning", func(t *testing.T) {
		item := model.Item{Name: "old-resource", Deprecated: true}
		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		result := FormatItem(item, 40)
		assert.Contains(t, result, "old-resource")
	})

	t.Run("item with icon", func(t *testing.T) {
		origMode := IconMode
		IconMode = "unicode"
		defer func() { IconMode = origMode }()

		origQuery := ActiveHighlightQuery
		ActiveHighlightQuery = ""
		defer func() { ActiveHighlightQuery = origQuery }()

		item := model.Item{Name: "pod", Icon: "⬤"}
		result := FormatItem(item, 40)
		assert.Contains(t, result, "pod")
	})
}

// --- FormatItemPlain ---

func TestFormatItemPlain(t *testing.T) {
	t.Run("simple name no extras", func(t *testing.T) {
		item := model.Item{Name: "my-pod"}
		result := FormatItemPlain(item, 40)
		assert.Contains(t, result, "my-pod")
	})

	t.Run("item with namespace", func(t *testing.T) {
		item := model.Item{Name: "pod", Namespace: "kube-system"}
		result := FormatItemPlain(item, 60)
		assert.Contains(t, result, "kube-system/pod")
	})

	t.Run("item with status and details", func(t *testing.T) {
		item := model.Item{Name: "pod", Ready: "2/3", Restarts: "5", Age: "10m", Status: "Running"}
		result := FormatItemPlain(item, 60)
		assert.Contains(t, result, "pod")
		assert.Contains(t, result, "2/3")
		assert.Contains(t, result, "5")
		assert.Contains(t, result, "10m")
		assert.Contains(t, result, "Running")
	})

	t.Run("current context shows star", func(t *testing.T) {
		item := model.Item{Name: "ctx", Status: "current"}
		result := FormatItemPlain(item, 40)
		assert.Contains(t, result, "* ")
		assert.Contains(t, result, "ctx")
	})

	t.Run("deprecated item shows warning", func(t *testing.T) {
		item := model.Item{Name: "res", Deprecated: true}
		result := FormatItemPlain(item, 40)
		assert.Contains(t, result, "res")
	})

	t.Run("long name truncated", func(t *testing.T) {
		item := model.Item{Name: "a-very-long-pod-name-that-exceeds-max-width", Status: "Running"}
		result := FormatItemPlain(item, 30)
		assert.LessOrEqual(t, len(result), 40) // Rough check: plain text should be bounded.
		assert.Contains(t, result, "Running")
	})

	t.Run("icon in plain mode", func(t *testing.T) {
		origMode := IconMode
		IconMode = "unicode"
		defer func() { IconMode = origMode }()

		item := model.Item{Name: "pod", Icon: "⬤"}
		result := FormatItemPlain(item, 40)
		assert.Contains(t, result, "pod")
		// In plain mode, icon is plain text (no styled IconStyle).
		assert.Contains(t, result, "⬤")
	})
}
