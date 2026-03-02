package ui

import (
	"encoding/json"
	"net/http"
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestJobsStartEditNilDetailNoMutation(t *testing.T) {
	model := NewJobsModel(nil)
	model.editFocus = jobEditFieldPriority
	model.editStatusIdx = 2
	model.editPriorityIdx = 1
	model.editDesc = "keep-me"
	model.editSaving = true

	model.startEdit()

	assert.Equal(t, jobEditFieldPriority, model.editFocus)
	assert.Equal(t, 2, model.editStatusIdx)
	assert.Equal(t, 1, model.editPriorityIdx)
	assert.Equal(t, "keep-me", model.editDesc)
	assert.True(t, model.editSaving)
}

func TestJobsHandleSubtaskInputTypingBackspaceAndEmptyEnter(t *testing.T) {
	model := NewJobsModel(nil)
	model.creatingSubtask = true

	updated, cmd := model.handleSubtaskInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'a'}})
	require.Nil(t, cmd)
	assert.Equal(t, "a", updated.subtaskBuf)

	updated, cmd = updated.handleSubtaskInput(tea.KeyMsg{Type: tea.KeySpace})
	require.Nil(t, cmd)
	assert.Equal(t, "a ", updated.subtaskBuf)

	updated, cmd = updated.handleSubtaskInput(tea.KeyMsg{Type: tea.KeyBackspace})
	require.Nil(t, cmd)
	assert.Equal(t, "a", updated.subtaskBuf)

	updated, cmd = updated.handleSubtaskInput(tea.KeyMsg{Type: tea.KeyBackspace})
	require.Nil(t, cmd)
	assert.Equal(t, "", updated.subtaskBuf)

	updated, cmd = updated.handleSubtaskInput(tea.KeyMsg{Type: tea.KeyEnter})
	require.Nil(t, cmd)
	assert.True(t, updated.creatingSubtask)
	assert.Equal(t, "", updated.subtaskBuf)
}

func TestFormatJobLineIncludesPriorityAndMetadataPreview(t *testing.T) {
	priority := "high"
	line := formatJobLine(api.Job{
		Title:    "\x1b[31mBuild\x1b[0m",
		Status:   "active",
		Priority: &priority,
		Metadata: api.JSONMap{"summary": "  ship clean output  "},
	})
	assert.Contains(t, line, "Build")
	assert.Contains(t, line, "active")
	assert.Contains(t, line, "high")
	assert.Contains(t, line, "ship clean output")
}

func TestFormatJobLineMetadataFallsBackToSortedKey(t *testing.T) {
	line := formatJobLine(api.Job{
		Title:    "Batch",
		Status:   "pending",
		Metadata: api.JSONMap{"z": "later", "a": "first"},
	})
	assert.Contains(t, line, "Batch · pending")
	assert.Contains(t, line, "first")
}

func TestJobsLoadScopeOptionsNilSuccessAndError(t *testing.T) {
	model := NewJobsModel(nil)
	assert.Nil(t, model.loadScopeOptions())

	_, client := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		if r.URL.Path != "/api/audit/scopes" {
			w.WriteHeader(http.StatusNotFound)
			return
		}
		err := json.NewEncoder(w).Encode(map[string]any{
			"data": []map[string]any{
				{"id": "scope-2", "name": "private"},
				{"id": "scope-1", "name": "public"},
			},
		})
		require.NoError(t, err)
	})

	model = NewJobsModel(client)
	cmd := model.loadScopeOptions()
	require.NotNil(t, cmd)
	msg := cmd()
	out, ok := msg.(jobsScopesLoadedMsg)
	require.True(t, ok)
	assert.Equal(t, []string{"private", "public"}, out.options)

	_, failingClient := testJobsClient(t, func(w http.ResponseWriter, r *http.Request) {
		http.Error(w, `{"error":{"code":"SCOPES_FAIL","message":"no scopes"}}`, http.StatusInternalServerError)
	})
	model = NewJobsModel(failingClient)
	cmd = model.loadScopeOptions()
	require.NotNil(t, cmd)
	msg = cmd()
	errOut, ok := msg.(errMsg)
	require.True(t, ok)
	assert.ErrorContains(t, errOut.err, "SCOPES_FAIL")
}

func TestJobsHandleLinkInputTypingBackspaceAndBack(t *testing.T) {
	model := NewJobsModel(nil)
	model.detail = &api.Job{ID: "job-1"}
	model.linkingRel = true

	updated, cmd := model.handleLinkInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
	require.Nil(t, cmd)
	assert.Equal(t, "e", updated.linkBuf)

	updated, cmd = updated.handleLinkInput(tea.KeyMsg{Type: tea.KeySpace})
	require.Nil(t, cmd)
	assert.Equal(t, "e ", updated.linkBuf)

	updated, cmd = updated.handleLinkInput(tea.KeyMsg{Type: tea.KeyBackspace})
	require.Nil(t, cmd)
	assert.Equal(t, "e", updated.linkBuf)

	updated, cmd = updated.handleLinkInput(tea.KeyMsg{Type: tea.KeyEsc})
	require.Nil(t, cmd)
	assert.False(t, updated.linkingRel)
	assert.Equal(t, "", updated.linkBuf)
}

func TestJobsToggleSelectAllNoItemsNoop(t *testing.T) {
	model := NewJobsModel(nil)
	model.selected = map[string]bool{"job-1": true}
	model.toggleSelectAll()
	assert.Equal(t, map[string]bool{"job-1": true}, model.selected)
}

func TestJobsHandleSubtaskInputBackAndNoopKey(t *testing.T) {
	model := NewJobsModel(nil)
	model.creatingSubtask = true
	model.subtaskBuf = "child"

	updated, cmd := model.handleSubtaskInput(tea.KeyMsg{Type: tea.KeyEsc})
	require.Nil(t, cmd)
	assert.False(t, updated.creatingSubtask)
	assert.Equal(t, "", updated.subtaskBuf)

	updated.creatingSubtask = true
	updated.subtaskBuf = "x"
	updated, cmd = updated.handleSubtaskInput(tea.KeyMsg{Type: tea.KeyTab})
	require.Nil(t, cmd)
	assert.True(t, updated.creatingSubtask)
	assert.Equal(t, "x", updated.subtaskBuf)
}

func TestJobsHandleLinkInputNoopKeyKeepsBuffer(t *testing.T) {
	model := NewJobsModel(nil)
	model.linkingRel = true
	model.linkBuf = "entity ent-1 owns"

	updated, cmd := model.handleLinkInput(tea.KeyMsg{Type: tea.KeyTab})
	require.Nil(t, cmd)
	assert.True(t, updated.linkingRel)
	assert.Equal(t, "entity ent-1 owns", updated.linkBuf)
}
