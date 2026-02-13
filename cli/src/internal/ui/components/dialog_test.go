package components

import (
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
