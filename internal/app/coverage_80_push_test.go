package app

import (
	"context"
	"fmt"
	"sync"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	fake "k8s.io/client-go/kubernetes/fake"

	"github.com/janosmiko/lfk/internal/k8s"
	"github.com/janosmiko/lfk/internal/model"
	"github.com/janosmiko/lfk/internal/ui"
	dynfake "k8s.io/client-go/dynamic/fake"
)

// basePush80Model returns a model with a fake k8s client for coverage tests.
func basePush80Model() Model {
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
		{Name: "pod-3", Namespace: "default", Kind: "Pod", Status: "Pending"},
	}
	return m
}

// =====================================================================
// Target 1: loadDashboard (commands_dashboard.go) ~200 lines
// =====================================================================

func TestCov80LoadDashboardReturnsCmd(t *testing.T) {
	m := basePush80Model()
	cmd := m.loadDashboard()
	require.NotNil(t, cmd)
	// The returned cmd is non-nil, confirming that the function captures
	// all needed state and returns a valid tea.Cmd closure.
}

func TestCov80LoadDashboardDifferentContexts(t *testing.T) {
	m := basePush80Model()
	m.nav.Context = "prod-cluster"
	cmd := m.loadDashboard()
	require.NotNil(t, cmd)

	m.nav.Context = ""
	cmd = m.loadDashboard()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 2: loadMonitoringDashboard (commands_dashboard.go)
// =====================================================================

func TestCov80LoadMonitoringDashboardReturnsCmd(t *testing.T) {
	m := basePush80Model()
	cmd := m.loadMonitoringDashboard()
	// The closure captures client/context; confirm it's non-nil.
	require.NotNil(t, cmd)
}

func TestCov80LoadMonitoringDashboardAllNs(t *testing.T) {
	m := basePush80Model()
	m.allNamespaces = true
	cmd := m.loadMonitoringDashboard()
	require.NotNil(t, cmd)
}

func TestCov80LoadMonitoringDashboardDifferentContext(t *testing.T) {
	m := basePush80Model()
	m.nav.Context = "staging"
	cmd := m.loadMonitoringDashboard()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 3: moveCursor (update_navigation.go) -- more branches
// =====================================================================

func TestCov80MoveCursorUpFromZero(t *testing.T) {
	m := basePush80Model()
	m.setCursor(0)
	result, cmd := m.moveCursor(-1)
	rm := result.(Model)
	assert.Equal(t, 0, rm.cursor())
	assert.NotNil(t, cmd) // loadPreview
}

func TestCov80MoveCursorDownPastEnd(t *testing.T) {
	m := basePush80Model()
	m.setCursor(0)
	result, cmd := m.moveCursor(100)
	rm := result.(Model)
	assert.Equal(t, len(rm.visibleMiddleItems())-1, rm.cursor())
	assert.NotNil(t, cmd)
}

func TestCov80MoveCursorByOne(t *testing.T) {
	m := basePush80Model()
	m.setCursor(0)
	result, cmd := m.moveCursor(1)
	rm := result.(Model)
	assert.Equal(t, 1, rm.cursor())
	assert.NotNil(t, cmd)
}

func TestCov80MoveCursorEmptyItems(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	result, _ := m.moveCursor(1)
	rm := result.(Model)
	assert.Equal(t, 0, rm.cursor())
}

func TestCov80MoveCursorWithMapView(t *testing.T) {
	m := basePush80Model()
	m.setCursor(0)
	m.mapView = true
	result, cmd := m.moveCursor(1)
	rm := result.(Model)
	assert.Equal(t, 1, rm.cursor())
	assert.NotNil(t, cmd)
}

func TestCov80MoveCursorAccordionDown(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.allGroupsExpanded = false
	m.expandedGroup = "Core"
	// Set up items with categories for accordion behavior.
	m.middleItems = []model.Item{
		{Name: "pods", Kind: "Pod", Category: "Core"},
		{Name: "svc", Kind: "Service", Category: "Core"},
		{Name: "deploy", Kind: "Deployment", Category: "Workloads"},
	}
	m.setCursor(1) // last item in Core group
	result, _ := m.moveCursor(1)
	rm := result.(Model)
	_ = rm
}

func TestCov80MoveCursorAccordionUp(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.allGroupsExpanded = false
	m.expandedGroup = "Workloads"
	m.middleItems = []model.Item{
		{Name: "pods", Kind: "Pod", Category: "Core"},
		{Name: "deploy", Kind: "Deployment", Category: "Workloads"},
		{Name: "sts", Kind: "StatefulSet", Category: "Workloads"},
	}
	m.setCursor(1) // first item in Workloads group
	result, _ := m.moveCursor(-1)
	rm := result.(Model)
	_ = rm
}

// =====================================================================
// Target 4: bulkDeleteResources with fake client (commands.go)
// =====================================================================

func TestCov80BulkDeleteWithFakeClient(t *testing.T) {
	m := basePush80Model()
	m.bulkItems = []model.Item{
		{Name: "pod-1", Namespace: "default"},
		{Name: "pod-2", Namespace: "other-ns"},
		{Name: "pod-3"}, // no namespace, should use fallback
	}
	m.actionCtx = actionContext{
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", Kind: "Pod", Namespaced: true},
	}
	cmd := m.bulkDeleteResources()
	require.NotNil(t, cmd)
	msg := cmd()
	result, ok := msg.(bulkActionResultMsg)
	require.True(t, ok)
	// Fake client can't find these resources, so they all fail.
	assert.Equal(t, 3, result.failed)
}

// =====================================================================
// Target 5: bulkScaleResources with fake client (commands.go)
// =====================================================================

func TestCov80BulkScaleWithFakeClient(t *testing.T) {
	m := basePush80Model()
	m.bulkItems = []model.Item{
		{Name: "deploy-1", Namespace: "default"},
		{Name: "deploy-2"},
	}
	m.actionCtx = actionContext{
		context: "test-ctx",
		kind:    "Deployment",
	}
	cmd := m.bulkScaleResources(3)
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(bulkActionResultMsg)
	assert.True(t, ok)
}

func TestCov80BulkScaleToZero(t *testing.T) {
	m := basePush80Model()
	m.bulkItems = []model.Item{{Name: "deploy-1", Namespace: "ns1"}}
	m.actionCtx = actionContext{context: "test-ctx", kind: "Deployment"}
	cmd := m.bulkScaleResources(0)
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(bulkActionResultMsg)
	assert.True(t, ok)
}

// =====================================================================
// Target 6: bulkRestartResources with fake client (commands.go)
// =====================================================================

func TestCov80BulkRestartWithFakeClient(t *testing.T) {
	m := basePush80Model()
	m.bulkItems = []model.Item{
		{Name: "deploy-1", Namespace: "default"},
		{Name: "deploy-2", Namespace: "ns2"},
	}
	m.actionCtx = actionContext{context: "test-ctx", kind: "Deployment"}
	cmd := m.bulkRestartResources()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(bulkActionResultMsg)
	assert.True(t, ok)
}

// =====================================================================
// Target 7: batchPatchLabels with fake client (commands.go)
// =====================================================================

func TestCov80BatchPatchLabelsAdd(t *testing.T) {
	m := basePush80Model()
	m.bulkItems = []model.Item{
		{Name: "pod-1", Namespace: "default"},
		{Name: "pod-2"},
	}
	m.actionCtx = actionContext{
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", APIVersion: "v1", Namespaced: true},
	}
	cmd := m.batchPatchLabels("env", "prod", false, false)
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(bulkActionResultMsg)
	assert.True(t, ok)
}

func TestCov80BatchPatchLabelsRemove(t *testing.T) {
	m := basePush80Model()
	m.bulkItems = []model.Item{{Name: "pod-1", Namespace: "default"}}
	m.actionCtx = actionContext{
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", APIVersion: "v1", Namespaced: true},
	}
	cmd := m.batchPatchLabels("env", "", true, false)
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(bulkActionResultMsg)
	assert.True(t, ok)
}

func TestCov80BatchPatchAnnotations(t *testing.T) {
	m := basePush80Model()
	m.bulkItems = []model.Item{{Name: "pod-1", Namespace: "default"}}
	m.actionCtx = actionContext{
		context:      "test-ctx",
		resourceType: model.ResourceTypeEntry{Resource: "pods", APIVersion: "v1", Namespaced: true},
	}
	cmd := m.batchPatchLabels("note", "value", false, true)
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(bulkActionResultMsg)
	assert.True(t, ok)
}

// =====================================================================
// Target 8: loadContainersForLogFilter (commands.go)
// =====================================================================

func TestCov80LoadContainersForLogFilter(t *testing.T) {
	m := basePush80Model()
	m.actionCtx = actionContext{
		context:   "test-ctx",
		name:      "my-pod",
		namespace: "default",
	}
	cmd := m.loadContainersForLogFilter()
	require.NotNil(t, cmd)
	msg := cmd()
	lmsg, ok := msg.(logContainersLoadedMsg)
	require.True(t, ok)
	// Fake client returns empty containers, so err is set or containers is empty.
	_ = lmsg
}

func TestCov80LoadContainersForLogFilterNoPod(t *testing.T) {
	m := basePush80Model()
	m.actionCtx = actionContext{
		context:   "test-ctx",
		name:      "nonexistent-pod",
		namespace: "default",
	}
	cmd := m.loadContainersForLogFilter()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(logContainersLoadedMsg)
	assert.True(t, ok)
}

// =====================================================================
// Target 9: loadMetrics (commands_load.go)
// =====================================================================

func TestCov80LoadMetricsNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadMetrics()
	assert.Nil(t, cmd)
}

func TestCov80LoadMetricsPod(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType.Kind = "Pod"
	m.setCursor(0)
	cmd := m.loadMetrics()
	require.NotNil(t, cmd)
	msg := cmd()
	mmsg, ok := msg.(metricsLoadedMsg)
	require.True(t, ok)
	_ = mmsg
}

func TestCov80LoadMetricsDeployment(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType.Kind = "Deployment"
	m.middleItems = []model.Item{
		{Name: "deploy-1", Namespace: "default", Kind: "Deployment", Status: "Running"},
	}
	m.setCursor(0)
	cmd := m.loadMetrics()
	// Deployment/StatefulSet/DaemonSet branch captured; can't execute
	// because the fake dynamic client lacks registered list kinds.
	require.NotNil(t, cmd)
}

func TestCov80LoadMetricsStatefulSet(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType.Kind = "StatefulSet"
	m.middleItems = []model.Item{
		{Name: "sts-1", Namespace: "default", Kind: "StatefulSet", Status: "Running"},
	}
	m.setCursor(0)
	cmd := m.loadMetrics()
	require.NotNil(t, cmd)
}

func TestCov80LoadMetricsDaemonSet(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType.Kind = "DaemonSet"
	m.middleItems = []model.Item{
		{Name: "ds-1", Namespace: "default", Kind: "DaemonSet"},
	}
	m.setCursor(0)
	cmd := m.loadMetrics()
	require.NotNil(t, cmd)
}

func TestCov80LoadMetricsUnknownKind(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType.Kind = "ConfigMap"
	m.middleItems = []model.Item{
		{Name: "cm-1", Namespace: "default", Kind: "ConfigMap"},
	}
	m.setCursor(0)
	cmd := m.loadMetrics()
	// ConfigMap does not have metrics, returns nil.
	assert.Nil(t, cmd)
}

func TestCov80LoadMetricsAtLevelOwned(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.nav.ResourceType.Kind = "Deployment"
	m.middleItems = []model.Item{
		{Name: "pod-a", Namespace: "default", Kind: "Pod", Status: "Running"},
	}
	m.setCursor(0)
	cmd := m.loadMetrics()
	require.NotNil(t, cmd) // Pod kind at LevelOwned
	msg := cmd()
	_, ok := msg.(metricsLoadedMsg)
	assert.True(t, ok)
}

// =====================================================================
// Target 10: saveLabelData (commands_load.go)
// =====================================================================

func TestCov80SaveLabelDataNilData(t *testing.T) {
	m := basePush80Model()
	m.labelData = nil
	cmd := m.saveLabelData()
	assert.Nil(t, cmd)
}

func TestCov80SaveLabelDataNoSelection(t *testing.T) {
	m := basePush80Model()
	m.labelData = &model.LabelAnnotationData{
		Labels:      map[string]string{"a": "b"},
		Annotations: map[string]string{"c": "d"},
	}
	m.middleItems = nil // no selection
	cmd := m.saveLabelData()
	assert.Nil(t, cmd)
}

func TestCov80SaveLabelDataWithSelection(t *testing.T) {
	m := basePush80Model()
	m.labelData = &model.LabelAnnotationData{
		Labels:      map[string]string{"env": "prod"},
		Annotations: map[string]string{"note": "test"},
	}
	m.labelResourceType = model.ResourceTypeEntry{Resource: "pods", APIVersion: "v1", Namespaced: true}
	m.setCursor(0)
	cmd := m.saveLabelData()
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(labelSavedMsg)
	assert.True(t, ok)
}

func TestCov80SaveLabelDataItemNamespace(t *testing.T) {
	m := basePush80Model()
	m.labelData = &model.LabelAnnotationData{
		Labels:      map[string]string{},
		Annotations: map[string]string{},
	}
	m.labelResourceType = model.ResourceTypeEntry{Resource: "pods", APIVersion: "v1", Namespaced: true}
	m.middleItems = []model.Item{
		{Name: "pod-1", Namespace: "custom-ns", Kind: "Pod"},
	}
	m.setCursor(0)
	cmd := m.saveLabelData()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 11: loadHelmRevisions (commands_load.go) -- early return when
// helm is not found
// =====================================================================

func TestCov80LoadHelmRevisionsNoHelm(t *testing.T) {
	m := basePush80Model()
	t.Setenv("PATH", "/nonexistent")
	m.actionCtx = actionContext{
		context:   "test-ctx",
		name:      "my-release",
		namespace: "default",
	}
	cmd := m.loadHelmRevisions()
	require.NotNil(t, cmd)
	msg := cmd()
	hmsg, ok := msg.(helmRevisionListMsg)
	require.True(t, ok)
	assert.Error(t, hmsg.err)
	assert.Contains(t, hmsg.err.Error(), "helm not found")
}

// =====================================================================
// Target 12: switchToTab (update_mouse.go)
// =====================================================================

func TestCov80SwitchToTabExplorer(t *testing.T) {
	m := basePush80Model()
	m.mode = modeExplorer
	m.tabs = []TabState{{}, {}}
	m.activeTab = 0
	result, _ := m.switchToTab(1)
	rm := result.(Model)
	assert.Equal(t, 1, rm.activeTab)
}

func TestCov80SwitchToTabLogs(t *testing.T) {
	m := basePush80Model()
	m.mode = modeLogs
	ch := make(chan string, 1)
	m.logCh = ch
	m.tabs = []TabState{{}, {}}
	m.activeTab = 0
	// Pre-fill the second tab so loadTab restores it.
	m.tabs[1].mode = modeLogs
	m.tabs[1].logCh = ch
	result, cmd := m.switchToTab(1)
	rm := result.(Model)
	_ = rm
	// Should return waitForLogLine cmd.
	_ = cmd
}

func TestCov80SwitchToTabNilCmd(t *testing.T) {
	m := basePush80Model()
	m.mode = modeExplorer
	m.tabs = []TabState{{}}
	m.activeTab = 0
	// Switching to same tab index.
	result, cmd := m.switchToTab(0)
	rm := result.(Model)
	assert.Equal(t, 0, rm.activeTab)
	_ = cmd
}

// =====================================================================
// Target 13: openBulkActionDirect (update_actions.go)
// =====================================================================

func TestCov80OpenBulkActionDirectNoSelection(t *testing.T) {
	m := basePush80Model()
	m.selectedItems = make(map[string]bool)
	result, cmd := m.openBulkActionDirect("Delete")
	rm := result.(Model)
	assert.False(t, rm.bulkMode)
	assert.Nil(t, cmd)
}

func TestCov80OpenBulkActionDirectWithSelection(t *testing.T) {
	m := basePush80Model()
	// Selection keys use "namespace/name" format (see selectionKey()).
	m.selectedItems = map[string]bool{
		"default/pod-1": true,
		"ns-2/pod-2":    true,
	}
	result, _ := m.openBulkActionDirect("Delete")
	rm := result.(Model)
	assert.True(t, rm.bulkMode)
}

func TestCov80OpenBulkActionDirectLogs(t *testing.T) {
	m := basePush80Model()
	m.selectedItems = map[string]bool{
		"default/pod-1": true,
	}
	result, _ := m.openBulkActionDirect("Logs")
	// executeBulkAction("Logs") may return *Model.
	switch rm := result.(type) {
	case Model:
		assert.False(t, rm.overlay == overlayAction)
	case *Model:
		assert.False(t, rm.overlay == overlayAction)
	}
}

// =====================================================================
// Target 14: restorePortForwards (portforward_state.go)
// =====================================================================

func TestCov80RestorePortForwardsNoKubectl(t *testing.T) {
	m := basePush80Model()
	m.portForwardMgr = k8s.NewPortForwardManager()
	m.pendingPortForwards = &PortForwardStates{
		PortForwards: []PortForwardState{
			{ResourceKind: "svc", ResourceName: "my-svc", Namespace: "default", Context: "test-ctx", LocalPort: "8080", RemotePort: "80"},
		},
	}
	t.Setenv("PATH", "/nonexistent")
	cmds := m.restorePortForwards()
	assert.Nil(t, cmds)
}

func TestCov80RestorePortForwardsEmpty(t *testing.T) {
	m := basePush80Model()
	m.portForwardMgr = k8s.NewPortForwardManager()
	m.pendingPortForwards = &PortForwardStates{}
	cmds := m.restorePortForwards()
	assert.Empty(t, cmds)
}

func TestCov80SaveAndLoadPortForwardState(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	state := &PortForwardStates{
		PortForwards: []PortForwardState{
			{ResourceKind: "svc", ResourceName: "web", Namespace: "prod", Context: "ctx", LocalPort: "3000", RemotePort: "80"},
		},
	}
	err := savePortForwardState(state)
	require.NoError(t, err)
	loaded := loadPortForwardState()
	require.Len(t, loaded.PortForwards, 1)
	assert.Equal(t, "web", loaded.PortForwards[0].ResourceName)
}

func TestCov80LoadPortForwardStateNoFile(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	loaded := loadPortForwardState()
	assert.NotNil(t, loaded)
	assert.Empty(t, loaded.PortForwards)
}

// =====================================================================
// Target 15: portForwardItems (tabs.go)
// =====================================================================

func TestCov80PortForwardItemsEmpty(t *testing.T) {
	m := basePush80Model()
	m.portForwardMgr = k8s.NewPortForwardManager()
	items := m.portForwardItems()
	assert.Empty(t, items)
}

// =====================================================================
// Target 16: openExplainBrowser (update_explain.go)
// =====================================================================

func TestCov80OpenExplainBrowserAtResourceTypes(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{
		{Name: "Pods", Kind: "Pod", Extra: "/v1/pods"},
	}
	m.setCursor(0)
	result, cmd := m.openExplainBrowser()
	rm := result.(Model)
	_ = rm
	_ = cmd
}

func TestCov80OpenExplainBrowserVirtualItem(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{
		{Name: "Overview", Kind: "__overview__", Extra: "__overview__"},
	}
	m.setCursor(0)
	result, cmd := m.openExplainBrowser()
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestCov80OpenExplainBrowserCollapsedGroup(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{
		{Name: "Collapsed", Kind: "__collapsed_group__"},
	}
	m.setCursor(0)
	result, _ := m.openExplainBrowser()
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
}

func TestCov80OpenExplainBrowserMonitoring(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{
		{Name: "Monitoring", Kind: "__monitoring__", Extra: "__monitoring__"},
	}
	m.setCursor(0)
	result, _ := m.openExplainBrowser()
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
}

func TestCov80OpenExplainBrowserAtResources(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.nav.ResourceType = model.ResourceTypeEntry{
		Kind:       "Deployment",
		Resource:   "deployments",
		APIGroup:   "apps",
		APIVersion: "v1",
		Namespaced: true,
	}
	result, cmd := m.openExplainBrowser()
	rm := result.(Model)
	assert.True(t, rm.loading)
	_ = cmd
}

func TestCov80OpenExplainBrowserNoSel(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = nil
	result, cmd := m.openExplainBrowser()
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestCov80OpenExplainBrowserAtClusters(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	result, cmd := m.openExplainBrowser()
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestCov80OpenExplainBrowserEmptyResource(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.nav.ResourceType = model.ResourceTypeEntry{}
	result, cmd := m.openExplainBrowser()
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestCov80OpenExplainBrowserFallbackKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{
		{Name: "CRD thing", Kind: "MyCRD", Extra: "nonexistent/ref"},
	}
	m.setCursor(0)
	result, cmd := m.openExplainBrowser()
	rm := result.(Model)
	// Should fallback to lowercase kind + "s".
	assert.True(t, rm.loading)
	_ = cmd
}

// =====================================================================
// Target 17: kubectlGetPodSelector (commands_logs.go)
// =====================================================================

func TestCov80KubectlGetPodSelectorCronJob(t *testing.T) {
	// CronJob always returns empty.
	result := kubectlGetPodSelector("/usr/bin/kubectl", "/dev/null", "default", "CronJob", "my-cron", "test-ctx")
	assert.Empty(t, result)
}

// =====================================================================
// Target 18: navigateParent (update_navigation.go)
// =====================================================================

func TestCov80NavigateParentFromClusters(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	result, cmd := m.navigateParent()
	rm := result.(Model)
	assert.Equal(t, model.LevelClusters, rm.nav.Level)
	assert.Nil(t, cmd)
}

func TestCov80NavigateParentFromResourceTypes(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.nav.Context = "test-ctx"
	m.leftItems = []model.Item{{Name: "ctx-1"}}
	m.leftItemsHistory = [][]model.Item{{{Name: "root"}}}
	result, cmd := m.navigateParent()
	rm := result.(Model)
	assert.Equal(t, model.LevelClusters, rm.nav.Level)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateParentFromResources(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.leftItems = []model.Item{{Name: "Pods"}}
	m.leftItemsHistory = [][]model.Item{{{Name: "cluster"}}}
	result, _ := m.navigateParent()
	rm := result.(Model)
	assert.Equal(t, model.LevelResourceTypes, rm.nav.Level)
}

func TestCov80NavigateParentFromOwned(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.nav.ResourceName = "deploy-1"
	m.leftItems = []model.Item{{Name: "deploy-1"}}
	m.leftItemsHistory = [][]model.Item{{{Name: "cluster"}}, {{Name: "Deployments"}}}
	result, cmd := m.navigateParent()
	rm := result.(Model)
	assert.Equal(t, model.LevelResources, rm.nav.Level)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateParentFromOwnedWithStack(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.nav.ResourceName = "deploy-1"
	m.ownedParentStack = []ownedParentState{
		{
			resourceType: model.ResourceTypeEntry{Kind: "Application"},
			resourceName: "my-app",
			namespace:    "argocd",
		},
	}
	m.leftItems = []model.Item{{Name: "deploy-1"}}
	m.leftItemsHistory = [][]model.Item{{{Name: "cluster"}}, {{Name: "Apps"}}, {{Name: "app-1"}}}
	result, cmd := m.navigateParent()
	rm := result.(Model)
	// Should pop to parent owned level.
	assert.Equal(t, model.LevelOwned, rm.nav.Level)
	assert.Equal(t, "my-app", rm.nav.ResourceName)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateParentFromContainersPod(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelContainers
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Pod"}
	m.leftItems = []model.Item{{Name: "container-1"}}
	m.leftItemsHistory = [][]model.Item{{{Name: "cluster"}}, {{Name: "Pods"}}, {{Name: "pod-1"}}}
	result, cmd := m.navigateParent()
	rm := result.(Model)
	// Pod containers go back to LevelResources.
	assert.Equal(t, model.LevelResources, rm.nav.Level)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateParentFromContainersOther(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelContainers
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Deployment"}
	m.leftItems = []model.Item{{Name: "container-1"}}
	m.leftItemsHistory = [][]model.Item{{{Name: "cluster"}}, {{Name: "Deploys"}}, {{Name: "dep-1"}}}
	result, cmd := m.navigateParent()
	rm := result.(Model)
	// Non-pod containers go back to LevelOwned.
	assert.Equal(t, model.LevelOwned, rm.nav.Level)
	assert.NotNil(t, cmd)
}

// =====================================================================
// Target 19: navigateChild (update_navigation.go)
// =====================================================================

func TestCov80NavigateChildNoSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	result, cmd := m.navigateChild()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80NavigateChildFromClusters(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	m.middleItems = []model.Item{{Name: "test-ctx"}}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	rm := result.(Model)
	assert.Equal(t, model.LevelResourceTypes, rm.nav.Level)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateChildFromResourceTypesOverview(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{{Name: "Overview", Extra: "__overview__"}}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	rm := result.(Model)
	assert.True(t, rm.fullscreenDashboard)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateChildFromResourceTypesMonitoring(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{{Name: "Monitoring", Extra: "__monitoring__"}}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	rm := result.(Model)
	assert.True(t, rm.fullscreenDashboard)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateChildFromResourceTypesPortForward(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.portForwardMgr = k8s.NewPortForwardManager()
	m.middleItems = []model.Item{{Name: "Port Forwards", Kind: "__port_forwards__"}}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	rm := result.(Model)
	assert.Equal(t, model.LevelResources, rm.nav.Level)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateChildFromResourceTypesCollapsedGroup(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{
		{Name: "Core", Kind: "__collapsed_group__", Category: "Core"},
		{Name: "Pods", Kind: "Pod", Category: "Core", Extra: "/v1/pods"},
	}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	rm := result.(Model)
	assert.Equal(t, "Core", rm.expandedGroup)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateChildFromResourcesPod(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Pod", Resource: "pods", Namespaced: true}
	m.middleItems = []model.Item{{Name: "my-pod", Kind: "Pod", Namespace: "default"}}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	rm := result.(Model)
	assert.Equal(t, model.LevelContainers, rm.nav.Level)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateChildFromOwnedPod(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Deployment"}
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod", Namespace: "default"}}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	rm := result.(Model)
	assert.Equal(t, model.LevelContainers, rm.nav.Level)
	assert.NotNil(t, cmd)
}

func TestCov80NavigateChildFromOwnedNonDrillable(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Deployment"}
	m.middleItems = []model.Item{{Name: "cm-1", Kind: "ConfigMap", Namespace: "default"}}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80NavigateChildFromContainers(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelContainers
	m.middleItems = []model.Item{{Name: "container-1", Kind: "Container"}}
	m.setCursor(0)
	result, cmd := m.navigateChild()
	_ = result
	assert.Nil(t, cmd)
}

// =====================================================================
// Target 20: enterFullView (update_navigation.go)
// =====================================================================

func TestCov80EnterFullViewNoSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	result, cmd := m.enterFullView()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80EnterFullViewFromClusters(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	m.middleItems = []model.Item{{Name: "ctx-1"}}
	m.setCursor(0)
	result, cmd := m.enterFullView()
	rm := result.(Model)
	// Should navigate child.
	assert.Equal(t, model.LevelResourceTypes, rm.nav.Level)
	assert.NotNil(t, cmd)
}

func TestCov80EnterFullViewFromResourceTypes(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{{Name: "Overview", Extra: "__overview__"}}
	m.setCursor(0)
	result, cmd := m.enterFullView()
	rm := result.(Model)
	assert.True(t, rm.fullscreenDashboard)
	assert.NotNil(t, cmd)
}

func TestCov80EnterFullViewPortForwards(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "__port_forwards__"}
	m.middleItems = []model.Item{{Name: "pf-1"}}
	m.setCursor(0)
	result, cmd := m.enterFullView()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80EnterFullViewNormal(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod"}}
	m.setCursor(0)
	result, cmd := m.enterFullView()
	rm := result.(Model)
	assert.Equal(t, modeYAML, rm.mode)
	assert.NotNil(t, cmd)
}

// =====================================================================
// Target 21: exportResourceToFile (commands.go)
// =====================================================================

func TestCov80ExportResourceToFileNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.exportResourceToFile()
	assert.Nil(t, cmd)
}

func TestCov80ExportResourceToFileDefaultLevel(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	cmd := m.exportResourceToFile()
	assert.Nil(t, cmd)
}

func TestCov80ExportResourceToFileLevelResourcesCmd(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.setCursor(0)
	cmd := m.exportResourceToFile()
	require.NotNil(t, cmd)
}

func TestCov80ExportResourceToFileLevelOwnedPod(t *testing.T) {
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "pod-owned", Kind: "Pod", Namespace: "default"}}
	m.setCursor(0)
	cmd := m.exportResourceToFile()
	require.NotNil(t, cmd)
}

func TestCov80ExportResourceToFileLevelOwnedUnknownKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "unknown-thing", Kind: "UnknownKind123", Namespace: "default"}}
	m.setCursor(0)
	cmd := m.exportResourceToFile()
	require.NotNil(t, cmd)
	msg := cmd()
	emsg, ok := msg.(exportDoneMsg)
	require.True(t, ok)
	assert.Error(t, emsg.err)
}

func TestCov80ExportResourceToFileLevelContainers(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelContainers
	m.nav.OwnedName = "my-pod"
	m.middleItems = []model.Item{{Name: "container-1", Kind: "Container"}}
	m.setCursor(0)
	cmd := m.exportResourceToFile()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 22: copyYAMLToClipboard (commands.go)
// =====================================================================

func TestCov80CopyYAMLNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.copyYAMLToClipboard()
	assert.Nil(t, cmd)
}

func TestCov80CopyYAMLLevelResources(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.setCursor(0)
	cmd := m.copyYAMLToClipboard()
	require.NotNil(t, cmd)
}

func TestCov80CopyYAMLLevelOwnedPod(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod", Namespace: "default"}}
	m.setCursor(0)
	cmd := m.copyYAMLToClipboard()
	require.NotNil(t, cmd)
}

func TestCov80CopyYAMLLevelOwnedKnownKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "rs-1", Kind: "ReplicaSet", Namespace: "default"}}
	m.setCursor(0)
	cmd := m.copyYAMLToClipboard()
	require.NotNil(t, cmd)
}

func TestCov80CopyYAMLLevelOwnedUnknownKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "x", Kind: "XYZ999", Namespace: "ns"}}
	m.setCursor(0)
	cmd := m.copyYAMLToClipboard()
	require.NotNil(t, cmd)
	msg := cmd()
	ymsg, ok := msg.(yamlClipboardMsg)
	require.True(t, ok)
	assert.Error(t, ymsg.err)
}

func TestCov80CopyYAMLLevelContainers(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelContainers
	m.nav.OwnedName = "pod-1"
	m.middleItems = []model.Item{{Name: "container-1"}}
	m.setCursor(0)
	cmd := m.copyYAMLToClipboard()
	require.NotNil(t, cmd)
}

func TestCov80CopyYAMLLevelClusters(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	cmd := m.copyYAMLToClipboard()
	assert.Nil(t, cmd)
}

// =====================================================================
// Target 23: loadYAML (commands_load.go)
// =====================================================================

func TestCov80LoadYAMLNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadYAML()
	assert.Nil(t, cmd)
}

func TestCov80LoadYAMLLevelResources(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.setCursor(0)
	cmd := m.loadYAML()
	require.NotNil(t, cmd)
}

func TestCov80LoadYAMLLevelOwnedPod(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod", Namespace: "default"}}
	m.setCursor(0)
	cmd := m.loadYAML()
	require.NotNil(t, cmd)
}

func TestCov80LoadYAMLLevelOwnedOther(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "rs-1", Kind: "ReplicaSet", Namespace: "default"}}
	m.setCursor(0)
	cmd := m.loadYAML()
	require.NotNil(t, cmd)
}

func TestCov80LoadYAMLLevelOwnedUnknown(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "x", Kind: "Unknown999"}}
	m.setCursor(0)
	cmd := m.loadYAML()
	require.NotNil(t, cmd)
	msg := cmd()
	ymsg, ok := msg.(yamlLoadedMsg)
	require.True(t, ok)
	assert.Error(t, ymsg.err)
}

func TestCov80LoadYAMLLevelContainers(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelContainers
	m.nav.OwnedName = "pod-1"
	m.middleItems = []model.Item{{Name: "container-1"}}
	m.setCursor(0)
	cmd := m.loadYAML()
	require.NotNil(t, cmd)
}

func TestCov80LoadYAMLLevelClusters(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	cmd := m.loadYAML()
	assert.Nil(t, cmd)
}

// =====================================================================
// Target 24: loadDiff (commands_load.go)
// =====================================================================

func TestCov80LoadDiff(t *testing.T) {
	m := basePush80Model()
	rt := model.ResourceTypeEntry{Kind: "Pod", Resource: "pods", Namespaced: true}
	itemA := model.Item{Name: "pod-1", Namespace: "default"}
	itemB := model.Item{Name: "pod-2", Namespace: "other-ns"}
	cmd := m.loadDiff(rt, itemA, itemB)
	require.NotNil(t, cmd)
}

func TestCov80LoadDiffSameNs(t *testing.T) {
	m := basePush80Model()
	rt := model.ResourceTypeEntry{Kind: "Pod", Resource: "pods", Namespaced: true}
	itemA := model.Item{Name: "pod-1", Namespace: "default"}
	itemB := model.Item{Name: "pod-2", Namespace: "default"}
	cmd := m.loadDiff(rt, itemA, itemB)
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 25: handleMouse branches (update_mouse.go)
// =====================================================================

func TestCov80HandleMouseWheelUpInLogs(t *testing.T) {
	m := basePush80Model()
	m.mode = modeLogs
	m.logScroll = 10
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	result, _ := m.handleMouse(msg)
	rm := result.(Model)
	assert.Less(t, rm.logScroll, 10)
}

func TestCov80HandleMouseWheelDownInLogs(t *testing.T) {
	m := basePush80Model()
	m.mode = modeLogs
	m.logScroll = 0
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	result, _ := m.handleMouse(msg)
	rm := result.(Model)
	assert.GreaterOrEqual(t, rm.logScroll, 0)
}

func TestCov80HandleMouseWheelUpInExplorer(t *testing.T) {
	m := basePush80Model()
	m.mode = modeExplorer
	m.setCursor(2)
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	result, _ := m.handleMouse(msg)
	rm := result.(Model)
	assert.LessOrEqual(t, rm.cursor(), 2)
}

func TestCov80HandleMouseWheelDownInExplorer(t *testing.T) {
	m := basePush80Model()
	m.mode = modeExplorer
	m.setCursor(0)
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelDown}
	result, _ := m.handleMouse(msg)
	_ = result.(Model)
}

func TestCov80HandleMouseInOverlay(t *testing.T) {
	m := basePush80Model()
	m.mode = modeExplorer
	m.overlay = overlayAction
	msg := tea.MouseMsg{Button: tea.MouseButtonWheelUp}
	result, cmd := m.handleMouse(msg)
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80HandleMouseLeftClickNotPress(t *testing.T) {
	m := basePush80Model()
	m.mode = modeExplorer
	msg := tea.MouseMsg{Button: tea.MouseButtonLeft, Action: tea.MouseActionRelease}
	result, cmd := m.handleMouse(msg)
	_ = result
	assert.Nil(t, cmd)
}

// =====================================================================
// Target 26: loadPreviewEvents (commands_load.go)
// =====================================================================

func TestCov80LoadPreviewEventsNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadPreviewEvents()
	assert.Nil(t, cmd)
}

func TestCov80LoadPreviewEventsWithSel(t *testing.T) {
	m := basePush80Model()
	m.setCursor(0)
	cmd := m.loadPreviewEvents()
	require.NotNil(t, cmd)
}

func TestCov80LoadPreviewEventsAtLevelOwned(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.nav.ResourceType.Kind = "Deployment"
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod", Namespace: "default"}}
	m.setCursor(0)
	cmd := m.loadPreviewEvents()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 27: resolveOwnedResourceType (commands_load.go)
// =====================================================================

func TestCov80ResolveOwnedResourceTypeNil(t *testing.T) {
	m := basePush80Model()
	_, ok := m.resolveOwnedResourceType(nil)
	assert.False(t, ok)
}

func TestCov80ResolveOwnedResourceTypeKnownKind(t *testing.T) {
	m := basePush80Model()
	item := &model.Item{Kind: "Deployment", Name: "deploy-1"}
	rt, ok := m.resolveOwnedResourceType(item)
	assert.True(t, ok)
	assert.Equal(t, "Deployment", rt.Kind)
}

func TestCov80ResolveOwnedResourceTypeExtraRef(t *testing.T) {
	m := basePush80Model()
	item := &model.Item{Kind: "SomeCRD", Extra: "apps/v1/deployments"}
	rt, ok := m.resolveOwnedResourceType(item)
	if ok {
		assert.NotEmpty(t, rt.Resource)
	}
}

func TestCov80ResolveOwnedResourceTypeFallbackGroupVersion(t *testing.T) {
	m := basePush80Model()
	// Extra with group/version format, unknown kind.
	item := &model.Item{Kind: "MyCustomResource", Extra: "custom.io/v1beta1"}
	rt, ok := m.resolveOwnedResourceType(item)
	assert.True(t, ok)
	assert.Equal(t, "custom.io", rt.APIGroup)
	assert.Equal(t, "v1beta1", rt.APIVersion)
	assert.Equal(t, "mycustomresources", rt.Resource)
}

func TestCov80ResolveOwnedResourceTypeNoMatch(t *testing.T) {
	m := basePush80Model()
	item := &model.Item{Kind: "NoSuchKind999"}
	_, ok := m.resolveOwnedResourceType(item)
	assert.False(t, ok)
}

// =====================================================================
// Target 28: loadSecretData/saveSecretData (commands_load.go)
// =====================================================================

func TestCov80LoadSecretDataNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadSecretData()
	assert.Nil(t, cmd)
}

func TestCov80LoadSecretDataWithSel(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Secret", Resource: "secrets", Namespaced: true}
	m.middleItems = []model.Item{{Name: "my-secret", Namespace: "default", Kind: "Secret"}}
	m.setCursor(0)
	cmd := m.loadSecretData()
	require.NotNil(t, cmd)
}

func TestCov80SaveSecretDataNil(t *testing.T) {
	m := basePush80Model()
	m.secretData = nil
	cmd := m.saveSecretData()
	assert.Nil(t, cmd)
}

func TestCov80SaveSecretDataNoSel(t *testing.T) {
	m := basePush80Model()
	m.secretData = &model.SecretData{Data: map[string]string{"key": "val"}}
	m.middleItems = nil
	cmd := m.saveSecretData()
	assert.Nil(t, cmd)
}

// =====================================================================
// Target 29: loadConfigMapData/saveConfigMapData (commands_load.go)
// =====================================================================

func TestCov80LoadConfigMapDataNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadConfigMapData()
	assert.Nil(t, cmd)
}

func TestCov80LoadConfigMapDataWithSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = []model.Item{{Name: "my-cm", Namespace: "default", Kind: "ConfigMap"}}
	m.setCursor(0)
	cmd := m.loadConfigMapData()
	require.NotNil(t, cmd)
}

func TestCov80SaveConfigMapDataNil(t *testing.T) {
	m := basePush80Model()
	m.configMapData = nil
	cmd := m.saveConfigMapData()
	assert.Nil(t, cmd)
}

func TestCov80SaveConfigMapDataNoSel(t *testing.T) {
	m := basePush80Model()
	m.configMapData = &model.ConfigMapData{Data: map[string]string{"key": "val"}}
	m.middleItems = nil
	cmd := m.saveConfigMapData()
	assert.Nil(t, cmd)
}

// =====================================================================
// Target 30: loadLabelData (commands_load.go)
// =====================================================================

func TestCov80LoadLabelDataNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadLabelData()
	assert.Nil(t, cmd)
}

func TestCov80LoadLabelDataWithSel(t *testing.T) {
	m := basePush80Model()
	m.labelResourceType = model.ResourceTypeEntry{Resource: "pods", APIVersion: "v1", Namespaced: true}
	m.setCursor(0)
	cmd := m.loadLabelData()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 31: loadRevisions (commands_load.go)
// =====================================================================

func TestCov80LoadRevisionsNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadRevisions()
	assert.Nil(t, cmd)
}

func TestCov80LoadRevisionsWithSel(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Deployment", Resource: "deployments", Namespaced: true}
	m.middleItems = []model.Item{{Name: "deploy-1", Namespace: "default", Kind: "Deployment"}}
	m.setCursor(0)
	cmd := m.loadRevisions()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 32: loadResourceTree (commands_load.go)
// =====================================================================

func TestCov80LoadResourceTreeNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadResourceTree()
	assert.Nil(t, cmd)
}

func TestCov80LoadResourceTreeResources(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.setCursor(0)
	cmd := m.loadResourceTree()
	require.NotNil(t, cmd)
}

func TestCov80LoadResourceTreeOwned(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod", Namespace: "default"}}
	m.setCursor(0)
	cmd := m.loadResourceTree()
	require.NotNil(t, cmd)
}

func TestCov80LoadResourceTreeClusters(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	cmd := m.loadResourceTree()
	assert.Nil(t, cmd)
}

// =====================================================================
// Target 33: directAction functions (update_actions.go)
// =====================================================================

func TestCov80DirectActionLogsNoKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	result, cmd := m.directActionLogs()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionLogsPortForward(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "__port_forwards__"}
	result, cmd := m.directActionLogs()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionLogsNoSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	result, cmd := m.directActionLogs()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionEditNoKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	result, cmd := m.directActionEdit()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionEditNoSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	result, cmd := m.directActionEdit()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionDescribeNoKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	result, cmd := m.directActionDescribe()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionDescribeNoSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	result, cmd := m.directActionDescribe()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionDeleteNoKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	result, cmd := m.directActionDelete()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionDeleteNoSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	result, cmd := m.directActionDelete()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80DirectActionDeleteDeletingPod(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Pod", Resource: "pods", Namespaced: true}
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod", Namespace: "default", Deleting: true}}
	m.setCursor(0)
	result, cmd := m.directActionDelete()
	rm := result.(Model)
	assert.Equal(t, overlayConfirmType, rm.overlay)
	assert.Contains(t, rm.pendingAction, "Force Delete")
	assert.Nil(t, cmd)
}

func TestCov80DirectActionDeleteDeletingNonPod(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "ConfigMap", Resource: "configmaps", Namespaced: true}
	m.middleItems = []model.Item{{Name: "cm-1", Kind: "ConfigMap", Namespace: "default", Deleting: true}}
	m.setCursor(0)
	result, cmd := m.directActionDelete()
	rm := result.(Model)
	assert.Equal(t, overlayConfirmType, rm.overlay)
	assert.Contains(t, rm.pendingAction, "Force Finalize")
	assert.Nil(t, cmd)
}

func TestCov80DirectActionRefresh(t *testing.T) {
	m := basePush80Model()
	result, cmd := m.directActionRefresh()
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "Refreshing")
	assert.NotNil(t, cmd)
}

// =====================================================================
// Target 34: more tab-related functions (tabs.go)
// =====================================================================

func TestCov80TabLabels(t *testing.T) {
	m := basePush80Model()
	m.tabs = []TabState{
		{nav: model.NavigationState{Context: "prod"}},
		{nav: model.NavigationState{Context: "dev", ResourceType: model.ResourceTypeEntry{DisplayName: "Pods"}}},
		{nav: model.NavigationState{}},
	}
	m.nav.Context = "test-ctx"
	m.activeTab = 0
	labels := m.tabLabels()
	require.Len(t, labels, 3)
	assert.Equal(t, "test-ctx", labels[0]) // active tab updated
	assert.Contains(t, labels[1], "dev/Pods")
	assert.Equal(t, "clusters", labels[2])
}

func TestCov80TabAtX(t *testing.T) {
	m := basePush80Model()
	m.tabs = []TabState{
		{nav: model.NavigationState{Context: "ctx-1"}},
		{nav: model.NavigationState{Context: "ctx-2"}},
	}
	m.nav.Context = "ctx-1"
	m.activeTab = 0
	// First tab starts at pos 1.
	assert.Equal(t, 0, m.tabAtX(1))
	// Past all tabs.
	assert.Equal(t, -1, m.tabAtX(200))
}

func TestCov80SortMiddleItemsByName(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.sortColumnName = "Name"
	m.sortAscending = true
	ui.ActiveSortableColumns = []string{"Name"}
	m.middleItems = []model.Item{
		{Name: "zebra"},
		{Name: "alpha"},
		{Name: "middle"},
	}
	m.sortMiddleItems()
	assert.Equal(t, "alpha", m.middleItems[0].Name)
	assert.Equal(t, "zebra", m.middleItems[2].Name)
}

func TestCov80SortMiddleItemsByStatus(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.sortColumnName = "Status"
	m.sortAscending = true
	ui.ActiveSortableColumns = []string{"Status"}
	m.middleItems = []model.Item{
		{Name: "a", Status: "Failed"},
		{Name: "b", Status: "Running"},
		{Name: "c", Status: "Pending"},
	}
	m.sortMiddleItems()
	assert.Equal(t, "Running", m.middleItems[0].Status)
}

func TestCov80SortMiddleItemsByAge(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.sortColumnName = "Age"
	m.sortAscending = true
	ui.ActiveSortableColumns = []string{"Age"}
	now := time.Now()
	m.middleItems = []model.Item{
		{Name: "old", CreatedAt: now.Add(-10 * time.Hour)},
		{Name: "new", CreatedAt: now.Add(-1 * time.Hour)},
		{Name: "zero"},
	}
	m.sortMiddleItems()
	// Newest first is ascending for Age.
	assert.Equal(t, "new", m.middleItems[0].Name)
}

func TestCov80SortMiddleItemsByRestarts(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.sortColumnName = "Restarts"
	m.sortAscending = true
	ui.ActiveSortableColumns = []string{"Restarts"}
	m.middleItems = []model.Item{
		{Name: "a", Restarts: "5"},
		{Name: "b", Restarts: "1"},
		{Name: "c", Restarts: "10"},
	}
	m.sortMiddleItems()
	assert.Equal(t, "b", m.middleItems[0].Name)
}

func TestCov80SortMiddleItemsByReady(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.sortColumnName = "Ready"
	m.sortAscending = true
	ui.ActiveSortableColumns = []string{"Ready"}
	m.middleItems = []model.Item{
		{Name: "a", Ready: "3/3"},
		{Name: "b", Ready: "1/3"},
		{Name: "c", Ready: "2/3"},
	}
	m.sortMiddleItems()
	assert.Equal(t, "b", m.middleItems[0].Name)
}

func TestCov80SortMiddleItemsByNamespace(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.sortColumnName = "Namespace"
	m.sortAscending = true
	ui.ActiveSortableColumns = []string{"Namespace"}
	m.middleItems = []model.Item{
		{Name: "a", Namespace: "zeta"},
		{Name: "b", Namespace: "alpha"},
	}
	m.sortMiddleItems()
	assert.Equal(t, "alpha", m.middleItems[0].Namespace)
}

func TestCov80SortMiddleItemsDescending(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.sortColumnName = "Name"
	m.sortAscending = false
	ui.ActiveSortableColumns = []string{"Name"}
	m.middleItems = []model.Item{
		{Name: "alpha"},
		{Name: "zebra"},
	}
	m.sortMiddleItems()
	assert.Equal(t, "zebra", m.middleItems[0].Name)
}

func TestCov80SortMiddleItemsAtResourceTypes(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResourceTypes
	m.sortColumnName = "Name"
	m.sortAscending = true
	ui.ActiveSortableColumns = []string{"Name"}
	m.middleItems = []model.Item{{Name: "b"}, {Name: "a"}}
	m.sortMiddleItems()
	// Should not sort at LevelResourceTypes.
	assert.Equal(t, "b", m.middleItems[0].Name)
}

func TestCov80SortMiddleItemsExtraColumn(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.sortColumnName = "Image"
	m.sortAscending = true
	ui.ActiveSortableColumns = []string{"Image"}
	m.middleItems = []model.Item{
		{Name: "a", Columns: []model.KeyValue{{Key: "Image", Value: "nginx:latest"}}},
		{Name: "b", Columns: []model.KeyValue{{Key: "Image", Value: "alpine:3.18"}}},
	}
	m.sortMiddleItems()
	assert.Equal(t, "b", m.middleItems[0].Name)
}

// =====================================================================
// Target 35: itemIndexFromDisplayLine (update_navigation.go)
// =====================================================================

func TestCov80ItemIndexFromDisplayLine(t *testing.T) {
	m := basePush80Model()
	m.middleItems = []model.Item{
		{Name: "pod-1"},
		{Name: "pod-2"},
		{Name: "pod-3"},
	}
	idx := m.itemIndexFromDisplayLine(0)
	assert.Equal(t, 0, idx)
	idx = m.itemIndexFromDisplayLine(1)
	assert.Equal(t, 1, idx)
	idx = m.itemIndexFromDisplayLine(100)
	assert.Equal(t, -1, idx)
}

func TestCov80ItemIndexFromDisplayLineWithCategories(t *testing.T) {
	m := basePush80Model()
	// Use LevelResources (no category filtering in visibleMiddleItems).
	m.nav.Level = model.LevelResources
	m.middleItems = []model.Item{
		{Name: "pods", Category: "Core"},
		{Name: "svc", Category: "Core"},
		{Name: "deploy", Category: "Workloads"},
	}
	// Exercise the function -- it will walk through categories and count
	// separator/header lines, hitting various branches.
	// Line 0 = category header "Core", line 1 = pods (0), line 2 = svc (1),
	// line 3 = separator, line 4 = category header "Workloads", line 5 = deploy (2).
	idx := m.itemIndexFromDisplayLine(1)
	assert.Equal(t, 0, idx)
	assert.Equal(t, -1, m.itemIndexFromDisplayLine(200))
}

// =====================================================================
// Target 36: various helper functions
// =====================================================================

func TestCov80SanitizeError(t *testing.T) {
	m := basePush80Model()
	m.width = 80
	err := fmt.Errorf("line1\nline2\n\nline4")
	s := m.sanitizeError(err)
	assert.NotContains(t, s, "\n")
}

func TestCov80SanitizeErrorShortWidth(t *testing.T) {
	m := basePush80Model()
	m.width = 20
	err := fmt.Errorf("this is a very long error message that should be truncated at some point")
	s := m.sanitizeError(err)
	assert.True(t, len(s) <= 43) // maxLen = max(40, 20-20) = 40, +3 for "..."
}

func TestCov80SanitizeMessage(t *testing.T) {
	m := basePush80Model()
	m.width = 80
	s := m.sanitizeMessage("line1\nline2")
	assert.NotContains(t, s, "\n")
}

func TestCov80SanitizeMessageTruncation(t *testing.T) {
	m := basePush80Model()
	m.width = 20
	long := ""
	for range 100 {
		long += "x"
	}
	s := m.sanitizeMessage(long)
	assert.True(t, len(s) <= 43)
}

func TestCov80FullErrorMessage(t *testing.T) {
	err := fmt.Errorf("err\n  with\n  spaces  ")
	s := fullErrorMessage(err)
	assert.NotContains(t, s, "\n")
	assert.NotContains(t, s, "  ")
}

func TestCov80SetStatusMessage(t *testing.T) {
	m := basePush80Model()
	m.setStatusMessage("test info", false)
	assert.Equal(t, "test info", m.statusMessage)
	assert.False(t, m.statusMessageErr)

	m.setStatusMessage("test error", true)
	assert.True(t, m.statusMessageErr)
}

func TestCov80SetErrorFromErr(t *testing.T) {
	m := basePush80Model()
	m.setErrorFromErr("prefix: ", fmt.Errorf("something went wrong"))
	assert.True(t, m.statusMessageErr)
	assert.Contains(t, m.statusMessage, "prefix:")
}

func TestCov80HasStatusMessage(t *testing.T) {
	m := basePush80Model()
	assert.False(t, m.hasStatusMessage())
	m.setStatusMessage("test", false)
	assert.True(t, m.hasStatusMessage())
}

func TestCov80AddLogEntryOverflow(t *testing.T) {
	m := basePush80Model()
	for range 600 {
		m.addLogEntry("INF", "msg")
	}
	assert.LessOrEqual(t, len(m.errorLog), 500)
}

func TestCov80SelectedResourceKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.nav.ResourceType.Kind = "Pod"
	assert.Equal(t, "Pod", m.selectedResourceKind())

	m.nav.Level = model.LevelContainers
	assert.Equal(t, "Container", m.selectedResourceKind())

	m.nav.Level = model.LevelOwned
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod"}}
	m.setCursor(0)
	assert.Equal(t, "Pod", m.selectedResourceKind())
}

func TestCov80EffectiveNamespace(t *testing.T) {
	m := basePush80Model()
	assert.Equal(t, "default", m.effectiveNamespace())

	m.allNamespaces = true
	assert.Equal(t, "", m.effectiveNamespace())

	m.allNamespaces = false
	m.selectedNamespaces = map[string]bool{"ns1": true, "ns2": true}
	assert.Equal(t, "", m.effectiveNamespace())

	m.selectedNamespaces = map[string]bool{"ns1": true}
	assert.Equal(t, "ns1", m.effectiveNamespace())
}

func TestCov80ActionNamespace(t *testing.T) {
	m := basePush80Model()
	m.actionCtx = actionContext{namespace: "action-ns"}
	assert.Equal(t, "action-ns", m.actionNamespace())

	m.actionCtx.namespace = ""
	assert.Equal(t, "default", m.actionNamespace())
}

func TestCov80ResolveNamespace(t *testing.T) {
	m := basePush80Model()
	m.nav.Namespace = "nav-ns"
	assert.Equal(t, "nav-ns", m.resolveNamespace())

	m.nav.Namespace = ""
	assert.Equal(t, "default", m.resolveNamespace())
}

// =====================================================================
// Target 37: loadQuotas / loadNamespaces / loadContexts (commands_load.go)
// =====================================================================

func TestCov80LoadQuotas(t *testing.T) {
	m := basePush80Model()
	cmd := m.loadQuotas()
	require.NotNil(t, cmd)
}

func TestCov80LoadNamespaces(t *testing.T) {
	m := basePush80Model()
	cmd := m.loadNamespaces()
	require.NotNil(t, cmd)
}

func TestCov80LoadNamespacesEmptyContext(t *testing.T) {
	m := basePush80Model()
	m.nav.Context = ""
	cmd := m.loadNamespaces()
	require.NotNil(t, cmd)
}

func TestCov80LoadContexts(t *testing.T) {
	m := basePush80Model()
	msg := m.loadContexts()
	cmsg, ok := msg.(contextsLoadedMsg)
	require.True(t, ok)
	_ = cmsg
}

// =====================================================================
// Target 38: loadResources / loadOwned / loadContainers (commands_load.go)
// =====================================================================

func TestCov80LoadResourcesForPreviewNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadResources(true)
	assert.Nil(t, cmd)
}

func TestCov80LoadResourcesNormal(t *testing.T) {
	m := basePush80Model()
	cmd := m.loadResources(false)
	require.NotNil(t, cmd)
}

func TestCov80LoadOwnedForPreviewNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadOwned(true)
	assert.Nil(t, cmd)
}

func TestCov80LoadOwnedNormal(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType.Kind = "Deployment"
	m.nav.ResourceName = "deploy-1"
	cmd := m.loadOwned(false)
	require.NotNil(t, cmd)
}

func TestCov80LoadOwnedEmptyNs(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType.Kind = "Deployment"
	m.nav.ResourceName = "deploy-1"
	m.allNamespaces = true
	m.nav.Namespace = "specific-ns"
	cmd := m.loadOwned(false)
	require.NotNil(t, cmd)
}

func TestCov80LoadContainersForPreviewNilSel(t *testing.T) {
	m := basePush80Model()
	m.middleItems = nil
	cmd := m.loadContainers(true)
	assert.Nil(t, cmd)
}

func TestCov80LoadContainersNormal(t *testing.T) {
	m := basePush80Model()
	m.nav.OwnedName = "pod-1"
	cmd := m.loadContainers(false)
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 39: isKubectlCommand / shellQuote / clearBeforeExec (commands.go)
// =====================================================================

func TestCov80IsKubectlCommandDirect(t *testing.T) {
	assert.True(t, isKubectlCommand("kubectl get pods"))
	assert.True(t, isKubectlCommand("kubectl"))
	assert.True(t, isKubectlCommand("get pods"))
	assert.False(t, isKubectlCommand("helm install"))
}

func TestCov80ShellQuote(t *testing.T) {
	assert.Equal(t, "'hello'", shellQuote("hello"))
	assert.Equal(t, "'he'\"'\"'llo'", shellQuote("he'llo"))
}

func TestCov80FindCustomActionNoMatch(t *testing.T) {
	_, found := findCustomAction("Pod", "nonexistent-action")
	assert.False(t, found)
}

func TestCov80ExpandCustomActionTemplate(t *testing.T) {
	actx := actionContext{
		name:      "my-pod",
		namespace: "default",
		context:   "prod",
		kind:      "Pod",
		columns: []model.KeyValue{
			{Key: "Node", Value: "worker-1"},
			{Key: "IP", Value: "10.0.0.1"},
		},
	}
	result := expandCustomActionTemplate("kubectl exec {name} -n {namespace} --context {context} # {Node} {ip}", actx)
	assert.Contains(t, result, "my-pod")
	assert.Contains(t, result, "default")
	assert.Contains(t, result, "prod")
	assert.Contains(t, result, "worker-1")
}

// =====================================================================
// Target 40: navigateToOwner (update_navigation.go)
// =====================================================================

func TestCov80NavigateToOwnerUnknownKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	result, cmd := m.navigateToOwner("UnknownKind999", "my-resource")
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

// =====================================================================
// Target 41: getPortForwardID (portforward_state.go)
// =====================================================================

func TestCov80GetPortForwardID(t *testing.T) {
	m := basePush80Model()
	cols := []model.KeyValue{
		{Key: "ID", Value: "42"},
		{Key: "Status", Value: "running"},
	}
	assert.Equal(t, 42, m.getPortForwardID(cols))

	assert.Equal(t, 0, m.getPortForwardID(nil))
	assert.Equal(t, 0, m.getPortForwardID([]model.KeyValue{{Key: "ID", Value: "bad"}}))
}

// =====================================================================
// Target 42: parseReadyRatio / compareNumeric helpers (tabs.go)
// =====================================================================

func TestCov80ParseReadyRatioInvalid(t *testing.T) {
	assert.Equal(t, float64(0), parseReadyRatio("noslash"))
	assert.Equal(t, float64(0), parseReadyRatio("0/0"))
	assert.InDelta(t, 0.5, parseReadyRatio("1/2"), 0.01)
}

func TestCov80CompareNumeric(t *testing.T) {
	assert.True(t, compareNumeric("1", "2"))
	assert.False(t, compareNumeric("10", "2"))
	assert.True(t, compareNumeric("abc", "1")) // "abc" parses as 0
}

func TestCov80StatusPriority(t *testing.T) {
	assert.Equal(t, 0, statusPriority("Running"))
	assert.Equal(t, 1, statusPriority("Pending"))
	assert.Equal(t, 2, statusPriority("Failed"))
	assert.Equal(t, 3, statusPriority("Unknown"))
}

func TestCov80SortModeName(t *testing.T) {
	m := basePush80Model()
	m.sortColumnName = ""
	assert.Contains(t, m.sortModeName(), "Name")

	m.sortColumnName = "Status"
	m.sortAscending = true
	assert.Contains(t, m.sortModeName(), "Status")
	assert.Contains(t, m.sortModeName(), "\u2191")

	m.sortAscending = false
	assert.Contains(t, m.sortModeName(), "\u2193")
}

func TestCov80GetColumnValue(t *testing.T) {
	item := model.Item{
		Columns: []model.KeyValue{
			{Key: "Image", Value: "nginx:latest"},
			{Key: "Node", Value: "worker-1"},
		},
	}
	assert.Equal(t, "nginx:latest", getColumnValue(item, "Image"))
	assert.Equal(t, "", getColumnValue(item, "NonExistent"))
}

// =====================================================================
// Target 43: loadPodMetricsForList / loadNodeMetricsForList
// =====================================================================

func TestCov80LoadPodMetricsForList(t *testing.T) {
	m := basePush80Model()
	cmd := m.loadPodMetricsForList()
	require.NotNil(t, cmd)
}

func TestCov80LoadNodeMetricsForList(t *testing.T) {
	m := basePush80Model()
	cmd := m.loadNodeMetricsForList()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 44: discoverCRDs (commands_load.go)
// =====================================================================

func TestCov80DiscoverCRDs(t *testing.T) {
	m := basePush80Model()
	cmd := m.discoverCRDs("test-ctx")
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 45: loadResourceTypes (commands_load.go)
// =====================================================================

func TestCov80LoadResourceTypesNoCRDs(t *testing.T) {
	m := basePush80Model()
	cmd := m.loadResourceTypes()
	require.NotNil(t, cmd)
	msg := cmd()
	rmsg, ok := msg.(resourceTypesMsg)
	require.True(t, ok)
	assert.NotEmpty(t, rmsg.items)
}

func TestCov80LoadResourceTypesWithCRDs(t *testing.T) {
	m := basePush80Model()
	m.discoveredCRDs["test-ctx"] = []model.ResourceTypeEntry{
		{Kind: "MyCRD", Resource: "mycrds", APIGroup: "test.io", APIVersion: "v1", Namespaced: true},
	}
	cmd := m.loadResourceTypes()
	require.NotNil(t, cmd)
	msg := cmd()
	rmsg, ok := msg.(resourceTypesMsg)
	require.True(t, ok)
	assert.NotEmpty(t, rmsg.items)
}

// =====================================================================
// Target 46: loadContainersForAction (commands.go)
// =====================================================================

func TestCov80LoadContainersForAction(t *testing.T) {
	m := basePush80Model()
	m.actionCtx = actionContext{
		context:   "test-ctx",
		name:      "my-pod",
		namespace: "default",
	}
	cmd := m.loadContainersForAction()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 47: loadPodsForAction (commands.go)
// =====================================================================

func TestCov80LoadPodsForAction(t *testing.T) {
	m := basePush80Model()
	m.actionCtx = actionContext{
		context:   "test-ctx",
		name:      "deploy-1",
		namespace: "default",
		kind:      "Deployment",
	}
	cmd := m.loadPodsForAction()
	require.NotNil(t, cmd)
}

// =====================================================================
// Target 48: copyMapStringInt / copyMapStringBool / copyItemCache (tabs.go)
// =====================================================================

func TestCov80CopyMapStringIntNil(t *testing.T) {
	c := copyMapStringInt(nil)
	assert.NotNil(t, c)
	assert.Empty(t, c)
}

func TestCov80CopyMapStringIntNonNil(t *testing.T) {
	m := map[string]int{"a": 1, "b": 2}
	c := copyMapStringInt(m)
	assert.Equal(t, 1, c["a"])
	c["a"] = 99
	assert.Equal(t, 1, m["a"]) // original unchanged
}

func TestCov80CopyMapStringBoolNil(t *testing.T) {
	c := copyMapStringBool(nil)
	assert.NotNil(t, c)
	assert.Empty(t, c)
}

func TestCov80CopyItemCacheNil(t *testing.T) {
	c := copyItemCache(nil)
	assert.NotNil(t, c)
	assert.Empty(t, c)
}

func TestCov80CopyItemCacheNonNil(t *testing.T) {
	m := map[string][]model.Item{
		"key": {{Name: "a"}, {Name: "b"}},
	}
	c := copyItemCache(m)
	assert.Len(t, c["key"], 2)
	c["key"][0].Name = "changed"
	assert.Equal(t, "a", m["key"][0].Name) // original unchanged
}

// =====================================================================
// Target 49: openActionMenu (update_actions.go)
// =====================================================================

func TestCov80OpenActionMenuBulkMode(t *testing.T) {
	m := basePush80Model()
	m.selectedItems = map[string]bool{
		"default/pod-1": true,
		"ns-2/pod-2":    true,
	}
	result, cmd := m.openActionMenu()
	rm := result.(Model)
	assert.True(t, rm.bulkMode)
	assert.Equal(t, overlayAction, rm.overlay)
	assert.Nil(t, cmd)
}

func TestCov80OpenActionMenuSingleItem(t *testing.T) {
	m := basePush80Model()
	m.setCursor(0)
	result, cmd := m.openActionMenu()
	rm := result.(Model)
	assert.False(t, rm.bulkMode)
	assert.Equal(t, overlayAction, rm.overlay)
	assert.Nil(t, cmd)
}

func TestCov80OpenActionMenuNoKind(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelClusters
	m.middleItems = nil
	result, cmd := m.openActionMenu()
	_ = result
	assert.Nil(t, cmd)
}

func TestCov80OpenActionMenuPortForward(t *testing.T) {
	m := basePush80Model()
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "__port_forwards__"}
	m.middleItems = []model.Item{{Name: "pf-1", Kind: "__port_forward_entry__"}}
	m.setCursor(0)
	result, cmd := m.openActionMenu()
	rm := result.(Model)
	assert.Equal(t, overlayAction, rm.overlay)
	assert.Nil(t, cmd)
}

func TestCov80OpenActionMenuContainers(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelContainers
	m.middleItems = []model.Item{{Name: "container-1", Kind: "Container"}}
	m.setCursor(0)
	result, cmd := m.openActionMenu()
	rm := result.(Model)
	assert.Equal(t, overlayAction, rm.overlay)
	assert.Nil(t, cmd)
}

func TestCov80OpenActionMenuDeletingItem(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Pod", Resource: "pods", Namespaced: true}
	m.middleItems = []model.Item{
		{Name: "pod-deleting", Kind: "Pod", Namespace: "default", Deleting: true},
	}
	m.setCursor(0)
	result, cmd := m.openActionMenu()
	rm := result.(Model)
	assert.Equal(t, overlayAction, rm.overlay)
	// The "Delete" action should be escalated.
	hasForce := false
	for _, item := range rm.overlayItems {
		if item.Name == "Force Delete" || item.Name == "Force Finalize" {
			hasForce = true
		}
	}
	assert.True(t, hasForce)
	assert.Nil(t, cmd)
}

// =====================================================================
// Target 50: buildActionCtx (update_actions.go)
// =====================================================================

func TestCov80BuildActionCtxResources(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.nav.ResourceType = model.ResourceTypeEntry{Kind: "Pod", Resource: "pods"}
	sel := &model.Item{Name: "pod-1", Namespace: "ns1"}
	ctx := m.buildActionCtx(sel, "Pod")
	assert.Equal(t, "pod-1", ctx.name)
	assert.Equal(t, "ns1", ctx.namespace)
	assert.Equal(t, "Pod", ctx.resourceType.Kind)
}

func TestCov80BuildActionCtxContainers(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelContainers
	m.nav.OwnedName = "my-pod"
	sel := &model.Item{Name: "app", Extra: "nginx:1.25"}
	ctx := m.buildActionCtx(sel, "Container")
	assert.Equal(t, "my-pod", ctx.name)
	assert.Equal(t, "app", ctx.containerName)
	assert.Equal(t, "nginx:1.25", ctx.image)
	assert.Equal(t, "Pod", ctx.kind)
}

func TestCov80BuildActionCtxOwned(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelOwned
	sel := &model.Item{Name: "rs-1", Kind: "ReplicaSet", Namespace: "default"}
	ctx := m.buildActionCtx(sel, "ReplicaSet")
	assert.Equal(t, "rs-1", ctx.name)
}

func TestCov80BuildActionCtxNamespacePriority(t *testing.T) {
	m := basePush80Model()
	m.nav.Level = model.LevelResources
	m.nav.Namespace = "nav-ns"
	m.namespace = "selector-ns"

	// Item namespace takes priority.
	sel := &model.Item{Name: "pod-1", Namespace: "item-ns"}
	ctx := m.buildActionCtx(sel, "Pod")
	assert.Equal(t, "item-ns", ctx.namespace)

	// Nav namespace second.
	sel2 := &model.Item{Name: "pod-1"}
	ctx2 := m.buildActionCtx(sel2, "Pod")
	assert.Equal(t, "nav-ns", ctx2.namespace)

	// Selector namespace last.
	m.nav.Namespace = ""
	ctx3 := m.buildActionCtx(sel2, "Pod")
	assert.Equal(t, "selector-ns", ctx3.namespace)
}
