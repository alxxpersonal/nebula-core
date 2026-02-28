package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestQueryAuditLogWithPaginationMinimalParams(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/audit", r.URL.Path)
		assert.Equal(t, "10", r.URL.Query().Get("limit"))
		assert.Equal(t, "5", r.URL.Query().Get("offset"))
		assert.Equal(t, "", r.URL.Query().Get("table"))
		assert.Equal(t, "", r.URL.Query().Get("action"))
		assert.Equal(t, "", r.URL.Query().Get("actor_type"))
		assert.Equal(t, "", r.URL.Query().Get("actor_id"))
		assert.Equal(t, "", r.URL.Query().Get("record_id"))
		assert.Equal(t, "", r.URL.Query().Get("scope_id"))
		_, err := w.Write(jsonResponse([]map[string]any{
			{"id": "audit-min", "table_name": "entities", "record_id": "ent-1"},
		}))
		require.NoError(t, err)
	})

	items, err := client.QueryAuditLogWithPagination("", "", "", "", "", "", 10, 5)
	require.NoError(t, err)
	require.Len(t, items, 1)
	assert.Equal(t, "audit-min", items[0].ID)
}

func TestListAuditActorsWithoutFilterOmitsQueryParam(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/audit/actors", r.URL.Path)
		assert.Equal(t, "", r.URL.Query().Get("actor_type"))
		_, err := w.Write(jsonResponse([]map[string]any{
			{"changed_by_type": "entity", "changed_by_id": "ent-1", "action_count": 1},
		}))
		require.NoError(t, err)
	})

	actors, err := client.ListAuditActors("")
	require.NoError(t, err)
	require.Len(t, actors, 1)
	assert.Equal(t, "ent-1", actors[0].ActorID)
}

func TestQueryAuditLogHTTPErrorPath(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusBadRequest)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "BAD_REQUEST",
				"message": "invalid audit filter",
			},
		})
	})

	_, err := client.QueryAuditLog(QueryParams{"table": "??"})
	require.Error(t, err)
	assert.ErrorContains(t, err, "BAD_REQUEST")
}

func TestListAuditScopesHTTPErrorPath(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusForbidden)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "FORBIDDEN",
				"message": "admin scope required",
			},
		})
	})

	_, err := client.ListAuditScopes()
	require.Error(t, err)
	assert.ErrorContains(t, err, "FORBIDDEN")
}

func TestListAuditActorsHTTPErrorPath(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusUnauthorized)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "UNAUTHORIZED",
				"message": "invalid api key",
			},
		})
	})

	_, err := client.ListAuditActors("agent")
	require.Error(t, err)
	assert.ErrorContains(t, err, "INVALID_API_KEY")
}

func TestHealthDecodeErrorPath(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		_, _ = w.Write([]byte("{invalid-json"))
	})

	_, err := client.Health()
	require.Error(t, err)
	assert.ErrorContains(t, err, "decode response")
}

func TestHealthHTTPErrorPath(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, _ *http.Request) {
		w.WriteHeader(http.StatusServiceUnavailable)
		_ = json.NewEncoder(w).Encode(map[string]any{
			"error": map[string]any{
				"code":    "UNAVAILABLE",
				"message": "api warming up",
			},
		})
	})

	_, err := client.Health()
	require.Error(t, err)
	assert.ErrorContains(t, err, "UNAVAILABLE")
}
