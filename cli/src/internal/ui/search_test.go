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

func searchTestClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestSearchModelQueryCallsEndpoints(t *testing.T) {
	var entityQuery, knowledgeQuery, jobQuery string
	_, client := searchTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/entities":
			entityQuery = r.URL.Query().Get("search_text")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "ent-1", "name": "alxx", "type": "person"},
				},
			})
		case "/api/knowledge":
			knowledgeQuery = r.URL.Query().Get("search_text")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "kn-1", "name": "Nebula Notes", "source_type": "note"},
				},
			})
		case "/api/jobs":
			jobQuery = r.URL.Query().Get("search_text")
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "job-1", "title": "Alpha Job", "status": "active"},
				},
			})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewSearchModel(client)
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.Equal(t, "a", entityQuery)
	assert.Equal(t, "a", knowledgeQuery)
	assert.Equal(t, "a", jobQuery)
	assert.Len(t, model.items, 3)
}

func TestSearchModelSelectionEmitsMsg(t *testing.T) {
	_, client := searchTestClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch r.URL.Path {
		case "/api/entities":
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{"id": "ent-1", "name": "alpha", "type": "tool"},
				},
			})
		case "/api/knowledge":
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		case "/api/jobs":
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewSearchModel(client)
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	msg := cmd()
	model, _ = model.Update(msg)

	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	selection := cmd().(searchSelectionMsg)

	assert.Equal(t, "entity", selection.kind)
	require.NotNil(t, selection.entity)
	assert.Equal(t, "ent-1", selection.entity.ID)
}
