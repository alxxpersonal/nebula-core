package ui

import (
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
