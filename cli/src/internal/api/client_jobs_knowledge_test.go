package api

import (
	"encoding/json"
	"net/http"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestUpdateJobEncodesBodyAndDecodesResponse(t *testing.T) {
	now := time.Now()
	title := "Updated Title"

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/jobs/job-1", r.URL.Path)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, title, body["title"])

		w.Write(jsonResponse(map[string]any{
			"id":          "job-1",
			"title":       title,
			"description": nil,
			"status":      "active",
			"priority":    nil,
			"metadata":    map[string]any{},
			"created_at":  now,
			"updated_at":  now,
		}))
	})

	out, err := client.UpdateJob("job-1", UpdateJobInput{Title: &title})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "job-1", out.ID)
	assert.Equal(t, title, out.Title)
}

func TestGetKnowledgeDecodesResponse(t *testing.T) {
	now := time.Now()

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "/api/knowledge/kn-1", r.URL.Path)

		w.Write(jsonResponse(map[string]any{
			"id":          "kn-1",
			"name":        "Doc",
			"source_type": "note",
			"status":      "active",
			"tags":        []string{"docs"},
			"metadata":    map[string]any{},
			"created_at":  now,
			"updated_at":  now,
		}))
	})

	out, err := client.GetKnowledge("kn-1")
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "kn-1", out.ID)
	assert.Equal(t, "Doc", out.Name)
}

func TestUpdateKnowledgeEncodesBodyAndDecodesResponse(t *testing.T) {
	now := time.Now()
	title := "New Title"

	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Equal(t, "/api/knowledge/kn-1", r.URL.Path)

		var body map[string]any
		require.NoError(t, json.NewDecoder(r.Body).Decode(&body))
		assert.Equal(t, title, body["title"])

		w.Write(jsonResponse(map[string]any{
			"id":          "kn-1",
			"name":        "New Title",
			"source_type": "note",
			"status":      "active",
			"tags":        []string{},
			"metadata":    map[string]any{"k": "v"},
			"created_at":  now,
			"updated_at":  now,
		}))
	})

	out, err := client.UpdateKnowledge("kn-1", UpdateKnowledgeInput{
		Title:    &title,
		Metadata: map[string]any{"k": "v"},
	})
	require.NoError(t, err)
	require.NotNil(t, out)
	assert.Equal(t, "kn-1", out.ID)
	assert.Equal(t, "New Title", out.Name)
}
