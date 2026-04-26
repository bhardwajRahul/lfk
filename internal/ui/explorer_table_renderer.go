package ui

import (
	"sort"
	"strings"
	"unsafe"

	"github.com/janosmiko/lfk/internal/model"
)

// TableRenderer caches the non-cursor row strings and column-width layout
// across renders, keyed by an input fingerprint. Cursor rows are always
// re-rendered.
type TableRenderer struct {
	fp     tableFingerprint
	rows   map[int]string
	layout TableLayoutCache
}

type tableFingerprint struct {
	itemsPtr     uintptr
	itemsLen     int
	middleRev    uint64
	selRev       uint64
	width        int
	height       int
	highlight    string
	hiddenCols   string
	columnOrder  string
	sessionCols  string
	contextLabel string
}

func NewTableRenderer() *TableRenderer {
	return &TableRenderer{rows: make(map[int]string)}
}

func (r *TableRenderer) Render(headerLabel string, items []model.Item, cursor int, width, height int, loading bool, spinnerView string, errMsg string, middleRev, selRev uint64) string {
	fp := tableFingerprint{
		itemsPtr:     itemsHeaderPtr(items),
		itemsLen:     len(items),
		middleRev:    middleRev,
		selRev:       selRev,
		width:        width,
		height:       height,
		highlight:    ActiveHighlightQuery,
		hiddenCols:   serializeBoolSet(ActiveHiddenBuiltinColumns),
		columnOrder:  strings.Join(ActiveColumnOrder, "|"),
		sessionCols:  strings.Join(ActiveSessionColumns, "|"),
		contextLabel: ActiveContext,
	}
	if r.fp != fp {
		r.fp = fp
		clear(r.rows)
		r.layout = TableLayoutCache{}
	}
	prevCache := ActiveRowCache
	prevLayout := ActiveTableLayout
	ActiveRowCache = r.rows
	ActiveTableLayout = &r.layout
	out := RenderTable(headerLabel, items, cursor, width, height, loading, spinnerView, errMsg)
	ActiveRowCache = prevCache
	ActiveTableLayout = prevLayout
	return out
}

func itemsHeaderPtr(items []model.Item) uintptr {
	if len(items) == 0 {
		return 0
	}
	return uintptr(unsafe.Pointer(&items[0]))
}

func serializeBoolSet(m map[string]bool) string {
	if len(m) == 0 {
		return ""
	}
	keys := make([]string, 0, len(m))
	for k, v := range m {
		if v {
			keys = append(keys, k)
		}
	}
	sort.Strings(keys)
	return strings.Join(keys, "|")
}
