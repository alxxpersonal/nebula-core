package components

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestConfirmDialogIncludesTitleMessageAndHints(t *testing.T) {
	out := ConfirmDialog("Confirm", "Are you sure?")
	clean := SanitizeText(out)

	assert.Contains(t, clean, "Confirm")
	assert.Contains(t, clean, "Are you sure?")
	assert.Contains(t, clean, "y: confirm | n: cancel")
}

func TestInputDialogIncludesTitleInputAndHints(t *testing.T) {
	out := InputDialog("Filter", "hello")
	clean := SanitizeText(out)

	assert.Contains(t, clean, "Filter")
	assert.Contains(t, clean, "> hello")
	assert.Contains(t, clean, "enter: submit | esc: cancel")
}

func TestConfirmPreviewDialogIncludesSummaryAndChanges(t *testing.T) {
	out := ConfirmPreviewDialog(
		"Archive Entity",
		[]TableRow{{Label: "Entity", Value: "Alpha"}},
		[]DiffRow{{Label: "status", From: "active", To: "archived"}},
		80,
	)
	clean := SanitizeText(out)

	assert.Contains(t, clean, "Archive Entity")
	assert.Contains(t, clean, "Summary")
	assert.Contains(t, clean, "Entity")
	assert.Contains(t, clean, "Alpha")
	assert.Contains(t, clean, "Changes")
	assert.Contains(t, clean, "status")
	assert.Contains(t, clean, "- active")
	assert.Contains(t, clean, "+ archived")
	assert.Contains(t, clean, "y: confirm | n: cancel")
	assert.Equal(t, 1, strings.Count(clean, "╭"))
	assert.Equal(t, 1, strings.Count(clean, "╮"))
}
