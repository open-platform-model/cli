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
	Version   string `json:"v,omitempty"`         // API version (excluded from identity)
	Component string `json:"component,omitempty"` // source component name
}

// Inventory is the current set of resources owned by a release.
type Inventory struct {
	Revision int              `json:"revision,omitempty"`
	Digest   string           `json:"digest,omitempty"`
	Count    int              `json:"count,omitempty"`
	Entries  []InventoryEntry `json:"entries"`
}
