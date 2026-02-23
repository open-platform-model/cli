package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"gopkg.in/yaml.v3"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/output"
)

// MissingResource identifies a resource that is tracked in the inventory but
// no longer exists on the cluster. Passed from the command layer (which fetches
// the inventory) into GetModuleStatus so that "Missing" entries appear in the output.
type MissingResource struct {
	Kind      string
	Namespace string
	Name      string
}

// StatusOptions configures a status operation.
type StatusOptions struct {
	// Namespace is the target namespace for resource lookup.
	Namespace string

	// ReleaseName is the release name.
	// Mutually exclusive with ReleaseID.
	ReleaseName string

	// ReleaseID is the release identity UUID for discovery.
	// Mutually exclusive with ReleaseName.
	ReleaseID string

	// Version is the module version, sourced from inv.Changes[latest].Source.Version.
	// Empty for local modules.
	Version string

	// ComponentMap maps "Kind/Namespace/Name" to component name, built from inventory entries.
	ComponentMap map[string]string

	// OutputFormat is the desired output format (table, yaml, json, wide).
	OutputFormat output.Format

	// InventoryLive is the list of live resources pre-fetched from the inventory
	// Secret by the caller. When empty or nil, GetReleaseStatus returns
	// noResourcesFoundError (unless MissingResources is also non-empty).
	InventoryLive []*unstructured.Unstructured

	// MissingResources is the list of resources tracked in the inventory that
	// no longer exist on the cluster. These are shown with "Missing" status.
	MissingResources []MissingResource

	// Wide enables extraction of workload-specific wide info (replicas, image).
	Wide bool

	// Verbose enables pod-level diagnostics for unhealthy workloads.
	Verbose bool
}

// resourceHealth contains health information for a single resource.
type resourceHealth struct {
	// Kind is the Kubernetes resource kind.
	Kind string `json:"kind" yaml:"kind"`
	// Name is the resource name.
	Name string `json:"name" yaml:"name"`
	// Namespace is the resource namespace.
	Namespace string `json:"namespace" yaml:"namespace"`
	// Component is the source component name (from inventory).
	Component string `json:"component,omitempty" yaml:"component,omitempty"`
	// Status is the evaluated health status.
	Status healthStatus `json:"status" yaml:"status"`
	// Age is the human-readable age of the resource.
	Age string `json:"age" yaml:"age"`
	// Wide holds extra workload-specific info (replicas, image), populated when Wide mode is on.
	Wide *wideInfo `json:"wide,omitempty" yaml:"wide,omitempty"`
	// Verbose holds pod-level diagnostics, populated when Verbose mode is on.
	Verbose *verboseInfo `json:"verbose,omitempty" yaml:"verbose,omitempty"`
}

// wideInfo holds workload-specific wide-format info extracted from the unstructured resource.
type wideInfo struct {
	Replicas string `json:"replicas,omitempty" yaml:"replicas,omitempty"` // "3/3", "10Gi (Bound)"
	Image    string `json:"image,omitempty" yaml:"image,omitempty"`       // "nginx:1.25", "app.local"
}

// verboseInfo holds pod-level diagnostics for a workload resource.
type verboseInfo struct {
	Pods []podInfo `json:"pods,omitempty" yaml:"pods,omitempty"`
}

// podInfo holds status info for a single pod.
type podInfo struct {
	Name     string `json:"name" yaml:"name"`
	Phase    string `json:"phase" yaml:"phase"` // Running, Pending, Failed
	Ready    bool   `json:"ready" yaml:"ready"`
	Reason   string `json:"reason,omitempty" yaml:"reason,omitempty"` // OOMKilled, ImagePullBackOff
	Restarts int    `json:"restarts" yaml:"restarts"`
}

// statusSummary contains aggregate resource counts.
type statusSummary struct {
	Total    int `json:"total" yaml:"total"`
	Ready    int `json:"ready" yaml:"ready"`
	NotReady int `json:"notReady" yaml:"notReady"`
}

// StatusResult contains the full status output.
type StatusResult struct {
	// ReleaseName is the human-readable release name.
	ReleaseName string `json:"releaseName" yaml:"releaseName"`
	// Version is the module version (empty for local modules).
	Version string `json:"version,omitempty" yaml:"version,omitempty"`
	// Namespace is the Kubernetes namespace.
	Namespace string `json:"namespace" yaml:"namespace"`
	// Resources is the list of resource health statuses.
	Resources []resourceHealth `json:"resources" yaml:"resources"`
	// AggregateStatus is the overall module health.
	AggregateStatus healthStatus `json:"aggregateStatus" yaml:"aggregateStatus"`
	// Summary contains aggregate resource counts.
	Summary statusSummary `json:"summary" yaml:"summary"`
}

// GetReleaseStatus evaluates health for all resources tracked in opts.InventoryLive.
// Returns noResourcesFoundError when no resources are present and no missing entries exist.
//
// opts.InventoryLive contains the live resources fetched from the inventory Secret by the
// caller. opts.MissingResources contains entries tracked in inventory that no longer exist
// on the cluster; these are appended with "Missing" status.
func GetReleaseStatus(ctx context.Context, client *Client, opts StatusOptions) (*StatusResult, error) {
	resources := opts.InventoryLive

	output.Debug("evaluating release status from inventory",
		"release", opts.ReleaseName,
		"liveCount", len(resources),
		"missingCount", len(opts.MissingResources),
	)

	// Return error when no resources found (and no missing resources to show)
	if len(resources) == 0 && len(opts.MissingResources) == 0 {
		return nil, &noResourcesFoundError{
			ReleaseName: opts.ReleaseName,
			ReleaseID:   opts.ReleaseID,
			Namespace:   opts.Namespace,
		}
	}

	result := &StatusResult{
		ReleaseName: opts.ReleaseName,
		Version:     opts.Version,
		Namespace:   opts.Namespace,
	}
	allReady := true

	for _, res := range resources {
		rh, healthy := buildResourceHealth(ctx, client, res, opts)
		result.Resources = append(result.Resources, rh)
		if !healthy {
			allReady = false
		}
	}

	// Append missing resources (tracked in inventory but not on cluster)
	for _, m := range opts.MissingResources {
		result.Resources = append(result.Resources, resourceHealth{
			Kind:      m.Kind,
			Name:      m.Name,
			Namespace: m.Namespace,
			Status:    healthMissing,
			Age:       "<unknown>",
		})
		allReady = false
	}

	// Compute aggregate status
	switch {
	case len(result.Resources) == 0:
		result.AggregateStatus = healthUnknown
	case allReady:
		result.AggregateStatus = healthReady
	default:
		result.AggregateStatus = healthNotReady
	}

	// Compute summary
	for _, r := range result.Resources {
		result.Summary.Total++
		if r.Status == healthReady || r.Status == healthComplete {
			result.Summary.Ready++
		} else {
			result.Summary.NotReady++
		}
	}

	return result, nil
}

// buildResourceHealth constructs a resourceHealth for a single live resource.
// Returns the populated struct and whether the resource is healthy.
func buildResourceHealth(ctx context.Context, client *Client, res *unstructured.Unstructured, opts StatusOptions) (resourceHealth, bool) {
	health := evaluateHealth(res)
	age := computeAge(res)

	key := res.GetKind() + "/" + res.GetNamespace() + "/" + res.GetName()

	rh := resourceHealth{
		Kind:      res.GetKind(),
		Name:      res.GetName(),
		Namespace: res.GetNamespace(),
		Component: opts.ComponentMap[key],
		Status:    health,
		Age:       age,
	}

	if opts.Wide {
		rh.Wide = extractWideInfo(res)
	}

	// listWorkloadPods is only called for live, unhealthy workloads.
	// Missing resources are never passed through buildResourceHealth — they are
	// appended directly in GetReleaseStatus, so health == healthMissing cannot
	// occur here.
	if opts.Verbose && health == healthNotReady {
		if pods, err := listWorkloadPods(ctx, client, res); err == nil && len(pods) > 0 {
			rh.Verbose = &verboseInfo{Pods: pods}
		}
	}

	healthy := health == healthReady || health == healthComplete
	return rh, healthy
}

// FormatStatusTable renders the status result as a formatted table (default format).
func FormatStatusTable(result *StatusResult) string {
	var sb strings.Builder

	// Render metadata header
	sb.WriteString(formatStatusHeader(result))
	sb.WriteString("\n")

	if len(result.Resources) == 0 {
		return sb.String()
	}

	tbl := output.NewTable("KIND", "NAME", "COMPONENT", "STATUS", "AGE")
	for _, r := range result.Resources {
		tbl.Row(r.Kind, r.Name, output.FormatComponent(r.Component), output.FormatHealthStatus(string(r.Status)), r.Age)
	}
	sb.WriteString(tbl.String())

	// Render verbose pod details below the table
	sb.WriteString(formatVerboseBlocks(result))

	return sb.String()
}

// formatStatusWide renders the status result as a wide table with replicas and image columns.
func formatStatusWide(result *StatusResult) string {
	var sb strings.Builder

	sb.WriteString(formatStatusHeader(result))
	sb.WriteString("\n")

	if len(result.Resources) == 0 {
		return sb.String()
	}

	tbl := output.NewTable("KIND", "NAME", "COMPONENT", "STATUS", "REPLICAS", "IMAGE", "AGE")
	for _, r := range result.Resources {
		replicas := "-"
		image := "-"
		if r.Wide != nil {
			if r.Wide.Replicas != "" {
				replicas = r.Wide.Replicas
			}
			if r.Wide.Image != "" {
				image = r.Wide.Image
			}
		}
		tbl.Row(r.Kind, r.Name, output.FormatComponent(r.Component), output.FormatHealthStatus(string(r.Status)), replicas, image, r.Age)
	}
	sb.WriteString(tbl.String())

	sb.WriteString(formatVerboseBlocks(result))

	return sb.String()
}

// formatStatusHeader renders the release metadata header block.
func formatStatusHeader(result *StatusResult) string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "Release:    %s\n", output.StyleNoun(result.ReleaseName))
	if result.Version != "" {
		fmt.Fprintf(&sb, "Version:    %s\n", result.Version)
	}
	fmt.Fprintf(&sb, "Namespace:  %s\n", output.StyleNoun(result.Namespace))
	fmt.Fprintf(&sb, "Status:     %s\n", output.FormatHealthStatus(string(result.AggregateStatus)))
	fmt.Fprintf(&sb, "Resources:  %d total (%d ready", result.Summary.Total, result.Summary.Ready)
	if result.Summary.NotReady > 0 {
		fmt.Fprintf(&sb, ", %d not ready", result.Summary.NotReady)
	}
	sb.WriteString(")\n")
	return sb.String()
}

// formatVerboseBlocks renders pod detail blocks for resources with verbose data.
// Column widths are computed dynamically per block from actual pod name and phase
// lengths, producing compact kubectl-style output with no excess whitespace.
func formatVerboseBlocks(result *StatusResult) string {
	var sb strings.Builder
	for _, r := range result.Resources {
		if r.Verbose == nil || len(r.Verbose.Pods) == 0 {
			continue
		}

		readyCount := 0
		for _, p := range r.Verbose.Pods {
			if p.Ready {
				readyCount++
			}
		}

		// Compute column widths for this block.
		nameWidth, phaseWidth := 0, 0
		for _, p := range r.Verbose.Pods {
			if len(p.Name) > nameWidth {
				nameWidth = len(p.Name)
			}
			if len(p.Phase) > phaseWidth {
				phaseWidth = len(p.Phase)
			}
		}

		fmt.Fprintf(&sb, "\n%s/%s (%d/%d ready):\n", r.Kind, r.Name, readyCount, len(r.Verbose.Pods))
		for _, p := range r.Verbose.Pods {
			// When a termination reason is available, it is the primary context.
			// Fall back to "(not ready)" only when no specific reason exists.
			var detail string
			switch {
			case p.Ready:
				detail = "(ready)"
			case p.Reason != "":
				detail = p.Reason
			default:
				detail = "(not ready)"
			}
			if p.Restarts > 0 {
				detail += fmt.Sprintf(", %d restarts", p.Restarts)
			}

			namePad := strings.Repeat(" ", nameWidth-len(p.Name))
			phasePad := strings.Repeat(" ", phaseWidth-len(p.Phase))
			fmt.Fprintf(&sb, "    %s%s   %s%s   %s\n", p.Name, namePad, p.Phase, phasePad, detail)
		}
	}
	return sb.String()
}

// FormatStatusJSON renders the status result as JSON.
func formatStatusJSON(result *StatusResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling status to JSON: %w", err)
	}
	return string(data), nil
}

// FormatStatusYAML renders the status result as YAML.
func formatStatusYAML(result *StatusResult) (string, error) {
	data, err := yaml.Marshal(result)
	if err != nil {
		return "", fmt.Errorf("marshaling status to YAML: %w", err)
	}
	return string(data), nil
}

// computeAge computes the human-readable age of a resource from its creation timestamp.
func computeAge(resource *unstructured.Unstructured) string {
	timestamp := resource.GetCreationTimestamp()
	if timestamp.IsZero() {
		return "<unknown>"
	}

	duration := time.Since(timestamp.Time)
	return formatDuration(duration)
}

// formatDuration converts a duration to a human-readable string (e.g., "5m", "2h", "3d").
func formatDuration(d time.Duration) string {
	if d < 0 {
		d = 0
	}

	switch {
	case d < time.Minute:
		return fmt.Sprintf("%ds", int(d.Seconds()))
	case d < time.Hour:
		return fmt.Sprintf("%dm", int(d.Minutes()))
	case d < 24*time.Hour:
		hours := int(d.Hours())
		mins := int(d.Minutes()) - hours*60
		if mins > 0 {
			return fmt.Sprintf("%dh%dm", hours, mins)
		}
		return fmt.Sprintf("%dh", hours)
	default:
		days := int(d.Hours() / 24)
		return fmt.Sprintf("%dd", days)
	}
}

// FormatStatus formats the status result based on the output format.
func FormatStatus(result *StatusResult, format output.Format) (string, error) {
	switch format {
	case output.FormatJSON:
		return formatStatusJSON(result)
	case output.FormatYAML:
		return formatStatusYAML(result)
	case output.FormatWide:
		return formatStatusWide(result), nil
	case output.FormatTable:
		return FormatStatusTable(result), nil
	case output.FormatDir:
		// Dir format not supported for status - fall through to table
		return FormatStatusTable(result), nil
	default:
		return FormatStatusTable(result), nil
	}
}
