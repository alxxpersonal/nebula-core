package ui

import (
	"strings"
	"testing"

	"github.com/gravitrone/nebula-core/cli/internal/api"
)

func TestTaxonomyLineSanitizeGapRepro(t *testing.T) {
	item := api.TaxonomyEntry{
		Name:      "\x1b]8;;https://evil.example\x07click\x1b]8;;\x07\nsecond-line",
		IsBuiltin: true,
		IsActive:  true,
	}

	out := formatTaxonomyLine(item)
	if !strings.Contains(out, "\x1b]8;;") {
		t.Fatalf("expected OSC escape to survive (sanitize gap repro), got %q", out)
	}
	if !strings.Contains(out, "\n") {
		t.Fatalf("expected newline to survive (sanitize gap repro), got %q", out)
	}
}

