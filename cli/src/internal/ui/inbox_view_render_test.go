package ui

import (
	"testing"
	"time"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
	"github.com/stretchr/testify/assert"
)

func TestInboxDetailViewRendersSummaryDiffAndNestedObjects(t *testing.T) {
	now := time.Now()
	jobID := "job-1"
	notes := "review note"

	model := NewInboxModel(nil)
	model.width = 90
	model.loading = false
	model.detail = &api.Approval{
		ID:          "ap-1",
		RequestType: "update_entity",
		Status:      "pending",
		RequestedBy: "agent:test",
		AgentName:   "test-agent",
		JobID:       &jobID,
		Notes:       &notes,
		CreatedAt:   now,
		ChangeDetails: api.JSONMap{
			"name": "Alpha",
			"tags": []any{"one", "two"},
			"metadata": map[string]any{
				"role": "founder",
				"yr":   2026,
			},
			"changes": map[string]any{
				"status": map[string]any{"from": "active", "to": "archived"},
				"metadata": map[string]any{
					"from": map[string]any{"role": "builder"},
					"to":   map[string]any{"role": "founder"},
				},
			},
		},
	}

	out := components.SanitizeText(model.View())
	assert.Contains(t, out, "Approval Request")
	assert.Contains(t, out, "update_entity")
	assert.Contains(t, out, "pending")
	assert.Contains(t, out, "Review Notes")
	assert.Contains(t, out, "review note")

	// Summary table.
	assert.Contains(t, out, "Change Details")
	assert.Contains(t, out, "name")
	assert.Contains(t, out, "Alpha")
	assert.Contains(t, out, "tags")
	assert.Contains(t, out, "one, two")

	// Diff table.
	assert.Contains(t, out, "Changes")
	assert.Contains(t, out, "status")
	assert.Contains(t, out, "active")
	assert.Contains(t, out, "archived")

	// Nested map renders as its own table.
	assert.Contains(t, out, "metadata")
	assert.Contains(t, out, "role")
	assert.Contains(t, out, "founder")
}

func TestInboxFilterInputAppliesAndClears(t *testing.T) {
	now := time.Now()
	model := NewInboxModel(nil)
	model.width = 80

	model, _ = model.Update(approvalsLoadedMsg{
		items: []api.Approval{
			{
				ID:          "ap-1",
				Status:      "pending",
				RequestType: "create_entity",
				AgentName:   "OpenAI",
				RequestedBy: "agent:openai",
				ChangeDetails: api.JSONMap{
					"name": "Alpha",
				},
				CreatedAt: now,
			},
			{
				ID:          "ap-2",
				Status:      "pending",
				RequestType: "create_entity",
				AgentName:   "Anthropic",
				RequestedBy: "agent:anthropic",
				ChangeDetails: api.JSONMap{
					"name": "Beta",
				},
				CreatedAt: now,
			},
		},
	})

	// Start filtering and type a filter.
	model.filtering = true
	for _, r := range []rune("agent:openai") {
		model, _ = model.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{r}})
	}
	assert.Equal(t, "agent:openai", model.filterBuf)
	assert.Len(t, model.filtered, 1)

	// Enter applies and exits filtering.
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEnter})
	assert.False(t, model.filtering)
	assert.Len(t, model.filtered, 1)

	// Esc clears filter and resets.
	model.filtering = true
	model, _ = model.Update(tea.KeyMsg{Type: tea.KeyEsc})
	assert.False(t, model.filtering)
	assert.Equal(t, "", model.filterBuf)
	assert.Len(t, model.filtered, 2)
}

