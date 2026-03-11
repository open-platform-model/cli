package inventory

// CreatedBy identifies which tool originally created a release inventory.
type CreatedBy string

const (
	CreatedByCLI        CreatedBy = "cli"
	CreatedByController CreatedBy = "controller"
)

// NormalizeCreatedBy converts the stored provenance to a supported value.
// Missing or unknown values are treated as legacy CLI-owned inventories.
func NormalizeCreatedBy(createdBy CreatedBy) CreatedBy {
	if createdBy == CreatedByController {
		return CreatedByController
	}
	return CreatedByCLI
}

// Package inventory provides the release inventory data model, serialization,
// naming, provenance, and change history helpers for the OPM inventory Secret.
//
// The inventory Secret records the exact set of resources applied per release,
// enabling automatic pruning, fast discovery, and change history.

// InventoryEntry represents a single tracked Kubernetes resource.
// Version is excluded from identity comparison to prevent false orphans
// during Kubernetes API version migrations.
//
//nolint:revive // Inventory* prefix is intentional: these types are referenced by name across packages
type InventoryEntry struct {
	Group     string `json:"group"`
	Kind      string `json:"kind"`
	Namespace string `json:"namespace"`
	Name      string `json:"name"`
	Version   string `json:"v"`         // API version (excluded from identity)
	Component string `json:"component"` // source component name
}

// ChangeEntry represents the full state for a single change (one apply operation).
type ChangeEntry struct {
	Source         ChangeSource  `json:"module"`         // JSON tag "module" preserved for backward compat
	Values         string        `json:"values"`         // resolved CUE values as CUE string
	ManifestDigest string        `json:"manifestDigest"` // SHA256 of rendered manifests
	Timestamp      string        `json:"timestamp"`      // RFC 3339
	Inventory      InventoryList `json:"inventory"`
}

// ChangeSource records the source context for a change entry: the module that
// was rendered and the release under which it was deployed.
type ChangeSource struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	// ReleaseName is the release name (the user-supplied --release-name value, e.g. "mc").
	// JSON tag "name" is preserved for backward compat with existing inventory Secrets.
	ReleaseName string `json:"name"`
	Local       bool   `json:"local,omitempty"` // true for local modules (no version)
}

// InventoryList is the list of tracked resources in a change.
//
//nolint:revive // Inventory prefix is intentional for cross-package clarity
type InventoryList struct {
	Entries []InventoryEntry `json:"entries"`
}

// ReleaseMetadata is the release-level metadata stored in the inventory Secret
// under the "releaseMetadata" key. Using kind/apiVersion matching a future CRD
// schema enables migration from Secret to CRD without changing the data model.
type ReleaseMetadata struct {
	Kind               string    `json:"kind"`               // "ModuleRelease"
	APIVersion         string    `json:"apiVersion"`         // "core.opmodel.dev/v1alpha1"
	ReleaseName        string    `json:"name"`               // release name (from --release-name), e.g. "mc"
	ReleaseNamespace   string    `json:"namespace"`          // Kubernetes namespace of the release
	ReleaseID          string    `json:"uuid"`               // deterministic UUID v5 release identity
	LastTransitionTime string    `json:"lastTransitionTime"` // RFC 3339
	CreatedBy          CreatedBy `json:"createdBy,omitempty"`
}

// NormalizedCreatedBy returns the effective creator, defaulting legacy
// inventories without the field to CLI ownership.
func (m ReleaseMetadata) NormalizedCreatedBy() CreatedBy {
	return NormalizeCreatedBy(m.CreatedBy)
}

// ModuleMetadata is the module-level metadata stored in the inventory Secret
// under the "moduleMetadata" key. Using kind/apiVersion matching a future CRD
// schema enables migration from Secret to CRD without changing the data model.
type ModuleMetadata struct {
	Kind       string `json:"kind"`
	APIVersion string `json:"apiVersion"`
	Name       string `json:"name"`
	UUID       string `json:"uuid,omitempty"`
}

// InventorySecret is the full in-memory representation of an inventory Secret.
// It maps 1:1 to a Kubernetes Secret stored in the release namespace.
//
//nolint:revive // Inventory prefix is intentional for cross-package clarity
type InventorySecret struct {
	ReleaseMetadata ReleaseMetadata
	ModuleMetadata  ModuleMetadata
	Index           []string
	Changes         map[string]*ChangeEntry

	resourceVersion string
}

// ResourceVersion returns the Kubernetes resourceVersion for optimistic concurrency.
// This is populated by UnmarshalFromSecret and used by WriteInventory.
func (s *InventorySecret) ResourceVersion() string {
	return s.resourceVersion
}

// SetResourceVersion stores the Kubernetes resourceVersion for optimistic concurrency.
func (s *InventorySecret) SetResourceVersion(resourceVersion string) {
	s.resourceVersion = resourceVersion
}
