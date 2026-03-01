package ui

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/gravitrone/nebula-core/cli/internal/config"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestCountViewLinesBranches(t *testing.T) {
	assert.Equal(t, 0, countViewLines(""))
	assert.Equal(t, 0, countViewLines(" \n\t "))
	assert.Equal(t, 1, countViewLines("one"))
	assert.Equal(t, 3, countViewLines("one\ntwo\nthree"))
}

func TestTabWantsArrowsProfileStateMatrix(t *testing.T) {
	app := NewApp(nil, &config.Config{})
	app.tab = tabProfile

	assert.False(t, app.tabWantsArrows())

	app.profile.creating = true
	assert.True(t, app.tabWantsArrows())
	app.profile.creating = false

	app.profile.createdKey = "nbl_created"
	assert.True(t, app.tabWantsArrows())
	app.profile.createdKey = ""

	app.profile.taxPromptMode = taxPromptEditName
	assert.True(t, app.tabWantsArrows())
}

func TestTabWantsArrowsFullSwitchCoverage(t *testing.T) {
	app := NewApp(nil, &config.Config{})

	app.tab = tabInbox
	assert.False(t, app.tabWantsArrows())
	app.inbox.rejecting = true
	assert.True(t, app.tabWantsArrows())

	app.tab = tabEntities
	app.entities.view = entitiesViewList
	assert.False(t, app.tabWantsArrows())
	app.entities.view = entitiesViewDetail
	assert.True(t, app.tabWantsArrows())

	app.tab = tabRelations
	app.rels.view = relsViewList
	assert.False(t, app.tabWantsArrows())
	app.rels.view = relsViewCreateSourceSearch
	assert.True(t, app.tabWantsArrows())

	app.tab = tabJobs
	app.jobs.view = jobsViewList
	app.jobs.detail = nil
	app.jobs.changingSt = false
	assert.False(t, app.tabWantsArrows())
	app.jobs.changingSt = true
	assert.True(t, app.tabWantsArrows())

	app.tab = tabLogs
	app.logs.view = logsViewList
	assert.False(t, app.tabWantsArrows())
	app.logs.view = logsViewDetail
	assert.True(t, app.tabWantsArrows())

	app.tab = tabFiles
	app.files.view = filesViewList
	assert.False(t, app.tabWantsArrows())
	app.files.view = filesViewDetail
	assert.True(t, app.tabWantsArrows())

	app.tab = tabProfile
	app.profile.creating = false
	app.profile.createdKey = ""
	app.profile.taxPromptMode = taxPromptNone
	assert.False(t, app.tabWantsArrows())

	app.tab = 999
	assert.False(t, app.tabWantsArrows())
}

func TestFinishQuickstartSuccessAndSkippedBranches(t *testing.T) {
	t.Setenv("HOME", t.TempDir())

	cfg := &config.Config{APIKey: "key", Username: "alxx", QuickstartPending: true}
	app := NewApp(nil, cfg)
	app.quickstartOpen = true
	app.quickstartStep = 2

	model, cmd := app.finishQuickstart(false)
	updated := model.(App)
	assert.False(t, updated.quickstartOpen)
	assert.Equal(t, 0, updated.quickstartStep)
	assert.False(t, updated.config.QuickstartPending)
	require.NotNil(t, cmd)
	require.NotNil(t, updated.toast)
	assert.Equal(t, "success", updated.toast.level)
	assert.Equal(t, "Quickstart complete.", updated.toast.text)

	model, cmd = updated.finishQuickstart(true)
	updated = model.(App)
	require.NotNil(t, cmd)
	require.NotNil(t, updated.toast)
	assert.Equal(t, "info", updated.toast.level)
	assert.Equal(t, "Quickstart skipped.", updated.toast.text)
}

func TestFinishQuickstartSaveErrorBranch(t *testing.T) {
	tmp := t.TempDir()
	homeAsFile := filepath.Join(tmp, "home-file")
	require.NoError(t, os.WriteFile(homeAsFile, []byte("x"), 0o644))
	t.Setenv("HOME", homeAsFile)

	cfg := &config.Config{APIKey: "key", Username: "alxx", QuickstartPending: true}
	app := NewApp(nil, cfg)
	app.quickstartOpen = true
	app.quickstartStep = 1

	model, cmd := app.finishQuickstart(false)
	updated := model.(App)
	assert.False(t, updated.quickstartOpen)
	assert.Equal(t, 0, updated.quickstartStep)
	assert.NotEmpty(t, updated.err)
	assert.Contains(t, updated.err, "save config")
	assert.Nil(t, cmd)
}
