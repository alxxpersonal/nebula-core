package components

import "github.com/charmbracelet/lipgloss"

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
