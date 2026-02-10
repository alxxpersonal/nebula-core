package components

import (
	"strings"
	"testing"

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
