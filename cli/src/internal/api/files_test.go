package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetFile(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/files/")

		w.Write(jsonResponse(map[string]any{
			"id":       "file-1",
			"filename": "demo.txt",
			"file_path": "/tmp/demo.txt",
		}))
	})

	file, err := client.GetFile("file-1")
	require.NoError(t, err)
	assert.Equal(t, "file-1", file.ID)
	assert.Equal(t, "demo.txt", file.Filename)
}

func TestCreateFile(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/files", r.URL.Path)

		var body CreateFileInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "demo.txt", body.Filename)

		w.Write(jsonResponse(map[string]any{
			"id":       "file-2",
			"filename": body.Filename,
			"file_path": body.FilePath,
		}))
	})

	file, err := client.CreateFile(CreateFileInput{
		Filename: "demo.txt",
		FilePath: "/tmp/demo.txt",
	})
	require.NoError(t, err)
	assert.Equal(t, "file-2", file.ID)
}

func TestQueryFiles(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "application/pdf", r.URL.Query().Get("mime_type"))
		assert.Equal(t, "archived", r.URL.Query().Get("status_category"))
		assert.Equal(t, "tag-1", r.URL.Query().Get("tags"))

		w.Write(jsonResponse([]map[string]any{
			{"id": "file-1", "filename": "demo.pdf"},
			{"id": "file-2", "filename": "spec.pdf"},
		}))
	})

	files, err := client.QueryFiles(QueryParams{
		"mime_type":      "application/pdf",
		"status_category": "archived",
		"tags":           "tag-1",
	})
	require.NoError(t, err)
	assert.Len(t, files, 2)
}

func TestUpdateFile(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Contains(t, r.URL.Path, "/api/files/")

		var body UpdateFileInput
		json.NewDecoder(r.Body).Decode(&body)
		require.NotNil(t, body.Status)
		assert.Equal(t, "archived", *body.Status)

		w.Write(jsonResponse(map[string]any{
			"id":       "file-3",
			"filename": "demo.txt",
			"status":   "archived",
		}))
	})

	status := "archived"
	file, err := client.UpdateFile("file-3", UpdateFileInput{Status: &status})
	require.NoError(t, err)
	assert.Equal(t, "archived", file.Status)
}
