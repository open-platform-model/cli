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

	// OutputFormat is the desired output format (table, yaml, json).
	OutputFormat output.Format

	// Watch enables continuous monitoring mode.
	Watch bool

	// Kubeconfig is the path to the kubeconfig file.
	Kubeconfig string

	// Context is the Kubernetes context to use.
	Context string
}

// resourceHealth contains health information for a single resource.
type resourceHealth struct {
	// Kind is the Kubernetes resource kind.
	Kind string `json:"kind" yaml:"kind"`
	// Name is the resource name.
	Name string `json:"name" yaml:"name"`
	// Namespace is the resource namespace.
	Namespace string `json:"namespace" yaml:"namespace"`
	// Status is the evaluated health status.
	Status healthStatus `json:"status" yaml:"status"`
	// Age is the human-readable age of the resource.
	Age string `json:"age" yaml:"age"`
}

// statusResult contains the full status output.
type statusResult struct {
	// Resources is the list of resource health statuses.
	Resources []resourceHealth `json:"resources" yaml:"resources"`
	// AggregateStatus is the overall module health.
	AggregateStatus healthStatus `json:"aggregateStatus" yaml:"aggregateStatus"`
	// ModuleID is the module identity UUID (if present on resources).
	ModuleID string `json:"moduleId,omitempty" yaml:"moduleId,omitempty"`
	// ReleaseID is the release identity UUID (if present on resources).
	ReleaseID string `json:"releaseId,omitempty" yaml:"releaseId,omitempty"`
}

// GetModuleStatus discovers resources by OPM labels and evaluates health per resource.
// Returns noResourcesFoundError when no resources match the selector.
func GetModuleStatus(ctx context.Context, client *Client, opts StatusOptions) (*statusResult, error) {
	// Discover resources via labels
	resources, err := DiscoverResources(ctx, client, DiscoveryOptions{
		ReleaseName: opts.ReleaseName,
		Namespace:   opts.Namespace,
		ReleaseID:   opts.ReleaseID,
	})
	if err != nil {
		return nil, fmt.Errorf("discovering release resources: %w", err)
	}

	// Return error when no resources found
	if len(resources) == 0 {
		return nil, &noResourcesFoundError{
			ReleaseName: opts.ReleaseName,
			ReleaseID:   opts.ReleaseID,
			Namespace:   opts.Namespace,
		}
	}

	result := &statusResult{}
	allReady := true

	// Extract identity labels from first resource (if available)
	if len(resources) > 0 {
		labels := resources[0].GetLabels()
		if labels != nil {
			result.ModuleID = labels[labelModuleID]
			result.ReleaseID = labels[labelReleaseID]
		}
	}

	for _, res := range resources {
		health := evaluateHealth(res)
		age := computeAge(res)

		result.Resources = append(result.Resources, resourceHealth{
			Kind:      res.GetKind(),
			Name:      res.GetName(),
			Namespace: res.GetNamespace(),
			Status:    health,
			Age:       age,
		})

		if health != healthReady && health != healthComplete {
			allReady = false
		}
	}

	// Compute aggregate status
	if len(result.Resources) == 0 {
		result.AggregateStatus = healthUnknown
	} else if allReady {
		result.AggregateStatus = healthReady
	} else {
		result.AggregateStatus = healthNotReady
	}

	return result, nil
}

// FormatStatusTable renders the status result as a formatted table.
func FormatStatusTable(result *statusResult) string {
	var sb strings.Builder

	// Show identity information if present
	if result.ModuleID != "" || result.ReleaseID != "" {
		if result.ModuleID != "" {
			sb.WriteString(fmt.Sprintf("Module ID:  %s\n", result.ModuleID))
		}
		if result.ReleaseID != "" {
			sb.WriteString(fmt.Sprintf("Release ID: %s\n", result.ReleaseID))
		}
		sb.WriteString("\n")
	}

	if len(result.Resources) == 0 {
		return sb.String()
	}

	tbl := output.NewTable("KIND", "NAME", "NAMESPACE", "STATUS", "AGE")
	for _, r := range result.Resources {
		tbl.Row(r.Kind, r.Name, r.Namespace, string(r.Status), r.Age)
	}
	sb.WriteString(tbl.String())
	return sb.String()
}

// FormatStatusJSON renders the status result as JSON.
func formatStatusJSON(result *statusResult) (string, error) {
	data, err := json.MarshalIndent(result, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling status to JSON: %w", err)
	}
	return string(data), nil
}

// FormatStatusYAML renders the status result as YAML.
func formatStatusYAML(result *statusResult) (string, error) {
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
func FormatStatus(result *statusResult, format output.Format) (string, error) {
	switch format {
	case output.FormatJSON:
		return formatStatusJSON(result)
	case output.FormatYAML:
		return formatStatusYAML(result)
	case output.FormatTable:
		return FormatStatusTable(result), nil
	case output.FormatDir:
		// Dir format not supported for status - fall through to table
		return FormatStatusTable(result), nil
	default:
		return FormatStatusTable(result), nil
	}
}
