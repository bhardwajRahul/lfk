package app

import (
	"testing"

	"github.com/stretchr/testify/assert"

	"github.com/janosmiko/lfk/internal/model"
)

// --- builtinFilterPresets kind-specific branches ---

func TestBuiltinFilterPresetsStatefulSet(t *testing.T) {
	presets := builtinFilterPresets("StatefulSet")
	names := presetNames(presets)
	assert.True(t, names["Not Ready"])
	assert.True(t, names["Failing"])
}

func TestBuiltinFilterPresetsDaemonSet(t *testing.T) {
	presets := builtinFilterPresets("DaemonSet")
	names := presetNames(presets)
	assert.True(t, names["Not Ready"])
	assert.True(t, names["Failing"])
}

func TestBuiltinFilterPresetsJob(t *testing.T) {
	presets := builtinFilterPresets("Job")
	names := presetNames(presets)
	assert.True(t, names["Failed"])

	failed := findPreset(presets, "Failed")
	assert.NotNil(t, failed)

	assert.True(t, failed.MatchFn(model.Item{Status: "Failed"}))
	assert.True(t, failed.MatchFn(model.Item{Status: "BackoffLimitExceeded"}))
	assert.False(t, failed.MatchFn(model.Item{Status: "Complete"}))
}

func TestBuiltinFilterPresetsCronJob(t *testing.T) {
	presets := builtinFilterPresets("CronJob")
	names := presetNames(presets)
	assert.True(t, names["Suspended"])

	suspended := findPreset(presets, "Suspended")
	assert.NotNil(t, suspended)

	assert.True(t, suspended.MatchFn(model.Item{
		Columns: []model.KeyValue{{Key: "Suspend", Value: "True"}},
	}))
	assert.False(t, suspended.MatchFn(model.Item{
		Columns: []model.KeyValue{{Key: "Suspend", Value: "False"}},
	}))
}

func TestBuiltinFilterPresetsApplication(t *testing.T) {
	presets := builtinFilterPresets("Application")
	names := presetNames(presets)
	assert.True(t, names["Out of Sync"])
	assert.True(t, names["Degraded"])

	outofsync := findPreset(presets, "Out of Sync")
	assert.NotNil(t, outofsync)
	assert.True(t, outofsync.MatchFn(model.Item{Status: "OutOfSync"}))
	assert.False(t, outofsync.MatchFn(model.Item{Status: "Synced"}))

	degraded := findPreset(presets, "Degraded")
	assert.NotNil(t, degraded)
	assert.True(t, degraded.MatchFn(model.Item{Status: "Degraded"}))
	assert.True(t, degraded.MatchFn(model.Item{Status: "Missing"}))
	assert.False(t, degraded.MatchFn(model.Item{Status: "Healthy"}))
}

func TestBuiltinFilterPresetsHelmRelease(t *testing.T) {
	presets := builtinFilterPresets("HelmRelease")
	names := presetNames(presets)
	assert.True(t, names["Suspended"])
	assert.True(t, names["Not Ready"])

	susp := findPreset(presets, "Suspended")
	assert.NotNil(t, susp)
	assert.True(t, susp.MatchFn(model.Item{Status: "Suspended"}))
	assert.False(t, susp.MatchFn(model.Item{Status: "Ready"}))

	notReady := findPreset(presets, "Not Ready")
	assert.NotNil(t, notReady)
	assert.True(t, notReady.MatchFn(model.Item{Status: "Failed"}))
	assert.False(t, notReady.MatchFn(model.Item{Status: "Ready"}))
	assert.False(t, notReady.MatchFn(model.Item{Status: "Applied"}))
	assert.False(t, notReady.MatchFn(model.Item{Status: "Suspended"}))
}

func TestBuiltinFilterPresetsKustomization(t *testing.T) {
	presets := builtinFilterPresets("Kustomization")
	names := presetNames(presets)
	assert.True(t, names["Suspended"])
	assert.True(t, names["Not Ready"])
}

func TestBuiltinFilterPresetsPVC(t *testing.T) {
	presets := builtinFilterPresets("PersistentVolumeClaim")
	names := presetNames(presets)
	assert.True(t, names["Pending"])
	assert.True(t, names["Lost"])

	pending := findPreset(presets, "Pending")
	assert.NotNil(t, pending)
	assert.True(t, pending.MatchFn(model.Item{Status: "Pending"}))
	assert.False(t, pending.MatchFn(model.Item{Status: "Bound"}))

	lost := findPreset(presets, "Lost")
	assert.NotNil(t, lost)
	assert.True(t, lost.MatchFn(model.Item{Status: "Lost"}))
	assert.False(t, lost.MatchFn(model.Item{Status: "Bound"}))
}

func TestBuiltinFilterPresetsCertificate(t *testing.T) {
	presets := builtinFilterPresets("Certificate")
	names := presetNames(presets)
	assert.True(t, names["Not Ready"])
	assert.True(t, names["Expiring Soon"])
}

func TestBuiltinFilterPresetsCertificateRequest(t *testing.T) {
	presets := builtinFilterPresets("CertificateRequest")
	names := presetNames(presets)
	assert.True(t, names["Not Ready"])
	assert.True(t, names["Expiring Soon"])
}

// --- Pod Pending preset ---

func TestPodPendingPreset(t *testing.T) {
	presets := builtinFilterPresets("Pod")
	pending := findPreset(presets, "Pending")
	assert.NotNil(t, pending)

	tests := []struct {
		status string
		want   bool
	}{
		{"Pending", true},
		{"ContainerCreating", true},
		{"PodInitializing", true},
		{"Init:0/1", true},
		{"Terminating", true},
		{"Unknown", true},
		{"Running", false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, pending.MatchFn(model.Item{Status: tt.status}), "Pending(%q)", tt.status)
	}
}

// --- Deployment Failing preset ---

func TestDeploymentFailingPreset(t *testing.T) {
	presets := builtinFilterPresets("Deployment")
	failing := findPreset(presets, "Failing")
	assert.NotNil(t, failing)

	tests := []struct {
		name string
		item model.Item
		want bool
	}{
		{"error status", model.Item{Status: "Error"}, true},
		{"degraded status", model.Item{Status: "Degraded"}, true},
		{"unavailable column", model.Item{
			Status:  "Progressing",
			Columns: []model.KeyValue{{Key: "Unavailable", Value: "2"}},
		}, true},
		{"ready mismatch", model.Item{Status: "Available", Ready: "1/3"}, true},
		{"healthy", model.Item{Status: "Available", Ready: "3/3"}, false},
	}
	for _, tt := range tests {
		assert.Equal(t, tt.want, failing.MatchFn(tt.item), "Failing(%s)", tt.name)
	}
}

// --- matchRestartsGt ---

func TestMatchRestartsGtZero(t *testing.T) {
	fn := matchRestartsGt(0)
	assert.True(t, fn(model.Item{Restarts: "1"}))
	assert.False(t, fn(model.Item{Restarts: "0"}))
	assert.False(t, fn(model.Item{Restarts: ""}))
}

// --- Universal Recent preset ---

func TestUniversalRecentPreset(t *testing.T) {
	presets := builtinFilterPresets("ConfigMap")
	recent := findPreset(presets, "Recent (<1h)")
	assert.NotNil(t, recent)

	assert.False(t, recent.MatchFn(model.Item{}))
}
