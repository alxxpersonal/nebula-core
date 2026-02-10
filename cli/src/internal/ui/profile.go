package ui

import (
	"fmt"
	"strings"

	tea "github.com/charmbracelet/bubbletea"

	"github.com/gravitrone/nebula-core/cli/internal/api"
	"github.com/gravitrone/nebula-core/cli/internal/config"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

// --- Messages ---

type keysLoadedMsg struct{ items []api.APIKey }
type agentsLoadedMsg struct{ items []api.Agent }
type keyCreatedMsg struct{ resp *api.CreateKeyResponse }
type keyRevokedMsg struct{}
type agentUpdatedMsg struct{}

// --- Profile Model ---

type ProfileModel struct {
	client *api.Client
	config *config.Config

	section int // 0 = keys, 1 = agents

	keys        []api.APIKey
	keyList     *components.List
	agents      []api.Agent
	agentList   *components.List
	agentDetail *api.Agent

	loading    bool
	creating   bool
	createBuf  string
	createdKey string

	width  int
	height int
}

func NewProfileModel(client *api.Client, cfg *config.Config) ProfileModel {
	return ProfileModel{
		client:    client,
		config:    cfg,
		keyList:   components.NewList(10),
		agentList: components.NewList(10),
	}
}

func (m ProfileModel) Init() tea.Cmd {
	m.loading = true
	m.agentDetail = nil
	return tea.Batch(m.loadKeys, m.loadAgents)
}

func (m ProfileModel) Update(msg tea.Msg) (ProfileModel, tea.Cmd) {
	switch msg := msg.(type) {
	case keysLoadedMsg:
		m.keys = msg.items
		labels := make([]string, len(msg.items))
		for i, k := range msg.items {
			labels[i] = formatKeyLine(k)
		}
		m.keyList.SetItems(labels)
		m.loading = false
		return m, nil

	case agentsLoadedMsg:
		m.agents = msg.items
		labels := make([]string, len(msg.items))
		for i, a := range msg.items {
			labels[i] = formatAgentLine(a)
		}
		m.agentList.SetItems(labels)
		return m, nil

	case keyCreatedMsg:
		m.creating = false
		m.createBuf = ""
		m.createdKey = msg.resp.APIKey
		return m, m.loadKeys

	case keyRevokedMsg:
		return m, m.loadKeys

	case agentUpdatedMsg:
		return m, m.loadAgents

	case tea.KeyMsg:
		if m.creating {
			return m.handleCreateInput(msg)
		}

		if m.agentDetail != nil {
			return m.handleAgentDetailKeys(msg)
		}

		if m.createdKey != "" {
			if isBack(msg) || isEnter(msg) {
				m.createdKey = ""
			}
			return m, nil
		}

		switch {
		case isKey(msg, "left"):
			m.section = (m.section - 1 + 2) % 2
		case isKey(msg, "right"):
			m.section = (m.section + 1) % 2
		case isDown(msg):
			m.activeList().Down()
		case isUp(msg):
			m.activeList().Up()
		case isKey(msg, "n"):
			if m.section == 0 {
				m.creating = true
				m.createBuf = ""
			}
		case isKey(msg, "r"):
			if m.section == 0 {
				return m.revokeSelected()
			}
		case isKey(msg, "t"):
			if m.section == 1 {
				return m.toggleTrust()
			}
		case isEnter(msg):
			if m.section == 1 {
				if idx := m.agentList.Selected(); idx < len(m.agents) {
					agent := m.agents[idx]
					m.agentDetail = &agent
				}
			}
		}
	}
	return m, nil
}

func (m ProfileModel) View() string {
	if m.loading {
		return "  " + MutedStyle.Render("Loading profile...")
	}

	if m.creating {
		return components.Indent(components.InputDialog("New Key Name", m.createBuf), 1)
	}

	if m.createdKey != "" {
		return components.Indent(components.ConfirmDialog("Key Created",
			fmt.Sprintf("Save this key, it won't be shown again:\n\n%s\n\nPress Enter to continue.", m.createdKey)), 1)
	}

	if m.agentDetail != nil {
		return m.renderAgentDetail()
	}

	var b strings.Builder

	// User info table
	b.WriteString(components.Indent(components.Table("Profile", []components.TableRow{
		{Label: "User", Value: m.config.Username},
		{Label: "Server", Value: m.config.ServerURL},
	}, m.width), 1))
	b.WriteString("\n\n")

	// Section tabs
	keysLabel := "API Keys"
	agentsLabel := "Agents"
	var tabs string
	if m.section == 0 {
		tabs = SelectedStyle.Render(keysLabel) + "   " + MutedStyle.Render(agentsLabel)
	} else {
		tabs = MutedStyle.Render(keysLabel) + "   " + SelectedStyle.Render(agentsLabel)
	}
	b.WriteString(components.CenterLine(tabs, m.width))
	b.WriteString("\n\n")

	if m.section == 0 {
		b.WriteString(m.renderKeys())
	} else {
		b.WriteString(m.renderAgents())
	}

	return b.String()
}

// --- Helpers ---

func (m *ProfileModel) activeList() *components.List {
	if m.section == 0 {
		return m.keyList
	}
	return m.agentList
}

func (m ProfileModel) loadKeys() tea.Msg {
	items, err := m.client.ListAllKeys()
	if err != nil {
		return errMsg{err}
	}
	return keysLoadedMsg{items}
}

func (m ProfileModel) loadAgents() tea.Msg {
	items, err := m.client.ListAgents("")
	if err != nil {
		return errMsg{err}
	}
	return agentsLoadedMsg{items}
}

func (m ProfileModel) handleCreateInput(msg tea.KeyMsg) (ProfileModel, tea.Cmd) {
	switch {
	case isBack(msg):
		m.creating = false
		m.createBuf = ""
	case isEnter(msg):
		name := m.createBuf
		m.creating = false
		m.createBuf = ""
		return m, func() tea.Msg {
			resp, err := m.client.CreateKey(name)
			if err != nil {
				return errMsg{err}
			}
			return keyCreatedMsg{resp}
		}
	case isKey(msg, "backspace"):
		if len(m.createBuf) > 0 {
			m.createBuf = m.createBuf[:len(m.createBuf)-1]
		}
	default:
		if len(msg.String()) == 1 || msg.String() == " " {
			m.createBuf += msg.String()
		}
	}
	return m, nil
}

func (m ProfileModel) revokeSelected() (ProfileModel, tea.Cmd) {
	if idx := m.keyList.Selected(); idx < len(m.keys) {
		id := m.keys[idx].ID
		return m, func() tea.Msg {
			err := m.client.RevokeKey(id)
			if err != nil {
				return errMsg{err}
			}
			return keyRevokedMsg{}
		}
	}
	return m, nil
}

func (m ProfileModel) toggleTrust() (ProfileModel, tea.Cmd) {
	if idx := m.agentList.Selected(); idx < len(m.agents) {
		agent := m.agents[idx]
		newVal := !agent.RequiresApproval
		return m, func() tea.Msg {
			_, err := m.client.UpdateAgent(agent.ID, api.UpdateAgentInput{
				RequiresApproval: &newVal,
			})
			if err != nil {
				return errMsg{err}
			}
			return agentUpdatedMsg{}
		}
	}
	return m, nil
}

func (m ProfileModel) renderKeys() string {
	if len(m.keys) == 0 {
		return components.Indent(components.Box(MutedStyle.Render("No API keys."), m.width), 1)
	}

	var rows strings.Builder
	visible := m.keyList.Visible()
	for i, label := range visible {
		absIdx := m.keyList.RelToAbs(i)
		if m.section == 0 && m.keyList.IsSelected(absIdx) {
			rows.WriteString(SelectedStyle.Render("  > " + label))
		} else {
			rows.WriteString(NormalStyle.Render("    " + label))
		}
		if i < len(visible)-1 {
			rows.WriteString("\n")
		}
	}

	title := "API Keys"
	countLine := MutedStyle.Render(fmt.Sprintf("%d keys", len(m.keys)))
	content := countLine + "\n\n" + rows.String()
	return components.Indent(components.TitledBox(title, content, m.width), 1)
}

func (m ProfileModel) renderAgents() string {
	if len(m.agents) == 0 {
		return components.Indent(components.Box(MutedStyle.Render("No agents registered."), m.width), 1)
	}

	var rows strings.Builder
	visible := m.agentList.Visible()
	for i, label := range visible {
		absIdx := m.agentList.RelToAbs(i)
		if m.section == 1 && m.agentList.IsSelected(absIdx) {
			rows.WriteString(SelectedStyle.Render("  > " + label))
		} else {
			rows.WriteString(NormalStyle.Render("    " + label))
		}
		if i < len(visible)-1 {
			rows.WriteString("\n")
		}
	}

	title := "Agents"
	countLine := MutedStyle.Render(fmt.Sprintf("%d agents", len(m.agents)))
	content := countLine + "\n\n" + rows.String()
	return components.Indent(components.TitledBox(title, content, m.width), 1)
}

func (m ProfileModel) renderAgentDetail() string {
	if m.agentDetail == nil {
		return ""
	}
	agent := m.agentDetail
	trust := "trusted"
	if agent.RequiresApproval {
		trust = "untrusted"
	}
	scopes := "-"
	if len(agent.Scopes) > 0 {
		scopes = strings.Join(agent.Scopes, ", ")
	}
	caps := "-"
	if len(agent.Capabilities) > 0 {
		caps = strings.Join(agent.Capabilities, ", ")
	}
	rows := []components.TableRow{
		{Label: "ID", Value: agent.ID},
		{Label: "Name", Value: agent.Name},
		{Label: "Status", Value: agent.Status},
		{Label: "Trust", Value: trust},
		{Label: "Scopes", Value: scopes},
		{Label: "Capabilities", Value: caps},
		{Label: "Created", Value: agent.CreatedAt.Format("2006-01-02 15:04")},
		{Label: "Updated", Value: agent.UpdatedAt.Format("2006-01-02 15:04")},
	}
	if agent.Description != nil && strings.TrimSpace(*agent.Description) != "" {
		rows = append(rows, components.TableRow{Label: "Description", Value: strings.TrimSpace(*agent.Description)})
	}
	return components.Indent(components.Table("Agent Details", rows, m.width), 1)
}

func (m ProfileModel) handleAgentDetailKeys(msg tea.KeyMsg) (ProfileModel, tea.Cmd) {
	if isBack(msg) || isEnter(msg) {
		m.agentDetail = nil
	}
	return m, nil
}

func formatKeyLine(k api.APIKey) string {
	prefix := k.KeyPrefix + "..."
	owner := "-"
	if k.EntityName != nil {
		owner = *k.EntityName
	} else if k.AgentName != nil {
		owner = "agent: " + *k.AgentName
	}
	return fmt.Sprintf("%-12s  %-20s  %-5s  %s", prefix, k.Name, k.CreatedAt.Format("01/02"), owner)
}

func formatAgentLine(a api.Agent) string {
	trust := "untrusted"
	if !a.RequiresApproval {
		trust = "trusted"
	}
	return fmt.Sprintf("[%s] %s (%s)", a.Status, a.Name, trust)
}
