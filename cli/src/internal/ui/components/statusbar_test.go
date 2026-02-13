package components

import (
	"strings"
	"testing"

	"github.com/charmbracelet/lipgloss"
	"github.com/stretchr/testify/assert"
)

func TestHintIncludesKeyAndDesc(t *testing.T) {
	out := Hint("↑/↓", "Scroll")
	assert.True(t, strings.Contains(out, "Scroll"))
	assert.True(t, strings.Contains(out, "↑/↓"))
}

func TestStatusBarRendersHints(t *testing.T) {
	out := StatusBar([]string{Hint("q", "Quit")}, 0)
	assert.True(t, strings.Contains(out, "Quit"))
	assert.True(t, strings.Contains(out, "q"))
}

func TestWrapSegmentsWrapsWhenNarrow(t *testing.T) {
	segments := []string{"123456", "abcdef", "ghijkl"}
	rows := wrapSegments(segments, 10)
	assert.Len(t, rows, 3)
	for _, row := range rows {
		assert.LessOrEqual(t, lipgloss.Width(row), 10)
	}
}
