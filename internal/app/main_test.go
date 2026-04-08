package app

import (
	"os"
	"path/filepath"
	"testing"
)

// TestMain installs a per-package temporary XDG_STATE_HOME so any test that
// reaches saveBookmarks, savePinned, saveSession, or any other state-file
// writer cannot accidentally overwrite the developer's real
// ~/.local/state/lfk/ files. Individual tests are still free to override
// XDG_STATE_HOME via t.Setenv when they need a specific path; t.Setenv
// restores the value automatically when the test finishes, so the package
// default is preserved between tests.
func TestMain(m *testing.M) {
	os.Exit(runTests(m))
}

// runTests is a helper so deferred cleanup actually runs (a defer in TestMain
// alongside os.Exit would never fire).
func runTests(m *testing.M) int {
	tmp, err := os.MkdirTemp("", "lfk-app-tests-")
	if err != nil {
		panic("test setup: cannot create temp XDG_STATE_HOME: " + err.Error())
	}
	defer func() { _ = os.RemoveAll(tmp) }()

	if err := os.Setenv("XDG_STATE_HOME", filepath.Join(tmp, "state")); err != nil {
		panic("test setup: cannot set XDG_STATE_HOME: " + err.Error())
	}

	return m.Run()
}
