package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/charmbracelet/lipgloss"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

// --- Messages ---

type knowledgeSavedMsg struct{}
type knowledgeLinkResultsMsg struct{ items []api.Entity }
type knowledgeListLoadedMsg struct{ items []api.Knowledge }
type knowledgeScopesLoadedMsg struct{ names map[string]string }
type knowledgeDetailLoadedMsg struct{ item api.Knowledge }
type knowledgeUpdatedMsg struct{ item api.Knowledge }

// --- Constants ---

var knowledgeTypes = []string{
	"note",
	"video",
	"article",
	"paper",
	"tool",
	"course",
	"thread",
}

var knowledgeStatusOptions = []string{"active", "archived"}

type knowledgeView int

const (
	knowledgeViewAdd knowledgeView = iota
	knowledgeViewList
	knowledgeViewDetail
	knowledgeViewEdit
)

// Field indices
const (
	fieldTitle    = 0
	fieldURL      = 1
	fieldType     = 2
	fieldTags     = 3
	fieldScopes   = 4
	fieldEntities = 5
	fieldNotes    = 6
	fieldMeta     = 7
	fieldCount    = 8
)

const (
	knowledgeEditFieldTitle = iota
	knowledgeEditFieldURL
	knowledgeEditFieldType
	knowledgeEditFieldStatus
	knowledgeEditFieldTags
	knowledgeEditFieldScopes
	knowledgeEditFieldNotes
	knowledgeEditFieldMeta
	knowledgeEditFieldCount
)

// --- Knowledge Model ---

// KnowledgeModel handles adding knowledge items manually.
type KnowledgeModel struct {
	client              *api.Client
	fields              []formField
	typeIdx             int
	typeSelecting       bool
	scopeOptions        []string
	scopeIdx            int
	scopeSelecting      bool
	focus               int
	modeFocus           bool
	saved               bool
	saving              bool
	view                knowledgeView
	errText             string
	tags                []string
	tagBuf              string
	scopes              []string
	scopeBuf            string
	linkSearching       bool
	linkLoading         bool
	linkQuery           string
	linkResults         []api.Entity
	linkList            *components.List
	linkEntities        []api.Entity
	list                *components.List
	items               []api.Knowledge
	loadingList         bool
	detail              *api.Knowledge
	knowledgeEditFields []formField
	editFocus           int
	editTypeIdx         int
	editTypeSelecting   bool
	editScopeSelecting  bool
	editStatusIdx       int
	editTags            []string
	editTagBuf          string
	editScopes          []string
	editScopeBuf        string
	editMeta            MetadataEditor
	editSaving          bool
	metaEditor          MetadataEditor
	metaExpanded        bool
	contentExpanded     bool
	vaultExpanded       bool
	scopeNames          map[string]string
	width               int
	height              int
}

type formField struct {
	label string
	value string
}

// NewKnowledgeModel builds the knowledge UI model.
func NewKnowledgeModel(client *api.Client) KnowledgeModel {
	return KnowledgeModel{
		client: client,
		fields: []formField{
			{label: "Title"},
			{label: "URL"},
			{label: "Type"},
			{label: "Tags"},
			{label: "Scopes"},
			{label: "Entities"},
			{label: "Notes"},
			{label: "Metadata"},
		},
		knowledgeEditFields: []formField{
			{label: "Title"},
			{label: "URL"},
			{label: "Type"},
			{label: "Status"},
			{label: "Tags"},
			{label: "Scopes"},
			{label: "Notes"},
			{label: "Metadata"},
		},
		linkList: components.NewList(6),
		list:     components.NewList(10),
	}
}

func (m KnowledgeModel) Init() tea.Cmd {
	m.saved = false
	m.errText = ""
	m.focus = 0
	m.modeFocus = false
	m.typeIdx = 0
	m.typeSelecting = false
	m.scopeIdx = 0
	m.scopeSelecting = false
	m.view = knowledgeViewAdd
	m.tags = nil
	m.tagBuf = ""
	m.scopes = nil
	m.scopeBuf = ""
	m.linkSearching = false
	m.linkLoading = false
	m.linkQuery = ""
	m.linkResults = nil
	m.linkEntities = nil
	m.detail = nil
	m.loadingList = false
	m.editFocus = 0
	m.editTypeIdx = 0
	m.editTypeSelecting = false
	m.editScopeSelecting = false
	m.editStatusIdx = statusIndex(knowledgeStatusOptions, "active")
	m.editTags = nil
	m.editTagBuf = ""
	m.editScopes = nil
	m.editScopeBuf = ""
	m.editMeta.Reset()
	m.editSaving = false
	m.metaEditor.Reset()
	m.metaExpanded = false
	m.contentExpanded = false
	m.vaultExpanded = false
	if m.scopeNames == nil {
		m.scopeNames = map[string]string{}
	}
	if m.linkList != nil {
		m.linkList.SetItems(nil)
	}
	if m.list != nil {
		m.list.SetItems(nil)
	}
	for i := range m.fields {
		m.fields[i].value = ""
	}
	return m.loadScopeNames()
}

func (m KnowledgeModel) Update(msg tea.Msg) (KnowledgeModel, tea.Cmd) {
	switch msg := msg.(type) {
	case knowledgeSavedMsg:
		m.saving = false
		m.saved = true
		return m, nil

	case errMsg:
		m.saving = false
		m.editSaving = false
		m.errText = msg.err.Error()
		return m, nil
	case knowledgeLinkResultsMsg:
		m.linkLoading = false
		m.linkResults = msg.items
		labels := make([]string, len(msg.items))
		for i, e := range msg.items {
			labels[i] = formatEntityLine(e)
		}
		if m.linkList != nil {
			m.linkList.SetItems(labels)
		}
		return m, nil

	case knowledgeListLoadedMsg:
		m.loadingList = false
		m.items = msg.items
		labels := make([]string, len(msg.items))
		for i, k := range msg.items {
			labels[i] = formatKnowledgeLine(k)
		}
		if m.list != nil {
			m.list.SetItems(labels)
		}
		return m, nil
	case knowledgeScopesLoadedMsg:
		if m.scopeNames == nil {
			m.scopeNames = map[string]string{}
		}
		for id, name := range msg.names {
			m.scopeNames[id] = name
		}
		m.scopeOptions = scopeNameList(m.scopeNames)
		m.metaEditor.SetScopeOptions(m.scopeOptions)
		m.editMeta.SetScopeOptions(m.scopeOptions)
		return m, nil
	case knowledgeDetailLoadedMsg:
		m.detail = &msg.item
		return m, nil
	case knowledgeUpdatedMsg:
		m.editSaving = false
		m.detail = &msg.item
		m.view = knowledgeViewDetail
		return m, nil

	case tea.KeyMsg:
		if m.metaEditor.Active {
			m.metaEditor.HandleKey(msg)
			return m, nil
		}
		if m.editMeta.Active {
			m.editMeta.HandleKey(msg)
			return m, nil
		}
		if m.view == knowledgeViewList {
			return m.handleListKeys(msg)
		}
		if m.view == knowledgeViewEdit {
			return m.handleEditKeys(msg)
		}
		if m.view == knowledgeViewDetail {
			return m.handleDetailKeys(msg)
		}
		if m.linkSearching {
			return m.handleLinkSearch(msg)
		}
		if m.modeFocus {
			return m.handleModeKeys(msg)
		}
		// Type selector field - press space to enter, then space/left/right to cycle
		if m.focus == fieldType {
			if m.typeSelecting {
				switch {
				case isKey(msg, "left"):
					m.typeIdx = (m.typeIdx - 1 + len(knowledgeTypes)) % len(knowledgeTypes)
					return m, nil
				case isKey(msg, "right"):
					m.typeIdx = (m.typeIdx + 1) % len(knowledgeTypes)
					return m, nil
				case isSpace(msg):
					m.typeSelecting = false
					return m, nil
				}
			} else if isSpace(msg) {
				m.typeSelecting = true
				return m, nil
			}
		}
		if m.focus == fieldScopes && m.scopeSelecting {
			switch {
			case isKey(msg, "left"):
				if len(m.scopeOptions) > 0 {
					m.scopeIdx = (m.scopeIdx - 1 + len(m.scopeOptions)) % len(m.scopeOptions)
				}
				return m, nil
			case isKey(msg, "right"):
				if len(m.scopeOptions) > 0 {
					m.scopeIdx = (m.scopeIdx + 1) % len(m.scopeOptions)
				}
				return m, nil
			case isSpace(msg):
				if len(m.scopeOptions) > 0 {
					scope := m.scopeOptions[m.scopeIdx]
					m.scopes = toggleScope(m.scopes, scope)
				}
				return m, nil
			case isEnter(msg), isBack(msg):
				m.scopeSelecting = false
				return m, nil
			}
		}

		switch {
		case isDown(msg):
			m.typeSelecting = false
			m.scopeSelecting = false
			m.focus = (m.focus + 1) % fieldCount
		case isUp(msg):
			if m.focus == 0 {
				m.typeSelecting = false
				m.scopeSelecting = false
				m.modeFocus = true
				return m, nil
			}
			m.typeSelecting = false
			m.scopeSelecting = false
			m.focus = (m.focus - 1 + fieldCount) % fieldCount
		case isKey(msg, "ctrl+s"):
			return m.save()
		case isBack(msg):
			m.resetForm()
		case isKey(msg, "backspace"):
			switch m.focus {
			case fieldTags:
				if len(m.tagBuf) > 0 {
					m.tagBuf = m.tagBuf[:len(m.tagBuf)-1]
				} else if len(m.tags) > 0 {
					m.tags = m.tags[:len(m.tags)-1]
				}
			case fieldScopes:
				if len(m.scopes) > 0 {
					m.scopes = m.scopes[:len(m.scopes)-1]
				}
			case fieldEntities:
				if len(m.linkEntities) > 0 {
					m.linkEntities = m.linkEntities[:len(m.linkEntities)-1]
				}
			default:
				if m.focus != fieldType {
					f := &m.fields[m.focus]
					if len(f.value) > 0 {
						f.value = f.value[:len(f.value)-1]
					}
				}
			}
		default:
			if m.focus == fieldTags {
				switch {
				case isSpace(msg) || isKey(msg, ",") || isEnter(msg):
					m.commitTag()
				default:
					ch := msg.String()
					if len(ch) == 1 && ch != "," {
						m.tagBuf += ch
					}
				}
			} else if m.focus == fieldScopes {
				if isSpace(msg) {
					m.scopeSelecting = true
				}
			} else if m.focus == fieldEntities {
				if isEnter(msg) {
					m.startLinkSearch()
				}
			} else if m.focus == fieldMeta {
				if isEnter(msg) {
					m.metaEditor.Active = true
				}
			} else if m.focus != fieldType {
				ch := msg.String()
				if len(ch) == 1 || ch == " " {
					m.fields[m.focus].value += ch
				}
			}
		}
		if m.focus == fieldEntities && !m.linkSearching {
			ch := msg.String()
			if len(ch) == 1 || ch == " " {
				m.startLinkSearch()
				m.linkQuery += ch
				return m, m.updateLinkSearch()
			}
		}
	}
	return m, nil
}

func (m KnowledgeModel) View() string {
	if m.saving {
		return "  " + MutedStyle.Render("Saving...")
	}

	if m.saved {
		return components.Indent(components.Box(SuccessStyle.Render("Knowledge saved! Press Esc to add another."), m.width), 1)
	}

	if m.editMeta.Active {
		return m.editMeta.Render(m.width)
	}

	if m.metaEditor.Active {
		return m.metaEditor.Render(m.width)
	}

	if m.linkSearching {
		return m.renderLinkSearch()
	}

	modeLine := m.renderModeLine()
	var body string
	switch m.view {
	case knowledgeViewList:
		body = m.renderList()
	case knowledgeViewDetail:
		body = m.renderDetail()
	case knowledgeViewEdit:
		body = m.renderEdit()
	default:
		body = m.renderAdd()
	}
	if modeLine != "" {
		body = components.CenterLine(modeLine, m.width) + "\n\n" + body
	}
	return components.Indent(body, 1)
}

func (m KnowledgeModel) renderAdd() string {
	var b strings.Builder
	for i, f := range m.fields {
		label := f.label

		if i == fieldType {
			// Type selector
			if i == m.focus && m.typeSelecting {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n  ")
				for j, t := range knowledgeTypes {
					if j == m.typeIdx {
						b.WriteString(AccentStyle.Render("[" + t + "]"))
					} else {
						b.WriteString(MutedStyle.Render(" " + t + " "))
					}
					if j < len(knowledgeTypes)-1 {
						b.WriteString(" ")
					}
				}
			} else if i == m.focus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + knowledgeTypes[m.typeIdx]))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + knowledgeTypes[m.typeIdx]))
			}
		} else if i == fieldTags {
			if i == m.focus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderTags(true)))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderTags(false)))
			}
		} else if i == fieldScopes {
			if i == m.focus && m.scopeSelecting {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + renderScopeOptions(m.scopes, m.scopeOptions, m.scopeIdx)))
			} else if i == m.focus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderScopes(true)))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderScopes(false)))
			}
		} else if i == fieldEntities {
			if i == m.focus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderLinkedEntities(true)))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderLinkedEntities(false)))
			}
		} else if i == fieldMeta {
			if i == m.focus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
			}
			b.WriteString("\n")
			meta := renderMetadataInput(m.metaEditor.Buffer)
			b.WriteString(NormalStyle.Render("  " + meta))
		} else if i == m.focus {
			b.WriteString(SelectedStyle.Render("> " + label + ":"))
			b.WriteString("\n")
			b.WriteString(NormalStyle.Render("  " + f.value))
			b.WriteString(AccentStyle.Render("█"))
		} else {
			b.WriteString(MutedStyle.Render("  " + label + ":"))
			b.WriteString("\n")
			val := f.value
			if val == "" {
				val = "-"
			}
			b.WriteString(NormalStyle.Render("  " + val))
		}

		if i < fieldCount-1 {
			b.WriteString("\n\n")
		}
	}

	if m.errText != "" {
		b.WriteString("\n\n")
		b.WriteString(components.ErrorBox("Error", m.errText, m.width))
	}

	return components.TitledBox("Add Knowledge", b.String(), m.width)
}

func (m KnowledgeModel) renderEdit() string {
	var b strings.Builder
	for i, f := range m.knowledgeEditFields {
		label := f.label
		switch i {
		case knowledgeEditFieldType:
			if i == m.editFocus && m.editTypeSelecting {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n  ")
				for j, t := range knowledgeTypes {
					if j == m.editTypeIdx {
						b.WriteString(AccentStyle.Render("[" + t + "]"))
					} else {
						b.WriteString(MutedStyle.Render(" " + t + " "))
					}
					if j < len(knowledgeTypes)-1 {
						b.WriteString(" ")
					}
				}
			} else if i == m.editFocus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + knowledgeTypes[m.editTypeIdx]))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + knowledgeTypes[m.editTypeIdx]))
			}
		case knowledgeEditFieldStatus:
			if i == m.editFocus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
			}
			b.WriteString("\n")
			status := knowledgeStatusOptions[m.editStatusIdx]
			b.WriteString(NormalStyle.Render("  " + status))
		case knowledgeEditFieldTags:
			if i == m.editFocus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderEditTags(true)))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderEditTags(false)))
			}
		case knowledgeEditFieldScopes:
			if i == m.editFocus && m.editScopeSelecting {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + renderScopeOptions(m.editScopes, m.scopeOptions, m.scopeIdx)))
			} else if i == m.editFocus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderEditScopes(true)))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + m.renderEditScopes(false)))
			}
		case knowledgeEditFieldMeta:
			if i == m.editFocus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
			}
			b.WriteString("\n")
			meta := renderMetadataInput(m.editMeta.Buffer)
			b.WriteString(NormalStyle.Render("  " + meta))
		default:
			if i == m.editFocus {
				b.WriteString(SelectedStyle.Render("> " + label + ":"))
				b.WriteString("\n")
				b.WriteString(NormalStyle.Render("  " + f.value))
				b.WriteString(AccentStyle.Render("█"))
			} else {
				b.WriteString(MutedStyle.Render("  " + label + ":"))
				b.WriteString("\n")
				val := f.value
				if val == "" {
					val = "-"
				}
				b.WriteString(NormalStyle.Render("  " + val))
			}
		}

		if i < knowledgeEditFieldCount-1 {
			b.WriteString("\n\n")
		}
	}

	if m.errText != "" {
		b.WriteString("\n\n")
		b.WriteString(components.ErrorBox("Error", m.errText, m.width))
	}

	if m.editSaving {
		b.WriteString("\n\n" + MutedStyle.Render("Saving..."))
	}

	return components.TitledBox("Edit Knowledge", b.String(), m.width)
}

func (m KnowledgeModel) renderModeLine() string {
	add := TabInactiveStyle.Render("Add")
	list := TabInactiveStyle.Render("Library")
	if m.view == knowledgeViewAdd {
		add = TabActiveStyle.Render("Add")
	} else {
		list = TabActiveStyle.Render("Library")
	}
	line := add + " " + list
	if m.modeFocus {
		return SelectedStyle.Render("› " + line)
	}
	return line
}

func (m KnowledgeModel) handleModeKeys(msg tea.KeyMsg) (KnowledgeModel, tea.Cmd) {
	switch {
	case isDown(msg):
		m.modeFocus = false
		if m.view == knowledgeViewEdit {
			m.editFocus = 0
		} else {
			m.focus = 0
		}
	case isUp(msg):
		m.modeFocus = false
	case isKey(msg, "left"), isKey(msg, "right"), isSpace(msg), isEnter(msg):
		return m.toggleMode()
	case isBack(msg):
		m.modeFocus = false
		if m.view == knowledgeViewEdit {
			m.editFocus = 0
		} else {
			m.focus = 0
		}
	}
	return m, nil
}

func (m KnowledgeModel) toggleMode() (KnowledgeModel, tea.Cmd) {
	m.modeFocus = false
	m.detail = nil
	m.metaExpanded = false
	m.contentExpanded = false
	m.vaultExpanded = false
	if m.view == knowledgeViewAdd {
		m.view = knowledgeViewList
		m.loadingList = true
		return m, m.loadKnowledgeList()
	}
	if m.view == knowledgeViewDetail || m.view == knowledgeViewEdit {
		m.view = knowledgeViewList
		return m, nil
	}
	m.view = knowledgeViewAdd
	return m, nil
}

func (m KnowledgeModel) handleListKeys(msg tea.KeyMsg) (KnowledgeModel, tea.Cmd) {
	switch {
	case isDown(msg):
		m.list.Down()
	case isUp(msg):
		if m.list.Selected() == 0 {
			m.modeFocus = true
		} else {
			m.list.Up()
		}
	case isEnter(msg):
		if idx := m.list.Selected(); idx < len(m.items) {
			item := m.items[idx]
			m.detail = &item
			m.view = knowledgeViewDetail
			return m, m.loadKnowledgeDetail(item.ID)
		}
	case isBack(msg):
		m.view = knowledgeViewAdd
	}
	return m, nil
}

func (m KnowledgeModel) handleDetailKeys(msg tea.KeyMsg) (KnowledgeModel, tea.Cmd) {
	switch {
	case isUp(msg):
		m.modeFocus = true
	case isBack(msg):
		m.detail = nil
		m.metaExpanded = false
		m.contentExpanded = false
		m.vaultExpanded = false
		m.view = knowledgeViewList
	case isKey(msg, "e"):
		m.startEdit()
		m.view = knowledgeViewEdit
	case isKey(msg, "m"):
		m.metaExpanded = !m.metaExpanded
	case isKey(msg, "c"):
		m.contentExpanded = !m.contentExpanded
	case isKey(msg, "v"):
		m.vaultExpanded = !m.vaultExpanded
	}
	return m, nil
}

func (m KnowledgeModel) handleEditKeys(msg tea.KeyMsg) (KnowledgeModel, tea.Cmd) {
	if m.editSaving {
		return m, nil
	}
	if m.modeFocus {
		return m.handleModeKeys(msg)
	}
	if m.editFocus == knowledgeEditFieldType {
		if m.editTypeSelecting {
			switch {
			case isKey(msg, "left"):
				m.editTypeIdx = (m.editTypeIdx - 1 + len(knowledgeTypes)) % len(knowledgeTypes)
				return m, nil
			case isKey(msg, "right"):
				m.editTypeIdx = (m.editTypeIdx + 1) % len(knowledgeTypes)
				return m, nil
			case isSpace(msg), isEnter(msg):
				m.editTypeSelecting = false
				return m, nil
			}
		} else if isSpace(msg) || isEnter(msg) {
			m.editTypeSelecting = true
			return m, nil
		}
	}
	if m.editFocus == knowledgeEditFieldScopes && m.editScopeSelecting {
		switch {
		case isKey(msg, "left"):
			if len(m.scopeOptions) > 0 {
				m.scopeIdx = (m.scopeIdx - 1 + len(m.scopeOptions)) % len(m.scopeOptions)
			}
			return m, nil
		case isKey(msg, "right"):
			if len(m.scopeOptions) > 0 {
				m.scopeIdx = (m.scopeIdx + 1) % len(m.scopeOptions)
			}
			return m, nil
		case isSpace(msg):
			if len(m.scopeOptions) > 0 {
				scope := m.scopeOptions[m.scopeIdx]
				m.editScopes = toggleScope(m.editScopes, scope)
			}
			return m, nil
		case isEnter(msg), isBack(msg):
			m.editScopeSelecting = false
			return m, nil
		}
	}
	if m.editFocus == knowledgeEditFieldStatus {
		switch {
		case isKey(msg, "left"):
			m.editStatusIdx = (m.editStatusIdx - 1 + len(knowledgeStatusOptions)) % len(knowledgeStatusOptions)
			return m, nil
		case isKey(msg, "right"), isSpace(msg):
			m.editStatusIdx = (m.editStatusIdx + 1) % len(knowledgeStatusOptions)
			return m, nil
		}
	}

	switch {
	case isDown(msg):
		m.editTypeSelecting = false
		m.editScopeSelecting = false
		m.editFocus = (m.editFocus + 1) % knowledgeEditFieldCount
	case isUp(msg):
		m.editTypeSelecting = false
		m.editScopeSelecting = false
		if m.editFocus == 0 {
			m.modeFocus = true
			return m, nil
		}
		m.editFocus = (m.editFocus - 1 + knowledgeEditFieldCount) % knowledgeEditFieldCount
	case isKey(msg, "ctrl+s"):
		return m.saveEdit()
	case isBack(msg):
		m.editScopeSelecting = false
		m.view = knowledgeViewDetail
	case isKey(msg, "backspace"):
		switch m.editFocus {
		case knowledgeEditFieldTags:
			if len(m.editTagBuf) > 0 {
				m.editTagBuf = m.editTagBuf[:len(m.editTagBuf)-1]
			} else if len(m.editTags) > 0 {
				m.editTags = m.editTags[:len(m.editTags)-1]
			}
		case knowledgeEditFieldScopes:
			if len(m.editScopes) > 0 {
				m.editScopes = m.editScopes[:len(m.editScopes)-1]
			}
		default:
			if m.editFocus != knowledgeEditFieldType && m.editFocus != knowledgeEditFieldStatus {
				f := &m.knowledgeEditFields[m.editFocus]
				if len(f.value) > 0 {
					f.value = f.value[:len(f.value)-1]
				}
			}
		}
	default:
		switch m.editFocus {
		case knowledgeEditFieldTags:
			switch {
			case isSpace(msg) || isKey(msg, ",") || isEnter(msg):
				m.commitEditTag()
			default:
				ch := msg.String()
				if len(ch) == 1 && ch != "," {
					m.editTagBuf += ch
				}
			}
		case knowledgeEditFieldScopes:
			if isSpace(msg) {
				m.editScopeSelecting = true
			}
		case knowledgeEditFieldMeta:
			if isEnter(msg) {
				m.editMeta.Active = true
			}
		default:
			if m.editFocus != knowledgeEditFieldType && m.editFocus != knowledgeEditFieldStatus {
				ch := msg.String()
				if len(ch) == 1 || ch == " " {
					m.knowledgeEditFields[m.editFocus].value += ch
				}
			}
		}
	}
	return m, nil
}

func (m KnowledgeModel) renderList() string {
	if m.loadingList {
		return components.Box(MutedStyle.Render("Loading knowledge..."), m.width)
	}

	if len(m.items) == 0 {
		return components.EmptyStateBox(
			"Knowledge",
			"No knowledge found.",
			[]string{"Press tab to switch Add/Library", "Press / for command palette"},
			m.width,
		)
	}

	contentWidth := components.BoxContentWidth(m.width)
	visible := m.list.Visible()

	previewWidth := contentWidth * 35 / 100
	if previewWidth < 40 {
		previewWidth = 40
	}
	if previewWidth > 60 {
		previewWidth = 60
	}

	gap := 3
	tableWidth := contentWidth
	sideBySide := contentWidth >= 110
	if sideBySide {
		tableWidth = contentWidth - previewWidth - gap
		if tableWidth < 60 {
			sideBySide = false
			tableWidth = contentWidth
		}
	}

	sepWidth := 1
	if b := lipgloss.RoundedBorder().Left; b != "" {
		sepWidth = lipgloss.Width(b)
	}

	// 4 columns -> 3 separators.
	availableCols := tableWidth - (3 * sepWidth)
	if availableCols < 30 {
		availableCols = 30
	}

	typeWidth := 10
	statusWidth := 10
	atWidth := 11
	titleWidth := availableCols - (typeWidth + statusWidth + atWidth)
	if titleWidth < 12 {
		titleWidth = 12
	}
	cols := []components.TableColumn{
		{Header: "Title", Width: titleWidth, Align: lipgloss.Left},
		{Header: "Type", Width: typeWidth, Align: lipgloss.Left},
		{Header: "Status", Width: statusWidth, Align: lipgloss.Left},
		{Header: "At", Width: atWidth, Align: lipgloss.Left},
	}

	tableRows := make([][]string, 0, len(visible))
	activeRowRel := -1
	var previewItem *api.Knowledge
	if idx := m.list.Selected(); idx >= 0 && idx < len(m.items) {
		previewItem = &m.items[idx]
	}

	for i := range visible {
		absIdx := m.list.RelToAbs(i)
		if absIdx < 0 || absIdx >= len(m.items) {
			continue
		}
		k := m.items[absIdx]

		title := components.ClampTextWidthEllipsis(components.SanitizeOneLine(k.Name), titleWidth)
		typ := strings.TrimSpace(components.SanitizeOneLine(k.SourceType))
		if typ == "" {
			typ = "note"
		}
		status := strings.TrimSpace(components.SanitizeOneLine(k.Status))
		if status == "" {
			status = "-"
		}
		at := k.UpdatedAt
		if at.IsZero() {
			at = k.CreatedAt
		}
		when := at.Format("01-02 15:04")

		if m.list.IsSelected(absIdx) {
			activeRowRel = len(tableRows)
		}
		tableRows = append(tableRows, []string{
			title,
			components.ClampTextWidthEllipsis(typ, typeWidth),
			components.ClampTextWidthEllipsis(status, statusWidth),
			when,
		})
	}

	countLine := fmt.Sprintf("%d total", len(m.items))
	countLine = MutedStyle.Render(countLine)

	table := components.TableGridWithActiveRow(cols, tableRows, tableWidth, activeRowRel)
	preview := ""
	if previewItem != nil {
		content := m.renderKnowledgePreview(*previewItem, previewBoxContentWidth(previewWidth))
		preview = renderPreviewBox(content, previewWidth)
	}

	body := table
	if sideBySide && preview != "" {
		body = lipgloss.JoinHorizontal(lipgloss.Top, table, strings.Repeat(" ", gap), preview)
	} else if preview != "" {
		body = table + "\n\n" + preview
	}

	content := countLine + "\n\n" + body + "\n"
	return components.TitledBox("Knowledge", content, m.width)
}

func (m KnowledgeModel) renderDetail() string {
	if m.detail == nil {
		return m.renderList()
	}

	k := m.detail
	rows := []components.TableRow{
		{Label: "ID", Value: k.ID},
		{Label: "Title", Value: k.Name},
	}
	if k.SourceType != "" {
		rows = append(rows, components.TableRow{Label: "Type", Value: k.SourceType})
	}
	if k.Status != "" {
		rows = append(rows, components.TableRow{Label: "Status", Value: k.Status})
	}
	if k.URL != nil && strings.TrimSpace(*k.URL) != "" {
		rows = append(rows, components.TableRow{Label: "URL", Value: *k.URL})
	}
	if len(k.PrivacyScopeIDs) > 0 {
		rows = append(rows, components.TableRow{Label: "Scopes", Value: m.formatKnowledgeScopes(k.PrivacyScopeIDs)})
	}
	if len(k.Tags) > 0 {
		rows = append(rows, components.TableRow{Label: "Tags", Value: strings.Join(k.Tags, ", ")})
	}
	rows = append(rows, components.TableRow{Label: "Created", Value: k.CreatedAt.Format("2006-01-02 15:04")})
	if !k.UpdatedAt.IsZero() {
		rows = append(rows, components.TableRow{Label: "Updated", Value: k.UpdatedAt.Format("2006-01-02 15:04")})
	}
	if k.VaultFilePath != nil && strings.TrimSpace(*k.VaultFilePath) != "" {
		path := *k.VaultFilePath
		if !m.vaultExpanded {
			path = truncateString(path, 60)
		}
		rows = append(rows, components.TableRow{Label: "Vault Path", Value: path})
	}

	sections := []string{components.Table("Knowledge", rows, m.width)}
	if k.Content != nil && strings.TrimSpace(*k.Content) != "" {
		content := strings.TrimSpace(components.SanitizeText(*k.Content))
		if !m.contentExpanded {
			content = truncateString(content, 220)
		}
		sections = append(sections, components.TitledBox("Content", content, m.width))
	}
	if len(k.Metadata) > 0 {
		metaTable := renderMetadataBlock(map[string]any(k.Metadata), m.width, m.metaExpanded)
		sections = append(sections, metaTable)
	}

	return strings.Join(sections, "\n\n")
}

func (m KnowledgeModel) renderKnowledgePreview(k api.Knowledge, width int) string {
	if width <= 0 {
		return ""
	}

	title := components.SanitizeOneLine(k.Name)
	typ := strings.TrimSpace(components.SanitizeOneLine(k.SourceType))
	if typ == "" {
		typ = "note"
	}
	status := strings.TrimSpace(components.SanitizeOneLine(k.Status))
	if status == "" {
		status = "-"
	}
	at := k.UpdatedAt
	if at.IsZero() {
		at = k.CreatedAt
	}

	var lines []string
	lines = append(lines, MetaKeyStyle.Render("Selected"))
	for _, part := range wrapPreviewText(title, width) {
		lines = append(lines, SelectedStyle.Render(part))
	}
	lines = append(lines, "")

	lines = append(lines, renderPreviewRow("Type", typ, width))
	lines = append(lines, renderPreviewRow("Status", status, width))
	lines = append(lines, renderPreviewRow("At", at.Format("01-02 15:04"), width))

	if k.URL != nil && strings.TrimSpace(*k.URL) != "" {
		lines = append(lines, renderPreviewRow("URL", strings.TrimSpace(*k.URL), width))
	}
	if len(k.PrivacyScopeIDs) > 0 {
		lines = append(lines, renderPreviewRow("Scopes", m.formatKnowledgeScopes(k.PrivacyScopeIDs), width))
	}
	if len(k.Tags) > 0 {
		lines = append(lines, renderPreviewRow("Tags", strings.Join(k.Tags, ", "), width))
	}

	snippet := ""
	if metaPreview := metadataPreview(map[string]any(k.Metadata), 80); metaPreview != "" {
		snippet = metaPreview
	} else if k.Content != nil {
		snippet = truncateString(strings.TrimSpace(components.SanitizeText(*k.Content)), 80)
	} else if k.URL != nil {
		snippet = truncateString(strings.TrimSpace(components.SanitizeText(*k.URL)), 80)
	}
	if strings.TrimSpace(snippet) != "" {
		lines = append(lines, renderPreviewRow("Preview", strings.TrimSpace(snippet), width))
	}

	return padPreviewLines(lines, width)
}

func (m *KnowledgeModel) startEdit() {
	if m.detail == nil {
		return
	}
	k := m.detail
	m.knowledgeEditFields[knowledgeEditFieldTitle].value = k.Name
	if k.URL != nil {
		m.knowledgeEditFields[knowledgeEditFieldURL].value = *k.URL
	} else {
		m.knowledgeEditFields[knowledgeEditFieldURL].value = ""
	}
	m.knowledgeEditFields[knowledgeEditFieldNotes].value = ""
	if k.Content != nil {
		m.knowledgeEditFields[knowledgeEditFieldNotes].value = *k.Content
	}
	m.editTypeIdx = statusIndex(knowledgeTypes, k.SourceType)
	m.editStatusIdx = statusIndex(knowledgeStatusOptions, k.Status)
	m.editTags = append([]string{}, k.Tags...)
	m.editTagBuf = ""
	m.editScopes = m.scopeNamesFromIDs(k.PrivacyScopeIDs)
	m.editScopeBuf = ""
	m.editScopeSelecting = false
	m.scopeIdx = 0
	m.editMeta.Load(map[string]any(k.Metadata))
	m.editMeta.Active = false
	m.editSaving = false
	m.editFocus = 0
}

func (m KnowledgeModel) saveEdit() (KnowledgeModel, tea.Cmd) {
	if m.detail == nil {
		return m, nil
	}
	m.commitEditTag()
	title := strings.TrimSpace(m.knowledgeEditFields[knowledgeEditFieldTitle].value)
	url := strings.TrimSpace(m.knowledgeEditFields[knowledgeEditFieldURL].value)
	content := strings.TrimSpace(m.knowledgeEditFields[knowledgeEditFieldNotes].value)
	sourceType := knowledgeTypes[m.editTypeIdx]
	status := knowledgeStatusOptions[m.editStatusIdx]
	tags := normalizeBulkTags(m.editTags)
	scopes := normalizeBulkScopes(m.editScopes)
	meta, err := parseMetadataInput(m.editMeta.Buffer)
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	meta = mergeMetadataScopes(meta, m.editMeta.Scopes)

	input := api.UpdateKnowledgeInput{
		Title:      &title,
		URL:        &url,
		SourceType: &sourceType,
		Content:    &content,
		Status:     &status,
		Tags:       &tags,
		Scopes:     &scopes,
		Metadata:   meta,
	}

	m.editSaving = true
	return m, func() tea.Msg {
		updated, err := m.client.UpdateKnowledge(m.detail.ID, input)
		if err != nil {
			return errMsg{err}
		}
		return knowledgeUpdatedMsg{item: *updated}
	}
}

// --- Helpers ---

func (m KnowledgeModel) loadKnowledgeList() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.QueryKnowledge(api.QueryParams{})
		if err != nil {
			return errMsg{err}
		}
		return knowledgeListLoadedMsg{items: items}
	}
}

func (m KnowledgeModel) loadKnowledgeDetail(id string) tea.Cmd {
	return func() tea.Msg {
		item, err := m.client.GetKnowledge(id)
		if err != nil {
			return errMsg{err}
		}
		return knowledgeDetailLoadedMsg{item: *item}
	}
}

func formatKnowledgeLine(k api.Knowledge) string {
	t := components.SanitizeText(k.SourceType)
	if t == "" {
		t = "note"
	}
	name := truncateKnowledgeName(components.SanitizeText(k.Name), maxKnowledgeNameLen)
	line := fmt.Sprintf("%s %s", name, TypeBadgeStyle.Render(components.SanitizeText(t)))
	if status := strings.TrimSpace(components.SanitizeText(k.Status)); status != "" {
		line = fmt.Sprintf("%s · %s", line, status)
	}
	preview := ""
	if metaPreview := metadataPreview(map[string]any(k.Metadata), 40); metaPreview != "" {
		preview = metaPreview
	} else if k.Content != nil {
		preview = truncateString(strings.TrimSpace(components.SanitizeText(*k.Content)), 40)
	} else if k.URL != nil {
		preview = truncateString(strings.TrimSpace(components.SanitizeText(*k.URL)), 40)
	}
	if preview != "" {
		line = fmt.Sprintf("%s · %s", line, preview)
	}
	return line
}

const maxKnowledgeNameLen = 80

func truncateKnowledgeName(s string, max int) string {
	if max <= 0 {
		return ""
	}
	runes := []rune(s)
	if len(runes) <= max {
		return s
	}
	return string(runes[:max]) + "..."
}

func (m KnowledgeModel) loadScopeNames() tea.Cmd {
	if m.client == nil {
		return nil
	}
	return func() tea.Msg {
		scopes, err := m.client.ListAuditScopes()
		if err != nil {
			return errMsg{err}
		}
		names := map[string]string{}
		for _, scope := range scopes {
			names[scope.ID] = scope.Name
		}
		return knowledgeScopesLoadedMsg{names: names}
	}
}

func (m KnowledgeModel) formatKnowledgeScopes(ids []string) string {
	if len(ids) == 0 {
		return "-"
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := m.scopeNames[id]; ok && name != "" {
			names = append(names, name)
		} else {
			names = append(names, id)
		}
	}
	return strings.Join(names, ", ")
}

func (m KnowledgeModel) scopeNamesFromIDs(ids []string) []string {
	if len(ids) == 0 {
		return nil
	}
	names := make([]string, 0, len(ids))
	for _, id := range ids {
		if name, ok := m.scopeNames[id]; ok && name != "" {
			names = append(names, name)
		} else {
			names = append(names, id)
		}
	}
	return names
}

func (m *KnowledgeModel) resetForm() {
	m.saved = false
	m.errText = ""
	m.typeSelecting = false
	m.focus = 0
	m.modeFocus = false
	m.typeIdx = 0
	m.scopeIdx = 0
	m.scopeSelecting = false
	m.tags = nil
	m.tagBuf = ""
	m.scopes = nil
	m.scopeBuf = ""
	m.linkSearching = false
	m.linkLoading = false
	m.linkQuery = ""
	m.linkResults = nil
	m.linkEntities = nil
	m.metaEditor.Reset()
	if m.linkList != nil {
		m.linkList.SetItems(nil)
	}
	for i := range m.fields {
		m.fields[i].value = ""
	}
}

func (m KnowledgeModel) save() (KnowledgeModel, tea.Cmd) {
	title := strings.TrimSpace(m.fields[fieldTitle].value)
	if title == "" {
		m.errText = "Title is required"
		return m, nil
	}

	url := strings.TrimSpace(m.fields[fieldURL].value)
	sourceType := knowledgeTypes[m.typeIdx]
	notes := strings.TrimSpace(m.fields[fieldNotes].value)

	m.commitTag()

	meta, err := parseMetadataInput(m.metaEditor.Buffer)
	if err != nil {
		m.errText = err.Error()
		return m, nil
	}
	meta = mergeMetadataScopes(meta, m.metaEditor.Scopes)

	scopes := normalizeBulkScopes(m.scopes)
	if len(scopes) == 0 {
		scopes = []string{"personal"}
	}

	input := api.CreateKnowledgeInput{
		Title:      title,
		URL:        url,
		SourceType: sourceType,
		Content:    notes,
		Scopes:     scopes,
		Tags:       m.tags,
		Metadata:   meta,
	}

	linkIDs := make([]string, 0, len(m.linkEntities))
	for _, e := range m.linkEntities {
		linkIDs = append(linkIDs, e.ID)
	}

	m.saving = true
	return m, func() tea.Msg {
		created, err := m.client.CreateKnowledge(input)
		if err != nil {
			return errMsg{err}
		}
		for _, id := range linkIDs {
			if err := m.client.LinkKnowledge(created.ID, id); err != nil {
				return errMsg{err}
			}
		}
		return knowledgeSavedMsg{}
	}
}

func (m *KnowledgeModel) renderTags(focused bool) string {
	if len(m.tags) == 0 && m.tagBuf == "" && !focused {
		return "-"
	}

	var b strings.Builder
	for i, t := range m.tags {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(AccentStyle.Render("[" + t + "]"))
	}
	if focused {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		if m.tagBuf != "" {
			b.WriteString(m.tagBuf)
		}
		b.WriteString(AccentStyle.Render("█"))
	} else if m.tagBuf != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(MutedStyle.Render(m.tagBuf))
	}
	return b.String()
}

func (m *KnowledgeModel) renderEditTags(focused bool) string {
	if len(m.editTags) == 0 && m.editTagBuf == "" && !focused {
		return "-"
	}

	var b strings.Builder
	for i, t := range m.editTags {
		if i > 0 {
			b.WriteString(" ")
		}
		b.WriteString(AccentStyle.Render("[" + t + "]"))
	}
	if focused {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		if m.editTagBuf != "" {
			b.WriteString(m.editTagBuf)
		}
		b.WriteString(AccentStyle.Render("█"))
	} else if m.editTagBuf != "" {
		if b.Len() > 0 {
			b.WriteString(" ")
		}
		b.WriteString(MutedStyle.Render(m.editTagBuf))
	}
	return b.String()
}

func (m *KnowledgeModel) renderScopes(focused bool) string {
	return renderScopePills(m.scopes, focused)
}

func (m *KnowledgeModel) renderEditScopes(focused bool) string {
	return renderScopePills(m.editScopes, focused)
}

func (m *KnowledgeModel) renderLinkedEntities(focused bool) string {
	if len(m.linkEntities) == 0 && !focused {
		return "-"
	}
	var b strings.Builder
	for i, e := range m.linkEntities {
		if i > 0 {
			b.WriteString(" ")
		}
		label := e.Name
		if label == "" {
			label = shortID(e.ID)
		}
		b.WriteString(AccentStyle.Render("[" + label + "]"))
	}
	return b.String()
}

func (m *KnowledgeModel) startLinkSearch() {
	m.linkSearching = true
	m.linkLoading = false
	m.linkQuery = ""
	m.linkResults = nil
	if m.linkList != nil {
		m.linkList.SetItems(nil)
	}
}

func (m KnowledgeModel) handleLinkSearch(msg tea.KeyMsg) (KnowledgeModel, tea.Cmd) {
	switch {
	case isBack(msg):
		m.linkSearching = false
		m.linkLoading = false
		m.linkQuery = ""
		m.linkResults = nil
		if m.linkList != nil {
			m.linkList.SetItems(nil)
		}
	case isDown(msg):
		if m.linkList != nil {
			m.linkList.Down()
		}
	case isUp(msg):
		if m.linkList != nil {
			m.linkList.Up()
		}
	case isEnter(msg):
		if m.linkList != nil {
			if idx := m.linkList.Selected(); idx < len(m.linkResults) {
				m.addLinkedEntity(m.linkResults[idx])
			}
		}
		m.linkSearching = false
		m.linkLoading = false
		m.linkQuery = ""
		m.linkResults = nil
		if m.linkList != nil {
			m.linkList.SetItems(nil)
		}
	case isKey(msg, "backspace"):
		if len(m.linkQuery) > 0 {
			m.linkQuery = m.linkQuery[:len(m.linkQuery)-1]
			return m, m.updateLinkSearch()
		}
	case isKey(msg, "cmd+backspace", "cmd+delete", "ctrl+u"):
		if m.linkQuery != "" {
			m.linkQuery = ""
			m.linkResults = nil
			if m.linkList != nil {
				m.linkList.SetItems(nil)
			}
			return m, nil
		}
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			if len(m.linkResults) > 0 {
				m.linkResults = nil
				if m.linkList != nil {
					m.linkList.SetItems(nil)
				}
			}
			m.linkQuery += msg.String()
			return m, m.updateLinkSearch()
		}
	}
	return m, nil
}

func (m KnowledgeModel) renderLinkSearch() string {
	var b strings.Builder
	b.WriteString("Search: " + m.linkQuery)
	b.WriteString(AccentStyle.Render("█"))
	b.WriteString("\n\n")
	if m.linkLoading {
		b.WriteString(MutedStyle.Render("Searching..."))
	} else if strings.TrimSpace(m.linkQuery) == "" {
		b.WriteString(MutedStyle.Render("Type to search."))
	} else if len(m.linkResults) == 0 {
		b.WriteString(MutedStyle.Render("No matches."))
	} else {
		visible := m.linkList.Visible()
		for i, label := range visible {
			absIdx := m.linkList.RelToAbs(i)
			if m.linkList.IsSelected(absIdx) {
				b.WriteString(SelectedStyle.Render("  > " + label))
			} else {
				b.WriteString(NormalStyle.Render("    " + label))
			}
			if i < len(visible)-1 {
				b.WriteString("\n")
			}
		}
	}
	return components.Indent(components.TitledBox("Link Entity", b.String(), m.width), 1)
}

func (m KnowledgeModel) searchLinkEntities(query string) tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.QueryEntities(api.QueryParams{"search_text": query})
		if err != nil {
			return errMsg{err}
		}
		return knowledgeLinkResultsMsg{items: items}
	}
}

func (m *KnowledgeModel) updateLinkSearch() tea.Cmd {
	query := strings.TrimSpace(m.linkQuery)
	if query == "" {
		m.linkLoading = false
		m.linkResults = nil
		if m.linkList != nil {
			m.linkList.SetItems(nil)
		}
		return nil
	}
	m.linkLoading = true
	return m.searchLinkEntities(query)
}

func (m *KnowledgeModel) addLinkedEntity(entity api.Entity) {
	for _, e := range m.linkEntities {
		if e.ID == entity.ID {
			return
		}
	}
	m.linkEntities = append(m.linkEntities, entity)
}

func (m *KnowledgeModel) commitTag() {
	raw := strings.TrimSpace(m.tagBuf)
	if raw == "" {
		m.tagBuf = ""
		return
	}

	tag := normalizeTag(raw)
	if tag == "" {
		m.tagBuf = ""
		return
	}

	for _, t := range m.tags {
		if t == tag {
			m.tagBuf = ""
			return
		}
	}
	m.tags = append(m.tags, tag)
	m.tagBuf = ""
}

func (m *KnowledgeModel) commitScope() {
	raw := strings.TrimSpace(m.scopeBuf)
	if raw == "" {
		m.scopeBuf = ""
		return
	}

	scope := normalizeScope(raw)
	if scope == "" {
		m.scopeBuf = ""
		return
	}

	for _, s := range m.scopes {
		if s == scope {
			m.scopeBuf = ""
			return
		}
	}
	m.scopes = append(m.scopes, scope)
	m.scopeBuf = ""
}

func (m *KnowledgeModel) commitEditTag() {
	raw := strings.TrimSpace(m.editTagBuf)
	if raw == "" {
		m.editTagBuf = ""
		return
	}

	tag := normalizeTag(raw)
	if tag == "" {
		m.editTagBuf = ""
		return
	}

	for _, t := range m.editTags {
		if t == tag {
			m.editTagBuf = ""
			return
		}
	}
	m.editTags = append(m.editTags, tag)
	m.editTagBuf = ""
}

func (m *KnowledgeModel) commitEditScope() {
	raw := strings.TrimSpace(m.editScopeBuf)
	if raw == "" {
		m.editScopeBuf = ""
		return
	}

	scope := normalizeScope(raw)
	if scope == "" {
		m.editScopeBuf = ""
		return
	}

	for _, s := range m.editScopes {
		if s == scope {
			m.editScopeBuf = ""
			return
		}
	}
	m.editScopes = append(m.editScopes, scope)
	m.editScopeBuf = ""
}

func normalizeTag(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	s = strings.ToLower(s)
	s = strings.ReplaceAll(s, "_", "-")
	s = strings.Join(strings.Fields(s), "-")
	return s
}

func normalizeScope(s string) string {
	s = strings.TrimSpace(s)
	s = strings.TrimPrefix(s, "#")
	s = strings.ToLower(s)
	s = strings.Join(strings.Fields(s), "-")
	return s
}
