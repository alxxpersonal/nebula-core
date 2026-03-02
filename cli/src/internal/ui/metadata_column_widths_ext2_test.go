package ui

import "testing"

import "github.com/stretchr/testify/assert"

func TestMetadataColumnWidthsOverflowReductionBranch(t *testing.T) {
	group, field, value := metadataColumnWidths(40)
	assert.Equal(t, 10, group)
	assert.Equal(t, 14, field)
	assert.Equal(t, 14, value)
}
