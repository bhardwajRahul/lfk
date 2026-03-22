package app

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSectionAtScrollPosEmpty(t *testing.T) {
	key := sectionAtScrollPos(0, nil, nil)
	assert.Empty(t, key)
}

func TestSectionAtScrollPosWithMapping(t *testing.T) {
	// Simple mapping: visible lines 0,1,2 map to original lines 0,1,2.
	mapping := []int{0, 1, 2, 3, 4}
	sections := []yamlSection{
		{key: "metadata", startLine: 0, endLine: 2, indent: 0},
		{key: "spec", startLine: 3, endLine: 4, indent: 0},
	}

	assert.Equal(t, "metadata", sectionAtScrollPos(0, mapping, sections))
	assert.Equal(t, "metadata", sectionAtScrollPos(1, mapping, sections))
	assert.Equal(t, "spec", sectionAtScrollPos(3, mapping, sections))
}

func TestSectionAtScrollPosOutOfRange(t *testing.T) {
	mapping := []int{0, 1}
	sections := []yamlSection{
		{key: "metadata", startLine: 0, endLine: 1, indent: 0},
	}
	// scrollPos beyond the mapping
	key := sectionAtScrollPos(10, mapping, sections)
	assert.Empty(t, key)
}
