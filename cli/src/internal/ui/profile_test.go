package ui

import (
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestProfileAgentDetailToggle(t *testing.T) {
	model := NewProfileModel(nil, &config.Config{Username: "alxx", ServerURL: "http://localhost"})
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
		ServerURL: srv.URL,
		APIKey:    "nbl_oldkey",
		Username:  "alxx",
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
