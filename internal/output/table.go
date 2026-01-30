// Package output provides terminal output utilities.
package output

import (
	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
)

// TableStyle defines the style for table output.
type TableStyle struct {
	// Border is the border style.
	Border lipgloss.Border

	// BorderColor is the color for borders.
	BorderColor lipgloss.Color

	// HeaderStyle is the style for header cells.
	HeaderStyle lipgloss.Style

	// CellStyle is the style for regular cells.
	CellStyle lipgloss.Style
}

// DefaultTableStyle returns the default table style.
func DefaultTableStyle() TableStyle {
	return TableStyle{
		Border:      lipgloss.NormalBorder(),
		BorderColor: lipgloss.Color("240"),
		HeaderStyle: lipgloss.NewStyle().Bold(true).Foreground(lipgloss.Color("12")),
		CellStyle:   lipgloss.NewStyle(),
	}
}

// Table represents a styled table.
type Table struct {
	headers []string
	rows    [][]string
	style   TableStyle
}

// NewTable creates a new table with the given headers.
func NewTable(headers ...string) *Table {
	return &Table{
		headers: headers,
		rows:    make([][]string, 0),
		style:   DefaultTableStyle(),
	}
}

// Row adds a row to the table.
func (t *Table) Row(cells ...string) *Table {
	t.rows = append(t.rows, cells)
	return t
}

// SetStyle sets the table style.
func (t *Table) SetStyle(style TableStyle) *Table {
	t.style = style
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

// RenderStatusTable renders a status table for resources.
func RenderStatusTable(resources []ResourceStatus) string {
	t := NewTable("KIND", "NAME", "STATUS", "AGE", "MESSAGE")

	for _, r := range resources {
		t.Row(r.Kind, r.Name, r.Status, r.Age, r.Message)
	}

	return t.String()
}

// ResourceStatus represents the status of a Kubernetes resource.
type ResourceStatus struct {
	Kind    string
	Name    string
	Status  string
	Age     string
	Message string
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
