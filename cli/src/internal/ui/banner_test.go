package ui

import (
	"strings"
	"testing"

	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
	"github.com/stretchr/testify/assert"
)

func TestSplitLinesSplitsOnNewlines(t *testing.T) {
	lines := splitLines("a\nb\nc")
	assert.Equal(t, []string{"a", "b", "c"}, lines)
}

func TestRenderBannerIncludesSubtitleAndNoOSC(t *testing.T) {
	out := RenderBanner()
	assert.NotContains(t, out, "\x1b]")

	clean := components.SanitizeText(out)
	assert.Contains(t, clean, "Context Infrastructure for Agents")
	assert.Contains(t, clean, "Command-Line Interface")
	assert.True(t, strings.Contains(clean, "─"))
}
