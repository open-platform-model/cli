package kubernetes

import (
	"time"
)

// HealthStatus represents the health state of a resource.
type HealthStatus string

const (
	// HealthReady indicates the resource is healthy and ready.
	HealthReady HealthStatus = "Ready"

	// HealthNotReady indicates the resource exists but is not ready.
	HealthNotReady HealthStatus = "NotReady"

	// HealthProgressing indicates the resource is being reconciled.
	HealthProgressing HealthStatus = "Progressing"

	// HealthFailed indicates the resource has failed.
	HealthFailed HealthStatus = "Failed"

	// HealthUnknown indicates the health cannot be determined.
	HealthUnknown HealthStatus = "Unknown"
)

// ResourceStatus represents the status of a single resource.
type ResourceStatus struct {
	// Kind is the Kubernetes resource kind.
	Kind string `json:"kind"`

	// APIVersion is the resource API version.
	APIVersion string `json:"apiVersion"`

	// Name is the resource name.
	Name string `json:"name"`

	// Namespace is the resource namespace (empty for cluster-scoped).
	Namespace string `json:"namespace,omitempty"`

	// Health is the evaluated health status.
	Health HealthStatus `json:"health"`

	// Message provides additional status information.
	Message string `json:"message,omitempty"`

	// Age is the time since creation.
	Age time.Duration `json:"age"`

	// Component is the OPM component name.
	Component string `json:"component,omitempty"`
}

// ModuleStatus represents the aggregate status of a module.
type ModuleStatus struct {
	// Name is the module name.
	Name string `json:"name"`

	// Version is the module version.
	Version string `json:"version,omitempty"`

	// Namespace the module is deployed to.
	Namespace string `json:"namespace"`

	// Resources is the list of resource statuses.
	Resources []ResourceStatus `json:"resources"`

	// Summary counts by health status.
	Summary StatusSummary `json:"summary"`
}

// StatusSummary provides aggregate counts.
type StatusSummary struct {
	Total       int `json:"total"`
	Ready       int `json:"ready"`
	NotReady    int `json:"notReady"`
	Progressing int `json:"progressing"`
	Failed      int `json:"failed"`
	Unknown     int `json:"unknown"`
}

// IsHealthy returns true if all resources are ready.
func (ms *ModuleStatus) IsHealthy() bool {
	return ms.Summary.Ready == ms.Summary.Total
}

// HasFailures returns true if any resources have failed.
func (ms *ModuleStatus) HasFailures() bool {
	return ms.Summary.Failed > 0
}

// IsProgressing returns true if any resources are progressing.
func (ms *ModuleStatus) IsProgressing() bool {
	return ms.Summary.Progressing > 0
}

// CalculateSummary recalculates the summary from resources.
func (ms *ModuleStatus) CalculateSummary() {
	ms.Summary = StatusSummary{}
	for _, r := range ms.Resources {
		ms.Summary.Total++
		switch r.Health {
		case HealthReady:
			ms.Summary.Ready++
		case HealthNotReady:
			ms.Summary.NotReady++
		case HealthProgressing:
			ms.Summary.Progressing++
		case HealthFailed:
			ms.Summary.Failed++
		default:
			ms.Summary.Unknown++
		}
	}
}

// BundleStatus represents the aggregate status of a bundle.
type BundleStatus struct {
	// Name is the bundle name.
	Name string `json:"name"`

	// Version is the bundle version.
	Version string `json:"version,omitempty"`

	// Namespace the bundle is deployed to.
	Namespace string `json:"namespace"`

	// Modules contains status for each module.
	Modules []ModuleStatus `json:"modules,omitempty"`

	// Resources is the flat list of all resource statuses.
	Resources []ResourceStatus `json:"resources"`

	// Summary counts by health status.
	Summary StatusSummary `json:"summary"`
}

// IsHealthy returns true if all resources are ready.
func (bs *BundleStatus) IsHealthy() bool {
	return bs.Summary.Ready == bs.Summary.Total
}

// CalculateSummary recalculates the summary from resources.
func (bs *BundleStatus) CalculateSummary() {
	bs.Summary = StatusSummary{}
	for _, r := range bs.Resources {
		bs.Summary.Total++
		switch r.Health {
		case HealthReady:
			bs.Summary.Ready++
		case HealthNotReady:
			bs.Summary.NotReady++
		case HealthProgressing:
			bs.Summary.Progressing++
		case HealthFailed:
			bs.Summary.Failed++
		default:
			bs.Summary.Unknown++
		}
	}
}
