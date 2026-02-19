package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestFitGridColumnsPrefersShrinkingWideColumns(t *testing.T) {
	columns := []TableColumn{
		{Header: "Rel", Width: 12, Align: lipgloss.Left},
		{Header: "Edge", Width: 42, Align: lipgloss.Left},
		{Header: "Status", Width: 9, Align: lipgloss.Left},
		{Header: "At", Width: 11, Align: lipgloss.Left},
	}

	// Force deficit so at least one column must shrink.
	fitted := fitGridColumns(columns, "|", 56)

	assert.Equal(t, 12, fitted[0].Width, "short system columns should remain stable")
	assert.Less(t, fitted[1].Width, 42, "wide edge column should absorb shrink first")
	assert.Equal(t, 9, fitted[2].Width, "status column should remain readable")
	assert.Equal(t, 11, fitted[3].Width, "time column should remain readable")
}

func TestShrinkColumnsStopsAtMinimums(t *testing.T) {
	columns := []TableColumn{
		{Header: "A", Width: 4},
		{Header: "B", Width: 4},
	}
	remaining := shrinkColumns(columns, []int{4, 4}, 10)
	assert.Equal(t, 10, remaining)
	assert.Equal(t, 4, columns[0].Width)
	assert.Equal(t, 4, columns[1].Width)
}

func TestTableGridWithActiveRowClampsWidthAndRendersRows(t *testing.T) {
	columns := []TableColumn{
		{Header: "Name", Width: 16, Align: lipgloss.Left},
		{Header: "Notes", Width: 28, Align: lipgloss.Left},
		{Header: "State", Width: 10, Align: lipgloss.Right},
	}
	rows := [][]string{
		{"alpha", strings.Repeat("very-long-", 8), "[X] ready"},
		{"beta", "short", "open"},
	}
	table := TableGridWithActiveRow(columns, rows, 64, 0)
	lines := strings.Split(table, "\n")
	assert.GreaterOrEqual(t, len(lines), 3)
	for _, line := range lines {
		assert.LessOrEqual(t, lipgloss.Width(line), 64)
	}
}

func TestRenderGridCellAlignModes(t *testing.T) {
	left := renderGridCell("x", 6, lipgloss.Left)
	right := renderGridCell("x", 6, lipgloss.Right)
	center := renderGridCell("x", 6, lipgloss.Center)

	assert.Equal(t, 6, lipgloss.Width(left))
	assert.Equal(t, 6, lipgloss.Width(right))
	assert.Equal(t, 6, lipgloss.Width(center))
	assert.True(t, strings.HasSuffix(left, " "))
	assert.True(t, strings.HasPrefix(right, " "))
	assert.True(t, strings.HasPrefix(center, " "))
}

func TestHighlightSelectionMarkersStylesKnownTokens(t *testing.T) {
	out := highlightSelectionMarkers(" [X] row [x] ")
	clean := SanitizeText(out)
	assert.Contains(t, clean, "[X]")
	assert.Contains(t, clean, "[x]")
}

func TestTableGridWrapperRendersSameContract(t *testing.T) {
	columns := []TableColumn{
		{Header: "Name", Width: 12, Align: lipgloss.Left},
		{Header: "Status", Width: 10, Align: lipgloss.Left},
	}
	rows := [][]string{{"alpha", "active"}}
	table := TableGrid(columns, rows, 40)
	assert.NotEmpty(t, table)
	for _, line := range strings.Split(table, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), 40)
	}
}

func TestTableGridWithActiveRowCanDisableHighlighting(t *testing.T) {
	columns := []TableColumn{
		{Header: "Name", Width: 12, Align: lipgloss.Left},
		{Header: "Status", Width: 10, Align: lipgloss.Left},
	}
	rows := [][]string{{"alpha", "active"}, {"beta", "idle"}}

	withoutActive := TableGridWithActiveRow(columns, rows, 40, -1)

	SetTableGridActiveRowsEnabled(false)
	defer SetTableGridActiveRowsEnabled(true)

	withSuppressedActive := TableGridWithActiveRow(columns, rows, 40, 0)
	assert.Equal(t, withoutActive, withSuppressedActive)
}
