package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCreateContext(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/context", r.URL.Path)

		var body CreateContextInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "video", body.SourceType)

		w.Write(jsonResponse(map[string]any{
			"id":          "know-1",
			"title":       body.Title,
			"source_type": body.SourceType,
			"url":         body.URL,
		}))
	})

	context, err := client.CreateContext(CreateContextInput{
		Title:      "Test Video",
		SourceType: "video",
		URL:        "https://youtube.com/watch?v=test",
		Scopes:     []string{"public"},
		Tags:       []string{},
	})
	require.NoError(t, err)
	assert.Equal(t, "know-1", context.ID)
	assert.Equal(t, "video", context.SourceType)
}

func TestQueryContext(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "video", r.URL.Query().Get("source_type"))

		w.Write(jsonResponse([]map[string]any{
			{"id": "k1", "title": "Video 1", "source_type": "video", "url": "url1"},
			{"id": "k2", "title": "Video 2", "source_type": "video", "url": "url2"},
		}))
	})

	items, err := client.QueryContext(QueryParams{"source_type": "video"})
	require.NoError(t, err)
	assert.Len(t, items, 2)
	assert.Equal(t, "video", items[0].SourceType)
}

func TestLinkContext(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/link")

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "ent-1", body["entity_id"])

		w.Write(jsonResponse(map[string]any{}))
	})

	err := client.LinkContext("know-1", "ent-1")
	require.NoError(t, err)
}

func TestCreateContextMissingURL(t *testing.T) {
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

	_, err := client.CreateContext(CreateContextInput{
		Title:      "Test",
		SourceType: "video",
	})
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "VALIDATION_ERROR")
}

func TestQueryContextEmpty(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.Write(jsonResponse([]map[string]any{}))
	})

	items, err := client.QueryContext(QueryParams{})
	require.NoError(t, err)
	assert.Len(t, items, 0)
}

func TestLinkContextInvalidEntity(t *testing.T) {
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

	err := client.LinkContext("know-1", "invalid-ent")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "NOT_FOUND")
}
