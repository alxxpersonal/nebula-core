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
	"github.com/gravitrone/nebula-core/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testProfileTaxonomyClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestProfileTaxonomyCreateFlowQueuesReload(t *testing.T) {
	now := time.Now()
	created := false
	listed := false

	_, client := testProfileTaxonomyClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/taxonomy/scopes" && r.Method == http.MethodPost:
			created = true
			json.NewEncoder(w).Encode(map[string]any{
				"data": map[string]any{
					"id":          "scope-new",
					"name":        "work",
					"description": "desc",
					"is_builtin":  false,
					"is_active":   true,
					"metadata":    map[string]any{},
					"created_at":  now,
					"updated_at":  now,
				},
			})
			return
		case strings.HasPrefix(r.URL.Path, "/api/taxonomy/scopes") && r.Method == http.MethodGet:
			listed = true
			json.NewEncoder(w).Encode(map[string]any{
				"data": []map[string]any{
					{
						"id":          "scope-new",
						"name":        "work",
						"description": "desc",
						"is_builtin":  false,
						"is_active":   true,
						"metadata":    map[string]any{},
						"created_at":  now,
						"updated_at":  now,
					},
				},
			})
			return
		default:
			// ProfileModel uses other endpoints on Init, but this test drives prompt flow only.
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewProfileModel(client, &config.Config{APIKey: "test-key"})
	model.section = 2

	// Open create prompt.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	assert.Equal(t, taxPromptCreateName, model.taxPromptMode)

	// Type name "work" then submit.
	for _, ch := range []rune("work") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.Equal(t, taxPromptCreateDescription, model.taxPromptMode)

	// Type description then submit, which triggers API call.
	for _, ch := range []rune("desc") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{ch}})
	}
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	model, cmd = model.Update(msg)
	require.NotNil(t, cmd)
	msg = cmd()
	model, _ = model.Update(msg)

	assert.True(t, created)
	assert.True(t, listed)
	assert.Equal(t, taxPromptNone, model.taxPromptMode)
	assert.False(t, model.taxLoading)
	assert.Len(t, model.taxItems, 1)
}
