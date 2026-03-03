package ui

import (
	"testing"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestEntitiesHandleAddKeysBranchMatrix(t *testing.T) {
	t.Run("saving and saved short-circuits", func(t *testing.T) {
		model := NewEntitiesModel(nil)
		model.addSaving = true
		next, cmd := model.handleAddKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'x'}})
		assert.Nil(t, cmd)
		assert.True(t, next.addSaving)

		model = NewEntitiesModel(nil)
		model.addSaved = true
		model.addFields[addFieldName].value = "keep"
		next, cmd = model.handleAddKeys(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Nil(t, cmd)
		assert.False(t, next.addSaved)
		assert.Equal(t, "", next.addFields[addFieldName].value)
	})

	t.Run("mode focus delegates to mode handler", func(t *testing.T) {
		model := NewEntitiesModel(nil)
		model.view = entitiesViewAdd
		model.modeFocus = true
		next, cmd := model.handleAddKeys(tea.KeyMsg{Type: tea.KeyRight})
		assert.Nil(t, cmd)
		assert.Equal(t, entitiesViewList, next.view)
		assert.False(t, next.modeFocus)
	})

	t.Run("status and scope selectors cycle and toggle", func(t *testing.T) {
		model := NewEntitiesModel(nil)
		model.addFocus = addFieldStatus
		model.addStatusIdx = 0

		next, _ := model.handleAddKeys(tea.KeyMsg{Type: tea.KeyLeft})
		assert.Equal(t, len(entityStatusOptions)-1, next.addStatusIdx)

		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyRight})
		assert.Equal(t, 0, next.addStatusIdx)

		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeySpace})
		assert.Equal(t, 1, next.addStatusIdx)

		next.addFocus = addFieldScopes
		next.scopeOptions = []string{"public", "private"}
		next.addScopeSelecting = true
		next.addScopeIdx = 0

		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyLeft})
		assert.Equal(t, 1, next.addScopeIdx)

		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyRight})
		assert.Equal(t, 0, next.addScopeIdx)

		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeySpace})
		assert.Equal(t, []string{"public"}, next.addScopes)

		next.scopeOptions = nil
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyLeft})
		assert.Equal(t, 0, next.addScopeIdx)

		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyEnter})
		assert.False(t, next.addScopeSelecting)
	})

	t.Run("navigation, save, delete and text input branches", func(t *testing.T) {
		model := NewEntitiesModel(nil)
		model.scopeOptions = []string{"public"}

		// Up from first field enters mode focus.
		model.addFocus = 0
		next, cmd := model.handleAddKeys(tea.KeyMsg{Type: tea.KeyUp})
		assert.Nil(t, cmd)
		assert.True(t, next.modeFocus)

		// Ctrl+S runs save validation path.
		next.modeFocus = false
		next.addFocus = addFieldName
		next, cmd = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyCtrlS})
		assert.Nil(t, cmd)
		assert.Equal(t, "Name is required", next.errText)

		// Tag input branches.
		next.addFocus = addFieldTags
		next.addTagBuf = "ab"
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "a", next.addTagBuf)
		next.addTagBuf = ""
		next.addTags = []string{"alpha", "beta"}
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, []string{"alpha"}, next.addTags)
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'z'}})
		assert.Equal(t, "z", next.addTagBuf)
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyEnter})
		assert.Equal(t, []string{"alpha", "z"}, next.addTags)
		assert.Equal(t, "", next.addTagBuf)

		// Scope delete and scope-select activation.
		next.addFocus = addFieldScopes
		next.addScopes = []string{"public"}
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Empty(t, next.addScopes)
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeySpace})
		assert.True(t, next.addScopeSelecting)

		// Metadata delete/activate branches.
		next.addScopeSelecting = false
		next.addFocus = addFieldMetadata
		next.addMeta.Buffer = "abc"
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "ab", next.addMeta.Buffer)
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyEnter})
		assert.True(t, next.addMeta.Active)

		// Default field input/delete branches.
		next.addFocus = addFieldType
		next.addFields[addFieldType].value = "pers"
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyBackspace})
		assert.Equal(t, "per", next.addFields[addFieldType].value)
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'s'}})
		assert.Equal(t, "pers", next.addFields[addFieldType].value)

		// Esc resets the whole form.
		next.addFields[addFieldName].value = "Alpha"
		next, _ = next.handleAddKeys(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Equal(t, "", next.addFields[addFieldName].value)
		assert.Equal(t, 0, next.addFocus)
	})
}

func TestEntitiesHandleDetailKeysBranchMatrix(t *testing.T) {
	restoreClipboard := copyEntityMetadataClipboard
	defer func() { copyEntityMetadataClipboard = restoreClipboard }()
	copyEntityMetadataClipboard = func(s string) error { return nil }

	model := NewEntitiesModel(nil)
	model.width = 90
	model.height = 30
	model.view = entitiesViewDetail
	model.detail = &api.Entity{
		ID:   "ent-1",
		Name: "Alpha",
		Type: "person",
		Metadata: map[string]any{
			"profile": map[string]any{
				"name": "Alpha",
				"bio":  "line1\nline2\nline3\nline4\nline5\nline6\nline7",
			},
		},
	}
	model.metaExpanded = true
	model.syncDetailMetadataRows()
	require.Greater(t, len(model.metaRows), 0)
	require.NotNil(t, model.metaList)

	t.Run("meta inspect navigation and copy", func(t *testing.T) {
		model.metaInspect = true
		model.metaInspectI = 0
		model.metaInspectO = 0

		next, cmd := model.handleDetailKeys(tea.KeyMsg{Type: tea.KeyDown})
		assert.Nil(t, cmd)
		assert.GreaterOrEqual(t, next.metaInspectO, 0)

		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyUp})
		assert.Nil(t, cmd)
		assert.Equal(t, 0, next.metaInspectO)

		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyEnter})
		require.NotNil(t, cmd)
		msg := cmd()
		copied, ok := msg.(entityMetadataCopiedMsg)
		require.True(t, ok)
		assert.Equal(t, 1, copied.count)

		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Nil(t, cmd)
		assert.False(t, next.metaInspect)
	})

	t.Run("expanded metadata selection and copy flows", func(t *testing.T) {
		next := model
		next.metaInspect = false
		next.metaExpanded = true
		next.metaSelected = map[int]bool{}
		next.metaSelectMode = false
		next.metaList = components.NewList(8)
		syncMetadataList(next.metaList, next.metaRows, metadataPanelPageSize(true))

		next, cmd := next.handleDetailKeys(tea.KeyMsg{Type: tea.KeySpace})
		assert.Nil(t, cmd)
		assert.True(t, next.metaSelectMode)
		assert.NotEmpty(t, next.metaSelected)

		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		assert.Nil(t, cmd)
		assert.Equal(t, len(next.metaRows), len(next.metaSelected))

		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'c'}})
		require.NotNil(t, cmd)
		msg := cmd()
		copied, ok := msg.(entityMetadataCopiedMsg)
		require.True(t, ok)
		assert.Equal(t, len(next.metaRows), copied.count)

		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Nil(t, cmd)
		assert.False(t, next.metaSelectMode)
		assert.Empty(t, next.metaSelected)
	})

	t.Run("selection mode exits when toggles clear all rows", func(t *testing.T) {
		next := model
		next.metaInspect = false
		next.metaExpanded = true
		next.syncDetailMetadataRows()
		require.Greater(t, len(next.metaRows), 0)
		next.metaList = components.NewList(8)
		syncMetadataList(next.metaList, next.metaRows, metadataPanelPageSize(true))
		next.metaList.Cursor = 0

		next.metaSelectMode = true
		next.metaSelected = map[int]bool{0: true}
		afterSpace, cmd := next.handleDetailKeys(tea.KeyMsg{Type: tea.KeySpace})
		assert.Nil(t, cmd)
		assert.False(t, afterSpace.metaSelectMode)
		assert.Empty(t, afterSpace.metaSelected)

		afterSpace.metaSelectMode = true
		afterSpace.metaSelected = make(map[int]bool, len(afterSpace.metaRows))
		for i := range afterSpace.metaRows {
			afterSpace.metaSelected[i] = true
		}
		afterBulk, cmd := afterSpace.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'b'}})
		assert.Nil(t, cmd)
		assert.False(t, afterBulk.metaSelectMode)
		assert.Empty(t, afterBulk.metaSelected)
	})

	t.Run("expanded metadata list arrows and inspect open", func(t *testing.T) {
		next := model
		next.metaInspect = false
		next.metaExpanded = true
		next.metaSelectMode = true
		next.metaSelected = map[int]bool{}
		next.syncDetailMetadataRows()

		next, cmd := next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyDown})
		assert.Nil(t, cmd)
		assert.GreaterOrEqual(t, next.metaList.Selected(), 0)

		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyUp})
		assert.Nil(t, cmd)
		assert.GreaterOrEqual(t, next.metaList.Selected(), 0)

		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyEnter})
		assert.Nil(t, cmd)
		assert.True(t, next.metaInspect)

		next.metaInspect = false
		next.metaSelectMode = true
		next.metaSelected = map[int]bool{}
		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Nil(t, cmd)
		assert.False(t, next.metaSelectMode)
		assert.Empty(t, next.metaSelected)
	})

	t.Run("detail-level shortcuts route to expected views", func(t *testing.T) {
		next := model
		next.metaInspect = true
		next.metaExpanded = true
		next.metaSelected = map[int]bool{0: true}
		next.metaSelectMode = true

		next, cmd := next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'m'}})
		assert.Nil(t, cmd)
		assert.False(t, next.metaExpanded)
		assert.False(t, next.metaInspect)
		assert.Empty(t, next.metaSelected)

		next.metaExpanded = false
		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'e'}})
		assert.Nil(t, cmd)
		assert.Equal(t, entitiesViewEdit, next.view)

		next.view = entitiesViewDetail
		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'d'}})
		assert.Nil(t, cmd)
		assert.Equal(t, entitiesViewConfirm, next.view)
		assert.Equal(t, "entity-archive", next.confirmKind)

		next.view = entitiesViewDetail
		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'r'}})
		assert.Equal(t, entitiesViewRelationships, next.view)
		assert.True(t, next.relLoading)
		require.NotNil(t, cmd)

		next.view = entitiesViewDetail
		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyRunes, Runes: []rune{'h'}})
		assert.Equal(t, entitiesViewHistory, next.view)
		assert.True(t, next.historyLoading)
		require.NotNil(t, cmd)

		next.view = entitiesViewDetail
		next, cmd = next.handleDetailKeys(tea.KeyMsg{Type: tea.KeyEsc})
		assert.Nil(t, cmd)
		assert.Equal(t, entitiesViewList, next.view)
		assert.Nil(t, next.detail)
	})
}
