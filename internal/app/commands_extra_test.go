package app

import (
	"os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- clearBeforeExec ---

func TestClearBeforeExec(t *testing.T) {
	t.Run("wraps simple command", func(t *testing.T) {
		original := exec.Command("kubectl", "get", "pods")
		wrapped := clearBeforeExec(original)

		assert.Equal(t, "sh", wrapped.Path[len(wrapped.Path)-2:])
		assert.Equal(t, "sh", wrapped.Args[0])
		assert.Equal(t, "-c", wrapped.Args[1])
		assert.Contains(t, wrapped.Args[2], "printf")
		assert.Contains(t, wrapped.Args[2], "'kubectl'")
		assert.Contains(t, wrapped.Args[2], "'get'")
		assert.Contains(t, wrapped.Args[2], "'pods'")
	})

	t.Run("preserves environment", func(t *testing.T) {
		original := exec.Command("kubectl", "get", "pods")
		original.Env = []string{"KUBECONFIG=/tmp/config"}
		original.Dir = "/some/dir"
		wrapped := clearBeforeExec(original)

		assert.Equal(t, []string{"KUBECONFIG=/tmp/config"}, wrapped.Env)
		assert.Equal(t, "/some/dir", wrapped.Dir)
	})

	t.Run("quotes args with special chars", func(t *testing.T) {
		original := exec.Command("kubectl", "get", "pods", "-l", "app=my app")
		wrapped := clearBeforeExec(original)

		assert.Contains(t, wrapped.Args[2], "'app=my app'")
	})

	t.Run("handles args with single quotes", func(t *testing.T) {
		original := exec.Command("echo", "it's")
		wrapped := clearBeforeExec(original)

		// shellQuote replaces ' with '"'"'
		assert.Contains(t, wrapped.Args[2], `'it'"'"'s'`)
	})
}

// --- SetVersion ---

func TestSetVersion(t *testing.T) {
	m := Model{}
	m.SetVersion("1.2.3")
	assert.Equal(t, "1.2.3", m.version)
}

func TestSetVersionOverwrite(t *testing.T) {
	m := Model{}
	m.SetVersion("1.0.0")
	m.SetVersion("2.0.0")
	assert.Equal(t, "2.0.0", m.version)
}

func TestSetVersionEmpty(t *testing.T) {
	m := Model{}
	m.SetVersion("")
	assert.Equal(t, "", m.version)
}

// --- SetStderrChan ---

func TestSetStderrChan(t *testing.T) {
	m := Model{}
	ch := make(chan string, 1)
	m.SetStderrChan(ch)
	assert.NotNil(t, m.stderrChan)
}

func TestSetStderrChanNil(t *testing.T) {
	m := Model{}
	m.SetStderrChan(nil)
	assert.Nil(t, m.stderrChan)
}

// --- scheduleStatusClear ---

func TestScheduleStatusClear(t *testing.T) {
	cmd := scheduleStatusClear()
	assert.NotNil(t, cmd)
}

// --- scheduleStartupTip ---

func TestScheduleStartupTip(t *testing.T) {
	cmd := scheduleStartupTip()
	assert.NotNil(t, cmd)
}

// --- scheduleWatchTick ---

func TestScheduleWatchTick(t *testing.T) {
	cmd := scheduleWatchTick(5)
	assert.NotNil(t, cmd)
}

// --- startupTips ---

func TestStartupTipsHasEntries(t *testing.T) {
	assert.True(t, len(startupTips) > 0)
}
