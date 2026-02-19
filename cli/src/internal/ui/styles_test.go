package ui

import (
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDividerRendersHorizontalLine(t *testing.T) {
	got := stripANSI(Divider(40))
	assert.NotEmpty(t, strings.TrimSpace(got))
	assert.GreaterOrEqual(t, len(strings.TrimSpace(got)), 10)
}
