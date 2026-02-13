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

func testProtocolsClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestProtocolsListToDetailToEditFlow(t *testing.T) {
	now := time.Now()
	_, client := testProtocolsClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case strings.HasPrefix(r.URL.Path, "/api/protocols") && r.Method == http.MethodGet:
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":         "proto-1",
						"name":       "p1",
						"title":      "Protocol 1",
						"content":    "hello",
						"status":     "active",
						"tags":       []string{},
						"metadata":   map[string]any{},
						"created_at": now,
						"updated_at": now,
					},
				},
			})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewProtocolsModel(client)
	cmd := model.Init()
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)

	assert.False(t, model.loading)
	assert.Len(t, model.items, 1)

	// Enter detail
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, model.detail)
	assert.Equal(t, protocolsViewDetail, model.view)

	// Enter edit
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	assert.Equal(t, protocolsViewEdit, model.view)
}

func TestProtocolsAddValidationErrorOnEmpty(t *testing.T) {
	_, client := testProtocolsClient(t, func(w http.ResponseWriter, r *http.Request) {})
	model := NewProtocolsModel(client)
	model.view = protocolsViewAdd

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyCtrlS})
	assert.Equal(t, "Name is required", model.addErr)
}
