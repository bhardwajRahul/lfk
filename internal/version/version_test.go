package version

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestFull_DefaultValues(t *testing.T) {
	got := Full()
	expected := "lfk dev (commit: unknown, built: unknown)"

	assert.Equal(t, expected, got)
}

func TestFull_CustomValues(t *testing.T) {
	origVersion := Version
	origCommit := GitCommit
	origDate := BuildDate

	t.Cleanup(func() {
		Version = origVersion
		GitCommit = origCommit
		BuildDate = origDate
	})

	Version = "v1.2.3"
	GitCommit = "abc1234"
	BuildDate = "2025-01-15T10:30:00Z"

	got := Full()
	expected := "lfk v1.2.3 (commit: abc1234, built: 2025-01-15T10:30:00Z)"

	assert.Equal(t, expected, got)
}

func TestShort_DefaultValue(t *testing.T) {
	got := Short()

	assert.Equal(t, "dev", got)
}

func TestShort_CustomValue(t *testing.T) {
	origVersion := Version

	t.Cleanup(func() {
		Version = origVersion
	})

	Version = "v2.0.0"

	got := Short()

	assert.Equal(t, "v2.0.0", got)
}
