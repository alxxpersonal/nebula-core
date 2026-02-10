package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

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
	cmd := model.Init()
	msg := cmd()
	model, cmd = model.Update(msg)
	msg = cmd()
	model, _ = model.Update(msg)

	require.Len(t, model.list.Items, 1)
	assert.Contains(t, model.list.Items[0], "uses · Nebula -> Postgres")
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
