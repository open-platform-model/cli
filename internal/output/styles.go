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

// FormatFQN formats a fully qualified name for display by replacing the
// first "#" (provider#path separator) with " - " for readability.
// Any "#" inside the path is preserved.
//
// Example: "kubernetes#deployment-transformer" → "kubernetes - deployment-transformer"
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

// FormatTransformerMatchVerbose renders a matched transformer line with reason.
//
// Format:
//
//	▸ <component> ← <provider> - <fqn>
//	     <reason>
//
// The first line is identical to FormatTransformerMatch. The reason is indented
// and dim-styled on the second line.
func FormatTransformerMatchVerbose(component, fqn, reason string) string {
	firstLine := FormatTransformerMatch(component, fqn)
	if reason == "" {
		return firstLine
	}
	indent := "     "
	styledReason := styleDim.Render(reason)
	return firstLine + "\n" + indent + styledReason
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

// FormatValuesValidationError renders a *opmerrors.ValuesValidationError as a
// colored, human-readable string for terminal output.
//
// Output format:
//
//	<file>: N errors                              ← file group header
//
//	  <line>:<col>  <path>                        ← yellow loc, bold path
//	               <message>                      ← indented below path
//
//	Cross-file conflicts: N conflicts             ← section header (if any)
//
//	  <path>                                      ← bold path
//	    <file>:<line>:<col> vs <file>:<line>:<col> ← yellow locations
//	    <message>                                 ← indented below
func FormatValuesValidationError(e *opmerrors.ValuesValidationError) string {
	styleLoc := lipgloss.NewStyle().Foreground(ColorYellow)
	stylePath := lipgloss.NewStyle().Bold(true)
	styleHeader := lipgloss.NewStyle().Bold(true)

	var sb strings.Builder

	// Group FieldErrors by file, preserving insertion order.
	type fileGroup struct {
		file   string
		errors []opmerrors.FieldError
	}
	var groups []fileGroup
	groupIdx := make(map[string]int)
	for _, fe := range e.Errors {
		file := fe.File
		if file == "" {
			file = "(unknown)"
		}
		idx, ok := groupIdx[file]
		if !ok {
			idx = len(groups)
			groups = append(groups, fileGroup{file: file})
			groupIdx[file] = idx
		}
		groups[idx].errors = append(groups[idx].errors, fe)
	}

	// Render per-file groups.
	for gi, g := range groups {
		if gi > 0 {
			sb.WriteByte('\n')
		}
		n := len(g.errors)
		noun := "errors"
		if n == 1 {
			noun = "error"
		}
		sb.WriteString(styleHeader.Render(fmt.Sprintf("%s: %d %s", g.file, n, noun)))
		sb.WriteByte('\n')
		for _, fe := range g.errors {
			sb.WriteByte('\n')
			if fe.Line > 0 {
				loc := styleLoc.Render(fmt.Sprintf("%d:%d", fe.Line, fe.Column))
				path := stylePath.Render(fe.Path)
				sb.WriteString(fmt.Sprintf("  %s  %s\n", loc, path))
			} else if fe.Path != "" {
				sb.WriteString("  ")
				sb.WriteString(stylePath.Render(fe.Path))
				sb.WriteByte('\n')
			}
			sb.WriteString("        ")
			sb.WriteString(fe.Message)
			sb.WriteByte('\n')
		}
	}

	// Render cross-file conflicts section.
	if len(e.Conflicts) > 0 {
		if len(groups) > 0 {
			sb.WriteByte('\n')
		}
		n := len(e.Conflicts)
		noun := "conflicts"
		if n == 1 {
			noun = "conflict"
		}
		sb.WriteString(styleHeader.Render(fmt.Sprintf("Cross-file conflicts: %d %s", n, noun)))
		sb.WriteByte('\n')
		for _, ce := range e.Conflicts {
			sb.WriteByte('\n')
			sb.WriteString("  ")
			sb.WriteString(stylePath.Render(ce.Path))
			sb.WriteByte('\n')
			// Render all locations joined with " vs "
			locParts := make([]string, 0, len(ce.Locations))
			for _, l := range ce.Locations {
				locParts = append(locParts, styleLoc.Render(fmt.Sprintf("%s:%d:%d", l.File, l.Line, l.Column)))
			}
			sb.WriteString("    ")
			sb.WriteString(strings.Join(locParts, " vs "))
			sb.WriteByte('\n')
			sb.WriteString("    ")
			sb.WriteString(ce.Message)
			sb.WriteByte('\n')
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
