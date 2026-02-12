package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestBoxWidthBounds(t *testing.T) {
	assert.Equal(t, 40, boxWidth(10))
	assert.Equal(t, 80, boxWidth(200))
	assert.Equal(t, 70, boxWidth(100))
}

func TestTitledBoxIncludesTitle(t *testing.T) {
	out := TitledBox("My Title", "Content", 80)
	assert.True(t, strings.Contains(out, "My Title"))
}

func TestTitledBoxEmptyTitleFallsBack(t *testing.T) {
	out := TitledBox("", "Content", 80)
	assert.True(t, strings.Contains(out, "Content"))
}

func TestErrorBoxIncludesMessage(t *testing.T) {
	out := ErrorBox("Error", "Something broke", 80)
	assert.True(t, strings.Contains(out, "Something broke"))
}

func TestTruncateRunes(t *testing.T) {
	assert.Equal(t, "", truncateRunes("hello", 0))
	assert.Equal(t, "he", truncateRunes("hello", 2))
	assert.Equal(t, "你", truncateRunes("你好", 1))
}

// TestTableClampsLongValues ensures table rows stay within the box width.
func TestTableClampsLongValues(t *testing.T) {
	rows := []TableRow{
		{
			Label: strings.Repeat("Label", 8),
			Value: strings.Repeat("value", 40),
		},
	}
	out := Table("Table", rows, 60)
	maxWidth := lipgloss.Width(strings.Split(Box("x", 60), "\n")[0])
	for _, line := range strings.Split(out, "\n") {
		assert.LessOrEqual(t, lipgloss.Width(line), maxWidth)
	}
}
