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

func testProfileClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestProfileAgentDetailToggle(t *testing.T) {
	model := NewProfileModel(nil, &config.Config{Username: "alxx"})
	model.section = 1
	model.agents = []api.Agent{
		{
			ID:           "agent-1",
			Name:         "Alpha",
			Status:       "active",
			Scopes:       []string{"public"},
			Capabilities: []string{"read"},
			CreatedAt:    time.Now(),
			UpdatedAt:    time.Now(),
		},
	}
	model.agentList.SetItems([]string{formatAgentLine(model.agents[0])})

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, model.agentDetail)
	assert.Equal(t, "agent-1", model.agentDetail.ID)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Nil(t, model.agentDetail)
}

func TestProfileSetAPIKeyPersistsAndUpdatesClient(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	var seenAuth string
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		seenAuth = r.Header.Get("Authorization")
		if seenAuth != "Bearer nbl_newkey" {
			w.WriteHeader(http.StatusUnauthorized)
			return
		}
		_, _ = w.Write([]byte(`{"data":{"id":"ent-1","name":"ok","tags":[]}}`))
	}))
	defer srv.Close()

	cfg := &config.Config{
		APIKey:   "nbl_oldkey",
		Username: "alxx",
	}
	require.NoError(t, cfg.Save())

	client := api.NewClient(srv.URL, "nbl_oldkey")
	model := NewProfileModel(client, cfg)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'k'}})
	require.True(t, model.editAPIKey)

	model.apiKeyBuf = "nbl_newkey"
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	msg := cmd()
	model, _ = model.Update(msg)
	require.False(t, model.editAPIKey)

	loaded, err := config.Load()
	require.NoError(t, err)
	assert.Equal(t, "nbl_newkey", loaded.APIKey)

	_, err = client.GetEntity("ent-1")
	require.NoError(t, err)
	assert.Equal(t, "Bearer nbl_newkey", seenAuth)
}

func TestProfileKeysLoadCreateAndRevokeFlows(t *testing.T) {
	now := time.Now()
	var createName string
	var revokedID string
	keysAllCalls := 0

	_, client := testProfileClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/keys/all" && r.Method == http.MethodGet:
			keysAllCalls++
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{
					"id":         "k1",
					"key_prefix": "nbl_abc123",
					"name":       "demo",
					"created_at": now,
				},
			}})
			return
		case r.URL.Path == "/api/keys" && r.Method == http.MethodPost:
			var body map[string]string
			require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
			createName = body["name"]
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{
				"api_key": "nbl_created_secret",
				"key_id":  "k2",
				"prefix":  "nbl_created",
				"name":    createName,
			}})
			return
		case strings.HasPrefix(r.URL.Path, "/api/keys/") && r.Method == http.MethodDelete:
			revokedID = strings.TrimPrefix(r.URL.Path, "/api/keys/")
			w.WriteHeader(http.StatusOK)
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	cfg := &config.Config{Username: "alxx", APIKey: "nbl_zzzzzzzzzz"}
	model := NewProfileModel(client, cfg)
	model.width = 100

	// Load keys.
	model, _ = model.Update(model.loadKeys())
	require.Len(t, model.keys, 1)
	assert.Contains(t, model.View(), "nbl_abc123...")

	// Create key flow.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	require.True(t, model.creating)
	for _, r := range []rune("my key") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	require.NotNil(t, cmd)
	model, cmd = model.Update(cmd())
	require.NotNil(t, cmd)
	model, _ = model.Update(cmd())
	assert.Equal(t, "my key", createName)
	assert.Equal(t, "nbl_created_secret", model.createdKey)
	require.GreaterOrEqual(t, keysAllCalls, 2)

	// Clear created key gate and revoke selected.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
	require.NotNil(t, cmd)
	model, cmd = model.Update(cmd())
	require.NotNil(t, cmd)
	model, _ = model.Update(cmd())
	assert.Equal(t, "k1", revokedID)
}

func TestProfileAgentsLoadAndToggleTrustFlow(t *testing.T) {
	now := time.Now()
	var patchedID string
	var patched api.UpdateAgentInput
	agentsCalls := 0

	_, client := testProfileClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/agents/" && r.Method == http.MethodGet:
			agentsCalls++
			json.NewEncoder(w).Encode(map[string]any{"data": []map[string]any{
				{
					"id":                "agent-1",
					"name":              "Alpha",
					"status":            "active",
					"requires_approval": true,
					"scopes":            []string{"public"},
					"capabilities":      []string{"read"},
					"created_at":        now,
					"updated_at":        now,
				},
			}})
			return
		case strings.HasPrefix(r.URL.Path, "/api/agents/") && r.Method == http.MethodPatch:
			patchedID = strings.TrimPrefix(r.URL.Path, "/api/agents/")
			require.NoError(t, json.NewDecoder(r.Body).Decode(&patched))
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": patchedID}})
			return
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	cfg := &config.Config{Username: "alxx", APIKey: "nbl_zzzzzzzzzz"}
	model := NewProfileModel(client, cfg)
	model.section = 1
	model.width = 100

	// Load agents.
	model, _ = model.Update(model.loadAgents())
	require.Len(t, model.agents, 1)
	assert.Contains(t, model.View(), "Agents")

	// Toggle trust.
	model, cmd := model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	require.NotNil(t, cmd)
	model, cmd = model.Update(cmd())
	require.NotNil(t, cmd)
	model, _ = model.Update(cmd())

	assert.Equal(t, "agent-1", patchedID)
	require.NotNil(t, patched.RequiresApproval)
	assert.False(t, *patched.RequiresApproval)
	require.GreaterOrEqual(t, agentsCalls, 2)
}

func TestMaskedAPIKey(t *testing.T) {
	assert.Equal(t, "-", maskedAPIKey(""))
	assert.Equal(t, "*****", maskedAPIKey("abcde"))
	assert.Equal(t, "abcdef...7890", maskedAPIKey("abcdef1234567890"))
}
