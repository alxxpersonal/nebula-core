package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

type taxonomyLoadedMsg struct {
	kind  string
	items []api.TaxonomyEntry
}

type taxonomyActionDoneMsg struct{}

type taxonomyPromptMode int

const (
	taxPromptNone taxonomyPromptMode = iota
	taxPromptCreateName
	taxPromptCreateDescription
	taxPromptEditName
	taxPromptEditDescription
	taxPromptFilter
)

var taxonomyKinds = []struct {
	Label string
	Path  string
}{
	{Label: "Scopes", Path: "scopes"},
	{Label: "Entity Types", Path: "entity-types"},
	{Label: "Relationship Types", Path: "relationship-types"},
	{Label: "Log Types", Path: "log-types"},
}

func (m ProfileModel) taxonomyKindPath() string {
	if m.taxKind < 0 || m.taxKind >= len(taxonomyKinds) {
		return taxonomyKinds[0].Path
	}
	return taxonomyKinds[m.taxKind].Path
}

func (m ProfileModel) loadTaxonomy() tea.Msg {
	kind := m.taxonomyKindPath()
	items, err := m.client.ListTaxonomy(kind, m.taxIncludeInactive, m.taxSearch, 200, 0)
	if err != nil {
		return errMsg{err}
	}
	return taxonomyLoadedMsg{kind: kind, items: items}
}

func (m *ProfileModel) setTaxonomyItems(items []api.TaxonomyEntry) {
	m.taxItems = items
	labels := make([]string, len(items))
	for i, item := range items {
		labels[i] = formatTaxonomyLine(item)
	}
	m.taxList.SetItems(labels)
}

func formatTaxonomyLine(item api.TaxonomyEntry) string {
	name := components.SanitizeOneLine(item.Name)
	parts := []string{name}
	if item.IsBuiltin {
		parts = append(parts, TypeBadgeStyle.Render("builtin"))
	}
	if !item.IsActive {
		parts = append(parts, WarningStyle.Render("inactive"))
	}
	if item.Description != nil && strings.TrimSpace(*item.Description) != "" {
		parts = append(parts, MutedStyle.Render(components.SanitizeOneLine(*item.Description)))
	}
	return strings.Join(parts, "  ")
}

func (m ProfileModel) selectedTaxonomy() *api.TaxonomyEntry {
	if m.taxList == nil {
		return nil
	}
	idx := m.taxList.Selected()
	if idx < 0 || idx >= len(m.taxItems) {
		return nil
	}
	item := m.taxItems[idx]
	return &item
}

func (m *ProfileModel) openTaxPrompt(mode taxonomyPromptMode, defaultValue string) {
	m.taxPromptMode = mode
	m.taxPromptBuf = defaultValue
}

func (m ProfileModel) taxonomyPromptTitle() string {
	switch m.taxPromptMode {
	case taxPromptCreateName:
		return "New Taxonomy Name"
	case taxPromptCreateDescription:
		return "New Taxonomy Description (optional)"
	case taxPromptEditName:
		return "Edit Taxonomy Name"
	case taxPromptEditDescription:
		return "Edit Taxonomy Description (optional)"
	case taxPromptFilter:
		return "Taxonomy Filter"
	default:
		return "Taxonomy"
	}
}

func (m ProfileModel) handleTaxonomyPrompt(msg tea.KeyMsg) (ProfileModel, tea.Cmd) {
	switch {
	case isBack(msg):
		m.taxPromptMode = taxPromptNone
		m.taxPromptBuf = ""
		m.taxPendingName = ""
		m.taxPendingDesc = ""
		m.taxEditID = ""
		return m, nil
	case isEnter(msg):
		return m.submitTaxonomyPrompt()
	case isKey(msg, "backspace"):
		if len(m.taxPromptBuf) > 0 {
			m.taxPromptBuf = m.taxPromptBuf[:len(m.taxPromptBuf)-1]
		}
		return m, nil
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.taxPromptBuf += msg.String()
		}
		return m, nil
	}
}

func (m ProfileModel) submitTaxonomyPrompt() (ProfileModel, tea.Cmd) {
	switch m.taxPromptMode {
	case taxPromptCreateName:
		name := strings.TrimSpace(m.taxPromptBuf)
		if name == "" {
			m.taxPromptMode = taxPromptNone
			m.taxPromptBuf = ""
			return m, func() tea.Msg { return errMsg{fmt.Errorf("taxonomy name required")} }
		}
		m.taxPendingName = name
		m.openTaxPrompt(taxPromptCreateDescription, "")
		return m, nil
	case taxPromptCreateDescription:
		desc := strings.TrimSpace(m.taxPromptBuf)
		input := api.CreateTaxonomyInput{
			Name:        m.taxPendingName,
			Description: desc,
		}
		kind := m.taxonomyKindPath()
		m.taxPromptMode = taxPromptNone
		m.taxPromptBuf = ""
		m.taxPendingName = ""
		m.taxPendingDesc = ""
		m.taxLoading = true
		return m, func() tea.Msg {
			if _, err := m.client.CreateTaxonomy(kind, input); err != nil {
				return errMsg{err}
			}
			return taxonomyActionDoneMsg{}
		}
	case taxPromptEditName:
		name := strings.TrimSpace(m.taxPromptBuf)
		if name == "" {
			m.taxPromptMode = taxPromptNone
			m.taxPromptBuf = ""
			return m, func() tea.Msg { return errMsg{fmt.Errorf("taxonomy name required")} }
		}
		m.taxPendingName = name
		m.openTaxPrompt(taxPromptEditDescription, m.taxPendingDesc)
		return m, nil
	case taxPromptEditDescription:
		name := m.taxPendingName
		desc := strings.TrimSpace(m.taxPromptBuf)
		id := m.taxEditID
		kind := m.taxonomyKindPath()
		m.taxPromptMode = taxPromptNone
		m.taxPromptBuf = ""
		m.taxPendingName = ""
		m.taxPendingDesc = ""
		m.taxEditID = ""
		m.taxLoading = true
		return m, func() tea.Msg {
			_, err := m.client.UpdateTaxonomy(kind, id, api.UpdateTaxonomyInput{
				Name:        &name,
				Description: &desc,
			})
			if err != nil {
				return errMsg{err}
			}
			return taxonomyActionDoneMsg{}
		}
	case taxPromptFilter:
		m.taxSearch = strings.TrimSpace(m.taxPromptBuf)
		m.taxPromptMode = taxPromptNone
		m.taxPromptBuf = ""
		m.taxLoading = true
		return m, m.loadTaxonomy
	default:
		return m, nil
	}
}

func (m ProfileModel) taxonomyArchiveSelected() (ProfileModel, tea.Cmd) {
	item := m.selectedTaxonomy()
	if item == nil {
		return m, nil
	}
	kind := m.taxonomyKindPath()
	m.taxLoading = true
	return m, func() tea.Msg {
		if _, err := m.client.ArchiveTaxonomy(kind, item.ID); err != nil {
			return errMsg{err}
		}
		return taxonomyActionDoneMsg{}
	}
}

func (m ProfileModel) taxonomyActivateSelected() (ProfileModel, tea.Cmd) {
	item := m.selectedTaxonomy()
	if item == nil {
		return m, nil
	}
	kind := m.taxonomyKindPath()
	m.taxLoading = true
	return m, func() tea.Msg {
		if _, err := m.client.ActivateTaxonomy(kind, item.ID); err != nil {
			return errMsg{err}
		}
		return taxonomyActionDoneMsg{}
	}
}

func (m ProfileModel) renderTaxonomy() string {
	var b strings.Builder

	kindTabs := make([]string, 0, len(taxonomyKinds))
	for i, kind := range taxonomyKinds {
		if i == m.taxKind {
			kindTabs = append(kindTabs, SelectedStyle.Render(kind.Label))
		} else {
			kindTabs = append(kindTabs, MutedStyle.Render(kind.Label))
		}
	}
	b.WriteString(components.CenterLine(strings.Join(kindTabs, "   "), m.width))
	b.WriteString("\n\n")

	if m.taxPromptMode != taxPromptNone {
		return b.String() + components.Indent(
			components.InputDialog(m.taxonomyPromptTitle(), m.taxPromptBuf),
			1,
		)
	}

	if m.taxLoading {
		return b.String() + components.Indent(
			components.Box(MutedStyle.Render("Loading taxonomy..."), m.width),
			1,
		)
	}

	if len(m.taxItems) == 0 {
		return b.String() + components.Indent(
			components.Box(MutedStyle.Render("No taxonomy rows found."), m.width),
			1,
		)
	}

	var rows strings.Builder
	visible := m.taxList.Visible()
	contentWidth := components.BoxContentWidth(m.width)
	maxLabelWidth := contentWidth - 4
	for i, label := range visible {
		if maxLabelWidth > 0 {
			label = components.ClampTextWidth(label, maxLabelWidth)
		}
		absIdx := m.taxList.RelToAbs(i)
		if m.taxList.IsSelected(absIdx) {
			rows.WriteString(SelectedStyle.Render("  > " + label))
		} else {
			rows.WriteString(NormalStyle.Render("    " + label))
		}
		if i < len(visible)-1 {
			rows.WriteString("\n")
		}
	}

	filterText := m.taxSearch
	if filterText == "" {
		filterText = "-"
	}
	info := fmt.Sprintf(
		"%d rows  ·  include inactive: %t  ·  filter: %s",
		len(m.taxItems),
		m.taxIncludeInactive,
		filterText,
	)
	content := MutedStyle.Render(info) + "\n\n" + rows.String()
	title := fmt.Sprintf("%s Taxonomy", taxonomyKinds[m.taxKind].Label)
	return b.String() + components.Indent(components.TitledBox(title, content, m.width), 1)
}
