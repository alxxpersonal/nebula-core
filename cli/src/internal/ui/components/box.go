package components

import (
	"encoding/json"
	"fmt"
	"sort"
	"strings"
	"unicode/utf8"

	"github.com/charmbracelet/lipgloss"
)

var (
	boxBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#273540")).
			Padding(1, 2)

	boxBorderActive = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7f57b4")).
			Padding(1, 2)

	boxHeaderStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#7f57b4")).
			Bold(true)

	diffLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#8fb3ff")).
			Bold(true)

	boxMutedStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#9ba0bf"))

	boxValueStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d7d9da"))

	boxLabelStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#436b77")).
			Bold(true)

	errorBorder = lipgloss.NewStyle().
			Border(lipgloss.RoundedBorder()).
			BorderForeground(lipgloss.Color("#7a2f3a")).
			Padding(1, 2)

	errorHeaderStyle = lipgloss.NewStyle().
				Foreground(lipgloss.Color("#e06c75")).
				Bold(true)

	errorBodyStyle = lipgloss.NewStyle().
			Foreground(lipgloss.Color("#d6b5b5"))
)

func boxWidth(width int) int {
	// Use ~70% of terminal width, capped at 80
	w := width * 70 / 100
	if w < 40 {
		w = 40
	}
	if w > 80 {
		w = 80
	}
	return w
}

// Box renders content inside a bordered box.
func Box(content string, width int) string {
	return boxBorder.Width(boxWidth(width)).Render(content)
}

// BoxContentWidth returns the inner content width excluding border and padding.
func BoxContentWidth(width int) int {
	w := boxWidth(width)
	if w <= 0 {
		return 0
	}
	// Border adds 2, padding adds 4 (left+right).
	inner := w - 6
	if inner < 0 {
		return 0
	}
	return inner
}

// ActiveBox renders content inside a highlighted bordered box.
func ActiveBox(content string, width int) string {
	return boxBorderActive.Width(boxWidth(width)).Render(content)
}

// ErrorBox renders a red bordered box for errors.
func ErrorBox(title, message string, width int) string {
	header := ""
	if title != "" {
		header = errorHeaderStyle.Render(title) + "\n\n"
	}
	body := errorBodyStyle.Render(message)
	return errorBorder.Width(boxWidth(width)).Render(header + body)
}

// TitledBox renders a box with a header title.
func TitledBox(title, content string, width int) string {
	return titledBoxWithStyle(title, content, width, boxBorder, boxHeaderStyle, lipgloss.Color("#273540"))
}

func titledBoxWithStyle(title, content string, width int, boxStyle, headerStyle lipgloss.Style, borderColor lipgloss.Color) string {
	if title == "" {
		return boxStyle.Width(boxWidth(width)).Render(content)
	}
	boxed := boxStyle.Width(boxWidth(width)).Render(content)
	lines := strings.Split(boxed, "\n")
	if len(lines) == 0 {
		return boxed
	}

	lineWidth := lipgloss.Width(lines[0])
	if lineWidth < 4 {
		return boxed
	}

	border := lipgloss.RoundedBorder()
	middleLen := lineWidth - 2
	titleText := fmt.Sprintf(" [ %s ] ", title)
	if lipgloss.Width(titleText) > middleLen {
		titleText = truncateRunes(titleText, middleLen)
	}

	titleWidth := lipgloss.Width(titleText)
	left := (middleLen - titleWidth) / 2
	if left < 0 {
		left = 0
	}
	right := middleLen - titleWidth - left
	if right < 0 {
		right = 0
	}

	borderStyle := lipgloss.NewStyle().Foreground(borderColor)
	leftSeg := borderStyle.Render(border.TopLeft + strings.Repeat(border.Top, left))
	rightSeg := borderStyle.Render(strings.Repeat(border.Top, right) + border.TopRight)
	line := leftSeg + headerStyle.Render(titleText) + rightSeg
	if w := lipgloss.Width(line); w < lineWidth {
		line += borderStyle.Render(strings.Repeat(border.Top, lineWidth-w))
	} else if w > lineWidth {
		line = truncateRunes(line, lineWidth)
	}

	lines[0] = line
	return strings.Join(lines, "\n")
}

func truncateRunes(s string, max int) string {
	if max <= 0 {
		return ""
	}
	if utf8.RuneCountInString(s) <= max {
		return s
	}
	var b strings.Builder
	b.Grow(max)
	n := 0
	for _, r := range s {
		if n >= max {
			break
		}
		b.WriteRune(r)
		n++
	}
	return b.String()
}

// InfoRow renders a label: value row for detail views.
func InfoRow(label, value string) string {
	return boxMutedStyle.Render(label+": ") + boxValueStyle.Render(value)
}

// Table renders a key-value table with aligned columns inside a bordered box.
func Table(title string, rows []TableRow, width int) string {
	if len(rows) == 0 {
		return ""
	}

	// Find max label width for alignment
	maxLabel := 0
	for _, r := range rows {
		if len(r.Label) > maxLabel {
			maxLabel = len(r.Label)
		}
	}

	var b strings.Builder
	for i, r := range rows {
		label := boxLabelStyle.Render(fmt.Sprintf("%-*s", maxLabel, r.Label))
		b.WriteString(label + "  " + boxValueStyle.Render(r.Value))
		if i < len(rows)-1 {
			b.WriteString("\n")
		}
	}

	if title != "" {
		return TitledBox(title, b.String(), width)
	}
	return Box(b.String(), width)
}

// TableRow is a single row in a key-value table.
type TableRow struct {
	Label string
	Value string
}

// Indent adds left padding to every line of a multi-line string.
func Indent(s string, spaces int) string {
	pad := strings.Repeat(" ", spaces)
	lines := strings.Split(s, "\n")
	for i, l := range lines {
		lines[i] = pad + l
	}
	return strings.Join(lines, "\n")
}

// CenterLine centers a single line within the standard box width.
func CenterLine(s string, width int) string {
	w := boxWidth(width)
	if w <= 0 {
		return s
	}
	lineWidth := lipgloss.Width(s)
	if lineWidth >= w {
		return s
	}
	pad := (w - lineWidth) / 2
	if pad <= 0 {
		return s
	}
	return strings.Repeat(" ", pad) + s
}

// DiffRow represents a single change with from/to values.
type DiffRow struct {
	Label string
	From  string
	To    string
}

// DiffTable renders a from/to diff table with - (red) and + (yellow) lines.
func DiffTable(title string, rows []DiffRow, width int) string {
	if len(rows) == 0 {
		return ""
	}

	removeStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ff6b6b"))
	addStyle := lipgloss.NewStyle().Foreground(lipgloss.Color("#ffd166"))
	renderValue := func(style lipgloss.Style, prefix string, value string) string {
		if value == "" {
			value = "-"
		}
		lines := strings.Split(value, "\n")
		var out strings.Builder
		for i, line := range lines {
			if i == 0 {
				out.WriteString(style.Render(prefix + line))
			} else {
				out.WriteString(style.Render(strings.Repeat(" ", len(prefix)) + line))
			}
			if i < len(lines)-1 {
				out.WriteString("\n")
			}
		}
		return out.String()
	}

	var b strings.Builder
	for i, r := range rows {
		b.WriteString(diffLabelStyle.Render(r.Label))
		b.WriteString("\n")
		b.WriteString(renderValue(removeStyle, "  - ", r.From))
		b.WriteString("\n")
		b.WriteString(renderValue(addStyle, "  + ", r.To))
		if i < len(rows)-1 {
			b.WriteString("\n\n")
		}
	}

	return TitledBox(title, b.String(), width)
}

// MetadataTable renders a nested metadata map as a bordered table.
func MetadataTable(data map[string]any, width int) string {
	if len(data) == 0 {
		return ""
	}

	lines := renderMetadataLines(data, 0)
	if len(lines) == 0 {
		return ""
	}
	return TitledBox("Metadata", strings.Join(lines, "\n"), width)
}

func renderMetadataLines(data map[string]any, indent int) []string {
	keys := make([]string, 0, len(data))
	for k := range data {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	var lines []string
	pad := strings.Repeat(" ", indent)
	for _, k := range keys {
		switch typed := data[k].(type) {
		case map[string]any:
			lines = append(lines, pad+k+":")
			lines = append(lines, renderMetadataLines(typed, indent+2)...)
		default:
			lines = append(lines, fmt.Sprintf("%s%s: %s", pad, k, formatMetadataValue(typed)))
		}
	}
	return lines
}

func formatMetadataValue(val any) string {
	switch typed := val.(type) {
	case []any:
		if len(typed) == 0 {
			return "[]"
		}
		parts := make([]string, 0, len(typed))
		for _, item := range typed {
			switch sub := item.(type) {
			case map[string]any:
				encoded, err := json.Marshal(sub)
				if err != nil {
					parts = append(parts, fmt.Sprintf("%v", sub))
				} else {
					parts = append(parts, string(encoded))
				}
			default:
				parts = append(parts, fmt.Sprintf("%v", sub))
			}
		}
		return "[" + strings.Join(parts, ", ") + "]"
	case map[string]any:
		encoded, err := json.Marshal(typed)
		if err != nil {
			return fmt.Sprintf("%v", typed)
		}
		return string(encoded)
	default:
		return fmt.Sprintf("%v", typed)
	}
}
