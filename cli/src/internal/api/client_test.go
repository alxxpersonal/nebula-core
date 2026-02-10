package api

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"sync"
	"sync/atomic"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func testServer(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *Client) {
	t.Helper()
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	client := NewClient(srv.URL, "nbl_testkey")
	return srv, client
}

func jsonResponse(data any) []byte {
	b, _ := json.Marshal(map[string]any{"data": data})
	return b
}

func TestGetEntity(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "Bearer nbl_testkey", r.Header.Get("Authorization"))
		assert.Contains(t, r.URL.Path, "/api/entities/")
		w.Write(jsonResponse(map[string]any{
			"id":   "abc-123",
			"name": "test entity",
			"tags": []string{"test"},
		}))
	})

	entity, err := client.GetEntity("abc-123")
	require.NoError(t, err)
	assert.Equal(t, "abc-123", entity.ID)
	assert.Equal(t, "test entity", entity.Name)
}

func TestQueryEntities(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "active", r.URL.Query().Get("status"))
		w.Write(jsonResponse([]map[string]any{
			{"id": "1", "name": "one", "tags": []string{}},
			{"id": "2", "name": "two", "tags": []string{}},
		}))
	})

	entities, err := client.QueryEntities(QueryParams{"status": "active"})
	require.NoError(t, err)
	assert.Len(t, entities, 2)
}

func TestQueryAuditLog(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/audit", r.URL.Path)
		assert.Equal(t, "entities", r.URL.Query().Get("table"))
		assert.Equal(t, "update", r.URL.Query().Get("action"))
		assert.Equal(t, "agent", r.URL.Query().Get("actor_type"))
		assert.Equal(t, "agent-1", r.URL.Query().Get("actor_id"))
		assert.Equal(t, "ent-1", r.URL.Query().Get("record_id"))
		assert.Equal(t, "scope-1", r.URL.Query().Get("scope_id"))
		assert.Equal(t, "25", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))
		w.Write(jsonResponse([]map[string]any{
			{"id": "audit-1", "table_name": "entities", "record_id": "ent-1"},
		}))
	})

	items, err := client.QueryAuditLogWithPagination(
		"entities",
		"update",
		"agent",
		"agent-1",
		"ent-1",
		"scope-1",
		25,
		0,
	)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "audit-1", items[0].ID)
}

func TestListAuditScopes(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/audit/scopes", r.URL.Path)
		w.Write(jsonResponse([]map[string]any{
			{"id": "scope-1", "name": "public", "agent_count": 2},
		}))
	})

	scopes, err := client.ListAuditScopes()
	require.NoError(t, err)
	require.Len(t, scopes, 1)
	assert.Equal(t, "scope-1", scopes[0].ID)
}

func TestListAuditActors(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/audit/actors", r.URL.Path)
		assert.Equal(t, "agent", r.URL.Query().Get("actor_type"))
		w.Write(jsonResponse([]map[string]any{
			{"changed_by_type": "agent", "changed_by_id": "agent-1", "action_count": 3},
		}))
	})

	actors, err := client.ListAuditActors("agent")
	require.NoError(t, err)
	require.Len(t, actors, 1)
	assert.Equal(t, "agent-1", actors[0].ActorID)
}
func TestCreateEntity(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		var body CreateEntityInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "new entity", body.Name)
		w.Write(jsonResponse(map[string]any{
			"id":   "new-id",
			"name": "new entity",
			"tags": []string{},
		}))
	})

	entity, err := client.CreateEntity(CreateEntityInput{
		Scopes: []string{"public"},
		Name:   "new entity",
		Type:   "person",
		Status: "active",
		Tags:   []string{},
	})
	require.NoError(t, err)
	assert.Equal(t, "new-id", entity.ID)
}

func TestGetPendingApprovals(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "/api/approvals/pending", r.URL.Path)
		w.Write(jsonResponse([]map[string]any{
			{"id": "ap-1", "agent_id": "ag-1", "action_type": "register", "status": "pending", "details": map[string]any{}},
		}))
	})

	approvals, err := client.GetPendingApprovals()
	require.NoError(t, err)
	assert.Len(t, approvals, 1)
	assert.Equal(t, "ap-1", approvals[0].ID)
}

func TestApproveRequest(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/approve")
		w.Write(jsonResponse(map[string]any{
			"id": "ap-1", "status": "approved", "agent_id": "ag-1",
			"action_type": "register", "details": map[string]any{},
		}))
	})

	approval, err := client.ApproveRequest("ap-1")
	require.NoError(t, err)
	assert.Equal(t, "approved", approval.Status)
}

func TestListAgents(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		w.Write(jsonResponse([]map[string]any{
			{"id": "ag-1", "name": "test-agent", "status": "active", "requires_approval": true, "scopes": []string{"public"}},
		}))
	})

	agents, err := client.ListAgents("")
	require.NoError(t, err)
	assert.Len(t, agents, 1)
	assert.Equal(t, "test-agent", agents[0].Name)
}

func TestLogin(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/api/keys/login")
		w.Write(jsonResponse(map[string]any{
			"api_key":   "nbl_newkey",
			"entity_id": "ent-1",
			"username":  "testuser",
		}))
	})

	resp, err := client.Login("testuser")
	require.NoError(t, err)
	assert.Equal(t, "nbl_newkey", resp.APIKey)
	assert.Equal(t, "testuser", resp.Username)
}

func TestHTTPError(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(404)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "NOT_FOUND",
				"message": "entity not found",
			},
		})
		w.Write(b)
	})

	_, err := client.GetEntity("nope")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_FOUND")
}

func TestBuildQuery(t *testing.T) {
	result := buildQuery("/api/entities", QueryParams{"status": "active", "type": "person"})
	assert.Contains(t, result, "/api/entities?")
	assert.Contains(t, result, "status=active")
	assert.Contains(t, result, "type=person")
}

func TestBuildQueryEmpty(t *testing.T) {
	result := buildQuery("/api/entities", nil)
	assert.Equal(t, "/api/entities", result)
}

func TestNewClientCustomTimeout(t *testing.T) {
	client := NewClient("http://example.com", "nbl_testkey", 5*time.Second)
	assert.Equal(t, 5*time.Second, client.httpClient.Timeout)
}

func TestClientConcurrentRequests(t *testing.T) {
	var count atomic.Int32
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		count.Add(1)
		w.Write(jsonResponse(map[string]any{
			"id":   "ent-1",
			"name": "test entity",
			"tags": []string{},
		}))
	}))
	t.Cleanup(srv.Close)

	client := NewClient(srv.URL, "nbl_testkey")

	const workers = 20
	var wg sync.WaitGroup
	errs := make(chan error, workers)
	for i := 0; i < workers; i++ {
		wg.Add(1)
		go func(idx int) {
			defer wg.Done()
			_, err := client.GetEntity(fmt.Sprintf("ent-%d", idx))
			errs <- err
		}(i)
	}
	wg.Wait()
	close(errs)

	for err := range errs {
		assert.NoError(t, err)
	}
	assert.Equal(t, int32(workers), count.Load())
}

func TestCreateKey(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.Write(jsonResponse(map[string]any{
			"api_key": "nbl_abc123",
			"key_id":  "key-1",
			"prefix":  "nbl_abc",
			"name":    "my-key",
		}))
	})

	resp, err := client.CreateKey("my-key")
	require.NoError(t, err)
	assert.Equal(t, "nbl_abc123", resp.APIKey)
	assert.Equal(t, "my-key", resp.Name)
}

func TestQueryJobs(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(jsonResponse([]map[string]any{
			{"id": "j-1", "title": "test job", "status": "pending"},
		}))
	})

	jobs, err := client.QueryJobs(nil)
	require.NoError(t, err)
	assert.Len(t, jobs, 1)
	assert.Equal(t, "test job", jobs[0].Title)
}

func TestRegisterAgent(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		w.Write(jsonResponse(map[string]any{
			"agent_id":            "ag-new",
			"approval_request_id": "ap-new",
			"status":              "pending_approval",
		}))
	})

	resp, err := client.RegisterAgent(RegisterAgentInput{
		Name:            "new-agent",
		RequestedScopes: []string{"public"},
	})
	require.NoError(t, err)
	assert.Equal(t, "ag-new", resp.AgentID)
	assert.Equal(t, "pending_approval", resp.Status)
}
