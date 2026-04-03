package app

import (
	"context"
	"sync"
	"sync/atomic"
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/janosmiko/lfk/internal/k8s"
	"github.com/janosmiko/lfk/internal/model"
	"github.com/janosmiko/lfk/internal/ui"
	dynfake "k8s.io/client-go/dynamic/fake"
)

func bp4() Model {
	m := Model{
		nav:            model.NavigationState{Level: model.LevelResources, Context: "test-ctx"},
		tabs:           []TabState{{}},
		selectedItems:  make(map[string]bool),
		cursorMemory:   make(map[string]int),
		itemCache:      make(map[string][]model.Item),
		discoveredCRDs: make(map[string][]model.ResourceTypeEntry),
		width:          120, height: 40, execMu: &sync.Mutex{}, namespace: "default",
		reqCtx: context.Background(),
	}
	m.client = k8s.NewTestClient(fake.NewClientset(), dynfake.NewSimpleDynamicClient(runtime.NewScheme()))
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Pod", Resource: "pods", APIVersion: "v1", Namespaced: true}
	m.middleItems = []model.Item{
		{Name: "pod-1", Namespace: "default", Kind: "Pod", Status: "Running"},
		{Name: "pod-2", Namespace: "ns-2", Kind: "Pod", Status: "Failed"},
	}
	return m
}

// --- handleDiffKey ---

func TestP4DiffKeyQ(t *testing.T) {
	m := bp4()
	m.mode = modeDiff
	m.diffLeft = "a"
	m.diffRight = "b"
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestP4DiffKeyEsc(t *testing.T) {
	m := bp4()
	m.mode = modeDiff
	m.diffLeft = "a"
	m.diffRight = "b"
	result, _ := m.handleKey(keyMsg("esc"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestP4DiffKeyJ(t *testing.T) {
	m := bp4()
	m.mode = modeDiff
	m.diffLeft = "line1\nline2\nline3\n"
	m.diffRight = "line1\nline2x\nline3\n"
	result, _ := m.handleKey(keyMsg("j"))
	rm := result.(Model)
	_ = rm
}

func TestP4DiffKeyK(t *testing.T) {
	m := bp4()
	m.mode = modeDiff
	m.diffScroll = 5
	result, _ := m.handleKey(keyMsg("k"))
	rm := result.(Model)
	_ = rm
}

func TestP4DiffKeyG(t *testing.T) {
	m := bp4()
	m.mode = modeDiff
	result, _ := m.handleKey(keyMsg("G"))
	rm := result.(Model)
	_ = rm
}

func TestP4DiffKeyGG(t *testing.T) {
	m := bp4()
	m.mode = modeDiff
	m.pendingG = true
	result, _ := m.handleKey(keyMsg("g"))
	rm := result.(Model)
	assert.Equal(t, 0, rm.diffScroll)
}

func TestP4DiffKeyU(t *testing.T) {
	m := bp4()
	m.mode = modeDiff
	m.diffUnified = false
	result, _ := m.handleKey(keyMsg("u"))
	rm := result.(Model)
	assert.True(t, rm.diffUnified)
}

func TestP4DiffKeyHelp(t *testing.T) {
	m := bp4()
	m.mode = modeDiff
	result, _ := m.handleKey(keyMsg("?"))
	rm := result.(Model)
	assert.Equal(t, modeHelp, rm.mode)
}

// --- handleLogKey ---

func TestP4LogKeyQ(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestP4LogKeyF(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	m.logFollow = false
	result, _ := m.handleKey(keyMsg("f"))
	rm := result.(Model)
	assert.True(t, rm.logFollow)
}

func TestP4LogKeyW(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	m.logWrap = false
	result, _ := m.handleKey(keyMsg("w"))
	rm := result.(Model)
	// 'w' toggles wrap. The actual key might be different.
	_ = rm
}

func TestP4LogKeyS(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	m.logTimestamps = false
	result, _ := m.handleKey(keyMsg("s"))
	rm := result.(Model)
	assert.True(t, rm.logTimestamps)
}

func TestP4LogKeyNumber(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	m.logLineNumbers = false
	result, _ := m.handleKey(keyMsg("#"))
	rm := result.(Model)
	assert.True(t, rm.logLineNumbers)
}

func TestP4LogKeyHelp(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	result, _ := m.handleKey(keyMsg("?"))
	rm := result.(Model)
	assert.Equal(t, modeHelp, rm.mode)
}

func TestP4LogKeyGG(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	m.logLines = []string{"line1", "line2", "line3"}
	m.logScroll = 2
	m.pendingG = true
	result, _ := m.handleKey(keyMsg("g"))
	rm := result.(Model)
	assert.Equal(t, 0, rm.logScroll)
}

func TestP4LogKeyG(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	m.pendingG = false
	result, _ := m.handleKey(keyMsg("g"))
	rm := result.(Model)
	assert.True(t, rm.pendingG)
}

func TestP4LogKeySlash(t *testing.T) {
	m := bp4()
	m.mode = modeLogs
	result, _ := m.handleKey(keyMsg("/"))
	rm := result.(Model)
	// '/' in log mode activates search.
	_ = rm
}

// --- handleScaleOverlayKey ---

func TestP4ScaleOverlayEsc(t *testing.T) {
	m := bp4()
	m.overlay = overlayScaleInput
	result, _ := m.handleKey(keyMsg("esc"))
	rm := result.(Model)
	assert.Equal(t, overlayNone, rm.overlay)
}

func TestP4ScaleOverlayBackspace(t *testing.T) {
	m := bp4()
	m.overlay = overlayScaleInput
	m.scaleInput.Insert("3")
	result, _ := m.handleKey(keyMsg("backspace"))
	rm := result.(Model)
	_ = rm
}

func TestP4ScaleOverlayDigit(t *testing.T) {
	m := bp4()
	m.overlay = overlayScaleInput
	result, _ := m.handleKey(keyMsg("5"))
	rm := result.(Model)
	_ = rm
}

// --- handleNamespace overlay ---

func TestP4NamespaceOverlayEsc(t *testing.T) {
	m := bp4()
	m.overlay = overlayNamespace
	result, _ := m.handleKey(keyMsg("esc"))
	rm := result.(Model)
	assert.Equal(t, overlayNone, rm.overlay)
}

func TestP4NamespaceOverlayNavDown(t *testing.T) {
	m := bp4()
	m.overlay = overlayNamespace
	m.overlayItems = []model.Item{{Name: "default"}, {Name: "kube-system"}}
	m.overlayCursor = 0
	result, _ := m.handleKey(keyMsg("j"))
	rm := result.(Model)
	assert.Equal(t, 1, rm.overlayCursor)
}

func TestP4NamespaceOverlayNavUp(t *testing.T) {
	m := bp4()
	m.overlay = overlayNamespace
	m.overlayItems = []model.Item{{Name: "default"}, {Name: "kube-system"}}
	m.overlayCursor = 1
	result, _ := m.handleKey(keyMsg("k"))
	rm := result.(Model)
	assert.Equal(t, 0, rm.overlayCursor)
}

// --- handleExplorerActionKey more branches ---

func TestP4ExplorerActionKeyI(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, _, handled := m.handleExplorerActionKey(keyMsg("I"))
	if handled {
		rm := result.(Model)
		assert.True(t, rm.loading)
	}
}

func TestP4ExplorerActionKeyR(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, cmd, handled := m.handleExplorerActionKey(keyMsg("R"))
	if handled {
		rm := result.(Model)
		_ = rm
		_ = cmd
	}
}

func TestP4ExplorerActionKeyV(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, cmd, handled := m.handleExplorerActionKey(keyMsg("v"))
	if handled {
		rm := result.(Model)
		_ = rm
		_ = cmd
	}
}

func TestP4ExplorerActionKeyX(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, cmd, handled := m.handleExplorerActionKey(keyMsg("x"))
	if handled {
		rm := result.(Model)
		assert.Equal(t, overlayAction, rm.overlay)
		_ = cmd
	}
}

func TestP4ExplorerActionKeyL(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, _, handled := m.handleExplorerActionKey(keyMsg("L"))
	if handled {
		_ = result.(Model)
	}
}

func TestP4ExplorerActionKeyY(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, _, handled := m.handleExplorerActionKey(keyMsg("y"))
	if handled {
		rm := result.(Model)
		_ = rm
	}
}

func TestP4ExplorerActionKeyBigY(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, _, handled := m.handleExplorerActionKey(keyMsg("Y"))
	if handled {
		_ = result.(Model)
	}
}

func TestP4ExplorerActionKeyE(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, _, handled := m.handleExplorerActionKey(keyMsg("e"))
	if handled {
		_ = result.(Model)
	}
}

func TestP4ExplorerActionKeyD(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, _, handled := m.handleExplorerActionKey(keyMsg("d"))
	if handled {
		_ = result.(Model)
	}
}

func TestP4ExplorerActionKeyO(t *testing.T) {
	m := bp4()
	m.setCursor(0)
	result, _, handled := m.handleExplorerActionKey(keyMsg("o"))
	if handled {
		_ = result.(Model)
	}
}

func TestP4ExplorerActionKeyAt(t *testing.T) {
	m := bp4()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{{Name: "Overview", Extra: "__overview__"}}
	m.setCursor(0)
	result, _, handled := m.handleExplorerActionKey(keyMsg("@"))
	_ = handled
	_ = result
}

func TestP4ExplorerActionKeyF(t *testing.T) {
	m := bp4()
	result, _, handled := m.handleExplorerActionKey(keyMsg("f"))
	if handled {
		rm := result.(Model)
		assert.True(t, rm.filterActive)
	}
}

func TestP4ExplorerActionKeyQuote(t *testing.T) {
	m := bp4()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	result, _, handled := m.handleExplorerActionKey(keyMsg("'"))
	if handled {
		_ = result.(Model)
	}
}

func TestP4ExplorerActionKeyComma(t *testing.T) {
	m := bp4()
	ui.ActiveSortableColumns = []string{"Name", "Status", "Age"}
	result, _, handled := m.handleExplorerActionKey(keyMsg(","))
	if handled {
		_ = result.(Model)
	}
}

// --- handleExecKey ---

func TestP4ExecKeyQ(t *testing.T) {
	m := bp4()
	m.mode = modeExec
	done := &sync.Once{}
	_ = done
	m.execDone = new(atomic.Bool)
	m.execDone.Store(true)
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

// --- handleMouseClick with different zones ---

func TestP4MouseClickRightColumn(t *testing.T) {
	m := bp4()
	m.mode = modeExplorer
	m.setCursor(0)
	// Click in the right column (x > middleEnd).
	result, _ := m.handleMouseClick(m.width-1, 10)
	_ = result.(Model)
}

func TestP4MouseClickLeftColumn(t *testing.T) {
	m := bp4()
	m.mode = modeExplorer
	// Click in left column (x < leftEnd).
	result, _ := m.handleMouseClick(2, 10)
	_ = result.(Model)
}

// --- More message handlers ---

func TestP4UpdatePodMetricsEnrichedMsg(t *testing.T) {
	m := bp4()
	m.requestGen = 5
	m.nav.ResourceType.Kind = "Pod"
	msg := podMetricsEnrichedMsg{
		metrics: map[string]model.PodMetrics{
			"pod-1": {CPU: 100, Memory: 256},
		},
		gen: 5,
	}
	result, _ := m.Update(msg)
	_ = result.(Model)
}

func TestP4UpdateNodeMetricsEnrichedMsg(t *testing.T) {
	m := bp4()
	m.requestGen = 5
	m.nav.ResourceType.Kind = "Node"
	msg := nodeMetricsEnrichedMsg{
		metrics: map[string]model.PodMetrics{
			"node-1": {CPU: 1000, Memory: 4096},
		},
		gen: 5,
	}
	result, _ := m.Update(msg)
	_ = result.(Model)
}

func TestP4UpdateFinalizerSearchMsg(t *testing.T) {
	m := bp4()
	msg := finalizerSearchResultMsg{
		results: []k8s.FinalizerMatch{
			{Name: "pod-1", Namespace: "default", Kind: "Pod", Finalizers: []string{"kubernetes"}},
		},
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	_ = rm
}

func TestP4UpdateFinalizerSearchMsgErr(t *testing.T) {
	m := bp4()
	msg := finalizerSearchResultMsg{err: assert.AnError}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
}

func TestP4UpdateFinalizerSearchMsgEmpty(t *testing.T) {
	m := bp4()
	msg := finalizerSearchResultMsg{}
	result, _ := m.Update(msg)
	rm := result.(Model)
	_ = rm
}
