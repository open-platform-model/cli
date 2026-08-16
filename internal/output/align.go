package output

import "strings"

// AlignColumns renders rows of cells as left-aligned columns separated by
// two spaces, each line prefixed by indent. Column widths are computed over
// all rows; the last cell of each row is never padded, so ragged rows do not
// leave trailing whitespace.
//
// This is the publish-refusal evidence form (enhancement 0011,
// 06-operational.md): "disagreements print both values in aligned columns
// with the file that declares each". Table is header-oriented and wrong for
// two-value disagreements, hence this dedicated helper.
func AlignColumns(indent string, rows [][]string) string {
	widths := columnWidths(rows)

	var b strings.Builder
	for i, row := range rows {
		if i > 0 {
			b.WriteString("\n")
		}
		b.WriteString(indent)
		for j, cell := range row {
			if j == len(row)-1 {
				b.WriteString(cell)
				break
			}
			b.WriteString(cell)
			b.WriteString(strings.Repeat(" ", widths[j]-len(cell)+2))
		}
	}
	return b.String()
}

// columnWidths returns the maximum cell width per column index. Cells that
// are the last in their row are excluded — they are never padded, so they
// must not widen the column for other rows either.
func columnWidths(rows [][]string) []int {
	var widths []int
	for _, row := range rows {
		for j, cell := range row {
			if j == len(row)-1 {
				continue
			}
			for len(widths) <= j {
				widths = append(widths, 0)
			}
			if len(cell) > widths[j] {
				widths[j] = len(cell)
			}
		}
	}
	return widths
}
