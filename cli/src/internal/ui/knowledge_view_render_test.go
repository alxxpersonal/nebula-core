package ui

import (
	"encoding/json"
	"net/http"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestKnowledgeAddLinkSearchSaveAndReset(t *testing.T) {
	now := time.Now()
	createCalled := false
	linkCalled := false

	_, client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/audit/scopes":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "scope-1", "name": "public", "agent_count": 1},
				},
			})
		case strings.HasPrefix(r.URL.Path, "/api/entities") && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "ent-1", "name": "OpenAI", "type": "organization", "status": "active", "tags": []string{}, "metadata": map[string]any{}},
				},
			})
		case r.URL.Path == "/api/knowledge" && r.Method == http.MethodPost:
			createCalled = true
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":                "k-1",
					"name":              "Alpha",
					"source_type":       "note",
					"status":            "active",
					"tags":              []string{"demo"},
					"privacy_scope_ids": []string{"scope-1"},
					"metadata":          map[string]any{},
					"created_at":        now,
					"updated_at":        now,
				},
			})
		case r.URL.Path == "/api/knowledge/k-1/link" && r.Method == http.MethodPost:
			linkCalled = true
			w.WriteHeader(http.StatusOK)
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewKnowledgeModel(client)
	model.width = 90

	// Init + load scopes.
	cmd := model.Init()
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)
	assert.Contains(t, model.scopeOptions, "public")

	// Move focus to Entities field and start link search.
	for i := 0; i < fieldEntities; i++ {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	}
	assert.Equal(t, fieldEntities, model.focus)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.True(t, model.linkSearching)
	assert.Contains(t, components.SanitizeText(model.View()), "Link Entity")

	// Type a query and run the search command.
	var searchCmd tea.Cmd
	for _, r := range []rune("Open") {
		model, searchCmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	require.NotNil(t, searchCmd)
	msg = searchCmd()
	model, _ = model.Update(msg)
	assert.False(t, model.linkLoading)
	assert.Len(t, model.linkResults, 1)

	// Select first result.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, model.linkSearching)
	assert.Len(t, model.linkEntities, 1)
	assert.Contains(t, components.SanitizeText(model.View()), "OpenAI")

	// Fill title.
	model.focus = fieldTitle
	for _, r := range []rune("Alpha") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}

	// Commit a tag.
	model.focus = fieldTags
	for _, r := range []rune("demo") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, model.tags, "demo")

	// Select a scope via selector.
	model.focus = fieldScopes
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace}) // enter selector
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeySpace}) // toggle current scope
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter}) // exit selector
	assert.Contains(t, model.scopes, "public")

	// Save knowledge (Create + Link).
	var saveCmd tea.Cmd
	model, saveCmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	require.NotNil(t, saveCmd)
	msg = saveCmd()
	model, _ = model.Update(msg)

	assert.True(t, createCalled)
	assert.True(t, linkCalled)
	assert.True(t, model.saved)
	assert.Contains(t, components.SanitizeText(model.View()), "Knowledge saved!")

	// Esc should reset add state.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, model.saved)
	assert.Equal(t, "", model.fields[fieldTitle].value)
	assert.Len(t, model.tags, 0)
	assert.Len(t, model.scopes, 0)
}

func TestKnowledgeLibraryDetailEditAndSave(t *testing.T) {
	now := time.Now()
	updateCalled := false
	vaultPath := "/vault/knowledge/alpha.md"
	content := "notes"
	url := "https://example.com"

	_, client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/audit/scopes":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "scope-1", "name": "public", "agent_count": 1},
				},
			})
		case r.URL.Path == "/api/knowledge" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":                "k-1",
						"name":              "Alpha",
						"url":               url,
						"source_type":       "note",
						"content":           content,
						"privacy_scope_ids": []string{"scope-1"},
						"status":            "active",
						"tags":              []string{"demo"},
						"metadata":          map[string]any{"role": "builder"},
						"vault_file_path":   vaultPath,
						"created_at":        now,
						"updated_at":        now,
					},
				},
			})
		case r.URL.Path == "/api/knowledge/k-1" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":                "k-1",
					"name":              "Alpha",
					"url":               url,
					"source_type":       "note",
					"content":           content,
					"privacy_scope_ids": []string{"scope-1"},
					"status":            "active",
					"tags":              []string{"demo"},
					"metadata":          map[string]any{"role": "builder"},
					"vault_file_path":   vaultPath,
					"created_at":        now,
					"updated_at":        now,
				},
			})
		case r.URL.Path == "/api/knowledge/k-1" && r.Method == http.MethodPatch:
			updateCalled = true
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":                "k-1",
					"name":              "Alpha",
					"url":               url,
					"source_type":       "note",
					"content":           content,
					"privacy_scope_ids": []string{"scope-1"},
					"status":            "active",
					"tags":              []string{"demo", "new"},
					"metadata":          map[string]any{"role": "builder"},
					"vault_file_path":   vaultPath,
					"created_at":        now,
					"updated_at":        now,
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewKnowledgeModel(client)
	model.width = 90

	// Init + load scopes.
	cmd := model.Init()
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	// Toggle to Library view via modeFocus.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.True(t, model.modeFocus)
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)
	assert.Equal(t, knowledgeViewList, model.view)

	out := components.SanitizeText(model.View())
	assert.Contains(t, out, "Knowledge")
	assert.Contains(t, out, "1 total")
	assert.Contains(t, out, "Alpha")

	// Open detail and load it.
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)
	assert.Equal(t, knowledgeViewDetail, model.view)

	out = components.SanitizeText(model.View())
	assert.Contains(t, out, "Title")
	assert.Contains(t, out, "Alpha")
	assert.Contains(t, out, "Scopes")
	assert.Contains(t, out, "public")
	assert.Contains(t, out, "Vault Path")

	// Enter edit mode.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	assert.Equal(t, knowledgeViewEdit, model.view)

	// Add a tag and save.
	model.editFocus = knowledgeEditFieldTags
	for _, r := range []rune("new") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Contains(t, model.editTags, "new")

	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)

	assert.True(t, updateCalled)
	assert.Equal(t, knowledgeViewDetail, model.view)
}
