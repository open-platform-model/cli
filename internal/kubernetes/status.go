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

	// Name is the module release name.
	Name string

	// OutputFormat is the desired output format (table, yaml, json).
	OutputFormat string

	// Watch enables continuous monitoring mode.
	Watch bool

	// Kubeconfig is the path to the kubeconfig file.
	Kubeconfig string

	// Context is the Kubernetes context to use.
	Context string
}

// ResourceHealth contains health information for a single resource.
type ResourceHealth struct {
	// Kind is the Kubernetes resource kind.
	Kind string `json:"kind" yaml:"kind"`
	// Name is the resource name.
	Name string `json:"name" yaml:"name"`
	// Namespace is the resource namespace.
	Namespace string `json:"namespace" yaml:"namespace"`
	// Status is the evaluated health status.
	Status HealthStatus `json:"status" yaml:"status"`
	// Age is the human-readable age of the resource.
	Age string `json:"age" yaml:"age"`
}

// StatusResult contains the full status output.
type StatusResult struct {
	// Resources is the list of resource health statuses.
	Resources []ResourceHealth `json:"resources" yaml:"resources"`
	// AggregateStatus is the overall module health.
	AggregateStatus HealthStatus `json:"aggregateStatus" yaml:"aggregateStatus"`
}

// GetModuleStatus discovers resources by OPM labels and evaluates health per resource.
func GetModuleStatus(ctx context.Context, client *Client, opts StatusOptions) (*StatusResult, error) {
	// Discover resources via labels
	resources, err := DiscoverResources(ctx, client, opts.Name, opts.Namespace)
	if err != nil {
		return nil, fmt.Errorf("discovering module resources: %w", err)
	}

	result := &StatusResult{}
	allReady := true

	for _, res := range resources {
		health := EvaluateHealth(res)
		age := computeAge(res)

		result.Resources = append(result.Resources, ResourceHealth{
			Kind:      res.GetKind(),
			Name:      res.GetName(),
			Namespace: res.GetNamespace(),
			Status:    health,
			Age:       age,
		})

		if health != HealthReady && health != HealthComplete {
			allReady = false
		}
	}

	// Compute aggregate status
	if len(result.Resources) == 0 {
		result.AggregateStatus = HealthUnknown
	} else if allReady {
		result.AggregateStatus = HealthReady
	} else {
		result.AggregateStatus = HealthNotReady
	}

	return result, nil
}

// FormatStatusTable renders the status result as a formatted table.
func FormatStatusTable(result *StatusResult) string {
	if len(result.Resources) == 0 {
		return ""
	}

	tbl := output.NewTable("KIND", "NAME", "NAMESPACE", "STATUS", "AGE")
	for _, r := range result.Resources {
		tbl.Row(r.Kind, r.Name, r.Namespace, string(r.Status), r.Age)
	}
	return tbl.String()
}

// FormatStatusJSON renders the status result as JSON.
func FormatStatusJSON(result *StatusResult) (string, error) {
	data, err := json.MarshalIndent(result.Resources, "", "  ")
	if err != nil {
		return "", fmt.Errorf("marshaling status to JSON: %w", err)
	}
	return string(data), nil
}

// FormatStatusYAML renders the status result as YAML.
func FormatStatusYAML(result *StatusResult) (string, error) {
	data, err := yaml.Marshal(result.Resources)
	if err != nil {
		return "", fmt.Errorf("marshaling status to YAML: %w", err)
	}
	return string(data), nil
}

// NoResourcesMessage returns a human-readable message when no resources are found.
func NoResourcesMessage(name, namespace string) string {
	return fmt.Sprintf("No resources found for module %q in namespace %q", name, namespace)
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
func FormatStatus(result *StatusResult, format string) (string, error) {
	switch strings.ToLower(format) {
	case "json":
		return FormatStatusJSON(result)
	case "yaml":
		return FormatStatusYAML(result)
	default:
		return FormatStatusTable(result), nil
	}
}
