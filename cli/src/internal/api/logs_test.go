package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetLog(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/logs/")

		w.Write(jsonResponse(map[string]any{
			"id":       "log-1",
			"log_type": "event",
			"status":   "active",
		}))
	})

	log, err := client.GetLog("log-1")
	require.NoError(t, err)
	assert.Equal(t, "log-1", log.ID)
	assert.Equal(t, "event", log.LogType)
}

func TestCreateLog(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Equal(t, "/api/logs", r.URL.Path)

		var body CreateLogInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "event", body.LogType)

		w.Write(jsonResponse(map[string]any{
			"id":       "log-2",
			"log_type": body.LogType,
			"status":   "active",
		}))
	})

	log, err := client.CreateLog(CreateLogInput{
		LogType: "event",
		Status:  "active",
	})
	require.NoError(t, err)
	assert.Equal(t, "log-2", log.ID)
}

func TestQueryLogs(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Equal(t, "event", r.URL.Query().Get("log_type"))
		assert.Equal(t, "archived", r.URL.Query().Get("status_category"))
		assert.Equal(t, "tag-1", r.URL.Query().Get("tags"))

		w.Write(jsonResponse([]map[string]any{
			{"id": "log-1", "log_type": "event"},
			{"id": "log-2", "log_type": "event"},
		}))
	})

	logs, err := client.QueryLogs(QueryParams{
		"log_type":        "event",
		"status_category": "archived",
		"tags":            "tag-1",
	})
	require.NoError(t, err)
	assert.Len(t, logs, 2)
}

func TestUpdateLog(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Contains(t, r.URL.Path, "/api/logs/")

		var body UpdateLogInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "archived", *body.Status)

		w.Write(jsonResponse(map[string]any{
			"id":       "log-3",
			"log_type": "event",
			"status":   "archived",
		}))
	})

	status := "archived"
	log, err := client.UpdateLog("log-3", UpdateLogInput{Status: &status})
	require.NoError(t, err)
	assert.Equal(t, "archived", log.Status)
}
