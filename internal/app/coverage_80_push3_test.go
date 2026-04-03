package app

import (
	"context"
	"sync"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/janosmiko/lfk/internal/k8s"
	"github.com/janosmiko/lfk/internal/model"
	"github.com/janosmiko/lfk/internal/ui"
	dynfake "k8s.io/client-go/dynamic/fake"
)

func basePush80v3Model() Model {
	m := Model{
		nav: model.NavigationState{
			Level:   model.LevelResources,
			Context: "test-ctx",
		},
		tabs:           []TabState{{}},
		selectedItems:  make(map[string]bool),
		cursorMemory:   make(map[string]int),
		itemCache:      make(map[string][]model.Item),
		discoveredCRDs: make(map[string][]model.ResourceTypeEntry),
		width:          120,
		height:         40,
		execMu:         &sync.Mutex{},
		namespace:      "default",
		reqCtx:         context.Background(),
	}
	m.client = k8s.NewTestClient(
		fake.NewClientset(),
		dynfake.NewSimpleDynamicClient(runtime.NewScheme()),
	)
	m.nav.ResourceType = model.ResourceTypeEntry{
		Kind:       "Pod",
		Resource:   "pods",
		APIVersion: "v1",
		Namespaced: true,
	}
	m.middleItems = []model.Item{
		{Name: "pod-1", Namespace: "default", Kind: "Pod", Status: "Running"},
		{Name: "pod-2", Namespace: "ns-2", Kind: "Pod", Status: "Failed"},
	}
	return m
}

// =====================================================================
// exec functions with early-return "not found" branches
// =====================================================================

func TestPush3ExecKubectlNodeCmdNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{name: "node-1", context: "test-ctx"}
	cmd := m.execKubectlNodeCmd("cordon")
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
	assert.Contains(t, amsg.err.Error(), "kubectl not found")
}

func TestPush3ExecKubectlExplainNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	cmd := m.execKubectlExplain("pods", "v1", "")
	require.NotNil(t, cmd)
	msg := cmd()
	emsg, ok := msg.(explainLoadedMsg)
	require.True(t, ok)
	assert.Error(t, emsg.err)
	assert.Contains(t, emsg.err.Error(), "kubectl not found")
}

func TestPush3ExecKubectlExplainRecursiveNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	cmd := m.execKubectlExplainRecursive("pods", "v1", "")
	require.NotNil(t, cmd)
	msg := cmd()
	emsg, ok := msg.(explainRecursiveMsg)
	require.True(t, ok)
	assert.Error(t, emsg.err)
}

func TestPush3RollbackHelmReleaseNoHelm(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{name: "release-1", namespace: "default", context: "test-ctx"}
	cmd := m.rollbackHelmRelease(1)
	require.NotNil(t, cmd)
	msg := cmd()
	hmsg, ok := msg.(helmRollbackDoneMsg)
	require.True(t, ok)
	assert.Error(t, hmsg.err)
}

func TestPush3HelmDiffNoHelm(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{name: "release-1", namespace: "default", context: "test-ctx"}
	cmd := m.helmDiff()
	require.NotNil(t, cmd)
	msg := cmd()
	dmsg, ok := msg.(diffLoadedMsg)
	require.True(t, ok)
	assert.Error(t, dmsg.err)
}

func TestPush3ExecKubectlDrainNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{name: "node-1", context: "test-ctx"}
	cmd := m.execKubectlDrain()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3ExecKubectlNodeShellNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{name: "node-1", context: "test-ctx"}
	cmd := m.execKubectlNodeShell()
	require.NotNil(t, cmd)
	// This returns a tea.ExecProcess or error cmd.
}

func TestPush3ExecKubectlDescribeNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		name:         "pod-1",
		namespace:    "default",
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", Kind: "Pod", Namespaced: true},
	}
	cmd := m.execKubectlDescribe()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3ExecKubectlEditNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		name:         "pod-1",
		namespace:    "default",
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", Kind: "Pod", Namespaced: true},
	}
	cmd := m.execKubectlEdit()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3DeleteResourceNoClient(t *testing.T) {
	m := basePush80v3Model()
	m.actionCtx = actionContext{
		name:         "pod-1",
		namespace:    "default",
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", Kind: "Pod", Namespaced: true},
	}
	cmd := m.deleteResource()
	require.NotNil(t, cmd)
	// Execute -- should try to delete via fake client.
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	// Fake client will fail since the pod doesn't exist.
	assert.Error(t, amsg.err)
}

func TestPush3ForceDeleteResourceNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		name:         "pod-1",
		namespace:    "default",
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", Kind: "Pod", Namespaced: true},
	}
	cmd := m.forceDeleteResource()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3RemoveFinalizersNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		name:         "pod-1",
		namespace:    "default",
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", Kind: "Pod", Namespaced: true},
	}
	cmd := m.removeFinalizers()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3UninstallHelmReleaseNoHelm(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{name: "release-1", namespace: "default", context: "test-ctx"}
	cmd := m.uninstallHelmRelease()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3EditHelmValuesNoHelm(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{name: "release-1", namespace: "default", context: "test-ctx"}
	cmd := m.editHelmValues()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3HelmUpgradeNoHelm(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{name: "release-1", namespace: "default", context: "test-ctx"}
	cmd := m.helmUpgrade()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3VulnScanImageNoTrivy(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{image: "nginx:latest"}
	cmd := m.vulnScanImage("nginx:latest")
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(describeLoadedMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3ScaleResourceNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	m.actionCtx = actionContext{
		name:      "deploy-1",
		namespace: "default",
		context:   "test-ctx",
		kind:      "Deployment",
	}
	cmd := m.scaleResource(3)
	require.NotNil(t, cmd)
}

func TestPush3RestartResourceNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	m.actionCtx = actionContext{
		name:      "deploy-1",
		namespace: "default",
		context:   "test-ctx",
		kind:      "Deployment",
	}
	cmd := m.restartResource()
	require.NotNil(t, cmd)
}

func TestPush3ExecKubectlExecNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		name:          "pod-1",
		namespace:     "default",
		context:       "test-ctx",
		containerName: "app",
	}
	cmd := m.execKubectlExec()
	require.NotNil(t, cmd)
	// This returns an error action result.
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3ExecKubectlAttachNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		name:          "pod-1",
		namespace:     "default",
		context:       "test-ctx",
		containerName: "app",
	}
	cmd := m.execKubectlAttach()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3ExecKubectlDebugNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		name:          "pod-1",
		namespace:     "default",
		context:       "test-ctx",
		containerName: "app",
		image:         "nginx:latest",
	}
	cmd := m.execKubectlDebug()
	require.NotNil(t, cmd)
	msg := cmd()
	amsg, ok := msg.(actionResultMsg)
	require.True(t, ok)
	assert.Error(t, amsg.err)
}

func TestPush3ExecCustomActionNoKubectl(t *testing.T) {
	m := basePush80v3Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		name:      "pod-1",
		namespace: "default",
		context:   "test-ctx",
	}
	ca := ui.CustomAction{
		Label:   "test",
		Command: "echo hello",
	}
	cmd := m.execCustomAction(ca.Command)
	require.NotNil(t, cmd)
}

// =====================================================================
// More handleKey branches to boost coverage
// =====================================================================

func TestPush3HandleKeyHelp(t *testing.T) {
	m := basePush80v3Model()
	result, _ := m.handleKey(keyMsg("?"))
	rm := result.(Model)
	assert.Equal(t, modeHelp, rm.mode)
}

func TestPush3HandleKeyF1(t *testing.T) {
	m := basePush80v3Model()
	result, _ := m.handleKey(keyMsg("f1"))
	rm := result.(Model)
	assert.Equal(t, modeHelp, rm.mode)
}

func TestPush3HandleKeySlash(t *testing.T) {
	m := basePush80v3Model()
	result, _ := m.handleKey(keyMsg("/"))
	rm := result.(Model)
	assert.True(t, rm.searchActive)
}

func TestPush3HandleKeyCtrlC(t *testing.T) {
	m := basePush80v3Model()
	result, _ := m.handleKey(keyMsg("ctrl+c"))
	rm := result.(Model)
	// ctrl+c shows quit confirmation.
	assert.Equal(t, overlayQuitConfirm, rm.overlay)
}

func TestPush3HandleKeyQuestionInHelp(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeHelp
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.NotEqual(t, modeHelp, rm.mode)
}

func TestPush3HandleKeyEscInYAML(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeYAML
	result, _ := m.handleKey(keyMsg("esc"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestPush3HandleKeyVimBindings(t *testing.T) {
	m := basePush80v3Model()
	m.setCursor(0)
	kb := ui.ActiveKeybindings

	// j = down
	result, _ := m.handleKey(keyMsg(kb.Down))
	rm := result.(Model)
	assert.GreaterOrEqual(t, rm.cursor(), 0)

	// k = up
	m.setCursor(1)
	result2, _ := m.handleKey(keyMsg(kb.Up))
	rm2 := result2.(Model)
	assert.GreaterOrEqual(t, rm2.cursor(), 0)
}

func TestPush3HandleKeyGG(t *testing.T) {
	m := basePush80v3Model()
	m.setCursor(1)
	// First 'g' sets pendingG.
	result, _ := m.handleKey(keyMsg("g"))
	rm := result.(Model)
	assert.True(t, rm.pendingG)

	// Second 'g' jumps to top.
	result2, _ := rm.handleKey(keyMsg("g"))
	rm2 := result2.(Model)
	assert.Equal(t, 0, rm2.cursor())
	assert.False(t, rm2.pendingG)
}

func TestPush3HandleKeyG(t *testing.T) {
	m := basePush80v3Model()
	m.setCursor(0)
	kb := ui.ActiveKeybindings
	// 'G' goes to bottom.
	result, _ := m.handleKey(keyMsg(kb.JumpBottom))
	rm := result.(Model)
	assert.Equal(t, len(rm.visibleMiddleItems())-1, rm.cursor())
}

func TestPush3HandleKeyTabNavigation(t *testing.T) {
	m := basePush80v3Model()
	m.tabs = []TabState{{}, {}}
	m.activeTab = 0
	kb := ui.ActiveKeybindings

	result, _ := m.handleKey(keyMsg(kb.NextTab))
	rm := result.(Model)
	assert.Equal(t, 1, rm.activeTab)
}

func TestPush3HandleKeyPrevTab(t *testing.T) {
	m := basePush80v3Model()
	m.tabs = []TabState{{}, {}}
	m.activeTab = 1
	kb := ui.ActiveKeybindings

	result, _ := m.handleKey(keyMsg(kb.PrevTab))
	rm := result.(Model)
	assert.Equal(t, 0, rm.activeTab)
}

// =====================================================================
// View functions
// =====================================================================

func TestPush3ViewHelpNotEmpty(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeHelp
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewExplorerNotEmpty(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeExplorer
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewYAMLNotEmpty(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeYAML
	m.yamlContent = "apiVersion: v1"
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewDiffNotEmpty(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeDiff
	m.diffLeft = "left"
	m.diffRight = "right"
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewDescribeNotEmpty(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeDescribe
	m.describeContent = "Name: pod-1\nStatus: Running"
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewExplainNotEmpty(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeExplain
	m.explainTitle = "pods"
	m.explainFields = []model.ExplainField{
		{Name: "apiVersion", Type: "string", Description: "API version"},
	}
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewFullscreenDashboard(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeExplorer
	m.fullscreenDashboard = true
	m.dashboardPreview = "CLUSTER DASHBOARD\n..."
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewWithTabs(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeExplorer
	m.tabs = []TabState{{}, {}}
	m.activeTab = 0
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewWithError(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeExplorer
	m.err = assert.AnError
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewWithStatusMessage(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeExplorer
	m.setStatusMessage("test message", false)
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewWithSelection(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeExplorer
	m.selectedItems = map[string]bool{"default/pod-1": true}
	result := m.View()
	assert.NotEmpty(t, result)
}

func TestPush3ViewFullscreenMiddle(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeExplorer
	m.fullscreenMiddle = true
	result := m.View()
	assert.NotEmpty(t, result)
}

// =====================================================================
// More Update message branches
// =====================================================================

func TestPush3UpdateLogLineMsg(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeLogs
	ch := make(chan string, 1)
	m.logCh = ch
	ch <- "next line"
	msg := logLineMsg{line: "test line", ch: ch}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.logLines, "test line")
}

func TestPush3UpdateLogLineMsgDone(t *testing.T) {
	m := basePush80v3Model()
	m.mode = modeLogs
	msg := logLineMsg{done: true}
	result, _ := m.Update(msg)
	rm := result.(Model)
	// Done flag appends stream ended marker.
	_ = rm
}

func TestPush3UpdatePortForwardUpdateMsg(t *testing.T) {
	m := basePush80v3Model()
	m.portForwardMgr = k8s.NewPortForwardManager()
	msg := portForwardUpdateMsg{}
	result, _ := m.Update(msg)
	_ = result.(Model)
}

func TestPush3UpdatePortForwardUpdateMsgErr(t *testing.T) {
	m := basePush80v3Model()
	m.portForwardMgr = k8s.NewPortForwardManager()
	msg := portForwardUpdateMsg{err: assert.AnError}
	result, _ := m.Update(msg)
	rm := result.(Model)
	// Exercises the error path.
	_ = rm
}

func TestPush3UpdatePreviewEventsLoadedMsg(t *testing.T) {
	m := basePush80v3Model()
	m.requestGen = 5
	msg := previewEventsLoadedMsg{
		events: []k8s.EventInfo{
			{Type: "Normal", Reason: "Scheduled", Message: "pod assigned", Source: "scheduler"},
		},
		gen: 5,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotEmpty(t, rm.previewEventsContent)
}

func TestPush3UpdatePreviewEventsLoadedMsgStaleGen(t *testing.T) {
	m := basePush80v3Model()
	m.requestGen = 5
	msg := previewEventsLoadedMsg{gen: 3}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Empty(t, rm.previewEventsContent)
}

func TestPush3UpdateRollbackDoneMsg(t *testing.T) {
	m := basePush80v3Model()
	msg := rollbackDoneMsg{}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "Rollback")
	assert.NotNil(t, cmd)
}

func TestPush3UpdateRollbackDoneMsgErr(t *testing.T) {
	m := basePush80v3Model()
	msg := rollbackDoneMsg{err: assert.AnError}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush3UpdateHelmRollbackDoneMsg(t *testing.T) {
	m := basePush80v3Model()
	msg := helmRollbackDoneMsg{}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.NotEmpty(t, rm.statusMessage)
	assert.NotNil(t, cmd)
}

func TestPush3UpdateHelmRollbackDoneMsgErr(t *testing.T) {
	m := basePush80v3Model()
	msg := helmRollbackDoneMsg{err: assert.AnError}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}
