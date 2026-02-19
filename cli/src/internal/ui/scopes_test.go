package ui

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestScopeSelectedReportsMembership(t *testing.T) {
	assert.True(t, scopeSelected([]string{"public", "private"}, "private"))
	assert.False(t, scopeSelected([]string{"public", "private"}, "admin"))
}

func TestRenderScopeOptionsShowsSelectionAndCursor(t *testing.T) {
	out := renderScopeOptions(
		[]string{"private"},
		[]string{"public", "private", "admin"},
		1,
	)
	clean := stripANSI(out)

	assert.Contains(t, clean, "[private]")
	assert.Contains(t, clean, "public")
	assert.Contains(t, clean, "admin")
}

func TestRenderScopeOptionsFallbacksToSelectedWhenOptionsEmpty(t *testing.T) {
	out := renderScopeOptions([]string{"sensitive"}, nil, 0)
	clean := stripANSI(out)
	assert.Contains(t, clean, "[sensitive]")
}
