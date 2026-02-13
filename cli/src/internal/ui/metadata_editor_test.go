package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestMetadataEditorOpenLoadsInitialAndActivates(t *testing.T) {
	var ed MetadataEditor
	ed.Open(map[string]any{
		"scopes": []any{"public"},
		"name":   "alex",
	})

	require.True(t, ed.Active)
	assert.Equal(t, []string{"public"}, ed.Scopes)
	assert.Contains(t, ed.Buffer, "name: alex")
}

func TestMetadataEditorHandleKeyTypingScopesAndExit(t *testing.T) {
	var ed MetadataEditor
	ed.Open(map[string]any{})
	ed.SetScopeOptions([]string{"public", "work"})

	// Typing appends to buffer.
	ed.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	assert.Equal(t, "a", ed.Buffer)

	// Backspace drops last rune.
	ed.HandleKey(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "", ed.Buffer)

	// Space on empty buffer opens scope selection.
	ed.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	assert.True(t, ed.scopeSelecting)

	// Move to "work" and toggle it on.
	ed.HandleKey(tea.KeyMsg{Type: tea.KeyRight})
	ed.HandleKey(tea.KeyMsg{Type: tea.KeySpace})
	assert.Contains(t, ed.Scopes, "work")

	// Exit scope selection.
	ed.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, ed.scopeSelecting)

	// Enter adds newline, tab adds spaces.
	ed.HandleKey(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
	ed.HandleKey(tea.KeyMsg{Type: tea.KeyEnter})
	ed.HandleKey(tea.KeyMsg{Type: tea.KeyTab})
	assert.Contains(t, ed.Buffer, "x\n  ")

	// Ctrl+U clears buffer.
	ed.HandleKey(tea.KeyMsg{Type: tea.KeyCtrlU})
	assert.Equal(t, "", ed.Buffer)

	// Esc closes editor and returns true to indicate done.
	done := ed.HandleKey(tea.KeyMsg{Type: tea.KeyEsc})
	assert.True(t, done)
	assert.False(t, ed.Active)
}

func TestMetadataEditorRenderIsStable(t *testing.T) {
	var ed MetadataEditor
	ed.Open(map[string]any{"name": "alex"})
	ed.Buffer = "name: alex"
	out := ed.Render(80)
	assert.Contains(t, out, "Metadata")
	assert.Contains(t, out, "name: alex")

	// Invalid YAML should render an error hint.
	ed.Buffer = "name alex"
	out = ed.Render(80)
	assert.Contains(t, out, "Metadata")
	assert.Contains(t, out, "expected 'key: value'")
}

func TestDropLastRuneHandlesMultibyteRunes(t *testing.T) {
	assert.Equal(t, "", dropLastRune(""))
	assert.Equal(t, "a", dropLastRune("ab"))
	assert.Equal(t, "a", dropLastRune("a😊"))
}
