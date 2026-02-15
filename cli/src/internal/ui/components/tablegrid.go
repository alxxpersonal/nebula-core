package components

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// TableColumn defines a single column for TableGrid.
//
// Width is the visual width of the column content (excluding separators).
// Align controls how cell text is aligned within the column.
type TableColumn struct {
	Header string
	Width  int
	Align  lipgloss.Position
}

var gridLineStyle = lipgloss.NewStyle().
	Foreground(lipgloss.Color("#273540"))

// TableGrid renders a table-like layout using the same rounded border glyphs
// used by Nebula's box components.
//
// The returned string has a visual width equal to tableWidth.
// Callers should pass a tableWidth that fits inside a box content area
// (typically components.BoxContentWidth(termWidth)).
func TableGrid(columns []TableColumn, rows [][]string, tableWidth int) string {
	if tableWidth <= 0 {
		return ""
	}
	if len(columns) == 0 {
		return padRight("", tableWidth)
	}

	border := lipgloss.RoundedBorder()
	v := border.Left
	if v == "" {
		v = "|"
	}
	h := border.Top
	if h == "" {
		h = "-"
	}
	cross := border.Middle
	if cross == "" {
		cross = "+"
	}

	cols := fitGridColumns(columns, v, tableWidth)

	// Build header and row lines.
	var out []string
	out = append(out, renderGridRow(cols, headerCells(cols), v, tableWidth, true))
	out = append(out, renderGridRule(cols, cross, h, tableWidth))

	for _, row := range rows {
		out = append(out, renderGridRow(cols, row, v, tableWidth, false))
	}

	return strings.Join(out, "\n")
}

func headerCells(columns []TableColumn) []string {
	hdr := make([]string, len(columns))
	for i, c := range columns {
		hdr[i] = SanitizeOneLine(c.Header)
	}
	return hdr
}

func fitGridColumns(columns []TableColumn, sep string, tableWidth int) []TableColumn {
	fitted := make([]TableColumn, len(columns))
	copy(fitted, columns)

	sepW := lipgloss.Width(sep)
	if sepW < 1 {
		sepW = 1
	}

	sum := 0
	for i := range fitted {
		if fitted[i].Width < 1 {
			fitted[i].Width = 1
		}
		sum += fitted[i].Width
	}
	// n columns => n-1 separators (only between columns, no outer table border).
	expected := sum
	if len(fitted) > 1 {
		expected += (len(fitted) - 1) * sepW
	}
	delta := tableWidth - expected
	if len(fitted) > 0 && delta != 0 {
		fitted[len(fitted)-1].Width += delta
		if fitted[len(fitted)-1].Width < 1 {
			fitted[len(fitted)-1].Width = 1
		}
	}
	return fitted
}

func renderGridRow(columns []TableColumn, cells []string, sep string, tableWidth int, header bool) string {
	headerStyle := boxLabelStyle
	if header {
		headerStyle = boxLabelStyle.Bold(true)
	}

	sepStyled := gridLineStyle.Inline(true).Render(sep)

	var b strings.Builder
	for i, col := range columns {
		if i > 0 {
			b.WriteString(sepStyled)
		}
		w := col.Width
		text := ""
		if i < len(cells) {
			text = cells[i]
		}

		rendered := renderGridCell(text, w, col.Align)
		if header {
			// Inline keeps this cell as exactly one rendered line.
			rendered = headerStyle.Inline(true).Render(rendered)
		}
		b.WriteString(rendered)
	}

	line := b.String()
	if lipgloss.Width(line) < tableWidth {
		line = padRight(line, tableWidth)
	}
	return line
}

func renderGridRule(columns []TableColumn, cross, horiz string, tableWidth int) string {
	if horiz == "" {
		horiz = "-"
	}
	var b strings.Builder
	for i, col := range columns {
		w := col.Width
		if w < 1 {
			w = 1
		}
		b.WriteString(strings.Repeat(horiz, w))
		if i < len(columns)-1 {
			b.WriteString(cross)
		}
	}
	line := b.String()
	if lipgloss.Width(line) < tableWidth {
		line = padRight(line, tableWidth)
	}
	return gridLineStyle.Inline(true).Render(line)
}

func renderGridCell(text string, width int, align lipgloss.Position) string {
	if width <= 0 {
		return ""
	}

	clamped := ClampTextWidth(text, width)
	w := lipgloss.Width(clamped)
	if w >= width {
		return truncateRunes(clamped, width)
	}

	pad := width - w
	switch align {
	case lipgloss.Right:
		return strings.Repeat(" ", pad) + clamped
	case lipgloss.Center:
		left := pad / 2
		right := pad - left
		return strings.Repeat(" ", left) + clamped + strings.Repeat(" ", right)
	default:
		return clamped + strings.Repeat(" ", pad)
	}
}
