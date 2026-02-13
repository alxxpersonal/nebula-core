package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

var dialogStyle = lipgloss.NewStyle().
	Border(lipgloss.RoundedBorder()).
	BorderForeground(lipgloss.Color("#273540")).
	Padding(1, 2).
	Width(40)

// ConfirmDialog renders a yes/no confirmation.
func ConfirmDialog(title, message string) string {
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7f57b4")).
		Bold(true).
		Render(title)

	body := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ba0bf")).
		Render(message)

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ba0bf")).
		Render("\ny: confirm | n: cancel")

	return dialogStyle.Render(header + "\n\n" + body + hint)
}

// InputDialog renders a text input prompt.
func InputDialog(title, input string) string {
	header := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#7f57b4")).
		Bold(true).
		Render(title)

	field := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#436b77")).
		Render("> " + input + "█")

	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ba0bf")).
		Render("\nenter: submit | esc: cancel")

	return dialogStyle.Render(header + "\n\n" + field + hint)
}

// ConfirmPreviewDialog renders a confirmation with summary rows and optional diffs.
func ConfirmPreviewDialog(title string, summary []TableRow, diffs []DiffRow, width int) string {
	sections := make([]string, 0, 4)
	if len(summary) > 0 {
		sections = append(sections, Table("Summary", summary, width))
	}
	if len(diffs) > 0 {
		sections = append(sections, DiffTable("Changes", diffs, width))
	}
	hint := lipgloss.NewStyle().
		Foreground(lipgloss.Color("#9ba0bf")).
		Render("y: confirm | n: cancel")
	sections = append(sections, hint)

	return TitledBox(title, strings.Join(sections, "\n\n"), width)
}
