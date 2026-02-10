package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetAgent(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/agents/")
		w.Write(jsonResponse(map[string]any{
			"id":                "ag-1",
			"name":              "test-agent",
			"status":            "active",
			"requires_approval": true,
			"scopes":            []string{"public"},
		}))
	})

	agent, err := client.GetAgent("test-agent")
	require.NoError(t, err)
	assert.Equal(t, "ag-1", agent.ID)
	assert.Equal(t, "test-agent", agent.Name)
	assert.True(t, agent.RequiresApproval)
}

func TestListAgentsWithFilter(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "active", r.URL.Query().Get("status_category"))
		w.Write(jsonResponse([]map[string]any{
			{
				"id":                "ag-1",
				"name":              "agent1",
				"status":            "active",
				"requires_approval": false,
				"scopes":            []string{"public"},
			},
		}))
	})

	agents, err := client.ListAgents("active")
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "agent1", agents[0].Name)
}

func TestListAgentsNoFilter(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Empty(t, r.URL.Query().Get("status_category"))
		w.Write(jsonResponse([]map[string]any{
			{"id": "ag-1", "name": "agent1", "status": "active", "requires_approval": false, "scopes": []string{}},
			{"id": "ag-2", "name": "agent2", "status": "inactive", "requires_approval": true, "scopes": []string{}},
		}))
	})

	agents, err := client.ListAgents("")
	require.NoError(t, err)
	assert.Len(t, agents, 2)
}

func TestUpdateAgent(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)

		var body UpdateAgentInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.NotNil(t, body.RequiresApproval)
		assert.False(t, *body.RequiresApproval)

		w.Write(jsonResponse(map[string]any{
			"id":                "ag-1",
			"name":              "agent1",
			"status":            "active",
			"requires_approval": false,
			"scopes":            []string{"public"},
		}))
	})

	falseVal := false
	agent, err := client.UpdateAgent("ag-1", UpdateAgentInput{
		RequiresApproval: &falseVal,
	})
	require.NoError(t, err)
	assert.False(t, agent.RequiresApproval)
}

func TestRegisterAgentDuplicate(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(409)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "DUPLICATE",
				"message": "agent already exists",
			},
		})
		w.Write(b)
	})

	_, err := client.RegisterAgent(RegisterAgentInput{
		Name:            "existing-agent",
		RequestedScopes: []string{"public"},
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "DUPLICATE")
}
