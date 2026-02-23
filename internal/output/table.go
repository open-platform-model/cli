// Package output provides terminal output utilities.
package output

import (
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Table renders a kubectl-style plain text table: space-padded columns with no
// border characters. Column widths are computed from the max content width
// (ANSI-aware via lipgloss.Width). Headers are bold cyan. Columns are separated
// by a 3-space gap. The last column is never padded.
type Table struct {
	headers []string
	rows    [][]string
}

// NewTable creates a new plain table with the given column headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
	}
}

// Row adds a data row to the table.
func (t *Table) Row(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

// String renders the table as plain column-aligned text.
func (t *Table) String() string {
	if len(t.headers) == 0 {
		return ""
	}

	// Compute column widths. Headers are plain ASCII; cells may contain ANSI
	// escape codes so we use lipgloss.Width for correct measurement.
	widths := make([]int, len(t.headers))
	for i, h := range t.headers {
		widths[i] = len(h)
	}
	for _, row := range t.rows {
		for i, cell := range row {
			if i < len(widths) {
				if w := lipgloss.Width(cell); w > widths[i] {
					widths[i] = w
				}
			}
		}
	}

	const colGap = "   " // 3-space gap between columns (kubectl convention)
	headerStyle := lipgloss.NewStyle().Bold(true).Foreground(ColorCyan)

	var sb strings.Builder

	// Header row
	for i, h := range t.headers {
		if i > 0 {
			sb.WriteString(colGap)
		}
		sb.WriteString(headerStyle.Render(h))
		if i < len(t.headers)-1 {
			sb.WriteString(strings.Repeat(" ", widths[i]-len(h)))
		}
	}
	sb.WriteString("\n")

	// Data rows
	for _, row := range t.rows {
		for i, cell := range row {
			if i > 0 {
				sb.WriteString(colGap)
			}
			sb.WriteString(cell)
			if i < len(row)-1 && i < len(widths) {
				sb.WriteString(strings.Repeat(" ", widths[i]-lipgloss.Width(cell)))
			}
		}
		sb.WriteString("\n")
	}

	return sb.String()
}

// RenderFileTree renders a file tree with aligned descriptions.
func RenderFileTree(files []FileEntry, alignColumn int) string {
	var result string
	for _, f := range files {
		padding := alignColumn - len(f.Path)
		if padding < 1 {
			padding = 1
		}
		spaces := make([]byte, padding)
		for i := range spaces {
			spaces[i] = ' '
		}
		result += f.Path + string(spaces) + f.Description + "\n"
	}
	return result
}

// FileEntry represents a file in a tree listing.
type FileEntry struct {
	Path        string
	Description string
}
