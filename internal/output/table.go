package output

import (
	"fmt"
	"strings"
	"time"

	"github.com/charmbracelet/lipgloss"
	"github.com/charmbracelet/lipgloss/table"
	"github.com/opmodel/cli/internal/kubernetes"
)

// StatusTable renders a status table for resources.
type StatusTable struct {
	styles *Styles
}

// NewStatusTable creates a new StatusTable with default styles.
func NewStatusTable() *StatusTable {
	return &StatusTable{
		styles: DefaultStyles(),
	}
}

// RenderModuleStatus renders a module status as a table.
func (t *StatusTable) RenderModuleStatus(status *kubernetes.ModuleStatus) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Module: %s", status.Name))
	if status.Version != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", status.Version))
	}
	sb.WriteString(fmt.Sprintf(" in namespace %s\n", status.Namespace))

	// Summary
	sb.WriteString(t.renderSummary(status.Summary))
	sb.WriteString("\n")

	// Resources table
	if len(status.Resources) > 0 {
		sb.WriteString(t.renderResourcesTable(status.Resources))
	} else {
		sb.WriteString("No resources found.\n")
	}

	return sb.String()
}

// RenderBundleStatus renders a bundle status as a table.
func (t *StatusTable) RenderBundleStatus(status *kubernetes.BundleStatus) string {
	var sb strings.Builder

	// Header
	sb.WriteString(fmt.Sprintf("Bundle: %s", status.Name))
	if status.Version != "" {
		sb.WriteString(fmt.Sprintf(" (%s)", status.Version))
	}
	sb.WriteString(fmt.Sprintf(" in namespace %s\n", status.Namespace))

	// Summary
	sb.WriteString(t.renderSummary(status.Summary))
	sb.WriteString("\n")

	// Resources table
	if len(status.Resources) > 0 {
		sb.WriteString(t.renderResourcesTable(status.Resources))
	} else {
		sb.WriteString("No resources found.\n")
	}

	return sb.String()
}

// renderSummary renders a status summary.
func (t *StatusTable) renderSummary(summary kubernetes.StatusSummary) string {
	ready := t.styles.StatusReady.Render(fmt.Sprintf("%d Ready", summary.Ready))
	parts := []string{
		fmt.Sprintf("Total: %d", summary.Total),
		ready,
	}

	if summary.NotReady > 0 {
		parts = append(parts, t.styles.StatusNotReady.Render(fmt.Sprintf("%d NotReady", summary.NotReady)))
	}
	if summary.Progressing > 0 {
		parts = append(parts, t.styles.StatusProgressing.Render(fmt.Sprintf("%d Progressing", summary.Progressing)))
	}
	if summary.Failed > 0 {
		parts = append(parts, t.styles.StatusFailed.Render(fmt.Sprintf("%d Failed", summary.Failed)))
	}
	if summary.Unknown > 0 {
		parts = append(parts, fmt.Sprintf("%d Unknown", summary.Unknown))
	}

	return strings.Join(parts, " | ")
}

// renderResourcesTable renders the resources as a table.
func (t *StatusTable) renderResourcesTable(resources []kubernetes.ResourceStatus) string {
	tbl := table.New().
		Border(lipgloss.NormalBorder()).
		BorderStyle(t.styles.TableBorder).
		Headers("KIND", "NAME", "NAMESPACE", "STATUS", "AGE", "MESSAGE")

	for _, r := range resources {
		ns := r.Namespace
		if ns == "" {
			ns = "-"
		}

		status := t.renderHealthStatus(r.Health)
		age := formatDuration(r.Age)
		message := truncate(r.Message, 40)

		tbl.Row(r.Kind, r.Name, ns, status, age, message)
	}

	return tbl.String()
}

// renderHealthStatus renders a health status with appropriate styling.
func (t *StatusTable) renderHealthStatus(health kubernetes.HealthStatus) string {
	switch health {
	case kubernetes.HealthReady:
		return t.styles.StatusReady.Render(string(health))
	case kubernetes.HealthNotReady:
		return t.styles.StatusNotReady.Render(string(health))
	case kubernetes.HealthProgressing:
		return t.styles.StatusProgressing.Render(string(health))
	case kubernetes.HealthFailed:
		return t.styles.StatusFailed.Render(string(health))
	default:
		return string(health)
	}
}

// formatDuration formats a duration in a human-readable way.
func formatDuration(d time.Duration) string {
	if d < time.Minute {
		return fmt.Sprintf("%ds", int(d.Seconds()))
	}
	if d < time.Hour {
		return fmt.Sprintf("%dm", int(d.Minutes()))
	}
	if d < 24*time.Hour {
		return fmt.Sprintf("%dh", int(d.Hours()))
	}
	return fmt.Sprintf("%dd", int(d.Hours()/24))
}

// truncate truncates a string to the given length.
func truncate(s string, maxLen int) string {
	if len(s) <= maxLen {
		return s
	}
	return s[:maxLen-3] + "..."
}
