package ui

import (
	"strings"

	tea "github.com/charmbracelet/bubbletea"
	"github.com/gravitrone/nebula-core/cli/internal/ui/components"
)

type MetadataEditor struct {
	Active bool
	Buffer string
}

func (m *MetadataEditor) Open(initial map[string]any) {
	m.Active = true
	m.Buffer = metadataToInput(initial)
}

func (m *MetadataEditor) Reset() {
	m.Active = false
	m.Buffer = ""
}

func (m *MetadataEditor) HandleKey(msg tea.KeyMsg) bool {
	switch {
	case isBack(msg):
		m.Active = false
		return true
	case isKey(msg, "backspace", "delete"):
		m.Buffer = dropLastRune(m.Buffer)
	case isKey(msg, "cmd+backspace", "cmd+delete", "ctrl+u"):
		m.Buffer = ""
	case isEnter(msg):
		m.Buffer += "\n"
	case isKey(msg, "tab"):
		m.Buffer += "  "
	default:
		ch := msg.String()
		if len(ch) == 1 || ch == " " {
			m.Buffer += ch
		}
	}
	return false
}

func (m MetadataEditor) Render(width int) string {
	content := renderMetadataInput(m.Buffer)
	if strings.TrimSpace(content) == "" {
		content = "-"
	}
	content += AccentStyle.Render("█")
	hint := MutedStyle.Render("tab indent  |  enter newline  |  esc back")
	if _, err := parseMetadataInput(m.Buffer); err != nil {
		hint = hint + "\n" + ErrorStyle.Render(err.Error())
	}
	return components.Indent(components.TitledBox("Metadata", content+"\n\n"+hint, width), 1)
}

func dropLastRune(s string) string {
	if s == "" {
		return ""
	}
	runes := []rune(s)
	return string(runes[:len(runes)-1])
}
