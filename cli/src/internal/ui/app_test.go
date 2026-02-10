package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildEntityPaletteActions(t *testing.T) {
	items := []api.Entity{{ID: "ent-123456789", Name: "Alpha", Type: "tool"}}
	actions := buildEntityPaletteActions(items, "")

	require.Len(t, actions, 1)
	assert.Equal(t, "entity:ent-123456789", actions[0].ID)
	assert.Equal(t, "Alpha", actions[0].Label)
	assert.Equal(t, "tool · ent-1234", actions[0].Desc)
}

func TestFilterPalette(t *testing.T) {
	items := []paletteAction{
		{ID: "tab:inbox", Label: "Inbox", Desc: "Approvals"},
		{ID: "tab:jobs", Label: "Jobs", Desc: "Tasks"},
	}
	filtered := filterPalette(items, "job")

	require.Len(t, filtered, 1)
	assert.Equal(t, "tab:jobs", filtered[0].ID)
}

func TestRunPaletteActionEntityJump(t *testing.T) {
	app := NewApp(nil, &config.Config{})
	app.paletteEntities = []api.Entity{{ID: "ent-1", Name: "Alpha", Type: "person"}}
	action := paletteAction{ID: "entity:ent-1", Label: "Alpha"}

	model, _ := app.runPaletteAction(action)
	updated := model.(App)

	assert.Equal(t, tabEntities, updated.tab)
	require.NotNil(t, updated.entities.detail)
	assert.Equal(t, "ent-1", updated.entities.detail.ID)
	assert.Equal(t, entitiesViewDetail, updated.entities.view)
}

func TestTabNavAllowsActionKeys(t *testing.T) {
	app := NewApp(nil, &config.Config{})
	app.tab = tabRelations
	app.tabNav = true
	app.rels.view = relsViewList

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated := model.(App)

	assert.False(t, updated.tabNav)
	assert.Equal(t, relsViewCreateSourceSearch, updated.rels.view)
}

func TestBuildEntityPaletteActionsFiltersQuery(t *testing.T) {
	items := []api.Entity{{ID: "ent-1", Name: "Alpha"}, {ID: "ent-2", Name: "Beta"}}
	actions := buildEntityPaletteActions(items, "al")

	require.Len(t, actions, 1)
	assert.Equal(t, "entity:ent-1", actions[0].ID)
}

func TestHelpToggle(t *testing.T) {
	app := NewApp(nil, &config.Config{})
	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'?'}})
	updated := model.(App)
	assert.True(t, updated.helpOpen)

	model, _ = updated.Update(tea.KeyMsg{Type: tea.KeyEsc})
	updated = model.(App)
	assert.False(t, updated.helpOpen)
}

func TestQuitConfirmWhenUnsaved(t *testing.T) {
	app := NewApp(nil, &config.Config{})
	app.know.view = knowledgeViewAdd
	app.know.fields[fieldTitle].value = "draft"

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'q'}})
	updated := model.(App)

	assert.True(t, updated.quitConfirm)
	assert.Nil(t, cmd)
}

func TestQuitConfirmAccepts(t *testing.T) {
	app := NewApp(nil, &config.Config{})
	app.quitConfirm = true

	model, cmd := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'y'}})
	updated := model.(App)

	assert.True(t, updated.quitConfirm)
	require.NotNil(t, cmd)
	msg := cmd()
	_, ok := msg.(tea.QuitMsg)
	assert.True(t, ok)
}

func TestQuitConfirmCancels(t *testing.T) {
	app := NewApp(nil, &config.Config{})
	app.quitConfirm = true

	model, _ := app.Update(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'n'}})
	updated := model.(App)

	assert.False(t, updated.quitConfirm)
}
