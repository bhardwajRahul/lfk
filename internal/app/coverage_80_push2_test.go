package app

import (
	"context"
	"errors"
	"fmt"
	"sync"
	"testing"

	"github.com/charmbracelet/bubbles/spinner"
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

func basePush80v2Model() Model {
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
// Update -- message dispatch branches
// =====================================================================

func TestPush2UpdateWindowSizeMsg(t *testing.T) {
	m := basePush80v2Model()
	result, cmd := m.Update(tea.WindowSizeMsg{Width: 200, Height: 60})
	rm := result.(Model)
	assert.Equal(t, 200, rm.width)
	assert.Equal(t, 60, rm.height)
	assert.Nil(t, cmd)
}

func TestPush2UpdateWindowSizeSmall(t *testing.T) {
	m := basePush80v2Model()
	result, cmd := m.Update(tea.WindowSizeMsg{Width: 20, Height: 10})
	rm := result.(Model)
	assert.Equal(t, 20, rm.width)
	assert.Nil(t, cmd)
}

func TestPush2UpdateSpinnerTickMsg(t *testing.T) {
	m := basePush80v2Model()
	m.spinner = spinner.New()
	result, _ := m.Update(m.spinner.Tick())
	_ = result.(Model)
}

func TestPush2UpdateContextsLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.loading = true
	msg := contextsLoadedMsg{
		items: []model.Item{
			{Name: "ctx-1"},
			{Name: "ctx-2"},
		},
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.False(t, rm.loading)
	assert.Len(t, rm.middleItems, 2)
}

func TestPush2UpdateContextsLoadedMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := contextsLoadedMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.NotNil(t, rm.err)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateContextsLoadedMsgCanceled(t *testing.T) {
	m := basePush80v2Model()
	msg := contextsLoadedMsg{err: context.Canceled}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Nil(t, rm.err)
	assert.Nil(t, cmd)
}

func TestPush2UpdateResourceTypesMsg(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelResourceTypes
	msg := resourceTypesMsg{
		items: []model.Item{{Name: "Pods"}, {Name: "Services"}},
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Len(t, rm.middleItems, 2)
}

func TestPush2UpdateResourcesLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := resourcesLoadedMsg{
		items: []model.Item{{Name: "pod-new"}},
		gen:   5,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Len(t, rm.middleItems, 1)
}

func TestPush2UpdateResourcesLoadedMsgStalegen(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := resourcesLoadedMsg{
		items: []model.Item{{Name: "pod-new"}},
		gen:   3, // stale
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	// Items should not change for stale gen.
	assert.Len(t, rm.middleItems, 3)
}

func TestPush2UpdateResourcesLoadedMsgErr(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := resourcesLoadedMsg{
		err: fmt.Errorf("fail"),
		gen: 5,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
}

func TestPush2UpdateResourcesLoadedMsgCanceled(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := resourcesLoadedMsg{
		err: context.Canceled,
		gen: 5,
	}
	result, cmd := m.Update(msg)
	_ = result
	assert.Nil(t, cmd)
}

func TestPush2UpdateResourcesLoadedMsgForPreview(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := resourcesLoadedMsg{
		items:      []model.Item{{Name: "preview-item"}},
		gen:        5,
		forPreview: true,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Len(t, rm.rightItems, 1)
}

func TestPush2UpdateOwnedLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelOwned
	m.requestGen = 5
	msg := ownedLoadedMsg{
		items: []model.Item{{Name: "rs-1", Kind: "ReplicaSet"}},
		gen:   5,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Len(t, rm.middleItems, 1)
}

func TestPush2UpdateOwnedLoadedMsgForPreview(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := ownedLoadedMsg{
		items:      []model.Item{{Name: "owned-prev"}},
		gen:        5,
		forPreview: true,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Len(t, rm.rightItems, 1)
}

func TestPush2UpdateContainersLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelContainers
	m.requestGen = 5
	msg := containersLoadedMsg{
		items: []model.Item{{Name: "container-1", Kind: "Container"}},
		gen:   5,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Len(t, rm.middleItems, 1)
}

func TestPush2UpdateContainersLoadedMsgForPreview(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := containersLoadedMsg{
		items:      []model.Item{{Name: "container-1"}},
		gen:        5,
		forPreview: true,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Len(t, rm.rightItems, 1)
}

func TestPush2UpdateYamlLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeYAML
	msg := yamlLoadedMsg{content: "apiVersion: v1\nkind: Pod\n"}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.yamlContent, "apiVersion")
}

func TestPush2UpdateYamlLoadedMsgErr(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeYAML
	msg := yamlLoadedMsg{err: fmt.Errorf("fail")}
	result, _ := m.Update(msg)
	rm := result.(Model)
	// Exercises the error branch of yamlLoadedMsg handling.
	_ = rm
}

func TestPush2UpdateNamespacesLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.overlay = overlayNamespace
	msg := namespacesLoadedMsg{
		items: []model.Item{{Name: "default"}, {Name: "kube-system"}, {Name: "prod"}},
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotEmpty(t, rm.overlayItems)
}

func TestPush2UpdateNamespacesLoadedMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := namespacesLoadedMsg{err: fmt.Errorf("fail")}
	result, _ := m.Update(msg)
	rm := result.(Model)
	_ = rm
}

func TestPush2UpdateStatusMessageExpiredMsg(t *testing.T) {
	m := basePush80v2Model()
	m.setStatusMessage("temp msg", false)
	result, cmd := m.Update(statusMessageExpiredMsg{})
	rm := result.(Model)
	assert.Empty(t, rm.statusMessage)
	assert.Nil(t, cmd)
}

func TestPush2UpdateWatchTickMsg(t *testing.T) {
	m := basePush80v2Model()
	m.watchMode = true
	result, cmd := m.Update(watchTickMsg{})
	rm := result.(Model)
	_ = rm
	assert.NotNil(t, cmd)
}

func TestPush2UpdateWatchTickMsgNotActive(t *testing.T) {
	m := basePush80v2Model()
	m.watchMode = false
	result, cmd := m.Update(watchTickMsg{})
	_ = result
	assert.Nil(t, cmd)
}

func TestPush2UpdateCrdDiscoveryMsg(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelResourceTypes
	msg := crdDiscoveryMsg{
		context: "test-ctx",
		entries: []model.ResourceTypeEntry{
			{Kind: "MyCRD", Resource: "mycrds"},
		},
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotEmpty(t, rm.discoveredCRDs["test-ctx"])
}

func TestPush2UpdateCrdDiscoveryMsgDiffContext(t *testing.T) {
	m := basePush80v2Model()
	msg := crdDiscoveryMsg{
		context: "other-ctx",
		entries: []model.ResourceTypeEntry{{Kind: "X"}},
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotEmpty(t, rm.discoveredCRDs["other-ctx"])
}

func TestPush2UpdateActionResultMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := actionResultMsg{message: "Deleted pod-1"}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "Deleted")
	assert.NotNil(t, cmd)
}

func TestPush2UpdateActionResultMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := actionResultMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateBulkActionResultMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := bulkActionResultMsg{succeeded: 3, failed: 1, errors: []string{"pod-4: not found"}}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "3")
	assert.NotNil(t, cmd)
}

func TestPush2UpdateBulkActionResultMsgNoErrors(t *testing.T) {
	m := basePush80v2Model()
	msg := bulkActionResultMsg{succeeded: 5}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "5")
	assert.NotNil(t, cmd)
}

func TestPush2UpdateDashboardLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := dashboardLoadedMsg{content: "dashboard content", events: "events", context: "test-ctx"}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Equal(t, "dashboard content", rm.dashboardPreview)
	assert.Equal(t, "events", rm.dashboardEventsPreview)
}

func TestPush2UpdateDashboardLoadedMsgWrongContext(t *testing.T) {
	m := basePush80v2Model()
	msg := dashboardLoadedMsg{content: "data", context: "other-ctx"}
	result, _ := m.Update(msg)
	rm := result.(Model)
	// Should be ignored since context doesn't match.
	assert.Empty(t, rm.dashboardPreview)
}

func TestPush2UpdateMonitoringDashboardMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := monitoringDashboardMsg{content: "monitoring data", context: "test-ctx"}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Equal(t, "monitoring data", rm.monitoringPreview)
}

func TestPush2UpdateMonitoringDashboardMsgWrongContext(t *testing.T) {
	m := basePush80v2Model()
	msg := monitoringDashboardMsg{content: "data", context: "other"}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Empty(t, rm.monitoringPreview)
}

func TestPush2UpdateStartupTipMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := startupTipMsg{tip: "Press ? for help"}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "Press ? for help")
	assert.NotNil(t, cmd)
}

func TestPush2UpdateExportDoneMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := exportDoneMsg{path: "/tmp/test.yaml"}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "/tmp/test.yaml")
	assert.NotNil(t, cmd)
}

func TestPush2UpdateExportDoneMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := exportDoneMsg{err: fmt.Errorf("write failed")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateYamlClipboardMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := yamlClipboardMsg{content: "apiVersion: v1"}
	result, cmd := m.Update(msg)
	_ = result.(Model)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateYamlClipboardMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := yamlClipboardMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateMetricsLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := metricsLoadedMsg{cpuUsed: 100, memUsed: 512, gen: 5}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotEmpty(t, rm.metricsContent)
}

func TestPush2UpdateMetricsLoadedMsgStaleGen(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := metricsLoadedMsg{cpuUsed: 100, gen: 3}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Empty(t, rm.metricsContent)
}

func TestPush2UpdateDescribeRefreshTickMsg(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeDescribe
	result, _ := m.Update(describeRefreshTickMsg{})
	_ = result
}

func TestPush2UpdateDescribeRefreshTickMsgNotDescribe(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeExplorer
	result, cmd := m.Update(describeRefreshTickMsg{})
	_ = result
	assert.Nil(t, cmd)
}

func TestPush2UpdateSecretDataLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.overlay = overlaySecretEditor
	msg := secretDataLoadedMsg{data: &model.SecretData{Data: map[string]string{"key": "val"}}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotNil(t, rm.secretData)
}

func TestPush2UpdateSecretSavedMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := secretSavedMsg{}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "Secret")
	assert.NotNil(t, cmd)
}

func TestPush2UpdateSecretSavedMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := secretSavedMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateLabelDataLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.overlay = overlayLabelEditor
	msg := labelDataLoadedMsg{data: &model.LabelAnnotationData{
		Labels:      map[string]string{"a": "b"},
		Annotations: map[string]string{"c": "d"},
	}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotNil(t, rm.labelData)
}

func TestPush2UpdateLabelSavedMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := labelSavedMsg{}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "saved")
	assert.NotNil(t, cmd)
}

func TestPush2UpdateLabelSavedMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := labelSavedMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateConfigMapDataLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := configMapDataLoadedMsg{data: &model.ConfigMapData{Data: map[string]string{"key": "val"}}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotNil(t, rm.configMapData)
}

func TestPush2UpdateConfigMapSavedMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := configMapSavedMsg{}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.Contains(t, rm.statusMessage, "ConfigMap")
	assert.NotNil(t, cmd)
}

func TestPush2UpdateRevisionListMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := revisionListMsg{revisions: []k8s.DeploymentRevision{
		{Revision: 1, Name: "deploy-1-abc"},
	}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotEqual(t, overlayNone, rm.overlay)
}

func TestPush2UpdateRevisionListMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := revisionListMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateHelmRevisionListMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := helmRevisionListMsg{revisions: []ui.HelmRevision{
		{Revision: 1, Status: "deployed"},
	}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotEqual(t, overlayNone, rm.overlay)
}

func TestPush2UpdateHelmRevisionListMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := helmRevisionListMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateResourceTreeLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	m.requestGen = 5
	msg := resourceTreeLoadedMsg{
		tree: &model.ResourceNode{Name: "deploy-1", Kind: "Deployment"},
		gen:  5,
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotNil(t, rm.resourceTree)
}

func TestPush2UpdateQuotaLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := quotaLoadedMsg{quotas: []k8s.QuotaInfo{
		{Name: "q1", Namespace: "default", Resources: []k8s.QuotaResource{
			{Name: "pods", Hard: "10", Used: "5", Percent: 50},
		}},
	}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.NotEqual(t, overlayNone, rm.overlay)
}

func TestPush2UpdateQuotaLoadedMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := quotaLoadedMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateCommandBarResultMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := commandBarResultMsg{output: "output here"}
	result, _ := m.Update(msg)
	_ = result.(Model)
	// Exercises the success path of commandBarResultMsg.
}

func TestPush2UpdateCommandBarResultMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := commandBarResultMsg{output: "error output", err: fmt.Errorf("fail")}
	result, _ := m.Update(msg)
	_ = result.(Model)
	// Exercises the error path of commandBarResultMsg handling.
}

func TestPush2UpdatePodSelectMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := podSelectMsg{items: []model.Item{
		{Name: "pod-a"},
		{Name: "pod-b"},
	}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	_ = rm // exercises the branch
}

func TestPush2UpdatePodSelectMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := podSelectMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateContainerSelectMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := containerSelectMsg{items: []model.Item{
		{Name: "container-a"},
	}}
	result, _ := m.Update(msg)
	rm := result.(Model)
	_ = rm
}

func TestPush2UpdateContainerSelectMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := containerSelectMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

func TestPush2UpdateDiffLoadedMsg(t *testing.T) {
	m := basePush80v2Model()
	msg := diffLoadedMsg{
		left:      "left content",
		right:     "right content",
		leftName:  "pod-1",
		rightName: "pod-2",
	}
	result, _ := m.Update(msg)
	rm := result.(Model)
	assert.Equal(t, modeDiff, rm.mode)
}

func TestPush2UpdateDiffLoadedMsgErr(t *testing.T) {
	m := basePush80v2Model()
	msg := diffLoadedMsg{err: fmt.Errorf("fail")}
	result, cmd := m.Update(msg)
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
	assert.NotNil(t, cmd)
}

// =====================================================================
// isContextCanceled
// =====================================================================

func TestPush2IsContextCanceled(t *testing.T) {
	assert.False(t, isContextCanceled(nil))
	assert.True(t, isContextCanceled(context.Canceled))
	assert.True(t, isContextCanceled(context.DeadlineExceeded))
	assert.True(t, isContextCanceled(fmt.Errorf("context canceled")))
	assert.True(t, isContextCanceled(fmt.Errorf("context deadline exceeded")))
	assert.False(t, isContextCanceled(errors.New("random error")))
}

// =====================================================================
// handleKey more branches
// =====================================================================

func TestPush2HandleKeySearchActive(t *testing.T) {
	m := basePush80v2Model()
	m.searchActive = true
	result, _ := m.handleKey(keyMsg("a"))
	_ = result.(Model)
}

func TestPush2HandleKeyFilterActive(t *testing.T) {
	m := basePush80v2Model()
	m.filterActive = true
	result, _ := m.handleKey(keyMsg("a"))
	_ = result.(Model)
}

func TestPush2HandleKeyCommandBarActive(t *testing.T) {
	m := basePush80v2Model()
	m.commandBarActive = true
	result, _ := m.handleKey(keyMsg("a"))
	_ = result.(Model)
}

func TestPush2HandleKeyOverlayActive(t *testing.T) {
	m := basePush80v2Model()
	m.overlay = overlayAction
	m.overlayItems = []model.Item{{Name: "action1"}}
	result, _ := m.handleKey(keyMsg("esc"))
	rm := result.(Model)
	assert.Equal(t, overlayNone, rm.overlay)
}

func TestPush2HandleKeyHelpMode(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeHelp
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.NotEqual(t, modeHelp, rm.mode)
}

func TestPush2HandleKeyYAMLMode(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeYAML
	m.yamlContent = "apiVersion: v1"
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestPush2HandleKeyExplainMode(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeExplain
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestPush2HandleKeyDescribeMode(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeDescribe
	m.describeContent = "some content"
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestPush2HandleKeyDiffMode(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeDiff
	m.diffLeft = "left"
	m.diffRight = "right"
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestPush2HandleKeyLogsMode(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeLogs
	result, _ := m.handleKey(keyMsg("q"))
	rm := result.(Model)
	assert.Equal(t, modeExplorer, rm.mode)
}

func TestPush2HandleKeyExplorerModeF(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeExplorer
	result, _ := m.handleKey(keyMsg("F"))
	rm := result.(Model)
	assert.True(t, rm.fullscreenMiddle)
}

func TestPush2HandleKeyExplorerModeW(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeExplorer
	kb := ui.ActiveKeybindings
	result, cmd := m.handleKey(keyMsg(kb.WatchMode))
	rm := result.(Model)
	assert.True(t, rm.watchMode)
	assert.NotNil(t, cmd)
}

func TestPush2HandleKeyExplorerModeEnter(t *testing.T) {
	m := basePush80v2Model()
	m.mode = modeExplorer
	m.setCursor(0)
	result, cmd := m.handleKey(keyMsg("enter"))
	rm := result.(Model)
	// Enter navigates -- either to containers or YAML view.
	_ = rm
	_ = cmd
}

func TestPush2HandleKeyExplorerModeSlash(t *testing.T) {
	m := basePush80v2Model()
	result, _ := m.handleKey(keyMsg("/"))
	rm := result.(Model)
	assert.True(t, rm.searchActive)
}

func TestPush2HandleKeyExplorerModeColon(t *testing.T) {
	m := basePush80v2Model()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	m.commandHistory = loadCommandHistory()
	result, _ := m.handleKey(keyMsg(":"))
	rm := result.(Model)
	assert.True(t, rm.commandBarActive)
}

func TestPush2HandleKeyExplorerModeQuestion(t *testing.T) {
	m := basePush80v2Model()
	result, _ := m.handleKey(keyMsg("?"))
	rm := result.(Model)
	assert.Equal(t, modeHelp, rm.mode)
}

// =====================================================================
// handleExplorerActionKey more branches
// =====================================================================

func TestPush2HandleExplorerActionKeyBackspace(t *testing.T) {
	m := basePush80v2Model()
	result, _, handled := m.handleExplorerActionKey(keyMsg("backspace"))
	if handled {
		_ = result.(Model)
	}
}

func TestPush2HandleExplorerActionKeyM(t *testing.T) {
	m := basePush80v2Model()
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	result, _, handled := m.handleExplorerActionKey(keyMsg("m"))
	// 'm' is handled by handleKey, not handleExplorerActionKey.
	// It may not be handled here.
	_ = result
	_ = handled
}

func TestPush2HandleExplorerActionKeyP(t *testing.T) {
	// 'p' pins a CRD group at LevelResourceTypes.
	m := basePush80v2Model()
	m.nav.Level = model.LevelResourceTypes
	m.middleItems = []model.Item{{Name: "Pods", Category: "Core"}}
	m.setCursor(0)
	t.Setenv("XDG_STATE_HOME", t.TempDir())
	result, _, handled := m.handleExplorerActionKey(keyMsg("p"))
	if handled {
		_ = result.(Model)
	}
}

func TestPush2HandleExplorerActionKeyEqualSign(t *testing.T) {
	m := basePush80v2Model()
	ui.ActiveSortableColumns = []string{"Name", "Status"}
	m.sortColumnName = "Name"
	m.sortAscending = true
	result, cmd, handled := m.handleExplorerActionKey(keyMsg("="))
	assert.True(t, handled)
	rm := result.(Model)
	assert.False(t, rm.sortAscending)
	assert.NotNil(t, cmd)
}

func TestPush2HandleExplorerActionKeyDash(t *testing.T) {
	m := basePush80v2Model()
	ui.ActiveSortableColumns = []string{"Name", "Status"}
	m.sortColumnName = "Status"
	result, cmd, handled := m.handleExplorerActionKey(keyMsg("-"))
	assert.True(t, handled)
	rm := result.(Model)
	// '-' resets sort -- sortColumnName becomes "Name" (default) or cleared.
	_ = rm
	assert.NotNil(t, cmd)
}

// =====================================================================
// executeBulkAction more branches
// =====================================================================

func TestPush2ExecuteBulkActionScale(t *testing.T) {
	m := basePush80v2Model()
	m.bulkItems = []model.Item{{Name: "deploy-1"}}
	m.actionCtx = actionContext{kind: "Deployment"}
	result, _ := m.executeBulkAction("Scale")
	rm := result.(Model)
	assert.Equal(t, overlayScaleInput, rm.overlay)
}

func TestPush2ExecuteBulkActionRestart(t *testing.T) {
	m := basePush80v2Model()
	m.bulkItems = []model.Item{{Name: "deploy-1"}}
	m.actionCtx = actionContext{context: "ctx", kind: "Deployment"}
	result, cmd := m.executeBulkAction("Restart")
	rm := result.(Model)
	// Restart triggers a confirmation overlay, not a direct status message.
	_ = rm
	_ = cmd
}

func TestPush2ExecuteBulkActionLabelsAnnotations(t *testing.T) {
	m := basePush80v2Model()
	m.bulkItems = []model.Item{{Name: "pod-1"}}
	m.batchLabelInput = TextInput{}
	result, _ := m.executeBulkAction("Labels / Annotations")
	rm := result.(Model)
	assert.Equal(t, overlayBatchLabel, rm.overlay)
}

// =====================================================================
// refreshCurrentLevel
// =====================================================================

func TestPush2RefreshCurrentLevelResources(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelResources
	cmd := m.refreshCurrentLevel()
	require.NotNil(t, cmd)
}

func TestPush2RefreshCurrentLevelOwned(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelOwned
	m.nav.ResourceType.Kind = "Deployment"
	m.nav.ResourceName = "deploy-1"
	cmd := m.refreshCurrentLevel()
	require.NotNil(t, cmd)
}

func TestPush2RefreshCurrentLevelContainers(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelContainers
	m.nav.OwnedName = "pod-1"
	cmd := m.refreshCurrentLevel()
	require.NotNil(t, cmd)
}

func TestPush2RefreshCurrentLevelClusters(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelClusters
	cmd := m.refreshCurrentLevel()
	require.NotNil(t, cmd)
}

func TestPush2RefreshCurrentLevelResourceTypes(t *testing.T) {
	m := basePush80v2Model()
	m.nav.Level = model.LevelResourceTypes
	cmd := m.refreshCurrentLevel()
	require.NotNil(t, cmd)
}

// =====================================================================
// closeTabOrQuit
// =====================================================================

func TestPush2CloseTabOrQuitSingleTab(t *testing.T) {
	m := basePush80v2Model()
	m.portForwardMgr = k8s.NewPortForwardManager()
	m.tabs = []TabState{{}}
	result, _ := m.closeTabOrQuit()
	rm := result.(Model)
	// Single tab: should show quit confirm or quit directly.
	_ = rm
}

func TestPush2CloseTabOrQuitMultipleTabs(t *testing.T) {
	m := basePush80v2Model()
	m.tabs = []TabState{{}, {}}
	m.activeTab = 0
	result, _ := m.closeTabOrQuit()
	rm := result.(Model)
	assert.Len(t, rm.tabs, 1)
}

// =====================================================================
// directActionScale / directActionForceDelete
// =====================================================================

func TestPush2DirectActionScaleNoSelection(t *testing.T) {
	m := basePush80v2Model()
	m.nav.ResourceType.Kind = "Deployment"
	m.middleItems = []model.Item{{Name: "deploy-1", Kind: "Deployment", Namespace: "default"}}
	m.setCursor(0)
	result, cmd := m.directActionScale()
	rm := result.(Model)
	assert.Equal(t, overlayScaleInput, rm.overlay)
	assert.Nil(t, cmd)
}

func TestPush2DirectActionScaleNonScaleable(t *testing.T) {
	m := basePush80v2Model()
	m.nav.ResourceType.Kind = "ConfigMap"
	m.middleItems = []model.Item{{Name: "cm-1", Kind: "ConfigMap"}}
	m.setCursor(0)
	result, _ := m.directActionScale()
	rm := result.(Model)
	assert.True(t, rm.statusMessageErr)
}

func TestPush2DirectActionForceDeleteNoSel(t *testing.T) {
	m := basePush80v2Model()
	m.middleItems = nil
	result, cmd := m.directActionForceDelete()
	_ = result
	assert.Nil(t, cmd)
}

func TestPush2DirectActionForceDeleteWithSel(t *testing.T) {
	m := basePush80v2Model()
	m.nav.ResourceType.Kind = "Pod"
	m.middleItems = []model.Item{{Name: "pod-1", Kind: "Pod", Namespace: "default"}}
	m.setCursor(0)
	result, cmd := m.directActionForceDelete()
	rm := result.(Model)
	assert.Equal(t, overlayConfirmType, rm.overlay)
	assert.Nil(t, cmd)
}
