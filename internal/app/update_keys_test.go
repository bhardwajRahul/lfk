package app

import (
	"strings"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// helper to make a rune key message.
func runeKey(r rune) tea.KeyMsg {
	return tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}}
}

// helper to make a special key message.
func specialKey(k tea.KeyType) tea.KeyMsg {
	return tea.KeyMsg{Type: k}
}

// baseExplorerModel returns a minimal Model for key handling tests in explorer mode.
func baseExplorerModel() Model {
	return Model{
		nav: model.NavigationState{
			Level:   model.LevelResources,
			Context: "test",
			ResourceType: model.ResourceTypeEntry{
				DisplayName: "Pods",
				Kind:        "Pod",
				Resource:    "pods",
			},
		},
		middleItems: []model.Item{
			{Name: "pod-a", Kind: "Pod"},
			{Name: "pod-b", Kind: "Pod"},
			{Name: "pod-c", Kind: "Pod"},
		},
		width:              120,
		height:             40,
		mode:               modeExplorer,
		namespace:          "default",
		tabs:               []TabState{{}},
		selectedItems:      make(map[string]bool),
		cursorMemory:       make(map[string]int),
		itemCache:          make(map[string][]model.Item),
		yamlCollapsed:      make(map[string]bool),
		selectedNamespaces: make(map[string]bool),
		selectionAnchor:    -1,
	}
}

// --- handleKey: dismiss status tip ---

func TestHandleKeyDismissesStatusTip(t *testing.T) {
	m := baseExplorerModel()
	m.statusMessage = "Press ? for help"
	m.statusMessageTip = true

	ret, _ := m.handleKey(runeKey('j'))
	result := ret.(Model)
	assert.Empty(t, result.statusMessage)
	assert.False(t, result.statusMessageTip)
}

// --- handleKey: cursor movement j/k ---

func TestHandleKeyJMovesDown(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(0)

	ret, _ := m.handleKey(runeKey('j'))
	result := ret.(Model)
	assert.Equal(t, 1, result.cursor())
}

func TestHandleKeyKMovesUp(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(2)

	ret, _ := m.handleKey(runeKey('k'))
	result := ret.(Model)
	assert.Equal(t, 1, result.cursor())
}

func TestHandleKeyDownArrow(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(0)

	ret, _ := m.handleKey(specialKey(tea.KeyDown))
	result := ret.(Model)
	assert.Equal(t, 1, result.cursor())
}

func TestHandleKeyUpArrow(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(2)

	ret, _ := m.handleKey(specialKey(tea.KeyUp))
	result := ret.(Model)
	assert.Equal(t, 1, result.cursor())
}

// --- handleKey: g / G for top/bottom ---

func TestHandleKeyGGGoesToTop(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(2)

	// First 'g' sets pendingG.
	ret, _ := m.handleKey(runeKey('g'))
	m = ret.(Model)
	assert.True(t, m.pendingG)

	// Second 'g' jumps to top.
	ret, _ = m.handleKey(runeKey('g'))
	m = ret.(Model)
	assert.False(t, m.pendingG)
	assert.Equal(t, 0, m.cursor())
}

func TestHandleKeyGGoesToBottom(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(0)

	ret, _ := m.handleKey(runeKey('G'))
	result := ret.(Model)
	assert.Equal(t, 2, result.cursor())
}

// --- handleKey: q opens quit confirm ---

func TestHandleKeyQOpensQuitConfirm(t *testing.T) {
	m := baseExplorerModel()

	ret, _ := m.handleKey(runeKey('q'))
	result := ret.(Model)
	assert.Equal(t, overlayQuitConfirm, result.overlay)
}

// --- handleKey: / opens search ---

func TestHandleKeySlashOpensSearch(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(1)

	ret, _ := m.handleKey(runeKey('/'))
	result := ret.(Model)
	assert.True(t, result.searchActive)
	assert.Equal(t, 1, result.searchPrevCursor)
}

// --- handleKey: f opens filter ---

func TestHandleKeyFOpensFilter(t *testing.T) {
	m := baseExplorerModel()

	ret, _ := m.handleKey(runeKey('f'))
	result := ret.(Model)
	assert.True(t, result.filterActive)
}

// --- handleKey: space toggles selection ---

func TestHandleKeySpaceTogglesSelection(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(0)

	ret, _ := m.handleKey(runeKey(' '))
	result := ret.(Model)
	assert.True(t, result.isSelected(model.Item{Name: "pod-a", Kind: "Pod"}))
	// Cursor should move down after selection.
	assert.Equal(t, 1, result.cursor())
}

// --- handleKey: esc behavior ---

func TestHandleKeyEscClearsFilter(t *testing.T) {
	m := baseExplorerModel()
	m.filterText = "nginx"
	m.setCursor(1)

	ret, _ := m.handleKey(specialKey(tea.KeyEsc))
	result := ret.(Model)
	assert.Empty(t, result.filterText)
	assert.Equal(t, 0, result.cursor())
}

func TestHandleKeyEscClearsSelection(t *testing.T) {
	m := baseExplorerModel()
	m.selectedItems["pod-a"] = true

	ret, _ := m.handleKey(specialKey(tea.KeyEsc))
	result := ret.(Model)
	assert.False(t, result.hasSelection())
}

func TestHandleKeyEscExitsFullscreenDashboard(t *testing.T) {
	m := baseExplorerModel()
	m.fullscreenDashboard = true

	ret, _ := m.handleKey(specialKey(tea.KeyEsc))
	result := ret.(Model)
	assert.False(t, result.fullscreenDashboard)
}

func TestHandleKeyEscNavigatesParent(t *testing.T) {
	m := baseExplorerModel()
	// At resources level with no filter/selection, esc navigates parent.
	ret, _ := m.handleKey(specialKey(tea.KeyEsc))
	result := ret.(Model)
	// Should have navigated to resource types level.
	assert.Equal(t, model.LevelResourceTypes, result.nav.Level)
}

// --- handleKey: h/left navigates parent ---

func TestHandleKeyHNavigatesParent(t *testing.T) {
	m := baseExplorerModel()

	ret, _ := m.handleKey(runeKey('h'))
	result := ret.(Model)
	assert.Equal(t, model.LevelResourceTypes, result.nav.Level)
}

// --- handleKey: enter opens full view ---

func TestHandleKeyEnterFullView(t *testing.T) {
	m := baseExplorerModel()
	m.setCursor(0)

	ret, _ := m.handleKey(specialKey(tea.KeyEnter))
	result := ret.(Model)
	// Enter on a resource should switch to YAML mode.
	assert.Equal(t, modeYAML, result.mode)
}

// --- handleKey: ? opens help ---

func TestHandleKeyQuestionMarkOpensHelp(t *testing.T) {
	m := baseExplorerModel()

	ret, _ := m.handleKey(runeKey('?'))
	result := ret.(Model)
	assert.Equal(t, modeHelp, result.mode)
	assert.Equal(t, 0, result.helpScroll)
}

// --- handleKey: w toggles watch ---

func TestHandleKeyWTogglesWatch(t *testing.T) {
	m := baseExplorerModel()
	assert.False(t, m.watchMode)

	ret, _ := m.handleKey(runeKey('w'))
	result := ret.(Model)
	assert.True(t, result.watchMode)

	ret, _ = result.handleKey(runeKey('w'))
	result = ret.(Model)
	assert.False(t, result.watchMode)
}

// --- handleKey: : opens command bar ---

func TestHandleKeyColonOpensCommandBar(t *testing.T) {
	m := baseExplorerModel()
	m.commandHistory = loadCommandHistory()

	ret, _ := m.handleKey(runeKey(':'))
	result := ret.(Model)
	assert.True(t, result.commandBarActive)
}

// --- handleKey: ctrl+s toggles secret values ---

func TestHandleKeyCtrlSTogglesSecrets(t *testing.T) {
	m := baseExplorerModel()
	assert.False(t, m.showSecretValues)

	ret, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	result := ret.(Model)
	assert.True(t, result.showSecretValues)

	ret, _ = result.handleKey(tea.KeyMsg{Type: tea.KeyCtrlS})
	result = ret.(Model)
	assert.False(t, result.showSecretValues)
}

// --- handleKey: P toggles full YAML preview ---

func TestHandleKeyPTogglesFullYAMLPreview(t *testing.T) {
	m := baseExplorerModel()
	assert.False(t, m.fullYAMLPreview)

	ret, _ := m.handleKey(runeKey('P'))
	result := ret.(Model)
	assert.True(t, result.fullYAMLPreview)

	ret, _ = result.handleKey(runeKey('P'))
	result = ret.(Model)
	assert.False(t, result.fullYAMLPreview)
}

// --- handleKey: F toggles fullscreen middle ---

func TestHandleKeyFTogglesFullscreenMiddle(t *testing.T) {
	m := baseExplorerModel()
	assert.False(t, m.fullscreenMiddle)

	ret, _ := m.handleKey(runeKey('F'))
	result := ret.(Model)
	assert.True(t, result.fullscreenMiddle)

	ret, _ = result.handleKey(runeKey('F'))
	result = ret.(Model)
	assert.False(t, result.fullscreenMiddle)
}

// --- handleKey: ctrl+a toggles select all ---

func TestHandleKeyCtrlASelectsAll(t *testing.T) {
	m := baseExplorerModel()

	ret, _ := m.handleKey(tea.KeyMsg{Type: tea.KeyCtrlA})
	result := ret.(Model)
	assert.Equal(t, 3, len(result.selectedItems))

	// Second ctrl+a deselects all.
	ret, _ = result.handleKey(tea.KeyMsg{Type: tea.KeyCtrlA})
	result = ret.(Model)
	assert.Equal(t, 0, len(result.selectedItems))
}

// --- handleKey: ' opens bookmarks ---

func TestHandleKeyQuoteOpensBookmarks(t *testing.T) {
	m := baseExplorerModel()

	ret, _ := m.handleKey(runeKey('\''))
	result := ret.(Model)
	assert.Equal(t, overlayBookmarks, result.overlay)
}

// --- handleKey: m sets pending mark ---

func TestHandleKeyMSetsPendingMark(t *testing.T) {
	m := baseExplorerModel()

	ret, _ := m.handleKey(runeKey('m'))
	result := ret.(Model)
	assert.True(t, result.pendingMark)
}

// --- handleKey: pending G cleared on non-g key ---

func TestHandleKeyPendingGClearedOnOtherKey(t *testing.T) {
	m := baseExplorerModel()
	m.pendingG = true

	ret, _ := m.handleKey(runeKey('j'))
	result := ret.(Model)
	assert.False(t, result.pendingG)
}

// --- handleKey: n/N with search ---

func TestHandleKeyNJumpsToNextSearch(t *testing.T) {
	m := baseExplorerModel()
	m.searchInput = TextInput{Value: "pod-b"}
	m.setCursor(0)

	ret, _ := m.handleKey(runeKey('n'))
	result := ret.(Model)
	// Should attempt to jump to next search match.
	assert.NotNil(t, result)
}

func TestHandleKeyNCapJumpsToPrevSearch(t *testing.T) {
	m := baseExplorerModel()
	m.searchInput = TextInput{Value: "pod-a"}
	m.setCursor(2)

	ret, _ := m.handleKey(runeKey('N'))
	result := ret.(Model)
	assert.NotNil(t, result)
}

// --- handleKey: j/k in fullscreen dashboard ---

func TestHandleKeyJKInFullscreenDashboard(t *testing.T) {
	m := baseExplorerModel()
	m.fullscreenDashboard = true
	m.previewScroll = 5
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{{Name: "Cluster Dashboard", Extra: "__overview__"}}
	m.dashboardPreview = strings.Repeat("dashboard line\n", 200)

	ret, _ := m.handleKey(runeKey('j'))
	result := ret.(Model)
	assert.Equal(t, 6, result.previewScroll)

	ret, _ = result.handleKey(runeKey('k'))
	result = ret.(Model)
	assert.Equal(t, 5, result.previewScroll)
}

// --- handleKey: T opens colorscheme ---

func TestHandleKeyTOpensColorscheme(t *testing.T) {
	m := baseExplorerModel()

	ret, _ := m.handleKey(runeKey('T'))
	result := ret.(Model)
	assert.Equal(t, overlayColorscheme, result.overlay)
}
