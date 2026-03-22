package k8s

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

// --- reorderYAMLFields: additional edge cases ---

func TestReorderYAMLFields_Preamble(t *testing.T) {
	t.Run("comment preamble before any section", func(t *testing.T) {
		input := `# This is a comment
apiVersion: v1
kind: Pod`
		result := reorderYAMLFields(input)
		// Comment should be preserved (as a preamble section).
		assert.Contains(t, result, "# This is a comment")
		assert.Contains(t, result, "apiVersion: v1")
	})

	t.Run("stringData field has priority 6", func(t *testing.T) {
		input := `status:
  phase: Active
stringData:
  key: value
kind: Secret
apiVersion: v1`
		result := reorderYAMLFields(input)
		// apiVersion(0) < kind(1) < stringData(6) < status(7)
		apiIdx := indexOf(result, "apiVersion:")
		kindIdx := indexOf(result, "kind:")
		stringDataIdx := indexOf(result, "stringData:")
		statusIdx := indexOf(result, "status:")
		assert.Less(t, apiIdx, kindIdx)
		assert.Less(t, kindIdx, stringDataIdx)
		assert.Less(t, stringDataIdx, statusIdx)
	})

	t.Run("type field has priority 3", func(t *testing.T) {
		input := `data:
  key: value
type: Opaque
metadata:
  name: my-secret
apiVersion: v1
kind: Secret`
		result := reorderYAMLFields(input)
		// apiVersion(0) < kind(1) < metadata(2) < type(3) < data(5)
		metadataIdx := indexOf(result, "metadata:")
		typeIdx := indexOf(result, "type:")
		// Use "\ndata:" to avoid matching "data:" inside "metadata:"
		dataIdx := indexOf(result, "\ndata:")
		assert.Less(t, metadataIdx, typeIdx)
		assert.Less(t, typeIdx, dataIdx)
	})

	t.Run("single line input", func(t *testing.T) {
		input := "apiVersion: v1"
		result := reorderYAMLFields(input)
		assert.Equal(t, input, result)
	})

	t.Run("only comments", func(t *testing.T) {
		input := "# just a comment"
		result := reorderYAMLFields(input)
		assert.Equal(t, input, result)
	})

	t.Run("indented lines without a section go to preamble", func(t *testing.T) {
		input := "  indented-line"
		result := reorderYAMLFields(input)
		assert.Equal(t, input, result)
	})
}
