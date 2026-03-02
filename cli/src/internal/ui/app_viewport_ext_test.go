package ui

import (
	"strings"
	"testing"

	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestClampBodyForViewportNoClampAndMinBudget(t *testing.T) {
	body := "line 1\nline 2"
	out, clipped := clampBodyForViewport(body, 3, 0, 0, 0)
	assert.False(t, clipped)
	assert.Equal(t, body, out)
}

func TestClampBodyForViewportAddsTopAndBottomMarkers(t *testing.T) {
	lines := make([]string, 0, 12)
	for i := 1; i <= 12; i++ {
		lines = append(lines, "row "+string('A'+rune(i-1)))
	}
	body := strings.Join(lines, "\n")

	out, clipped := clampBodyForViewport(body, 10, 1, 1, 2)
	require.True(t, clipped)
	clean := components.SanitizeText(out)
	split := strings.Split(clean, "\n")
	require.Len(t, split, 6)
	assert.Contains(t, split[0], "... ↑ more")
	assert.Contains(t, split[len(split)-1], "... ↓ more")
}

func TestClampBodyForViewportScrollBoundsAndEndClamp(t *testing.T) {
	lines := make([]string, 0, 10)
	for i := 1; i <= 10; i++ {
		lines = append(lines, "row "+string('A'+rune(i-1)))
	}
	body := strings.Join(lines, "\n")

	out, clipped := clampBodyForViewport(body, 10, 0, 0, 100)
	require.True(t, clipped)
	clean := components.SanitizeText(out)
	split := strings.Split(clean, "\n")
	require.Len(t, split, 8)
	assert.Contains(t, split[0], "... ↑ more")
	assert.NotContains(t, split[len(split)-1], "... ↓ more")
}

func TestClampBodyForViewportNegativeScrollAndTightViewport(t *testing.T) {
	lines := []string{
		"row 1", "row 2", "row 3", "row 4",
		"row 5", "row 6", "row 7", "row 8",
	}
	body := strings.Join(lines, "\n")

	// totalHeight forces min budget branch (budget < 6 -> 6).
	out, clipped := clampBodyForViewport(body, 5, 2, 2, -42)
	require.True(t, clipped)
	clean := components.SanitizeText(out)
	split := strings.Split(clean, "\n")
	require.Len(t, split, 6)
	assert.NotContains(t, split[0], "... ↑ more")
	assert.Contains(t, split[len(split)-1], "... ↓ more")
}
