package ui

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestHistoryModelScopesAndActorsSelectionLoadsHistory(t *testing.T) {
	var lastPath string
	var lastScopeID string
	var lastActorType string
	var lastActorID string

	_, client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		lastPath = r.URL.Path
		switch r.URL.Path {
		case "/api/audit":
			lastScopeID = r.URL.Query().Get("scope_id")
			lastActorType = r.URL.Query().Get("actor_type")
			lastActorID = r.URL.Query().Get("actor_id")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":         "audit-1",
						"table_name": "entities",
						"record_id":  "ent-1",
						"action":     "update",
						"changed_at": time.Now(),
					},
				},
			})
		case "/api/audit/scopes":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "scope-1", "name": "public", "agent_count": 1},
				},
			})
		case "/api/audit/actors":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"changed_by_type": "agent", "changed_by_id": "agent-1", "action_count": 2},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewHistoryModel(client)
	model.width = 80

	// Init loads history.
	cmd := model.Init()
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)
	assert.Equal(t, "/api/audit", lastPath)

	// Load scopes.
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)
	assert.Equal(t, historyViewScopes, model.view)

	// Select scope and verify it is applied to the next history load.
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)
	assert.Equal(t, "scope-1", model.filter.scopeID)
	assert.Equal(t, "scope-1", lastScopeID)

	// Load actors.
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)
	assert.Equal(t, historyViewActors, model.view)

	// Select actor and verify it is applied to the next history load.
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)
	assert.Equal(t, "agent", model.filter.actorType)
	assert.Equal(t, "agent-1", model.filter.actorID)
	assert.Equal(t, "agent", lastActorType)
	assert.Equal(t, "agent-1", lastActorID)
}

func TestHistoryModelFilterPromptAppliesAndLoads(t *testing.T) {
	var gotTable string
	var gotAction string

	_, client := testClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/audit":
			gotTable = r.URL.Query().Get("table")
			gotAction = r.URL.Query().Get("action")
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewHistoryModel(client)
	model.width = 80

	// Enter filtering mode.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	assert.True(t, model.filtering)

	// Type a filter and apply.
	for _, r := range []rune("table:entities action:update") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	var cmd tea.Cmd
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.False(t, model.filtering)
	assert.Equal(t, "entities", gotTable)
	assert.Equal(t, "update", gotAction)
}
