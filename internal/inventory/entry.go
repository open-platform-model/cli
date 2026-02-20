package inventory

import (
	"github.com/opmodel/cli/internal/core"
)

// NewEntryFromResource constructs an InventoryEntry from a rendered *core.Resource.
// Group and Kind come from the GVK, Version from the GVK's Version field,
// Namespace and Name from the resource metadata, Component from core.Resource.Component.
func NewEntryFromResource(r *core.Resource) InventoryEntry {
	gvk := r.GVK()
	return InventoryEntry{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: r.Namespace(),
		Name:      r.Name(),
		Version:   gvk.Version,
		Component: r.Component,
	}
}

// IdentityEqual reports whether two InventoryEntries represent the same resource.
// Comparison uses Group, Kind, Namespace, Name, and Component.
// Version is intentionally excluded to prevent false orphans during API version migrations.
func IdentityEqual(a, b InventoryEntry) bool {
	return a.Group == b.Group &&
		a.Kind == b.Kind &&
		a.Namespace == b.Namespace &&
		a.Name == b.Name &&
		a.Component == b.Component
}

// K8sIdentityEqual reports whether two InventoryEntries refer to the same
// Kubernetes resource, ignoring both Version and Component.
// Used by the component-rename safety check to detect when the same K8s resource
// appears under a different component name.
func K8sIdentityEqual(a, b InventoryEntry) bool {
	return a.Group == b.Group &&
		a.Kind == b.Kind &&
		a.Namespace == b.Namespace &&
		a.Name == b.Name
}
