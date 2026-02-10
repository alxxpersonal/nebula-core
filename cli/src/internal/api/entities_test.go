package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateEntity(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Contains(t, r.URL.Path, "/api/entities/")

		var body UpdateEntityInput
		json.NewDecoder(r.Body).Decode(&body)

		w.Write(jsonResponse(map[string]any{
			"id":   "ent-1",
			"name": body.Name,
			"tags": body.Tags,
		}))
	})

	entity, err := client.UpdateEntity("ent-1", UpdateEntityInput{
		Name: stringPtr("updated name"),
		Tags: stringSlicePtr([]string{"new-tag"}),
	})
	require.NoError(t, err)
	assert.Equal(t, "ent-1", entity.ID)
	assert.Equal(t, "updated name", entity.Name)
}

func TestSearchEntities(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/entities/search", r.URL.Path)

		var body map[string]any
		json.NewDecoder(r.Body).Decode(&body)
		assert.NotNil(t, body["metadata_query"])

		w.Write(jsonResponse([]map[string]any{
			{"id": "1", "name": "match1", "tags": []string{}},
			{"id": "2", "name": "match2", "tags": []string{}},
		}))
	})

	results, err := client.SearchEntities(map[string]any{"role": "professor"})
	require.NoError(t, err)
	assert.Len(t, results, 2)
}

func TestSearchEntitiesEmpty(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(jsonResponse([]map[string]any{}))
	})

	results, err := client.SearchEntities(map[string]any{})
	require.NoError(t, err)
	assert.Len(t, results, 0)
}

func TestGetEntityNotFound(t *testing.T) {
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

	_, err := client.GetEntity("nonexistent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_FOUND")
}

func TestQueryEntitiesWithMultipleParams(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "active", r.URL.Query().Get("status"))
		assert.Equal(t, "person", r.URL.Query().Get("type"))
		w.Write(jsonResponse([]map[string]any{
			{"id": "1", "name": "test", "tags": []string{}},
		}))
	})

	entities, err := client.QueryEntities(QueryParams{
		"status": "active",
		"type":   "person",
	})
	require.NoError(t, err)
	assert.Len(t, entities, 1)
}

func TestCreateEntityMissingFields(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "missing required field: name",
			},
		})
		w.Write(b)
	})

	_, err := client.CreateEntity(CreateEntityInput{
		Scopes: []string{"public"},
		Type:   "person",
		Status: "active",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VALIDATION_ERROR")
}

func TestGetEntityHistory(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/entities/ent-1/history")
		assert.Equal(t, "50", r.URL.Query().Get("limit"))
		assert.Equal(t, "0", r.URL.Query().Get("offset"))

		now := time.Now().UTC().Format(time.RFC3339)
		w.Write(jsonResponse([]map[string]any{
			{
				"id":             "audit-1",
				"table_name":     "entities",
				"record_id":      "ent-1",
				"action":         "update",
				"changed_fields": []string{"tags"},
				"changed_at":     now,
			},
		}))
	})

	rows, err := client.GetEntityHistory("ent-1", 50, 0)
	require.NoError(t, err)
	if assert.Len(t, rows, 1) {
		assert.Equal(t, "audit-1", rows[0].ID)
		assert.Equal(t, "update", rows[0].Action)
	}
}

func TestRevertEntity(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/api/entities/ent-1/revert")

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "audit-1", body["audit_id"])

		w.Write(jsonResponse(map[string]any{
			"id":   "ent-1",
			"name": "Restored",
			"tags": []string{},
		}))
	})

	entity, err := client.RevertEntity("ent-1", "audit-1")
	require.NoError(t, err)
	assert.Equal(t, "ent-1", entity.ID)
	assert.Equal(t, "Restored", entity.Name)
}

func stringPtr(s string) *string {
	return &s
}

func stringSlicePtr(v []string) *[]string {
	return &v
}
