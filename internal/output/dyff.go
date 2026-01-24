package output

// Dyff integration utilities for YAML-aware diffing.
// The main dyff integration is in internal/kubernetes/diff.go to avoid import cycles.
// This file provides any output-specific utilities for diff rendering.

import (
	"strings"
)

// RenderDiffResult renders a kubernetes.DiffResult with styles.
// This is kept separate to avoid import cycles with kubernetes package.
type DiffRenderer struct {
	styles *Styles
}

// NewDiffRenderer creates a new DiffRenderer with default styles.
func NewDiffRenderer() *DiffRenderer {
	return &DiffRenderer{
		styles: GetStyles(),
	}
}

// NewDiffRendererWithStyles creates a DiffRenderer with custom styles.
func NewDiffRendererWithStyles(styles *Styles) *DiffRenderer {
	return &DiffRenderer{
		styles: styles,
	}
}

// RenderAdded renders an added resource line.
func (r *DiffRenderer) RenderAdded(name string) string {
	return "  + " + r.styles.Success.Render(name)
}

// RenderRemoved renders a removed resource line.
func (r *DiffRenderer) RenderRemoved(name string) string {
	return "  - " + r.styles.Error.Render(name)
}

// RenderModified renders a modified resource header.
func (r *DiffRenderer) RenderModified(name string) string {
	return "  ~ " + r.styles.Warning.Render(name)
}

// RenderAddedHeader renders the "Added:" section header.
func (r *DiffRenderer) RenderAddedHeader() string {
	return r.styles.Success.Render("Added:")
}

// RenderRemovedHeader renders the "Removed:" section header.
func (r *DiffRenderer) RenderRemovedHeader() string {
	return r.styles.Error.Render("Removed:")
}

// RenderModifiedHeader renders the "Modified:" section header.
func (r *DiffRenderer) RenderModifiedHeader() string {
	return r.styles.Warning.Render("Modified:")
}

// IndentDiff indents a diff string for display under a resource name.
func IndentDiff(diff string, indent string) string {
	if diff == "" {
		return ""
	}

	var sb strings.Builder
	lines := strings.Split(diff, "\n")
	for _, line := range lines {
		if line != "" {
			sb.WriteString(indent)
			sb.WriteString(line)
			sb.WriteString("\n")
		}
	}
	return sb.String()
}
