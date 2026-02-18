// Package inventory provides the release inventory data model, serialization,
// and Kubernetes CRUD operations for the OPM inventory Secret.
//
// The inventory Secret records the exact set of resources applied per release,
// enabling automatic pruning, fast discovery, and change history.
package inventory

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
	Module         ModuleRef     `json:"module"`
	Values         string        `json:"values"`         // resolved CUE values as CUE string
	ManifestDigest string        `json:"manifestDigest"` // SHA256 of rendered manifests
	Timestamp      string        `json:"timestamp"`      // RFC 3339
	Inventory      InventoryList `json:"inventory"`
}

// ModuleRef identifies the source module for a change.
type ModuleRef struct {
	Path    string `json:"path"`
	Version string `json:"version,omitempty"`
	// Name is the module release name (the user-supplied --release-name value, e.g. "mc"),
	// not the canonical module definition name (e.g. "minecraft"). It records which
	// deployment of the module produced this change entry.
	Name  string `json:"name"`
	Local bool   `json:"local,omitempty"` // true for local modules (no version)
}

// InventoryList is the list of tracked resources in a change.
//
//nolint:revive // Inventory prefix is intentional for cross-package clarity
type InventoryList struct {
	Entries []InventoryEntry `json:"entries"`
}

// InventoryMetadata is the release-level metadata stored in the inventory Secret.
// Using kind/apiVersion matching a future CRD schema enables migration from
// Secret to CRD without changing the data model.
//
//nolint:revive // Inventory prefix is intentional for cross-package clarity
type InventoryMetadata struct {
	Kind               string `json:"kind"`        // "ModuleRelease"
	APIVersion         string `json:"apiVersion"`  // "core.opmodel.dev/v1alpha1"
	Name               string `json:"name"`        // canonical module name, e.g. "minecraft"
	ReleaseName        string `json:"releaseName"` // release name (from --release-name), e.g. "mc"
	Namespace          string `json:"namespace"`
	ReleaseID          string `json:"releaseId"`
	LastTransitionTime string `json:"lastTransitionTime"` // RFC 3339
}

// InventorySecret is the full in-memory representation of an inventory Secret.
// It maps 1:1 to a Kubernetes Secret stored in the release namespace.
//
//nolint:revive // Inventory prefix is intentional for cross-package clarity
type InventorySecret struct {
	Metadata InventoryMetadata
	Index    []string                // ordered change IDs (newest first)
	Changes  map[string]*ChangeEntry // keyed by "change-sha1-<8chars>"

	// resourceVersion holds the K8s resourceVersion from the last read,
	// used for optimistic concurrency on writes. Only set by UnmarshalFromSecret.
	resourceVersion string
}

// ResourceVersion returns the Kubernetes resourceVersion for optimistic concurrency.
// This is populated by UnmarshalFromSecret and used by WriteInventory.
func (s *InventorySecret) ResourceVersion() string {
	return s.resourceVersion
}
