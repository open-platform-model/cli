package output

import (
	"strings"
)

// RenderDiff renders a diff result using the provided data.
// This function exists to avoid import cycles - it takes raw data rather than
// importing kubernetes.DiffResult directly.
func RenderDiff(added, removed []string, modified []ModifiedItem, styles *Styles) string {
	if len(added) == 0 && len(removed) == 0 && len(modified) == 0 {
		return "No changes detected."
	}

	var sb strings.Builder

	// Render added resources
	if len(added) > 0 {
		sb.WriteString(styles.Success.Render("Added:"))
		sb.WriteString("\n")
		for _, name := range added {
			sb.WriteString("  + ")
			sb.WriteString(styles.Success.Render(name))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Render removed resources
	if len(removed) > 0 {
		sb.WriteString(styles.Error.Render("Removed:"))
		sb.WriteString("\n")
		for _, name := range removed {
			sb.WriteString("  - ")
			sb.WriteString(styles.Error.Render(name))
			sb.WriteString("\n")
		}
		sb.WriteString("\n")
	}

	// Render modified resources
	if len(modified) > 0 {
		sb.WriteString(styles.Warning.Render("Modified:"))
		sb.WriteString("\n")
		for _, mod := range modified {
			sb.WriteString("  ~ ")
			sb.WriteString(styles.Warning.Render(mod.Name))
			sb.WriteString("\n")
			if mod.Diff != "" {
				// Indent the diff output
				lines := strings.Split(mod.Diff, "\n")
				for _, line := range lines {
					if line != "" {
						sb.WriteString("    ")
						sb.WriteString(line)
						sb.WriteString("\n")
					}
				}
			}
			sb.WriteString("\n")
		}
	}

	// Add summary
	sb.WriteString("Summary: ")
	sb.WriteString(diffSummary(len(added), len(removed), len(modified)))
	sb.WriteString("\n")

	return sb.String()
}

// ModifiedItem represents a modified resource for rendering.
type ModifiedItem struct {
	Name string
	Diff string
}

// diffSummary returns a summary string of changes.
func diffSummary(added, removed, modified int) string {
	if added == 0 && removed == 0 && modified == 0 {
		return "No changes"
	}

	parts := make([]string, 0, 3)
	if added > 0 {
		parts = append(parts, pluralize(added, "added"))
	}
	if removed > 0 {
		parts = append(parts, pluralize(removed, "removed"))
	}
	if modified > 0 {
		parts = append(parts, pluralize(modified, "modified"))
	}

	return strings.Join(parts, ", ")
}

// pluralize returns "N item" or "N items" appropriately.
func pluralize(count int, label string) string {
	return strings.Join([]string{itoa(count), label}, " ")
}

// itoa converts an int to a string without importing strconv.
func itoa(n int) string {
	if n == 0 {
		return "0"
	}

	var negative bool
	if n < 0 {
		negative = true
		n = -n
	}

	var digits []byte
	for n > 0 {
		digits = append([]byte{byte('0' + n%10)}, digits...)
		n /= 10
	}

	if negative {
		digits = append([]byte{'-'}, digits...)
	}

	return string(digits)
}
