package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestFilterInputAcrossTabs(t *testing.T) {
	t.Run("context", func(t *testing.T) {
		model := NewContextModel(nil)
		model.filtering = true
		model.items = []api.Context{{ID: "ctx-1", Title: "Alpha Note"}}
		model.applyContextFilter()

		updated, _ := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		assert.Equal(t, "a", updated.filterBuf)
		assert.True(t, updated.filtering)

		updated, _ = updated.handleFilterInput(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, updated.filtering)
	})

	t.Run("entities", func(t *testing.T) {
		model := NewEntitiesModel(nil)
		model.filtering = true
		model.searchBuf = "alpha"
		updated, cmd := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "alph", updated.searchBuf)
		require.NotNil(t, cmd)

		updated, _ = updated.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updated.filtering)
	})

	t.Run("files", func(t *testing.T) {
		model := NewFilesModel(nil)
		model.filtering = true
		model.items = []api.File{{ID: "f-1", Filename: "Alpha.txt"}}
		model.applyFileSearch()

		updated, _ := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		assert.Equal(t, "a", updated.searchBuf)

		updated, _ = updated.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updated.filtering)
		assert.Equal(t, "", updated.searchBuf)
	})

	t.Run("jobs", func(t *testing.T) {
		model := NewJobsModel(nil)
		model.filtering = true
		model.items = []api.Job{{ID: "job-1", Title: "Alpha Job"}}
		model.applyJobSearch()

		updated, _ := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		assert.Equal(t, "a", updated.searchBuf)

		updated, _ = updated.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updated.filtering)
	})

	t.Run("logs", func(t *testing.T) {
		now := time.Now()
		model := NewLogsModel(nil)
		model.filtering = true
		model.items = []api.Log{{ID: "log-1", LogType: "event", Status: "active", Timestamp: now}}
		model.applyLogSearch()

		updated, _ := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("e")})
		assert.Equal(t, "e", updated.searchBuf)

		updated, _ = updated.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updated.filtering)
	})

	t.Run("protocols", func(t *testing.T) {
		now := time.Now()
		content := "rules"
		model := NewProtocolsModel(nil)
		model.filtering = true
		model.items = []api.Protocol{{ID: "p-1", Name: "alpha", Title: "Alpha", Content: &content, Status: "active", CreatedAt: now, UpdatedAt: now}}
		model.applySearch()

		updated, _ := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("a")})
		assert.Equal(t, "a", updated.searchBuf)

		updated, _ = updated.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updated.filtering)
	})

	t.Run("relationships", func(t *testing.T) {
		now := time.Now()
		model := NewRelationshipsModel(nil)
		model.filtering = true
		model.items = []api.Relationship{{
			ID:         "rel-1",
			SourceType: "entity",
			SourceID:   "ent-1",
			TargetType: "entity",
			TargetID:   "ent-2",
			Type:       "related-to",
			Status:     "active",
			CreatedAt:  now,
		}}
		model.applyListFilter()

		updated, _ := model.handleFilterInput(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune("r")})
		assert.Equal(t, "r", updated.filterBuf)

		updated, _ = updated.handleFilterInput(tea.KeyMsg{Type: tea.KeyEsc})
		assert.False(t, updated.filtering)
		assert.Equal(t, "", updated.filterBuf)
	})
}
