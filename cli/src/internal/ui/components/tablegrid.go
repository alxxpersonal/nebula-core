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
	sep := border.Left
	if sep == "" {
		sep = "|"
	}

	// Build header and row lines.
	var out []string
	out = append(out, renderGridRow(columns, headerCells(columns), sep, tableWidth, true))
	out = append(out, renderGridRule(columns, border.Top, border.Middle, sep, tableWidth))

	for i, row := range rows {
		out = append(out, renderGridRow(columns, row, sep, tableWidth, false))
		if i < len(rows)-1 {
			out = append(out, renderGridRule(columns, border.Top, border.Middle, sep, tableWidth))
		}
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

func renderGridRow(columns []TableColumn, cells []string, sep string, tableWidth int, header bool) string {
	headerStyle := boxLabelStyle
	if header {
		headerStyle = boxLabelStyle.Bold(true)
	}

	var b strings.Builder
	for i, col := range columns {
		if i > 0 {
			b.WriteString(sep)
		}
		w := col.Width
		if w < 1 {
			w = 1
		}

		text := ""
		if i < len(cells) {
			text = cells[i]
		}
		// Caller should sanitize user-provided strings. We still ensure a single line.
		text = SanitizeOneLine(text)

		style := lipgloss.NewStyle().Width(w).MaxWidth(w).Align(col.Align)
		rendered := style.Render(text)
		if header {
			rendered = headerStyle.Render(rendered)
		}
		b.WriteString(rendered)
	}

	line := b.String()
	if lipgloss.Width(line) < tableWidth {
		line = padRight(line, tableWidth)
	} else if lipgloss.Width(line) > tableWidth {
		line = truncateRunes(line, tableWidth)
	}
	return line
}

func renderGridRule(columns []TableColumn, horiz, cross, sep string, tableWidth int) string {
	if tableWidth <= 0 {
		return ""
	}
	if horiz == "" {
		horiz = "-"
	}
	if cross == "" {
		cross = "+"
	}
	if sep == "" {
		sep = "|"
	}

	// Build a full-width rule line. Put a cross at each column boundary.
	// Boundaries are at cumulative column widths plus separator widths.
	boundaries := map[int]struct{}{}
	pos := 0
	for i, col := range columns {
		w := col.Width
		if w < 1 {
			w = 1
		}
		pos += w
		if i < len(columns)-1 {
			boundaries[pos] = struct{}{}
			pos += lipgloss.Width(sep)
		}
	}

	var b strings.Builder
	// Ensure we always render exactly tableWidth cells.
	for i := 0; i < tableWidth; i++ {
		if _, ok := boundaries[i]; ok {
			b.WriteString(cross)
			continue
		}
		b.WriteString(horiz)
	}
	line := b.String()
	if lipgloss.Width(line) < tableWidth {
		return padRight(line, tableWidth)
	}
	return truncateRunes(line, tableWidth)
}
