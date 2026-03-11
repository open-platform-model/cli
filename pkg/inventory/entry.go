package inventory

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	pkgcore "github.com/opmodel/cli/pkg/core"
)

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

func IdentityEqual(a, b InventoryEntry) bool {
	return a.Group == b.Group &&
		a.Kind == b.Kind &&
		a.Namespace == b.Namespace &&
		a.Name == b.Name &&
		a.Component == b.Component
}

func K8sIdentityEqual(a, b InventoryEntry) bool {
	return a.Group == b.Group &&
		a.Kind == b.Kind &&
		a.Namespace == b.Namespace &&
		a.Name == b.Name
}
