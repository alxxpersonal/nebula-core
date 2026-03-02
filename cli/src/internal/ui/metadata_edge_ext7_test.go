package ui

import (
	"strings"
	"testing"

	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type blankStringer struct{}

func (blankStringer) String() string { return "" }

func TestParseMetadataInputPipeRowValueParseError(t *testing.T) {
	_, err := parseMetadataInput("profile | settings | {}")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "inline objects not supported")
}

func TestMetadataPreviewBranchMatrixAdditional(t *testing.T) {
	assert.Equal(t, "None", formatMetadataValue(blankStringer{}))

	assert.Equal(t, "", metadataValuePreview("alpha", 0))
	assert.Equal(t, "alpha", metadataValuePreview([]any{"   ", "alpha"}, 20))
	assert.Equal(t, "note", metadataValuePreview(map[string]any{"text": " note "}, 20))
	assert.Equal(t, "", metadataValuePreview(map[string]any{"text": "   "}, 20))
}

func TestRenderMetadataInputAndEditorPreviewAdditionalBranches(t *testing.T) {
	out := components.SanitizeText(renderMetadataInput("owner: alxx\n\nstatus: active"))
	assert.Contains(t, out, "owner")
	assert.Contains(t, out, "status")

	preview := components.SanitizeText(
		renderMetadataEditorPreview(
			"profile | timezone | Europe/Warsaw\nprofile | locale | pl_PL",
			nil,
			90,
			0, // maxRows lower-bound branch
		),
	)
	assert.Contains(t, preview, "profile |")
	assert.True(t, strings.Contains(preview, "timezone") || strings.Contains(preview, "locale"))
	assert.Contains(t, preview, "+1 more rows")
}

func TestRenderMetadataBlockAndSelectableNarrowWidthBranches(t *testing.T) {
	assert.Equal(t, "", renderMetadataBlockWithTitle("Metadata", map[string]any{}, 80, false))

	rows := []metadataDisplayRow{
		{field: "profile.timezone", value: "Europe/Warsaw"},
	}
	list := components.NewList(4)
	syncMetadataList(list, rows, 4)

	out := components.SanitizeText(
		renderMetadataSelectableBlockWithTitle("Metadata", rows, 50, list, nil, false),
	)
	assert.Contains(t, out, "profile")
	assert.Contains(t, out, "Rows 1-1 of 1")
}

func TestMetadataGridRowsWrappedFieldDominatesLineCount(t *testing.T) {
	rows := metadataGridRowsWrapped(
		"root",
		"very long field label that wraps",
		"value",
		10,
		8,
		20,
	)

	require.Greater(t, len(rows), 1)
	assert.Equal(t, "value", rows[0][2])
	assert.Equal(t, "", rows[len(rows)-1][2])
}
