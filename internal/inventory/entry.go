package inventory

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	pkgcore "github.com/opmodel/cli/pkg/core"
)

// NewEntryFromResource constructs an InventoryEntry from a rendered *unstructured.Unstructured.
// Group, Kind, and Version come from the GVK, Namespace and Name from the object metadata.
// Component is read from the "component.opmodel.dev/name" label injected by the CUE catalog.
func NewEntryFromResource(r *unstructured.Unstructured) InventoryEntry {
	gvk := r.GroupVersionKind()
	labels := r.GetLabels()
	component := labels[pkgcore.LabelComponentName]
	return InventoryEntry{
		Group:     gvk.Group,
		Kind:      gvk.Kind,
		Namespace: r.GetNamespace(),
		Name:      r.GetName(),
		Version:   gvk.Version,
		Component: component,
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
