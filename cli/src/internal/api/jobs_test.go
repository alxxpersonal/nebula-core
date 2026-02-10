package api

import (
	"encoding/json"
	"net/http"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestGetJob(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodGet, r.Method)
		assert.Contains(t, r.URL.Path, "/api/jobs/")

		w.Write(jsonResponse(map[string]any{
			"id":     "2026Q1-0001",
			"title":  "Test job",
			"status": "pending",
		}))
	})

	job, err := client.GetJob("2026Q1-0001")
	require.NoError(t, err)
	assert.Equal(t, "2026Q1-0001", job.ID)
	assert.Equal(t, "pending", job.Status)
}

func TestCreateJob(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)

		var body CreateJobInput
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "New task", body.Title)

		w.Write(jsonResponse(map[string]any{
			"id":     "2026Q1-0002",
			"title":  body.Title,
			"status": "pending",
		}))
	})

	job, err := client.CreateJob(CreateJobInput{
		Title: "New task",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026Q1-0002", job.ID)
}

func TestUpdateJobStatus(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPatch, r.Method)
		assert.Contains(t, r.URL.Path, "/status")

		var body map[string]string
		json.NewDecoder(r.Body).Decode(&body)
		assert.Equal(t, "in-progress", body["status"])

		w.Write(jsonResponse(map[string]any{
			"id":     "2026Q1-0001",
			"title":  "Test",
			"status": "in-progress",
		}))
	})

	job, err := client.UpdateJobStatus("2026Q1-0001", "in-progress")
	require.NoError(t, err)
	assert.Equal(t, "in-progress", job.Status)
}

func TestCreateSubtask(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, http.MethodPost, r.Method)
		assert.Contains(t, r.URL.Path, "/subtasks")

		w.Write(jsonResponse(map[string]any{
			"id":     "2026Q1-0001-01",
			"title":  "Subtask",
			"status": "pending",
		}))
	})

	job, err := client.CreateSubtask("2026Q1-0001", map[string]string{
		"title": "Subtask",
	})
	require.NoError(t, err)
	assert.Equal(t, "2026Q1-0001-01", job.ID)
}

func TestQueryJobsWithFilters(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		assert.Equal(t, "pending", r.URL.Query().Get("status"))
		assert.Equal(t, "high", r.URL.Query().Get("priority"))

		w.Write(jsonResponse([]map[string]any{
			{"id": "j1", "title": "Task 1", "status": "pending"},
			{"id": "j2", "title": "Task 2", "status": "pending"},
		}))
	})

	jobs, err := client.QueryJobs(QueryParams{
		"status":   "pending",
		"priority": "high",
	})
	require.NoError(t, err)
	assert.Len(t, jobs, 2)
}

func TestUpdateJobStatusInvalid(t *testing.T) {
	_, client := testServer(t, func(w http.ResponseWriter, r *http.Request) {
		w.WriteHeader(400)
		b, _ := json.Marshal(map[string]any{
			"error": map[string]any{
				"code":    "INVALID_STATUS",
				"message": "invalid status transition",
			},
		})
		w.Write(b)
	})

	_, err := client.UpdateJobStatus("2026Q1-0001", "invalid")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "INVALID_STATUS")
}
