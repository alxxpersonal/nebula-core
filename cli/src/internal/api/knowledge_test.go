package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateKnowledge(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/knowledge", r.URL.Path)

		var body CreateKnowledgeInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "video", body.SourceType)

		w.Write(jsonResponse(map[string]any{
			"id":          "know-1",
			"title":       body.Title,
			"source_type": body.SourceType,
			"url":         body.URL,
		}))
	})

	knowledge, err := client.CreateKnowledge(CreateKnowledgeInput{
		Title:      "Test Video",
		SourceType: "video",
		URL:        "https://youtube.com/watch?v=test",
		Scopes:     []string{"public"},
		Tags:       []string{},
	})
	require.NoError(t, err)
	assert.Equal(t, "know-1", knowledge.ID)
	assert.Equal(t, "video", knowledge.SourceType)
}

func TestQueryKnowledge(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "video", r.URL.Query().Get("source_type"))

		w.Write(jsonResponse([]map[string]any{
			{"id": "k1", "title": "Video 1", "source_type": "video", "url": "url1"},
			{"id": "k2", "title": "Video 2", "source_type": "video", "url": "url2"},
		}))
	})

	items, err := client.QueryKnowledge(QueryParams{"source_type": "video"})
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "video", items[0].SourceType)
}

func TestLinkKnowledge(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/link")

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "ent-1", body["entity_id"])

		w.Write(jsonResponse(map[string]any{}))
	})

	err := client.LinkKnowledge("know-1", "ent-1")
	require.NoError(t, err)
}

func TestCreateKnowledgeMissingURL(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "VALIDATION_ERROR",
				"message": "url required for source type video",
			},
		})
		w.Write(b)
	})

	_, err := client.CreateKnowledge(CreateKnowledgeInput{
		Title:      "Test",
		SourceType: "video",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VALIDATION_ERROR")
}

func TestQueryKnowledgeEmpty(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(jsonResponse([]map[string]any{}))
	})

	items, err := client.QueryKnowledge(QueryParams{})
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestLinkKnowledgeInvalidEntity(t *testing.T) {
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

	err := client.LinkKnowledge("know-1", "invalid-ent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_FOUND")
}
