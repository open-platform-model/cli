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
	ColorCyan = lipgloss.Color("14")

	// ColorGreen is used for the "created" resource status (bright, high-visibility).
	ColorGreen = lipgloss.Color("82")

	// ColorYellow is used for the "configured" resource status (medium visibility).
	ColorYellow = lipgloss.Color("220")

	// ColorRed is used for the "deleted" resource status.
	ColorRed = lipgloss.Color("196")

	// ColorBoldRed is used for the "failed" resource status (matches ERROR level).
	ColorBoldRed = lipgloss.Color("204")

	// ColorGreenCheck is used for the completion checkmark (✔).
	ColorGreenCheck = lipgloss.Color("10")

	// ColorDimGray is used for borders and other structural chrome.
	ColorDimGray = lipgloss.Color("240")
)

// Semantic styles — map domain concepts to visual presentation.
var (
	// StyleNoun styles identifiable nouns (module paths, resource names, namespaces).
	StyleNoun = lipgloss.NewStyle().Foreground(ColorCyan)

	// StyleAction styles action verbs (applying, installing, upgrading, deleting).
	StyleAction = lipgloss.NewStyle().Bold(true)

	// StyleDim styles structural chrome (scope prefixes, separators, timestamps).
	StyleDim = lipgloss.NewStyle().Faint(true)

	// StyleSummary styles completion and summary lines.
	StyleSummary = lipgloss.NewStyle().Bold(true)
)

// Resource status constants.
const (
	StatusCreated    = "created"
	StatusConfigured = "configured"
	StatusUnchanged  = "unchanged"
	StatusDeleted    = "deleted"
	StatusFailed     = "failed"
)

// StatusStyle returns the lipgloss style for a given resource status string.
// Unknown statuses return an unstyled default.
func StatusStyle(status string) lipgloss.Style {
	switch status {
	case StatusCreated:
		return lipgloss.NewStyle().Foreground(ColorGreen)
	case StatusConfigured:
		return lipgloss.NewStyle().Foreground(ColorYellow)
	case StatusUnchanged:
		return lipgloss.NewStyle().Faint(true)
	case StatusDeleted:
		return lipgloss.NewStyle().Foreground(ColorRed)
	case StatusFailed:
		return lipgloss.NewStyle().Bold(true).Foreground(ColorBoldRed)
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
	prefix := StyleDim.Render("r:")
	styledPath := StyleNoun.Render(path)
	styledStatus := StatusStyle(status).Render(status)

	return prefix + styledPath + strings.Repeat(" ", padding) + styledStatus
}

// FormatCheckmark renders a green checkmark with a message for stdout output.
func FormatCheckmark(msg string) string {
	check := lipgloss.NewStyle().Foreground(ColorGreenCheck).Render("✔")
	return check + " " + msg
}
