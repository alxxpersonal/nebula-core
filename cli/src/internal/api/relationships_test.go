package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateRelationship(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/relationships", r.URL.Path)

		var body CreateRelationshipInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "source-1", body.SourceID)
		assert.Equal(t, "target-1", body.TargetID)

		w.Write(jsonResponse(map[string]any{
			"id":                "rel-1",
			"source_id":         body.SourceID,
			"target_id":         body.TargetID,
			"relationship_type": body.Type,
		}))
	})

	rel, err := client.CreateRelationship(CreateRelationshipInput{
		SourceID: "source-1",
		TargetID: "target-1",
		Type:     "works-on",
	})
	require.NoError(t, err)
	assert.Equal(t, "rel-1", rel.ID)
	assert.Equal(t, "source-1", rel.SourceID)
}

func TestGetRelationships(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/relationships/person/source-1")

		w.Write(jsonResponse([]map[string]any{
			{"id": "rel-1", "source_id": "source-1", "target_id": "target-1", "relationship_type": "works-on"},
			{"id": "rel-2", "source_id": "source-1", "target_id": "target-2", "relationship_type": "friends-with"},
		}))
	})

	rels, err := client.GetRelationships("person", "source-1")
	require.NoError(t, err)
	assert.Len(t, rels, 2)
	assert.Equal(t, "works-on", rels[0].Type)
}

func TestQueryRelationships(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "works-on", r.URL.Query().Get("relationship_types"))

		w.Write(jsonResponse([]map[string]any{
			{"id": "rel-1", "source_id": "s1", "target_id": "t1", "relationship_type": "works-on"},
		}))
	})

	rels, err := client.QueryRelationships(QueryParams{"relationship_types": "works-on"})
	require.NoError(t, err)
	assert.Len(t, rels, 1)
}

func TestUpdateRelationship(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Contains(t, r.URL.Path, "/api/relationships/rel-1")

		var body UpdateRelationshipInput
		json.NewDecoder(r.Body).Decode(&body)

		w.Write(jsonResponse(map[string]any{
			"id":                "rel-1",
			"source_id":         "s1",
			"target_id":         "t1",
			"relationship_type": "dating",
		}))
	})

	rel, err := client.UpdateRelationship("rel-1", UpdateRelationshipInput{
		Properties: map[string]any{"status": "active"},
	})
	require.NoError(t, err)
	assert.Equal(t, "rel-1", rel.ID)
}

func TestGetRelationshipsEmpty(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(jsonResponse([]map[string]any{}))
	})

	rels, err := client.GetRelationships("person", "nonexistent")
	require.NoError(t, err)
	assert.Len(t, rels, 0)
}

func TestCreateRelationshipInvalidType(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "INVALID_TYPE",
				"message": "invalid relationship type",
			},
		})
		w.Write(b)
	})

	_, err := client.CreateRelationship(CreateRelationshipInput{
		SourceID: "s1",
		TargetID: "t1",
		Type:     "invalid-type",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_TYPE")
}
