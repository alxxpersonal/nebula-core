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

func testLogsClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestLogsInitLoadsLogsAndScopes(t *testing.T) {
	now := time.Now()
	_, client := testLogsClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/logs") && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":         "log-1",
						"log_type":   "workout",
						"timestamp":  now,
						"value":      map[string]any{"note": "x"},
						"status":     "active",
						"tags":       []string{},
						"metadata":   map[string]any{},
						"created_at": now,
						"updated_at": now,
					},
				},
			})
			return
		case r.URL.Path == "/api/audit/scopes" && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":              "scope-1",
						"name":            "public",
						"description":     nil,
						"agent_count":     0,
						"entity_count":    0,
						"knowledge_count": 0,
					},
				},
			})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewLogsModel(client)
	cmd := model.Init()
	require.NotNil(t, cmd)
	msg := cmd()
	model, cmd = model.Update(msg)

	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)

	assert.False(t, model.loading)
	assert.Len(t, model.items, 1)
	assert.Equal(t, "log-1", model.items[0].ID)
	assert.Contains(t, model.scopeOptions, "public")
}

func TestLogsAddValidationErrorOnEmpty(t *testing.T) {
	_, client := testLogsClient(t, func(w http.ResponseWriter, r *http.Request) {})
	model := NewLogsModel(client)
	model.view = logsViewAdd

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	assert.Equal(t, "Type is required", model.addErr)
}
