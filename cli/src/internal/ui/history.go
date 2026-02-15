package ui

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"time"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

type historyLoadedMsg struct{ items []api.AuditEntry }
type historyScopesLoadedMsg struct{ items []api.AuditScope }
type historyActorsLoadedMsg struct{ items []api.AuditActor }

type historyView int

const (
	historyViewList historyView = iota
	historyViewDetail
	historyViewScopes
	historyViewActors
)

type auditFilter struct {
	tableName string
	action    string
	actorType string
	actorID   string
	recordID  string
	scopeID   string
	actor     string
	terms     []string
}

type HistoryModel struct {
	client    *api.Client
	items     []api.AuditEntry
	list      *components.List
	loading   bool
	width     int
	height    int
	view      historyView
	detail    *api.AuditEntry
	filtering bool
	filterBuf string
	filter    auditFilter
	errText   string
	scopes    []api.AuditScope
	actors    []api.AuditActor
	scopeList *components.List
	actorList *components.List
}

// NewHistoryModel builds the audit history UI model.
func NewHistoryModel(client *api.Client) HistoryModel {
	return HistoryModel{
		client:    client,
		list:      components.NewList(10),
		scopeList: components.NewList(10),
		actorList: components.NewList(10),
		view:      historyViewList,
	}
}

func (m HistoryModel) Init() tea.Cmd {
	m.loading = true
	return m.loadHistory()
}

func (m HistoryModel) Update(msg tea.Msg) (HistoryModel, tea.Cmd) {
	switch msg := msg.(type) {
	case historyLoadedMsg:
		m.loading = false
		m.errText = ""
		m.items = m.applyLocalFilters(msg.items)
		labels := make([]string, len(m.items))
		for i, entry := range m.items {
			labels[i] = formatAuditLine(entry)
		}
		m.list.SetItems(labels)
		return m, nil
	case historyScopesLoadedMsg:
		m.loading = false
		m.errText = ""
		m.scopes = msg.items
		labels := make([]string, len(m.scopes))
		for i, scope := range m.scopes {
			labels[i] = formatScopeLine(scope)
		}
		m.scopeList.SetItems(labels)
		return m, nil
	case historyActorsLoadedMsg:
		m.loading = false
		m.errText = ""
		m.actors = msg.items
		labels := make([]string, len(m.actors))
		for i, actor := range m.actors {
			labels[i] = formatActorLine(actor)
		}
		m.actorList.SetItems(labels)
		return m, nil
	case errMsg:
		m.loading = false
		m.errText = msg.err.Error()
		return m, nil
	case tea.KeyMsg:
		if m.filtering {
			return m.handleFilterKeys(msg)
		}
		switch m.view {
		case historyViewList:
			return m.handleListKeys(msg)
		case historyViewDetail:
			if isBack(msg) {
				m.view = historyViewList
				m.detail = nil
				return m, nil
			}
		case historyViewScopes:
			return m.handleScopeKeys(msg)
		case historyViewActors:
			return m.handleActorKeys(msg)
		}
	}

	return m, nil
}

func (m HistoryModel) View() string {
	if m.filtering {
		return components.Indent(components.InputDialog("Filter Audit Log", m.filterBuf), 1)
	}
	if m.loading {
		label := "Loading history..."
		if m.view == historyViewScopes {
			label = "Loading scopes..."
		} else if m.view == historyViewActors {
			label = "Loading actors..."
		}
		return "  " + MutedStyle.Render(label)
	}
	if m.errText != "" {
		return components.Indent(components.ErrorBox("Error", m.errText, m.width), 1)
	}
	if m.view == historyViewDetail && m.detail != nil {
		return m.renderDetail(*m.detail)
	}
	if m.view == historyViewScopes {
		return m.renderScopes()
	}
	if m.view == historyViewActors {
		return m.renderActors()
	}
	return m.renderList()
}

func (m HistoryModel) handleListKeys(msg tea.KeyMsg) (HistoryModel, tea.Cmd) {
	switch {
	case isDown(msg):
		m.list.Down()
	case isUp(msg):
		m.list.Up()
	case isEnter(msg):
		if idx := m.list.Selected(); idx < len(m.items) {
			entry := m.items[idx]
			m.detail = &entry
			m.view = historyViewDetail
		}
	case isKey(msg, "f"):
		m.filtering = true
	case isKey(msg, "s"):
		m.view = historyViewScopes
		m.loading = true
		return m, m.loadScopes()
	case isKey(msg, "a"):
		m.view = historyViewActors
		m.loading = true
		return m, m.loadActors()
	}
	return m, nil
}

func (m HistoryModel) handleFilterKeys(msg tea.KeyMsg) (HistoryModel, tea.Cmd) {
	switch {
	case isEnter(msg):
		m.filtering = false
		m.filter = parseAuditFilter(m.filterBuf)
		m.loading = true
		return m, m.loadHistory()
	case isBack(msg):
		m.filtering = false
		m.filterBuf = ""
		m.filter = auditFilter{}
		m.loading = true
		return m, m.loadHistory()
	case msg.Type == tea.KeyBackspace:
		if len(m.filterBuf) > 0 {
			m.filterBuf = m.filterBuf[:len(m.filterBuf)-1]
		}
	case msg.Type == tea.KeyRunes:
		m.filterBuf += msg.String()
	}
	return m, nil
}

func (m HistoryModel) handleScopeKeys(msg tea.KeyMsg) (HistoryModel, tea.Cmd) {
	switch {
	case isDown(msg):
		m.scopeList.Down()
	case isUp(msg):
		m.scopeList.Up()
	case isEnter(msg):
		if idx := m.scopeList.Selected(); idx < len(m.scopes) {
			scope := m.scopes[idx]
			m.filter.scopeID = scope.ID
			m.view = historyViewList
			m.loading = true
			return m, m.loadHistory()
		}
	case isBack(msg):
		m.view = historyViewList
	}
	return m, nil
}

func (m HistoryModel) handleActorKeys(msg tea.KeyMsg) (HistoryModel, tea.Cmd) {
	switch {
	case isDown(msg):
		m.actorList.Down()
	case isUp(msg):
		m.actorList.Up()
	case isEnter(msg):
		if idx := m.actorList.Selected(); idx < len(m.actors) {
			actor := m.actors[idx]
			m.filter.actorType = actor.ActorType
			m.filter.actorID = actor.ActorID
			m.view = historyViewList
			m.loading = true
			return m, m.loadHistory()
		}
	case isBack(msg):
		m.view = historyViewList
	}
	return m, nil
}

func (m HistoryModel) loadHistory() tea.Cmd {
	filter := m.filter
	return func() tea.Msg {
		items, err := m.client.QueryAuditLogWithPagination(
			filter.tableName,
			filter.action,
			filter.actorType,
			filter.actorID,
			filter.recordID,
			filter.scopeID,
			50,
			0,
		)
		if err != nil {
			return errMsg{err}
		}
		return historyLoadedMsg{items: items}
	}
}

func (m HistoryModel) loadScopes() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.ListAuditScopes()
		if err != nil {
			return errMsg{err}
		}
		return historyScopesLoadedMsg{items: items}
	}
}

func (m HistoryModel) loadActors() tea.Cmd {
	return func() tea.Msg {
		items, err := m.client.ListAuditActors("")
		if err != nil {
			return errMsg{err}
		}
		return historyActorsLoadedMsg{items: items}
	}
}

func (m HistoryModel) renderList() string {
	if len(m.items) == 0 {
		content := MutedStyle.Render("No audit entries yet.")
		return components.Indent(components.Box(content, m.width), 1)
	}
	var rows strings.Builder
	filterLine := formatAuditFilters(m.filter)
	if filterLine != "" {
		rows.WriteString(MutedStyle.Render(filterLine))
		rows.WriteString("\n\n")
	}
	contentWidth := components.BoxContentWidth(m.width)
	maxLabelWidth := contentWidth - 4
	visible := m.list.Visible()
	for i, label := range visible {
		if maxLabelWidth > 0 {
			label = components.ClampTextWidth(label, maxLabelWidth)
		}
		absIdx := m.list.RelToAbs(i)
		if m.list.IsSelected(absIdx) {
			rows.WriteString(SelectedStyle.Render("  > " + label))
		} else {
			rows.WriteString(NormalStyle.Render("    " + label))
		}
		if i < len(visible)-1 {
			rows.WriteString("\n")
		}
	}
	return components.Indent(components.TitledBox("History", rows.String(), m.width), 1)
}

func (m HistoryModel) renderScopes() string {
	if len(m.scopes) == 0 {
		content := MutedStyle.Render("No scopes found.")
		return components.Indent(components.Box(content, m.width), 1)
	}
	var rows strings.Builder
	contentWidth := components.BoxContentWidth(m.width)
	maxLabelWidth := contentWidth - 4
	visible := m.scopeList.Visible()
	for i, label := range visible {
		if maxLabelWidth > 0 {
			label = components.ClampTextWidth(label, maxLabelWidth)
		}
		absIdx := m.scopeList.RelToAbs(i)
		if m.scopeList.IsSelected(absIdx) {
			rows.WriteString(SelectedStyle.Render("  > " + label))
		} else {
			rows.WriteString(NormalStyle.Render("    " + label))
		}
		if i < len(visible)-1 {
			rows.WriteString("\n")
		}
	}
	return components.Indent(components.TitledBox("Scopes", rows.String(), m.width), 1)
}

func (m HistoryModel) renderActors() string {
	if len(m.actors) == 0 {
		content := MutedStyle.Render("No actors found.")
		return components.Indent(components.Box(content, m.width), 1)
	}
	var rows strings.Builder
	contentWidth := components.BoxContentWidth(m.width)
	maxLabelWidth := contentWidth - 4
	visible := m.actorList.Visible()
	for i, label := range visible {
		if maxLabelWidth > 0 {
			label = components.ClampTextWidth(label, maxLabelWidth)
		}
		absIdx := m.actorList.RelToAbs(i)
		if m.actorList.IsSelected(absIdx) {
			rows.WriteString(SelectedStyle.Render("  > " + label))
		} else {
			rows.WriteString(NormalStyle.Render("    " + label))
		}
		if i < len(visible)-1 {
			rows.WriteString("\n")
		}
	}
	return components.Indent(components.TitledBox("Actors", rows.String(), m.width), 1)
}

func (m HistoryModel) renderDetail(entry api.AuditEntry) string {
	when := entry.ChangedAt.Format("2006-01-02 15:04")
	actor := formatAuditActor(entry)
	fields := ""
	if len(entry.ChangedFields) > 0 {
		fields = strings.Join(entry.ChangedFields, ", ")
	}
	rows := []components.TableRow{
		{Label: "Table", Value: entry.TableName},
		{Label: "Action", Value: entry.Action},
		{Label: "Record", Value: entry.RecordID},
		{Label: "Actor", Value: actor},
		{Label: "At", Value: when},
	}
	if fields != "" {
		rows = append(rows, components.TableRow{Label: "Fields", Value: fields})
	}
	if entry.ChangeReason != nil && *entry.ChangeReason != "" {
		rows = append(rows, components.TableRow{Label: "Reason", Value: *entry.ChangeReason})
	}
	section := components.Table("Audit Entry", rows, m.width)

	diffRows := buildAuditDiffRows(entry)
	if len(diffRows) > 0 {
		diff := components.DiffTable("Changes", diffRows, m.width)
		section = section + "\n\n" + diff
	}
	return components.Indent(section, 1)
}

func parseAuditFilter(input string) auditFilter {
	filter := auditFilter{}
	input = strings.TrimSpace(input)
	if input == "" {
		return filter
	}
	for _, token := range strings.Fields(input) {
		switch {
		case strings.HasPrefix(token, "table:"):
			filter.tableName = strings.TrimPrefix(token, "table:")
		case strings.HasPrefix(token, "action:"):
			filter.action = strings.TrimPrefix(token, "action:")
		case strings.HasPrefix(token, "actor_type:"):
			filter.actorType = strings.TrimPrefix(token, "actor_type:")
		case strings.HasPrefix(token, "actor_id:"):
			filter.actorID = strings.TrimPrefix(token, "actor_id:")
		case strings.HasPrefix(token, "record:"):
			filter.recordID = strings.TrimPrefix(token, "record:")
		case strings.HasPrefix(token, "record_id:"):
			filter.recordID = strings.TrimPrefix(token, "record_id:")
		case strings.HasPrefix(token, "scope:"):
			filter.scopeID = strings.TrimPrefix(token, "scope:")
		case strings.HasPrefix(token, "scope_id:"):
			filter.scopeID = strings.TrimPrefix(token, "scope_id:")
		case strings.HasPrefix(token, "actor:"):
			filter.actor = strings.ToLower(strings.TrimPrefix(token, "actor:"))
		default:
			filter.terms = append(filter.terms, strings.ToLower(token))
		}
	}
	return filter
}

func (m HistoryModel) applyLocalFilters(items []api.AuditEntry) []api.AuditEntry {
	filter := m.filter
	if filter.actor == "" && len(filter.terms) == 0 {
		return items
	}
	filtered := make([]api.AuditEntry, 0, len(items))
	for _, entry := range items {
		if filter.actor != "" {
			actor := strings.ToLower(formatAuditActor(entry))
			if !strings.Contains(actor, filter.actor) {
				continue
			}
		}
		if len(filter.terms) > 0 {
			haystack := strings.ToLower(fmt.Sprintf("%s %s %s", entry.TableName, entry.RecordID, formatAuditActor(entry)))
			matched := true
			for _, term := range filter.terms {
				if !strings.Contains(haystack, term) {
					matched = false
					break
				}
			}
			if !matched {
				continue
			}
		}
		filtered = append(filtered, entry)
	}
	return filtered
}

func formatAuditActor(entry api.AuditEntry) string {
	if entry.ActorName != nil && *entry.ActorName != "" {
		return *entry.ActorName
	}
	if entry.ChangedByType != nil && entry.ChangedByID != nil {
		return fmt.Sprintf("%s:%s", *entry.ChangedByType, shortID(*entry.ChangedByID))
	}
	if entry.ChangedByType != nil {
		return *entry.ChangedByType
	}
	return "system"
}

func formatAuditLine(entry api.AuditEntry) string {
	when := entry.ChangedAt.Format("01-02 15:04")
	actor := formatAuditActor(entry)
	action := entry.Action
	if action == "" {
		action = "update"
	}
	return fmt.Sprintf("%s  %s  %s  %s", when, strings.ToUpper(action), entry.TableName, actor)
}

func formatScopeLine(scope api.AuditScope) string {
	desc := ""
	if scope.Description != nil && *scope.Description != "" {
		desc = " - " + *scope.Description
	}
	return fmt.Sprintf(
		"%s  agents:%d entities:%d knowledge:%d%s",
		scope.Name,
		scope.AgentCount,
		scope.EntityCount,
		scope.KnowledgeCount,
		desc,
	)
}

func formatActorLine(actor api.AuditActor) string {
	name := "unknown"
	if actor.ActorName != nil && *actor.ActorName != "" {
		name = *actor.ActorName
	}
	when := actor.LastSeen.Format("01-02 15:04")
	return fmt.Sprintf(
		"%s  %s:%s  actions:%d  last:%s",
		name,
		actor.ActorType,
		shortID(actor.ActorID),
		actor.ActionCount,
		when,
	)
}

func formatAuditFilters(filter auditFilter) string {
	parts := []string{}
	if filter.tableName != "" {
		parts = append(parts, "table:"+filter.tableName)
	}
	if filter.action != "" {
		parts = append(parts, "action:"+filter.action)
	}
	if filter.actorType != "" {
		parts = append(parts, "actor_type:"+filter.actorType)
	}
	if filter.actorID != "" {
		parts = append(parts, "actor_id:"+shortID(filter.actorID))
	}
	if filter.recordID != "" {
		parts = append(parts, "record:"+shortID(filter.recordID))
	}
	if filter.scopeID != "" {
		parts = append(parts, "scope:"+shortID(filter.scopeID))
	}
	if filter.actor != "" {
		parts = append(parts, "actor:"+filter.actor)
	}
	if len(parts) == 0 {
		return ""
	}
	if len(parts) == 1 {
		return "Filters: " + parts[0]
	}
	// Avoid lipgloss word-wrapping splitting filter tokens (for example "scope:...").
	return "Filters:\n  " + strings.Join(parts, "\n  ")
}

func buildAuditDiffRows(entry api.AuditEntry) []components.DiffRow {
	keys := make([]string, 0)
	seen := map[string]bool{}
	if len(entry.ChangedFields) > 0 {
		for _, k := range entry.ChangedFields {
			if k == "" {
				continue
			}
			seen[k] = true
			keys = append(keys, k)
		}
	} else {
		for k := range entry.OldData {
			if !seen[k] {
				seen[k] = true
				keys = append(keys, k)
			}
		}
		for k := range entry.NewData {
			if !seen[k] {
				keys = append(keys, k)
			}
		}
	}
	if len(keys) == 0 {
		return nil
	}
	sort.Strings(keys)
	rows := make([]components.DiffRow, 0, len(keys))
	for _, key := range keys {
		from := entry.OldData[key]
		to := entry.NewData[key]
		if formatAuditValue(from) == formatAuditValue(to) {
			continue
		}
		rows = append(rows, components.DiffRow{
			Label: key,
			From:  formatAuditValue(from),
			To:    formatAuditValue(to),
		})
	}
	return rows
}

func formatAuditValue(value any) string {
	if value == nil {
		return "null"
	}
	switch v := value.(type) {
	case string:
		if v == "" {
			return "-"
		}
		return v
	case time.Time:
		return v.Format("2006-01-02 15:04")
	default:
		b, err := json.Marshal(v)
		if err != nil {
			return fmt.Sprintf("%v", v)
		}
		return string(b)
	}
}
