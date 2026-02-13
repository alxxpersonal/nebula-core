package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func relTestClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestRelationshipsInitLoadsNames(t *testing.T) {
	_, client := relTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/relationships":
			resp := map[string]any{
				"data": []map[string]any{
					{"id": "rel-1", "source_id": "ent-1", "target_id": "ent-2", "relationship_type": "uses", "properties": map[string]any{}},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case "/api/entities/ent-1":
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "ent-1", "name": "Nebula", "tags": []string{}}})
		case "/api/entities/ent-2":
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "ent-2", "name": "Postgres", "tags": []string{}}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewRelationshipsModel(client)
	cmds := []tea.Cmd{
		model.loadRelationships(),
		model.loadScopeOptions(),
		model.loadEntityCache(),
	}
	for _, cmd := range cmds {
		if cmd == nil {
			continue
		}
		model = applyMsg(model, cmd())
	}

	require.Len(t, model.list.Items, 1)
	assert.Contains(t, model.list.Items[0], "uses · Nebula -> Postgres")
}

func applyMsg(model RelationshipsModel, msg tea.Msg) RelationshipsModel {
	var cmd tea.Cmd
	model, cmd = model.Update(msg)
	if cmd == nil {
		return model
	}
	next := cmd()
	if next == nil {
		return model
	}
	return applyMsg(model, next)
}

func TestRelationshipsCreateSubmitCallsAPI(t *testing.T) {
	var captured api.CreateRelationshipInput
	_, client := relTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/relationships" && r.Method == http.MethodPost {
			var body api.CreateRelationshipInput
			json.NewDecoder(r.Body).Decode(&body)
			captured = body
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "rel-1"}})
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	model := NewRelationshipsModel(client)
	model.view = relsViewCreateType
	model.createSource = &api.Entity{ID: "ent-1", Type: "tool"}
	model.createTarget = &api.Entity{ID: "ent-2", Type: "project"}
	model.createType = "uses"

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.Equal(t, "ent-1", captured.SourceID)
	assert.Equal(t, "ent-2", captured.TargetID)
	assert.Equal(t, "uses", captured.Type)
	assert.Equal(t, "entity", captured.SourceType)
	assert.Equal(t, "entity", captured.TargetType)
}

func TestRelationshipsCreateLiveSearch(t *testing.T) {
	var capturedQuery string
	_, client := relTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path == "/api/entities" {
			capturedQuery = r.URL.Query().Get("search_text")
			resp := map[string]any{
				"data": []map[string]any{
					{"id": "ent-1", "name": "alxx", "type": "person"},
				},
			}
			json.NewEncoder(w).Encode(resp)
			return
		}
		w.WriteHeader(http.StatusNotFound)
	})

	model := NewRelationshipsModel(client)
	model.view = relsViewCreateSourceSearch

	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.Equal(t, "a", capturedQuery)
	require.Len(t, model.createResults, 1)
	assert.Equal(t, "ent-1", model.createResults[0].ID)
}

func TestRelationshipTypeSuggestions(t *testing.T) {
	model := NewRelationshipsModel(api.NewClient("http://example.com", "key"))
	model.typeOptions = []string{"works-on", "created-by"}
	model.view = relsViewCreateType

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'w'}})
	require.NotEmpty(t, model.createTypeResults)
	assert.Equal(t, "works-on", model.createTypeResults[0])
}

func TestRelationshipsInitViewDetailEditAndConfirmFlow(t *testing.T) {
	now := time.Now()
	var patched bool
	var cmd tea.Cmd

	_, client := relTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/relationships" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{
					"id":                "rel-1",
					"source_type":       "entity",
					"source_id":         "ent-1",
					"source_name":       "Alpha",
					"target_type":       "entity",
					"target_id":         "ent-2",
					"target_name":       "Beta",
					"relationship_type": "uses",
					"status":            "active",
					"properties":        map[string]any{},
					"created_at":        now,
				},
			}})
			return
		case r.URL.Path == "/api/audit/scopes" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{"id": "s1", "name": "public", "agent_count": 0, "entity_count": 0, "knowledge_count": 0},
			}})
			return
		case r.URL.Path == "/api/entities" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{"id": "ent-1", "name": "Alpha", "type": "entity", "tags": []string{}},
				{"id": "ent-2", "name": "Beta", "type": "entity", "tags": []string{}},
			}})
			return
		case strings.HasPrefix(r.URL.Path, "/api/relationships/") && r.Method == http.MethodPatch:
			patched = true
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"id":                "rel-1",
				"source_id":         "ent-1",
				"target_id":         "ent-2",
				"relationship_type": "uses",
				"status":            "archived",
				"properties":        map[string]any{},
				"created_at":        now,
			}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewRelationshipsModel(client)
	model.width = 80
	_ = model.Init() // covers Init() branch; run the cmds explicitly to avoid tea.BatchMsg handling here.
	for _, cmd := range []tea.Cmd{model.loadRelationships(), model.loadScopeOptions(), model.loadEntityCache()} {
		if cmd == nil {
			continue
		}
		model = applyMsg(model, cmd())
	}
	assert.False(t, model.loading)
	require.Len(t, model.items, 1)
	assert.Contains(t, model.View(), "Relationships")

	// Enter detail.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, relsViewDetail, model.view)
	assert.Contains(t, model.View(), "Relationship")

	// Open archive confirm and accept.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
	assert.Equal(t, relsViewConfirm, model.view)
	assert.Contains(t, model.View(), "Archive Relationship")

	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	require.NotNil(t, cmd)
	model = applyMsg(model, cmd())
	require.True(t, patched)
}

func TestRelationshipsModeFocusTogglesToAddFlow(t *testing.T) {
	_, client := relTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/relationships" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
			return
		case r.URL.Path == "/api/audit/scopes" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
			return
		case r.URL.Path == "/api/entities" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewRelationshipsModel(client)
	model.width = 80
	_ = model.Init()
	for _, cmd := range []tea.Cmd{model.loadRelationships(), model.loadScopeOptions(), model.loadEntityCache()} {
		if cmd == nil {
			continue
		}
		model = applyMsg(model, cmd())
	}

	// Focus mode line from list selection 0, then toggle into add flow.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.True(t, model.modeFocus)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, model.modeFocus)
	assert.True(t, model.isAddView())
	assert.Equal(t, relsViewCreateSourceSearch, model.view)
}

func TestRelationshipsCreateFlowSubmitsAndReturnsToList(t *testing.T) {
	now := time.Now()
	var createdType string

	_, client := relTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/relationships" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
			return
		case r.URL.Path == "/api/entities" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{"id": "ent-1", "name": "Alpha", "type": "entity", "tags": []string{}},
				{"id": "ent-2", "name": "Beta", "type": "entity", "tags": []string{}},
			}})
			return
		case r.URL.Path == "/api/audit/scopes" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
			return
		case r.URL.Path == "/api/relationships" && r.Method == http.MethodPost:
			var body api.CreateRelationshipInput
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			createdType = body.Type
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"id":                "rel-1",
				"relationship_type": body.Type,
				"created_at":        now,
			}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewRelationshipsModel(client)
	model.width = 80
	_ = model.Init()
	for _, cmd := range []tea.Cmd{model.loadRelationships(), model.loadScopeOptions(), model.loadEntityCache()} {
		if cmd == nil {
			continue
		}
		model = applyMsg(model, cmd())
	}

	// Start create flow.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Equal(t, relsViewCreateSourceSearch, model.view)
	assert.Contains(t, model.View(), "Source Entity")

	// Type query to filter from cache.
	for _, r := range []rune("al") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	require.NotEmpty(t, model.createResults)

	// Select source.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, relsViewCreateTargetSearch, model.view)

	// Type query and select target.
	for _, r := range []rune("be") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	require.NotEmpty(t, model.createResults)
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, relsViewCreateType, model.view)
	assert.Contains(t, model.View(), "Relationship Type")

	// Type relationship type and submit.
	for _, r := range []rune("knows") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model = applyMsg(model, cmd())

	assert.Equal(t, "knows", createdType)
	assert.Equal(t, relsViewList, model.view)
}
