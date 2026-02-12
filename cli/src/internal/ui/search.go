package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

type searchResultsMsg struct {
	query     string
	mode      string
	entities  []api.Entity
	knowledge []api.Knowledge
	jobs      []api.Job
	semantic  []api.SemanticSearchResult
}

type searchSelectionMsg struct {
	kind      string
	entity    *api.Entity
	knowledge *api.Knowledge
	job       *api.Job
}

type searchEntry struct {
	kind      string
	id        string
	label     string
	desc      string
	entity    *api.Entity
	knowledge *api.Knowledge
	job       *api.Job
}

type SearchModel struct {
	client  *api.Client
	query   string
	mode    string
	loading bool
	list    *components.List
	items   []searchEntry
	width   int
	height  int
}

const (
	searchModeText     = "text"
	searchModeSemantic = "semantic"
)

// NewSearchModel builds the search UI model.
func NewSearchModel(client *api.Client) SearchModel {
	return SearchModel{
		client: client,
		mode:   searchModeText,
		list:   components.NewList(12),
	}
}

func (m SearchModel) Init() tea.Cmd {
	return nil
}

func (m SearchModel) Update(msg tea.Msg) (SearchModel, tea.Cmd) {
	switch msg := msg.(type) {
	case searchResultsMsg:
		if strings.TrimSpace(msg.query) != strings.TrimSpace(m.query) {
			return m, nil
		}
		if msg.mode != m.mode {
			return m, nil
		}
		m.loading = false
		if m.mode == searchModeSemantic {
			m.items = buildSemanticEntries(msg.semantic)
		} else {
			m.items = buildSearchEntries(msg.query, msg.entities, msg.knowledge, msg.jobs)
		}
		labels := make([]string, len(m.items))
		for i, item := range m.items {
			labels[i] = fmt.Sprintf(
				"%s  %s",
				components.SanitizeText(item.label),
				MutedStyle.Render(components.SanitizeText(item.desc)),
			)
		}
		m.list.SetItems(labels)
		return m, nil
	case tea.KeyMsg:
		switch {
		case isBack(msg):
			if m.query != "" {
				m.query = ""
				m.items = nil
				m.list.SetItems(nil)
				m.loading = false
				return m, nil
			}
		case isKey(msg, "cmd+backspace", "cmd+delete", "ctrl+u"):
			if m.query != "" {
				m.query = ""
				m.items = nil
				m.list.SetItems(nil)
				m.loading = false
				return m, nil
			}
		case isKey(msg, "backspace", "delete"):
			if len(m.query) > 0 {
				m.query = m.query[:len(m.query)-1]
				return m, m.search(m.query)
			}
		case isDown(msg):
			m.list.Down()
		case isUp(msg):
			m.list.Up()
		case isKey(msg, "tab"):
			if m.mode == searchModeText {
				m.mode = searchModeSemantic
			} else {
				m.mode = searchModeText
			}
			if strings.TrimSpace(m.query) == "" {
				m.loading = false
				m.items = nil
				m.list.SetItems(nil)
				return m, nil
			}
			return m, m.search(m.query)
		case isEnter(msg):
			if idx := m.list.Selected(); idx < len(m.items) {
				entry := m.items[idx]
				return m, m.emitSelection(entry)
			}
		default:
			ch := msg.String()
			if len(ch) == 1 || ch == " " {
				if ch == " " && m.query == "" {
					return m, nil
				}
				m.query += ch
				return m, m.search(m.query)
			}
		}
	}
	return m, nil
}

func (m SearchModel) View() string {
	var b strings.Builder
	b.WriteString(MutedStyle.Render(fmt.Sprintf("Mode: %s (tab to toggle)", m.mode)))
	b.WriteString("\n\n")
	b.WriteString("  > " + components.SanitizeText(m.query))
	b.WriteString(AccentStyle.Render("█"))
	b.WriteString("\n\n")

	if m.loading {
		b.WriteString(MutedStyle.Render("Searching..."))
	} else if strings.TrimSpace(m.query) == "" {
		b.WriteString(MutedStyle.Render("Type to search."))
	} else if len(m.items) == 0 {
		b.WriteString(MutedStyle.Render("No matches."))
	} else {
		contentWidth := components.BoxContentWidth(m.width)
		maxLabelWidth := contentWidth - 4
		visible := m.list.Visible()
		for i, label := range visible {
			if maxLabelWidth > 0 {
				label = components.ClampTextWidth(label, maxLabelWidth)
			}
			absIdx := m.list.RelToAbs(i)
			if m.list.IsSelected(absIdx) {
				b.WriteString(SelectedStyle.Render("  > " + label))
			} else {
				b.WriteString(NormalStyle.Render("    " + label))
			}
			if i < len(visible)-1 {
				b.WriteString("\n")
			}
		}
	}

	return components.Indent(components.TitledBox("Search", b.String(), m.width), 1)
}

func (m *SearchModel) search(query string) tea.Cmd {
	q := strings.TrimSpace(query)
	if q == "" {
		m.loading = false
		m.items = nil
		m.list.SetItems(nil)
		return nil
	}
	m.loading = true
	mode := m.mode
	return func() tea.Msg {
		if mode == searchModeSemantic {
			results, err := m.client.SemanticSearch(q, []string{"entity", "knowledge"}, 20)
			if err != nil {
				return errMsg{err}
			}
			return searchResultsMsg{
				query:    q,
				mode:     mode,
				semantic: results,
			}
		}
		entities, err := m.client.QueryEntities(api.QueryParams{
			"search_text": q,
			"limit":       "20",
		})
		if err != nil {
			return errMsg{err}
		}
		knowledge, err := m.client.QueryKnowledge(api.QueryParams{
			"search_text": q,
			"limit":       "20",
		})
		if err != nil {
			return errMsg{err}
		}
		jobs, err := m.client.QueryJobs(api.QueryParams{
			"search_text": q,
			"limit":       "20",
		})
		if err != nil {
			return errMsg{err}
		}
		return searchResultsMsg{
			query:     q,
			mode:      mode,
			entities:  filterEntitiesByQuery(entities, q),
			knowledge: filterKnowledgeByQuery(knowledge, q),
			jobs:      filterJobsByQuery(jobs, q),
		}
	}
}

func (m SearchModel) emitSelection(entry searchEntry) tea.Cmd {
	return func() tea.Msg {
		switch entry.kind {
		case "entity":
			if entry.entity == nil {
				item, err := m.client.GetEntity(entry.id)
				if err != nil {
					return errMsg{err}
				}
				return searchSelectionMsg{kind: entry.kind, entity: item}
			}
		case "knowledge":
			if entry.knowledge == nil {
				item, err := m.client.GetKnowledge(entry.id)
				if err != nil {
					return errMsg{err}
				}
				return searchSelectionMsg{kind: entry.kind, knowledge: item}
			}
		case "job":
			if entry.job == nil {
				item, err := m.client.GetJob(entry.id)
				if err != nil {
					return errMsg{err}
				}
				return searchSelectionMsg{kind: entry.kind, job: item}
			}
		}
		return searchSelectionMsg{
			kind:      entry.kind,
			entity:    entry.entity,
			knowledge: entry.knowledge,
			job:       entry.job,
		}
	}
}

func buildSemanticEntries(items []api.SemanticSearchResult) []searchEntry {
	out := make([]searchEntry, 0, len(items))
	for _, item := range items {
		title := strings.TrimSpace(item.Title)
		if title == "" {
			title = item.ID
		}
		descParts := []string{
			fmt.Sprintf("%.2f", item.Score),
		}
		if strings.TrimSpace(item.Subtitle) != "" {
			descParts = append(descParts, item.Subtitle)
		}
		if strings.TrimSpace(item.Snippet) != "" {
			descParts = append(descParts, item.Snippet)
		}
		out = append(out, searchEntry{
			kind:  item.Kind,
			id:    item.ID,
			label: components.SanitizeText(title),
			desc:  components.SanitizeText(strings.Join(descParts, " · ")),
		})
	}
	return out
}

func buildSearchEntries(query string, entities []api.Entity, knowledge []api.Knowledge, jobs []api.Job) []searchEntry {
	items := make([]searchEntry, 0, len(entities)+len(knowledge)+len(jobs))
	for _, e := range entities {
		kind := "entity"
		descType := e.Type
		if descType == "" {
			descType = "entity"
		}
		entity := e
		items = append(items, searchEntry{
			kind:   kind,
			id:     e.ID,
			label:  components.SanitizeText(e.Name),
			desc:   components.SanitizeText(fmt.Sprintf("%s · %s", descType, shortID(e.ID))),
			entity: &entity,
		})
	}
	for _, k := range knowledge {
		kind := "knowledge"
		descType := k.SourceType
		if descType == "" {
			descType = "knowledge"
		}
		knowledgeItem := k
		items = append(items, searchEntry{
			kind:      kind,
			id:        k.ID,
			label:     components.SanitizeText(k.Name),
			desc:      components.SanitizeText(fmt.Sprintf("%s · %s", descType, shortID(k.ID))),
			knowledge: &knowledgeItem,
		})
	}
	for _, j := range jobs {
		kind := "job"
		desc := j.Status
		if desc == "" {
			desc = "job"
		}
		job := j
		items = append(items, searchEntry{
			kind:  kind,
			id:    j.ID,
			label: components.SanitizeText(j.Title),
			desc:  components.SanitizeText(fmt.Sprintf("%s · %s", desc, shortID(j.ID))),
			job:   &job,
		})
	}
	return items
}

func filterEntitiesByQuery(items []api.Entity, query string) []api.Entity {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return items
	}
	out := make([]api.Entity, 0, len(items))
	for _, e := range items {
		name, typ := normalizeEntityNameType(e.Name, e.Type)
		haystack := strings.ToLower(strings.Join([]string{name, typ, e.ID}, " "))
		if strings.Contains(haystack, q) {
			out = append(out, e)
		}
	}
	return out
}

func filterKnowledgeByQuery(items []api.Knowledge, query string) []api.Knowledge {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return items
	}
	out := make([]api.Knowledge, 0, len(items))
	for _, k := range items {
		if strings.Contains(strings.ToLower(k.Name), q) || strings.Contains(strings.ToLower(k.ID), q) {
			out = append(out, k)
		}
	}
	return out
}

func filterJobsByQuery(items []api.Job, query string) []api.Job {
	q := strings.ToLower(strings.TrimSpace(query))
	if q == "" {
		return items
	}
	out := make([]api.Job, 0, len(items))
	for _, j := range items {
		if strings.Contains(strings.ToLower(j.Title), q) || strings.Contains(strings.ToLower(j.ID), q) {
			out = append(out, j)
		}
	}
	return out
}
