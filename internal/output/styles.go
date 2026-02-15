package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"
)

// Color palette — named constants for all ANSI 256 colors used in the CLI.
// These are the single source of truth; never use inline lipgloss.Color literals.
var (
	// ColorCyan is used for identifiable nouns: module paths, resource names, namespaces.
	// Exported for use in build package error formatting.
	ColorCyan = lipgloss.Color("14")

	// colorGreen is used for the "created" resource status (bright, high-visibility).
	colorGreen = lipgloss.Color("82")

	// ColorYellow is used for the "configured" resource status and position markers (line:col).
	// Exported for use in build package error formatting.
	ColorYellow = lipgloss.Color("220")

	// colorRed is used for the "deleted" resource status.
	colorRed = lipgloss.Color("196")

	// colorBoldRed is used for the "failed" resource status (matches ERROR level).
	colorBoldRed = lipgloss.Color("204")

	// colorGreenCheck is used for the completion checkmark (✔).
	colorGreenCheck = lipgloss.Color("10")

	// colorDimGray is used for borders and other structural chrome.
	colorDimGray = lipgloss.Color("240")
)

// Semantic styles — map domain concepts to visual presentation.
var (
	// styleNoun styles identifiable nouns (module paths, resource names, namespaces).
	styleNoun = lipgloss.NewStyle().Foreground(ColorCyan)

	// styleDim styles structural chrome (scope prefixes, separators, timestamps).
	styleDim = lipgloss.NewStyle().Faint(true)
)

// Resource status constants.
const (
	StatusCreated    = "created"
	StatusConfigured = "configured"
	StatusUnchanged  = "unchanged"
	StatusDeleted    = "deleted"
	StatusValid      = "valid"
	statusFailed     = "failed"
)

// StatusStyle returns the lipgloss style for a given resource status string.
// Unknown statuses return an unstyled default.
func statusStyle(status string) lipgloss.Style {
	switch status {
	case StatusCreated:
		return lipgloss.NewStyle().Foreground(colorGreen)
	case StatusValid:
		return lipgloss.NewStyle().Foreground(colorGreen)
	case StatusConfigured:
		return lipgloss.NewStyle().Foreground(ColorYellow)
	case StatusUnchanged:
		return lipgloss.NewStyle().Faint(true)
	case StatusDeleted:
		return lipgloss.NewStyle().Foreground(colorRed)
	case statusFailed:
		return lipgloss.NewStyle().Bold(true).Foreground(colorBoldRed)
	default:
		return lipgloss.NewStyle()
	}
}

// minResourceColumnWidth is the minimum width for the resource path column
// before the status suffix. This ensures status words align consistently.
const minResourceColumnWidth = 48

// FormatResourceLine renders a resource identifier with a right-aligned,
// color-coded status suffix.
//
// Format: r:<Kind/namespace/name>  <status>
// For cluster-scoped resources (empty namespace): r:<Kind/name>
//
// The "r:" prefix is dim, the path is cyan, and the status uses StatusStyle.
func FormatResourceLine(kind, namespace, name, status string) string {
	// Build the resource path
	var path string
	if namespace != "" {
		path = fmt.Sprintf("%s/%s/%s", kind, namespace, name)
	} else {
		path = fmt.Sprintf("%s/%s", kind, name)
	}

	// Calculate padding for right-alignment
	padding := minResourceColumnWidth - len(path)
	if padding < 2 {
		padding = 2
	}

	// Render styled components
	prefix := styleDim.Render("r:")
	styledPath := styleNoun.Render(path)
	styledStatus := statusStyle(status).Render(status)

	return prefix + styledPath + strings.Repeat(" ", padding) + styledStatus
}

// FormatCheckmark renders a green checkmark with a message for stdout output.
func FormatCheckmark(msg string) string {
	check := lipgloss.NewStyle().Foreground(colorGreenCheck).Render("✔")
	return check + " " + msg
}

// FormatNotice renders a yellow arrow with a message for action-required output.
// Use this for "next steps" guidance where user action is needed.
func FormatNotice(msg string) string {
	arrow := lipgloss.NewStyle().Foreground(ColorYellow).Render("▶")
	return arrow + " " + msg
}

// FormatFQN formats a fully qualified name for display by replacing the
// first "#" (provider#path separator) with " - " for readability.
// Any "#" inside the path (e.g. @v0#TransformerName) is preserved.
//
// Example: "kubernetes#opmodel.dev/...@v0#Name" → "kubernetes - opmodel.dev/...@v0#Name"
func FormatFQN(fqn string) string {
	return strings.Replace(fqn, "#", " - ", 1)
}

// FormatTransformerMatch renders a matched transformer line.
//
// Format: ▸ <component> ← <provider> - <fqn>
//
// The bullet and component name are cyan. The arrow and FQN are dim.
func FormatTransformerMatch(component, fqn string) string {
	bullet := styleNoun.Render("▸")
	comp := styleNoun.Render(component)
	arrow := styleDim.Render("←")
	styledFQN := styleDim.Render(FormatFQN(fqn))
	return bullet + " " + comp + " " + arrow + " " + styledFQN
}

// FormatTransformerUnmatched renders an unmatched component line.
//
// Format: ▸ <component> (no matching transformer)
//
// The bullet is yellow. The component name is unstyled. The parenthetical is dim.
func FormatTransformerUnmatched(component string) string {
	bullet := lipgloss.NewStyle().Foreground(ColorYellow).Render("▸")
	detail := styleDim.Render("(no matching transformer)")
	return bullet + " " + component + " " + detail
}

// vetCheckColumnWidth is the alignment column for detail text in FormatVetCheck.
const vetCheckColumnWidth = 34

// FormatVetCheck renders a validation check result with a green checkmark, label,
// and optional right-aligned detail text.
//
// Format: ✔ <label>                      <detail>
//
// The checkmark is green. The detail text (if provided) is dim/faint and
// right-aligned at column 34 from the start of the label. If detail is empty,
// no trailing whitespace is added.
func FormatVetCheck(label, detail string) string {
	check := lipgloss.NewStyle().Foreground(colorGreenCheck).Render("✔")
	result := check + " " + label

	if detail != "" {
		// Calculate padding for right-alignment
		padding := vetCheckColumnWidth - len(label)
		if padding < 2 {
			padding = 2
		}
		styledDetail := styleDim.Render(detail)
		result += strings.Repeat(" ", padding) + styledDetail
	}

	return result
}
