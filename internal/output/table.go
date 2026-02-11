// Package output provides terminal output utilities.
package output

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// tableStyle defines the style for table output.
type tableStyle struct {
	// Border is the border style.
	Border lipgloss.Border

	// BorderColor is the color for borders.
	BorderColor lipgloss.Color

	// HeaderStyle is the style for header cells.
	HeaderStyle lipgloss.Style

	// CellStyle is the style for regular cells.
	CellStyle lipgloss.Style
}

// defaultTableStyle returns the default table style.
func defaultTableStyle() tableStyle {
	return tableStyle{
		Border:      lipgloss.NormalBorder(),
		BorderColor: colorDimGray,
		HeaderStyle: lipgloss.NewStyle().Bold(true).Foreground(colorCyan),
		CellStyle:   lipgloss.NewStyle(),
	}
}

// Table represents a styled table.
type Table struct {
	headers []string
	rows    [][]string
	style   tableStyle
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		style:   defaultTableStyle(),
	}
}

// Row adds a row to the table.
func (t *Table) Row(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

// String renders the table as a string.
func (t *Table) String() string {
	tbl := table.New().
		Border(t.style.Border).
		BorderStyle(lipgloss.NewStyle().Foreground(t.style.BorderColor)).
		Headers(t.headers...).
		StyleFunc(func(row, col int) lipgloss.Style {
			if row == table.HeaderRow {
				return t.style.HeaderStyle
			}
			return t.style.CellStyle
		})

	for _, row := range t.rows {
		tbl.Row(row...)
	}

	return tbl.String()
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
