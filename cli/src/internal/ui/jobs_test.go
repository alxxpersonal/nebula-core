package ui

import (
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/stretchr/testify/assert"
)

func testJobsClient(t *testing.T, handler http.HandlerFunc) (*httptest.Server, *api.Client) {
	srv := httptest.NewServer(handler)
	t.Cleanup(srv.Close)
	return srv, api.NewClient(srv.URL, "test-key")
}

func TestJobsModelInit(t *testing.T) {
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"data": []map[string]any{}}
		json.NewEncoder(w).Encode(resp)
	})

	model := NewJobsModel(client)
	cmd := model.Init()
	assert.NotNil(t, cmd)
}

func TestJobsModelLoadsJobs(t *testing.T) {
	priority := "high"
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "job-1", "status": "pending", "title": "Test Job", "priority": priority, "created_at": time.Now()},
				{"id": "job-2", "status": "active", "title": "Another Job", "created_at": time.Now()},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	model := NewJobsModel(client)

	cmd := model.Init()
	msg := cmd()
	model, _ = model.Update(msg)
	model.applyJobSearch()

	assert.False(t, model.loading)
	assert.Len(t, model.items, 2)
	assert.Equal(t, "job-1", model.items[0].ID)
}

func TestJobsModelNavigationKeys(t *testing.T) {
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "job-1", "status": "pending", "title": "Job 1", "created_at": time.Now()},
				{"id": "job-2", "status": "active", "title": "Job 2", "created_at": time.Now()},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	model := NewJobsModel(client)
	cmd := model.Init()
	msg := cmd()
	model, _ = model.Update(msg)
	model.applyJobSearch()

	// Navigate down
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyDown})
	assert.Equal(t, 1, model.list.Selected())

	// Navigate up
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyUp})
	assert.Equal(t, 0, model.list.Selected())
}

func TestJobsModelEnterShowsDetail(t *testing.T) {
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "job-1", "status": "pending", "title": "Test Job", "created_at": time.Now(), "metadata": map[string]any{}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	model := NewJobsModel(client)
	cmd := model.Init()
	msg := cmd()
	model, _ = model.Update(msg)
	model.applyJobSearch()

	// Press enter
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})

	assert.NotNil(t, model.detail)
	assert.Equal(t, "job-1", model.detail.ID)
}

func TestJobsModelEscapeBackFromDetail(t *testing.T) {
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "job-1", "status": "pending", "title": "Test Job", "created_at": time.Now(), "metadata": map[string]any{}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	model := NewJobsModel(client)
	cmd := model.Init()
	msg := cmd()
	model, _ = model.Update(msg)
	model.applyJobSearch()

	// Enter detail
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.NotNil(t, model.detail)

	// Escape back
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.Nil(t, model.detail)
}

func TestJobsModelStatusChangeFlow(t *testing.T) {
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "job-1", "status": "pending", "title": "Test Job", "created_at": time.Now(), "metadata": map[string]any{}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	model := NewJobsModel(client)
	cmd := model.Init()
	msg := cmd()
	model, _ = model.Update(msg)

	// Press 's' to change status
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	assert.True(t, model.changingSt)
	assert.NotNil(t, model.detail)
}

func TestJobsModelStatusInputHandling(t *testing.T) {
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{
			"data": []map[string]any{
				{"id": "job-1", "status": "pending", "title": "Test Job", "created_at": time.Now(), "metadata": map[string]any{}},
			},
		}
		json.NewEncoder(w).Encode(resp)
	})

	model := NewJobsModel(client)
	cmd := model.Init()
	msg := cmd()
	model, _ = model.Update(msg)

	// Start status change
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})

	// Type "active"
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'t'}})
	assert.Equal(t, "act", model.statusBuf)

	// Backspace
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyBackspace})
	assert.Equal(t, "ac", model.statusBuf)

	// Escape to cancel
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, model.changingSt)
	assert.Equal(t, "", model.statusBuf)
}

func TestJobsModelRenderEmpty(t *testing.T) {
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		resp := map[string]any{"data": []map[string]any{}}
		json.NewEncoder(w).Encode(resp)
	})

	model := NewJobsModel(client)
	cmd := model.Init()
	msg := cmd()
	model, _ = model.Update(msg)

	view := model.View()
	assert.Contains(t, view, "No jobs found")
}

func TestJobsModelRenderLoading(t *testing.T) {
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {})

	model := NewJobsModel(client)
	model.loading = true

	view := model.View()
	assert.Contains(t, view, "Loading jobs")
}

func TestJobsModelCreateSubtask(t *testing.T) {
	var subtaskTitle string
	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		switch {
		case r.URL.Path == "/api/jobs":
			resp := map[string]any{
				"data": []map[string]any{
					{"id": "job-1", "status": "pending", "title": "Test Job", "created_at": time.Now()},
				},
			}
			json.NewEncoder(w).Encode(resp)
		case r.URL.Path == "/api/jobs/job-1/subtasks" && r.Method == http.MethodPost:
			var body map[string]string
			json.NewDecoder(r.Body).Decode(&body)
			subtaskTitle = body["title"]
			json.NewEncoder(w).Encode(map[string]any{"data": map[string]any{"id": "job-1"}})
		default:
			w.WriteHeader(http.StatusNotFound)
		}
	})

	model := NewJobsModel(client)
	cmd := model.Init()
	msg := cmd()
	model, _ = model.Update(msg)

	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'f'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'o'}})
	model, cmd = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	msg = cmd()
	model, _ = model.Update(msg)

	assert.Equal(t, "foo", subtaskTitle)
}

func TestJobsSearchFiltersList(t *testing.T) {
	model := NewJobsModel(nil)
	model.allItems = []api.Job{{ID: "job-1", Title: "Alpha", Status: "pending"}, {ID: "job-2", Title: "Beta", Status: "active"}}
	model.searchBuf = "al"
	model.applyJobSearch()

	assert.Len(t, model.items, 1)
	assert.Equal(t, "job-1", model.items[0].ID)
}
