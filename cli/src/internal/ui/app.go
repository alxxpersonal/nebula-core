package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

// --- Tab Constants ---

const (
	tabInbox     = 0
	tabEntities  = 1
	tabRelations = 2
	tabKnow      = 3
	tabJobs      = 4
	tabLogs      = 5
	tabFiles     = 6
	tabProtocols = 7
	tabHistory   = 8
	tabSearch    = 9
	tabProfile   = 10
	tabCount     = 11
)

var tabNames = []string{"Inbox", "Entities", "Relationships", "Knowledge", "Jobs", "Logs", "Files", "Protocols", "History", "Search", "Profile"}

// --- Messages ---

type errMsg struct{ err error }
type paletteEntitiesLoadedMsg struct {
	query string
	items []api.Entity
}

type paletteAction struct {
	ID    string
	Label string
	Desc  string
}

// --- App Model ---

// App is the root TUI model that routes between tabs.
type App struct {
	client      *api.Client
	config      *config.Config
	tab         int
	tabNav      bool
	width       int
	height      int
	err         string
	helpOpen    bool
	quitConfirm bool

	paletteOpen          bool
	paletteQuery         string
	paletteIndex         int
	paletteActions       []paletteAction
	paletteFiltered      []paletteAction
	paletteEntityQuery   string
	paletteEntityLoading bool
	paletteEntities      []api.Entity

	importExportOpen bool

	inbox     InboxModel
	entities  EntitiesModel
	rels      RelationshipsModel
	know      KnowledgeModel
	jobs      JobsModel
	logs      LogsModel
	files     FilesModel
	protocols ProtocolsModel
	history   HistoryModel
	search    SearchModel
	profile   ProfileModel
	impex     ImportExportModel
}

// NewApp creates the root application model.
func NewApp(client *api.Client, cfg *config.Config) App {
	inbox := NewInboxModel(client)
	inbox.confirmBulk = true
	return App{
		client:         client,
		config:         cfg,
		tab:            tabInbox,
		tabNav:         true,
		paletteActions: defaultPaletteActions(),
		inbox:          inbox,
		entities:       NewEntitiesModel(client),
		rels:           NewRelationshipsModel(client),
		know:           NewKnowledgeModel(client),
		jobs:           NewJobsModel(client),
		logs:           NewLogsModel(client),
		files:          NewFilesModel(client),
		protocols:      NewProtocolsModel(client),
		history:        NewHistoryModel(client),
		search:         NewSearchModel(client),
		profile:        NewProfileModel(client, cfg),
		impex:          NewImportExportModel(client),
	}
}

func (a App) Init() tea.Cmd {
	return a.inbox.Init()
}

func (a App) Update(msg tea.Msg) (tea.Model, tea.Cmd) {
	switch msg := msg.(type) {
	case tea.WindowSizeMsg:
		a.width = msg.Width
		a.height = msg.Height
		a.inbox.width = msg.Width
		a.inbox.height = msg.Height
		a.entities.width = msg.Width
		a.entities.height = msg.Height
		a.rels.width = msg.Width
		a.rels.height = msg.Height
		a.know.width = msg.Width
		a.know.height = msg.Height
		a.jobs.width = msg.Width
		a.jobs.height = msg.Height
		a.logs.width = msg.Width
		a.logs.height = msg.Height
		a.files.width = msg.Width
		a.files.height = msg.Height
		a.protocols.width = msg.Width
		a.protocols.height = msg.Height
		a.history.width = msg.Width
		a.history.height = msg.Height
		a.search.width = msg.Width
		a.search.height = msg.Height
		a.profile.width = msg.Width
		a.profile.height = msg.Height
		a.impex.width = msg.Width
		a.impex.height = msg.Height
		return a, nil

	case errMsg:
		a.err = msg.err.Error()
		return a, nil
	case importExportDoneMsg:
		if a.importExportOpen {
			var cmd tea.Cmd
			a.impex, cmd = a.impex.Update(msg)
			if a.impex.closed {
				a.importExportOpen = false
			}
			return a, cmd
		}
	case importExportErrorMsg:
		if a.importExportOpen {
			var cmd tea.Cmd
			a.impex, cmd = a.impex.Update(msg)
			if a.impex.closed {
				a.importExportOpen = false
			}
			return a, cmd
		}
	case paletteEntitiesLoadedMsg:
		if msg.query != a.paletteEntityQuery {
			return a, nil
		}
		a.paletteEntityLoading = false
		a.paletteEntities = msg.items
		a.paletteFiltered = buildEntityPaletteActions(msg.items, a.paletteEntityQuery)
		a.paletteIndex = 0
		return a, nil
	case searchSelectionMsg:
		return a.applySearchSelection(msg)

	case tea.KeyMsg:
		if a.err != "" {
			a.err = ""
		}
		if a.importExportOpen {
			var cmd tea.Cmd
			a.impex, cmd = a.impex.Update(msg)
			if a.impex.closed {
				a.importExportOpen = false
			}
			return a, cmd
		}
		if a.quitConfirm {
			switch {
			case isKey(msg, "y"):
				return a, tea.Quit
			case isKey(msg, "n"), isBack(msg):
				a.quitConfirm = false
			}
			return a, nil
		}
		if a.helpOpen {
			if isBack(msg) || isKey(msg, "?") {
				a.helpOpen = false
			}
			return a, nil
		}
		if a.paletteOpen {
			return a.handlePaletteKeys(msg)
		}

		// Global keys
		if isKey(msg, "?") {
			a.helpOpen = true
			return a, nil
		}
		if isQuit(msg) {
			if a.hasUnsaved() {
				a.quitConfirm = true
				return a, nil
			}
			return a, tea.Quit
		}

		// Command palette
		if isKey(msg, "/") {
			a.openPalette()
			return a, nil
		}

		if idx, ok := tabIndexForKey(msg.String()); ok {
			app, cmd := a.switchTab(idx)
			return app, cmd
		}

		// Arrow tab navigation until user enters content with Down
		if a.tabNav {
			if isKey(msg, "left") {
				newTab := (a.tab - 1 + tabCount) % tabCount
				app, cmd := a.switchTab(newTab)
				return app, cmd
			}
			if isKey(msg, "right") {
				newTab := (a.tab + 1) % tabCount
				app, cmd := a.switchTab(newTab)
				return app, cmd
			}
			if isDown(msg) {
				a.tabNav = false
				return a, nil
			}

			// Any other key exits tab nav so the active tab can handle it.
			a.tabNav = false
		} else {
			if isUp(msg) && a.canExitToTabNav() {
				a.tabNav = true
				return a, nil
			}
		}
	}

	// Delegate to active tab
	var cmd tea.Cmd
	switch a.tab {
	case tabInbox:
		a.inbox, cmd = a.inbox.Update(msg)
	case tabEntities:
		a.entities, cmd = a.entities.Update(msg)
	case tabRelations:
		a.rels, cmd = a.rels.Update(msg)
	case tabKnow:
		a.know, cmd = a.know.Update(msg)
	case tabJobs:
		a.jobs, cmd = a.jobs.Update(msg)
	case tabLogs:
		a.logs, cmd = a.logs.Update(msg)
	case tabFiles:
		a.files, cmd = a.files.Update(msg)
	case tabProtocols:
		a.protocols, cmd = a.protocols.Update(msg)
	case tabHistory:
		a.history, cmd = a.history.Update(msg)
	case tabSearch:
		a.search, cmd = a.search.Update(msg)
	case tabProfile:
		a.profile, cmd = a.profile.Update(msg)
	}
	return a, cmd
}

func (a App) View() string {
	banner := centerBlockUniform(RenderBanner(), a.width)
	tabs := centerBlockUniform(a.renderTabs(), a.width)

	var content string
	switch a.tab {
	case tabInbox:
		content = a.inbox.View()
	case tabEntities:
		content = a.entities.View()
	case tabRelations:
		content = a.rels.View()
	case tabKnow:
		content = a.know.View()
	case tabJobs:
		content = a.jobs.View()
	case tabLogs:
		content = a.logs.View()
	case tabFiles:
		content = a.files.View()
	case tabProtocols:
		content = a.protocols.View()
	case tabHistory:
		content = a.history.View()
	case tabSearch:
		content = a.search.View()
	case tabProfile:
		content = a.profile.View()
	}
	content = centerBlockUniform(content, a.width)

	errorBox := ""
	if a.err != "" {
		errorBox = "\n\n" + centerBlockUniform(components.ErrorBox("Error", a.err, a.width), a.width)
	}

	hints := components.StatusBar(a.statusHints(), a.width)

	if a.quitConfirm {
		content = a.renderQuitConfirm()
		content = centerBlockUniform(content, a.width)
	} else if a.helpOpen {
		content = a.renderHelp()
		content = centerBlockUniform(content, a.width)
	} else if a.paletteOpen {
		content = a.renderPalette()
		content = centerBlockUniform(content, a.width)
	} else if a.importExportOpen {
		content = a.impex.View()
		content = centerBlockUniform(content, a.width)
	}

	return fmt.Sprintf("%s\n%s\n\n%s\n\n\n%s%s", banner, tabs, content, hints, errorBox)
}

func (a *App) switchTab(newTab int) (App, tea.Cmd) {
	oldTab := a.tab
	a.tab = newTab
	if oldTab != newTab {
		return *a, a.initTab(newTab)
	}
	return *a, nil
}

// tabWantsArrows returns true when the active tab needs left/right arrow keys.
func (a App) tabWantsArrows() bool {
	switch a.tab {
	case tabKnow:
		return true // type selector uses left/right
	case tabInbox:
		return a.inbox.detail != nil || a.inbox.rejecting
	case tabEntities:
		return a.entities.view != entitiesViewList
	case tabRelations:
		return a.rels.view != relsViewList
	case tabJobs:
		return a.jobs.detail != nil || a.jobs.changingSt
	case tabLogs:
		return a.logs.view != logsViewList
	case tabFiles:
		return a.files.view != filesViewList
	case tabSearch:
		return false
	case tabProfile:
		return a.profile.creating || a.profile.createdKey != ""
	}
	return false
}

func (a App) renderTabs() string {
	segments := make([]string, 0, len(tabNames))
	for i, name := range tabNames {
		label := name
		if i == a.tab {
			segments = append(segments, TabActiveStyle.Render(label))
		} else {
			segments = append(segments, TabInactiveStyle.Render(label))
		}
	}
	return lipgloss.JoinHorizontal(lipgloss.Top, segments...)
}

func (a App) initTab(tab int) tea.Cmd {
	switch tab {
	case tabInbox:
		return a.inbox.Init()
	case tabEntities:
		return a.entities.Init()
	case tabRelations:
		return a.rels.Init()
	case tabKnow:
		return a.know.Init()
	case tabJobs:
		return a.jobs.Init()
	case tabLogs:
		return a.logs.Init()
	case tabFiles:
		return a.files.Init()
	case tabProtocols:
		return a.protocols.Init()
	case tabHistory:
		return a.history.Init()
	case tabSearch:
		return a.search.Init()
	case tabProfile:
		return a.profile.Init()
	}
	return nil
}

func (a App) statusHints() []string {
	if a.quitConfirm {
		return []string{
			components.Hint("y", "Confirm"),
			components.Hint("n", "Cancel"),
		}
	}
	if a.helpOpen {
		return []string{
			components.Hint("esc", "Back"),
		}
	}
	return a.statusHintsForTab()
}

func (a App) statusHintsForTab() []string {
	base := []string{
		components.Hint("1-9/0/-", "Tabs"),
		components.Hint("/", "Command"),
		components.Hint("?", "Help"),
		components.Hint("q", "Quit"),
	}

	switch a.tab {
	case tabInbox:
		if a.inbox.filtering {
			return append(base,
				components.Hint("enter", "Apply"),
				components.Hint("esc", "Clear"),
			)
		}
		if a.inbox.rejecting {
			return append(base,
				components.Hint("enter", "Submit"),
				components.Hint("esc", "Cancel"),
			)
		}
		if a.inbox.detail != nil {
			return append(base,
				components.Hint("a", "Approve"),
				components.Hint("r", "Reject"),
				components.Hint("esc", "Back"),
			)
		}
		return append(base,
			components.Hint("↑/↓", "Scroll"),
			components.Hint("space", "Select"),
			components.Hint("b", "Select All"),
			components.Hint("A", "Approve All"),
			components.Hint("a", "Approve"),
			components.Hint("r", "Reject"),
			components.Hint("enter", "Details"),
			components.Hint("f", "Filter"),
		)
	case tabEntities:
		if a.entities.bulkPrompt != "" {
			return append(base,
				components.Hint("enter", "Apply"),
				components.Hint("esc", "Cancel"),
			)
		}
		switch a.entities.view {
		case entitiesViewDetail:
			return append(base,
				components.Hint("e", "Edit"),
				components.Hint("h", "History"),
				components.Hint("r", "Relationships"),
				components.Hint("d", "Archive"),
				components.Hint("esc", "Back"),
			)
		case entitiesViewEdit:
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("space", "Select"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Cancel"),
			)
		case entitiesViewRelationships:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("n", "New"),
				components.Hint("e", "Edit"),
				components.Hint("d", "Archive"),
				components.Hint("esc", "Back"),
			)
		case entitiesViewRelateSearch:
			return append(base,
				components.Hint("enter", "Search"),
				components.Hint("esc", "Back"),
			)
		case entitiesViewRelateSelect:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("enter", "Select"),
				components.Hint("esc", "Back"),
			)
		case entitiesViewRelateType:
			return append(base,
				components.Hint("enter", "Create"),
				components.Hint("esc", "Back"),
			)
		case entitiesViewRelEdit:
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("space", "Select"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Cancel"),
			)
		case entitiesViewAdd:
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("space", "Select"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Back"),
			)
		case entitiesViewHistory:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("enter", "Revert"),
				components.Hint("esc", "Back"),
			)
		case entitiesViewSearch:
			return append(base,
				components.Hint("enter", "Search"),
				components.Hint("esc", "Back"),
			)
		case entitiesViewConfirm:
			return append(base,
				components.Hint("y", "Confirm"),
				components.Hint("n", "Cancel"),
			)
		default:
			hints := append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("tab", "Complete"),
				components.Hint("enter", "Details"),
			)
			if strings.TrimSpace(a.entities.searchBuf) == "" {
				hints = append(hints, components.Hint("space", "Select"))
			}
			if a.entities.bulkCount() > 0 {
				hints = append(hints,
					components.Hint("t", "Tags"),
					components.Hint("p", "Scopes"),
					components.Hint("c", "Clear"),
				)
			}
			return hints
		}
	case tabRelations:
		switch a.rels.view {
		case relsViewDetail:
			return append(base,
				components.Hint("e", "Edit"),
				components.Hint("d", "Archive"),
				components.Hint("esc", "Back"),
			)
		case relsViewEdit:
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("space", "Select"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Cancel"),
			)
		case relsViewConfirm:
			return append(base,
				components.Hint("y", "Confirm"),
				components.Hint("n", "Cancel"),
			)
		case relsViewCreateSourceSearch, relsViewCreateTargetSearch, relsViewCreateSourceSelect, relsViewCreateTargetSelect:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("enter", "Select"),
				components.Hint("esc", "Back"),
			)
		case relsViewCreateType:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("enter", "Create"),
				components.Hint("esc", "Back"),
			)
		default:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("enter", "Details"),
				components.Hint("n", "New"),
			)
		}
	case tabKnow:
		if a.know.linkSearching {
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("enter", "Select"),
				components.Hint("esc", "Cancel"),
			)
		}
		switch a.know.view {
		case knowledgeViewList:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("enter", "Details"),
				components.Hint("esc", "Back"),
			)
		case knowledgeViewDetail:
			return append(base,
				components.Hint("m", "Metadata"),
				components.Hint("c", "Content"),
				components.Hint("v", "Vault"),
				components.Hint("esc", "Back"),
			)
		default:
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("space", "Select"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Cancel"),
			)
		}
	case tabJobs:
		if a.jobs.view == jobsViewAdd || a.jobs.view == jobsViewEdit {
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("space", "Select"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Cancel"),
			)
		}
		if a.jobs.detail != nil {
			return append(base,
				components.Hint("s", "Status"),
				components.Hint("n", "Subtask"),
				components.Hint("esc", "Back"),
			)
		}
		return append(base,
			components.Hint("↑/↓", "Scroll"),
			components.Hint("tab", "Complete"),
			components.Hint("enter", "Details"),
			components.Hint("s", "Status"),
		)
	case tabLogs:
		switch a.logs.view {
		case logsViewDetail:
			return append(base,
				components.Hint("e", "Edit"),
				components.Hint("v", "Value"),
				components.Hint("m", "Metadata"),
				components.Hint("esc", "Back"),
			)
		case logsViewAdd, logsViewEdit:
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("space", "Select"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Back"),
			)
		default:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("tab", "Complete"),
				components.Hint("enter", "Details"),
			)
		}
	case tabFiles:
		switch a.files.view {
		case filesViewDetail:
			return append(base,
				components.Hint("e", "Edit"),
				components.Hint("m", "Metadata"),
				components.Hint("esc", "Back"),
			)
		case filesViewAdd, filesViewEdit:
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("space", "Select"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Back"),
			)
		default:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("tab", "Complete"),
				components.Hint("enter", "Details"),
			)
		}
	case tabProtocols:
		switch a.protocols.view {
		case protocolsViewDetail:
			return append(base,
				components.Hint("e", "Edit"),
				components.Hint("esc", "Back"),
			)
		case protocolsViewEdit, protocolsViewAdd:
			return append(base,
				components.Hint("↑/↓", "Fields"),
				components.Hint("←/→", "Cycle"),
				components.Hint("ctrl+s", "Save"),
				components.Hint("esc", "Cancel"),
			)
		default:
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("n", "New"),
				components.Hint("enter", "Details"),
			)
		}
	case tabHistory:
		if a.history.filtering {
			return append(base,
				components.Hint("enter", "Apply"),
				components.Hint("esc", "Clear"),
			)
		}
		if a.history.view == historyViewScopes || a.history.view == historyViewActors {
			return append(base,
				components.Hint("↑/↓", "Scroll"),
				components.Hint("enter", "Select"),
				components.Hint("esc", "Back"),
			)
		}
		if a.history.view == historyViewDetail {
			return append(base,
				components.Hint("esc", "Back"),
			)
		}
		return append(base,
			components.Hint("↑/↓", "Scroll"),
			components.Hint("enter", "Details"),
			components.Hint("f", "Filter"),
			components.Hint("s", "Scopes"),
			components.Hint("a", "Actors"),
		)
	case tabSearch:
		return append(base,
			components.Hint("↑/↓", "Scroll"),
			components.Hint("enter", "Open"),
			components.Hint("esc", "Clear"),
		)
	case tabProfile:
		if a.profile.agentDetail != nil {
			return append(base,
				components.Hint("esc", "Back"),
			)
		}
		hints := []string{
			components.Hint("↑/↓", "Scroll"),
			components.Hint("←/→", "Section"),
		}
		if a.profile.section == 0 {
			hints = append(hints,
				components.Hint("n", "New Key"),
				components.Hint("r", "Revoke"),
			)
		} else {
			hints = append(hints,
				components.Hint("enter", "Details"),
				components.Hint("t", "Toggle Trust"),
			)
		}
		return append(base, hints...)
	}
	return base
}

func (a App) renderTips() string {
	return ""
}

func (a App) renderHelp() string {
	hints := a.statusHintsForTab()
	lines := make([]string, 0, len(hints)+2)
	lines = append(lines, MutedStyle.Render("esc to close"))
	lines = append(lines, "")
	for _, hint := range hints {
		lines = append(lines, "  "+hint)
	}
	body := strings.Join(lines, "\n")
	return components.Indent(components.TitledBox("Help", body, a.width), 1)
}

func (a App) renderQuitConfirm() string {
	body := "You have unsaved changes. Quit anyway?"
	return components.Indent(components.ConfirmDialog("Quit", body), 1)
}

func (a *App) openPalette() {
	a.paletteOpen = true
	a.paletteQuery = ""
	a.paletteIndex = 0
	a.paletteEntityQuery = ""
	a.paletteEntityLoading = false
	a.paletteEntities = nil
	a.paletteFiltered = filterPalette(a.paletteActions, "")
}

func (a App) renderPalette() string {
	title := "Command Palette"
	query := components.SanitizeOneLine(a.paletteQuery)
	if query == "" {
		query = ""
	}

	var b strings.Builder
	b.WriteString("  > " + query)
	b.WriteString(AccentStyle.Render("█"))
	b.WriteString("\n\n")

	items := a.paletteFiltered
	if strings.HasPrefix(a.paletteQuery, ":") && a.paletteEntityLoading {
		b.WriteString(MutedStyle.Render("Searching entities..."))
	} else if len(items) == 0 {
		b.WriteString(MutedStyle.Render("No matches."))
	} else {
		for i, item := range items {
			label := components.SanitizeOneLine(item.Label)
			desc := components.SanitizeOneLine(item.Desc)
			line := fmt.Sprintf("%s  %s", label, MutedStyle.Render(desc))
			if i == a.paletteIndex {
				b.WriteString(SelectedStyle.Render("  > " + line))
			} else {
				b.WriteString(NormalStyle.Render("    " + line))
			}
			if i < len(items)-1 {
				b.WriteString("\n")
			}
		}
	}

	return components.TitledBox(title, b.String(), a.width)
}

func (a *App) refreshPaletteFiltered() tea.Cmd {
	if strings.HasPrefix(a.paletteQuery, ":") {
		query := strings.TrimSpace(strings.TrimPrefix(a.paletteQuery, ":"))
		if query == "" {
			a.paletteEntityQuery = ""
			a.paletteEntityLoading = false
			a.paletteEntities = nil
			a.paletteFiltered = nil
			a.paletteIndex = 0
			return nil
		}
		if query != a.paletteEntityQuery {
			a.paletteEntityQuery = query
			a.paletteEntityLoading = true
			a.paletteEntities = nil
			a.paletteFiltered = nil
			a.paletteIndex = 0
			return a.loadPaletteEntities(query)
		}
		a.paletteFiltered = buildEntityPaletteActions(a.paletteEntities, a.paletteEntityQuery)
		if a.paletteIndex >= len(a.paletteFiltered) {
			a.paletteIndex = 0
		}
		return nil
	}

	a.paletteEntityQuery = ""
	a.paletteEntityLoading = false
	a.paletteEntities = nil
	a.paletteFiltered = filterPalette(a.paletteActions, a.paletteQuery)
	if a.paletteIndex >= len(a.paletteFiltered) {
		a.paletteIndex = 0
	}
	return nil
}

func (a App) loadPaletteEntities(query string) tea.Cmd {
	return func() tea.Msg {
		items, err := a.client.QueryEntities(api.QueryParams{
			"search_text": query,
			"limit":       "15",
		})
		if err != nil {
			return errMsg{err}
		}
		return paletteEntitiesLoadedMsg{query: query, items: items}
	}
}

func buildEntityPaletteActions(items []api.Entity, query string) []paletteAction {
	query = strings.TrimSpace(strings.ToLower(query))
	actions := make([]paletteAction, 0, len(items))
	for _, e := range items {
		if query != "" {
			name := strings.ToLower(e.Name)
			id := strings.ToLower(e.ID)
			if !strings.Contains(name, query) && !strings.Contains(id, query) {
				continue
			}
		}
		kind := e.Type
		if kind == "" {
			kind = "entity"
		}
		desc := fmt.Sprintf("%s · %s", kind, shortID(e.ID))
		actions = append(actions, paletteAction{
			ID:    "entity:" + e.ID,
			Label: e.Name,
			Desc:  desc,
		})
	}
	return actions
}

func (a App) handlePaletteKeys(msg tea.KeyMsg) (tea.Model, tea.Cmd) {
	switch {
	case isBack(msg):
		a.paletteOpen = false
		return a, nil
	case isEnter(msg):
		if len(a.paletteFiltered) == 0 {
			return a, nil
		}
		action := a.paletteFiltered[a.paletteIndex]
		a.paletteOpen = false
		a.paletteQuery = ""
		return a.runPaletteAction(action)
	case isUp(msg):
		if a.paletteIndex > 0 {
			a.paletteIndex--
		}
	case isDown(msg):
		if a.paletteIndex < len(a.paletteFiltered)-1 {
			a.paletteIndex++
		}
	case isKey(msg, "backspace"):
		if len(a.paletteQuery) > 0 {
			a.paletteQuery = a.paletteQuery[:len(a.paletteQuery)-1]
			return a, a.refreshPaletteFiltered()
		}
	default:
		ch := msg.String()
		if len(ch) == 1 || ch == " " {
			a.paletteQuery += ch
			return a, a.refreshPaletteFiltered()
		}
	}
	return a, nil
}

func (a *App) runPaletteAction(action paletteAction) (tea.Model, tea.Cmd) {
	a.tabNav = true
	switch action.ID {
	default:
		if strings.HasPrefix(action.ID, "entity:") {
			id := strings.TrimPrefix(action.ID, "entity:")
			for _, e := range a.paletteEntities {
				if e.ID == id {
					entity := e
					a.tab = tabEntities
					a.tabNav = false
					a.entities.detail = &entity
					a.entities.view = entitiesViewDetail
					return *a, nil
				}
			}
			return *a, nil
		}
	case "tab:inbox":
		return a.switchTab(tabInbox)
	case "tab:entities":
		return a.switchTab(tabEntities)
	case "tab:relationships":
		return a.switchTab(tabRelations)
	case "tab:knowledge":
		return a.switchTab(tabKnow)
	case "tab:jobs":
		return a.switchTab(tabJobs)
	case "tab:history":
		return a.switchTab(tabHistory)
	case "tab:search":
		return a.switchTab(tabSearch)
	case "tab:profile":
		return a.switchTab(tabProfile)
	case "entities:search":
		a.tab = tabEntities
		a.tabNav = false
		a.entities.view = entitiesViewSearch
		a.entities.searchBuf = ""
		return *a, nil
	case "profile:keys":
		a.tab = tabProfile
		a.profile.section = 0
		return *a, nil
	case "profile:agents":
		a.tab = tabProfile
		a.profile.section = 1
		return *a, nil
	case "ops:import":
		a.tabNav = false
		a.importExportOpen = true
		a.impex.Start(importMode)
		return *a, nil
	case "ops:export":
		a.tabNav = false
		a.importExportOpen = true
		a.impex.Start(exportMode)
		return *a, nil
	case "quit":
		if a.hasUnsaved() {
			a.quitConfirm = true
			return *a, nil
		}
		return *a, tea.Quit
	}
	return *a, nil
}

func (a *App) applySearchSelection(msg searchSelectionMsg) (tea.Model, tea.Cmd) {
	a.tabNav = false
	switch msg.kind {
	case "entity":
		if msg.entity != nil {
			entity := *msg.entity
			a.tab = tabEntities
			a.entities.detail = &entity
			a.entities.view = entitiesViewDetail
		}
	case "knowledge":
		if msg.knowledge != nil {
			knowledge := *msg.knowledge
			a.tab = tabKnow
			a.know.detail = &knowledge
			a.know.view = knowledgeViewDetail
		}
	case "job":
		if msg.job != nil {
			job := *msg.job
			a.tab = tabJobs
			a.jobs.detail = &job
		}
	}
	return *a, nil
}

func (a App) hasUnsaved() bool {
	if a.inbox.rejecting {
		return true
	}
	switch a.entities.view {
	case entitiesViewEdit, entitiesViewRelEdit, entitiesViewRelateSearch, entitiesViewRelateSelect, entitiesViewRelateType:
		return true
	}
	switch a.rels.view {
	case relsViewEdit, relsViewCreateSourceSearch, relsViewCreateSourceSelect, relsViewCreateTargetSearch, relsViewCreateTargetSelect, relsViewCreateType:
		return true
	}
	if a.know.view == knowledgeViewAdd && !a.know.saved && !a.know.saving {
		if knowledgeHasInput(a.know) {
			return true
		}
	}
	if a.jobs.changingSt || a.jobs.creatingSubtask {
		return true
	}
	if a.profile.creating {
		return true
	}
	return false
}

func knowledgeHasInput(m KnowledgeModel) bool {
	for _, f := range m.fields {
		if strings.TrimSpace(f.value) != "" {
			return true
		}
	}
	if len(m.tags) > 0 || strings.TrimSpace(m.tagBuf) != "" {
		return true
	}
	if len(m.linkEntities) > 0 || strings.TrimSpace(m.linkQuery) != "" {
		return true
	}
	return false
}

func defaultPaletteActions() []paletteAction {
	return []paletteAction{
		{ID: "tab:inbox", Label: "Inbox", Desc: "Go to inbox"},
		{ID: "tab:entities", Label: "Entities", Desc: "Browse entities"},
		{ID: "tab:relationships", Label: "Relationships", Desc: "Browse relationships"},
		{ID: "tab:knowledge", Label: "Knowledge", Desc: "Add knowledge"},
		{ID: "tab:jobs", Label: "Jobs", Desc: "View jobs"},
		{ID: "tab:history", Label: "History", Desc: "Audit log"},
		{ID: "tab:search", Label: "Search", Desc: "Global search"},
		{ID: "tab:profile", Label: "Profile", Desc: "Keys and agents"},
		{ID: "ops:import", Label: "Import", Desc: "Bulk import from file"},
		{ID: "ops:export", Label: "Export", Desc: "Export data to file"},
		{ID: "entities:search", Label: "Search entities", Desc: "Open entity search"},
		{ID: "profile:keys", Label: "Profile: API keys", Desc: "Manage keys"},
		{ID: "profile:agents", Label: "Profile: agents", Desc: "Manage agents"},
		{ID: "quit", Label: "Quit", Desc: "Exit CLI"},
	}
}

func filterPalette(items []paletteAction, query string) []paletteAction {
	if query == "" {
		return items
	}
	q := strings.ToLower(strings.TrimSpace(query))
	filtered := make([]paletteAction, 0, len(items))
	for _, item := range items {
		label := strings.ToLower(item.Label)
		desc := strings.ToLower(item.Desc)
		if strings.Contains(label, q) || strings.Contains(desc, q) {
			filtered = append(filtered, item)
		}
	}
	return filtered
}

func centerBlock(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	for i, line := range lines {
		lineWidth := lipgloss.Width(line)
		if lineWidth >= width {
			continue
		}
		pad := (width - lineWidth) / 2
		lines[i] = strings.Repeat(" ", pad) + line
	}
	return strings.Join(lines, "\n")
}

func centerBlockUniform(s string, width int) string {
	if width <= 0 {
		return s
	}
	lines := strings.Split(s, "\n")
	maxWidth := 0
	for _, line := range lines {
		w := lipgloss.Width(line)
		if w > maxWidth {
			maxWidth = w
		}
	}
	if maxWidth <= 0 || maxWidth >= width {
		return s
	}
	pad := (width - maxWidth) / 2
	if pad <= 0 {
		return s
	}
	prefix := strings.Repeat(" ", pad)
	for i := range lines {
		if lines[i] != "" {
			lines[i] = prefix + lines[i]
		}
	}
	return strings.Join(lines, "\n")
}

func (a App) canExitToTabNav() bool {
	switch a.tab {
	case tabInbox:
		if a.inbox.detail != nil || a.inbox.rejecting {
			return false
		}
		return a.inbox.list == nil || a.inbox.list.Selected() == 0
	case tabEntities:
		if a.entities.view != entitiesViewList {
			return false
		}
		return a.entities.list == nil || a.entities.list.Selected() == 0
	case tabRelations:
		if a.rels.view != relsViewList {
			return false
		}
		return a.rels.list == nil || a.rels.list.Selected() == 0
	case tabKnow:
		if a.know.view == knowledgeViewList {
			return a.know.list == nil || a.know.list.Selected() == 0
		}
		if a.know.view != knowledgeViewAdd {
			return false
		}
		return !a.know.modeFocus && a.know.focus == fieldTitle
	case tabJobs:
		if a.jobs.detail != nil || a.jobs.changingSt {
			return false
		}
		return a.jobs.list == nil || a.jobs.list.Selected() == 0
	case tabHistory:
		if a.history.filtering || a.history.view != historyViewList {
			return false
		}
		return a.history.list == nil || a.history.list.Selected() == 0
	case tabSearch:
		if strings.TrimSpace(a.search.query) == "" {
			return true
		}
		return a.search.list == nil || a.search.list.Selected() == 0
	case tabProfile:
		if a.profile.creating || a.profile.createdKey != "" || a.profile.agentDetail != nil {
			return false
		}
		if a.profile.section == 0 {
			return a.profile.keyList == nil || a.profile.keyList.Selected() == 0
		}
		return a.profile.agentList == nil || a.profile.agentList.Selected() == 0
	}
	return false
}

func tabIndexForKey(key string) (int, bool) {
	switch key {
	case "1", "2", "3", "4", "5", "6", "7", "8", "9":
		idx := int(key[0] - '1')
		if idx >= 0 && idx < tabCount {
			return idx, true
		}
	case "0":
		if tabCount > 9 {
			return 9, true
		}
	case "-":
		if tabCount > 10 {
			return 10, true
		}
	}
	return 0, false
}
