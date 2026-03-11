package output

import (
	"fmt"
	"strings"

	"github.com/charmbracelet/lipgloss"

	opmerrors "github.com/opmodel/cli/pkg/errors"
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
)

// Semantic styles — map domain concepts to visual presentation.
var (
	// styleNoun styles identifiable nouns (module paths, resource names, namespaces).
	styleNoun = lipgloss.NewStyle().Foreground(ColorCyan)

	// styleTransformer styles transformer names in match output.
	styleTransformer = lipgloss.NewStyle().Foreground(ColorYellow)

	// styleDim styles structural chrome (scope prefixes, separators, timestamps).
	styleDim = lipgloss.NewStyle().Faint(true)

	// styledGreenCheck is the pre-rendered green checkmark glyph (✔).
	styledGreenCheck = lipgloss.NewStyle().Foreground(colorGreenCheck).Render("✔")
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

// statusIcon returns the icon prefix for a resource status.
// Validation and apply statuses use distinct icon vocabularies:
//   - Validation: ✓ (checkmark — "validation passed")
//   - Apply: diff-style symbols (+, ~, =, -, !)
func statusIcon(status string) string {
	switch status {
	case StatusValid:
		return "✓"
	case StatusCreated:
		return "+"
	case StatusConfigured:
		return "~"
	case StatusUnchanged:
		return "="
	case StatusDeleted:
		return "-"
	case statusFailed:
		return "!"
	default:
		return " "
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
	styledStatus := statusStyle(status).Render(statusIcon(status) + " " + status)

	return prefix + styledPath + strings.Repeat(" ", padding) + styledStatus
}

// FormatCheckmark renders a green checkmark with a message for stdout output.
func FormatCheckmark(msg string) string {
	return styledGreenCheck + " " + msg
}

// FormatNotice renders a yellow arrow with a message for action-required output.
// Use this for "next steps" guidance where user action is needed.
func FormatNotice(msg string) string {
	arrow := lipgloss.NewStyle().Foreground(ColorYellow).Render("▶")
	return arrow + " " + msg
}

// FormatEventType renders an event type label with semantic color.
// Warning is yellow, Normal is dim, and unknown values are left unstyled.
func FormatEventType(eventType string) string {
	switch eventType {
	case "Warning":
		return lipgloss.NewStyle().Foreground(ColorYellow).Render(eventType)
	case "Normal":
		return Dim(eventType)
	default:
		return eventType
	}
}

// FormatEventResource renders an event resource identity in noun style.
func FormatEventResource(kind, name string) string {
	return StyleNoun(kind + "/" + name)
}

// FormatFQN formats a fully qualified name for display by replacing the
// first "#" (provider#path separator) with " - " for readability.
// Any "#" inside the path is preserved.
//
// Example: "kubernetes#deployment-transformer" → "kubernetes - deployment-transformer"
func FormatFQN(fqn string) string {
	return strings.Replace(fqn, "#", " - ", 1)
}

// styledFQN renders a fully-qualified transformer path with the transformer
// name highlighted in the noun style (cyan) and the surrounding path dim.
//
// For module-path FQNs like "opmodel.dev/providers/kubernetes/transformers/hpa-transformer@v1":
//
//	dim("opmodel.dev/providers/kubernetes/transformers/") + cyan("hpa-transformer") + dim("@v1")
//
// For simple "#"-separated FQNs like "kubernetes#statefulset-transformer":
// the "#" is replaced with " - " and the transformer name (after the separator) is cyan.
//
// Falls back to plain dim rendering when no recognizable separator is found.
func styledFQN(fqn string) string {
	// Handle module-path style: last "/" segment is "name@version".
	if idx := strings.LastIndex(fqn, "/"); idx >= 0 {
		prefix := fqn[:idx+1]  // "opmodel.dev/.../transformers/"
		nameVer := fqn[idx+1:] // "hpa-transformer@v1"
		name := nameVer
		ver := ""
		if at := strings.LastIndex(nameVer, "@"); at >= 0 {
			name = nameVer[:at] // "hpa-transformer"
			ver = nameVer[at:]  // "@v1"
		}
		return styleDim.Render(prefix) + styleTransformer.Render(name) + styleDim.Render(ver)
	}

	// Handle "#"-separated style: "provider#transformer-name".
	if idx := strings.Index(fqn, "#"); idx >= 0 {
		provider := fqn[:idx] // "kubernetes"
		name := fqn[idx+1:]   // "statefulset-transformer"
		return styleDim.Render(provider+" - ") + styleTransformer.Render(name)
	}

	// No separator — render the whole thing dim.
	return styleDim.Render(fqn)
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
	return bullet + " " + comp + " " + arrow + " " + styledFQN(fqn)
}

// FormatTransformerSkipped renders a non-matching transformer line for debug output.
//
// Format: ✗ <component> ↛ <provider> - <fqn>
//
// The cross and component name are dim. The arrow and FQN are dim.
// Use with Debug-level logging and structured key-value pairs for the reasons.
func FormatTransformerSkipped(component, fqn string) string {
	cross := styleDim.Render("✗")
	comp := styleDim.Render(component)
	arrow := styleDim.Render("↛")
	return cross + " " + comp + " " + arrow + " " + styledFQN(fqn)
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

// StyleNoun renders a string using the noun style (cyan). Exported for use outside
// the output package (e.g., kubernetes/status.go).
func StyleNoun(s string) string {
	return styleNoun.Render(s)
}

// FormatHealthStatus renders a health status string with the appropriate color.
// Ready/Complete/Bound → green, NotReady/Missing → red, Unknown/Pending/Lost → yellow, others → unstyled.
func FormatHealthStatus(status string) string {
	switch status {
	case "Ready", "Complete", "Bound":
		return lipgloss.NewStyle().Foreground(colorGreen).Render(status)
	case "NotReady", "Missing":
		return lipgloss.NewStyle().Foreground(colorRed).Render(status)
	case "Unknown", "Pending", "Lost":
		return lipgloss.NewStyle().Foreground(ColorYellow).Render(status)
	default:
		return status
	}
}

// FormatComponent renders a component name in cyan. Returns "-" unstyled for empty names.
func FormatComponent(name string) string {
	if name == "" {
		return "-"
	}
	return styleNoun.Render(name)
}

// Dim renders a string in the faint/dim style. Use for supplementary or fallback
// text that should recede visually (e.g. "(ready)", "(not ready)" in verbose output).
func Dim(s string) string {
	return styleDim.Render(s)
}

// FormatPodPhase renders a pod phase string with severity-based color.
//
// Color mapping:
//   - ready=true or Succeeded: green
//   - Running (not ready), Pending, ContainerCreating, PodInitializing, Unknown: yellow
//   - All other phases (CrashLoop, Failed, ImagePullBackOff, ErrImagePull, …): red
//
// The "all other → red" default cleanly covers waiting-reason overrides
// (e.g. CrashLoopBackOff → CrashLoop) without enumerating error states.
func FormatPodPhase(phase string, ready bool) string {
	if ready {
		return lipgloss.NewStyle().Foreground(colorGreen).Render(phase)
	}
	switch phase {
	case "Succeeded":
		return lipgloss.NewStyle().Foreground(colorGreen).Render(phase)
	case "Running", "Pending", "ContainerCreating", "PodInitializing", "Unknown":
		return lipgloss.NewStyle().Foreground(ColorYellow).Render(phase)
	default:
		return lipgloss.NewStyle().Foreground(colorRed).Render(phase)
	}
}

// FormatReadyRatio renders a "(ready/total ready)" string with health-status color.
//
// Color mapping:
//   - all ready (ready == total): green
//   - partially ready (ready > 0): yellow
//   - none ready (ready == 0): red
func FormatReadyRatio(ready, total int) string {
	s := fmt.Sprintf("(%d/%d ready)", ready, total)
	switch {
	case ready == total:
		return lipgloss.NewStyle().Foreground(colorGreen).Render(s)
	case ready > 0:
		return lipgloss.NewStyle().Foreground(ColorYellow).Render(s)
	default:
		return lipgloss.NewStyle().Foreground(colorRed).Render(s)
	}
}

// FormatGroupedErrors renders a slice of GroupedError values as a colored,
// human-readable string for terminal output.
//
// Output format:
//
//	<message>                     ← bold error message
//	  <file>:<line>:<col> -> <path>  ← yellow loc, arrow, then path
//	  <file>:<line>:<col> -> <path>  ← additional locations (e.g. conflict)
//
// Errors with multiple locations (CUE value conflicts) appear as a single
// message block with one line per contributing source position, making
// cross-file conflicts immediately readable without a separate section.
func FormatGroupedErrors(groups []opmerrors.GroupedError) string {
	styleLoc := lipgloss.NewStyle().Foreground(ColorYellow)
	styleMsg := lipgloss.NewStyle().Bold(true)

	var sb strings.Builder

	for gi, g := range groups {
		if gi > 0 {
			sb.WriteByte('\n')
		}
		sb.WriteString(styleMsg.Render(g.Message))
		sb.WriteByte('\n')
		pathPrinted := false
		for _, loc := range g.Locations {
			if !pathPrinted && loc.Path != "" {
				sb.WriteString("  ")
				sb.WriteString(loc.Path)
				sb.WriteByte('\n')
				pathPrinted = true
			}
			if loc.Line > 0 {
				locStr := styleLoc.Render(fmt.Sprintf("%s:%d:%d", loc.File, loc.Line, loc.Column))
				sb.WriteString("    ")
				sb.WriteString(styleDim.Render("> "))
				sb.WriteString(locStr)
				sb.WriteByte('\n')
			}
		}
		if !pathPrinted && len(g.Locations) == 0 {
			sb.WriteString("  values\n")
		}
	}

	return sb.String()
}

// FormatRestartCount renders a restart count suffix with churn-severity color.
// text is the pre-formatted string (e.g. ", 22 restarts").
//
// Color mapping:
//   - 10+ restarts: red (persistent crash loop)
//   - 1–9 restarts: yellow (elevated but not critical)
func FormatRestartCount(count int, text string) string {
	if count >= 10 {
		return lipgloss.NewStyle().Foreground(colorRed).Render(text)
	}
	return lipgloss.NewStyle().Foreground(ColorYellow).Render(text)
}

// FormatVetCheck renders a validation check result with a green checkmark, label,
// and optional right-aligned detail text.
//
// Format: ✔ <label>                      <detail>
//
// The checkmark is green. The detail text (if provided) is dim/faint and
// right-aligned at column 34 from the start of the label. If detail is empty,
// no trailing whitespace is added.
func FormatVetCheck(label, detail string) string {
	result := styledGreenCheck + " " + label

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
