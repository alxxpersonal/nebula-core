package ui

import "testing"

func TestFormatScopePreview(t *testing.T) {
	got := formatScopePreview([]string{"public", "admin"})
	want := "[public] [admin]"
	if got != want {
		t.Fatalf("expected %q, got %q", want, got)
	}

	if empty := formatScopePreview(nil); empty != "-" {
		t.Fatalf("expected dash for empty scopes, got %q", empty)
	}
}
