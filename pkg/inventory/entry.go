package inventory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

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

func ComputeStaleSet(previous, current []InventoryEntry) []InventoryEntry {
	if len(previous) == 0 {
		return []InventoryEntry{}
	}

	stale := make([]InventoryEntry, 0)
	for _, prev := range previous {
		found := false
		for _, cur := range current {
			if IdentityEqual(prev, cur) {
				found = true
				break
			}
		}
		if !found {
			stale = append(stale, prev)
		}
	}

	return stale
}

func ComputeDigest(entries []InventoryEntry) string {
	sorted := make([]InventoryEntry, len(entries))
	copy(sorted, entries)
	if len(sorted) == 0 {
		sum := sha256.Sum256(nil)
		return fmt.Sprintf("sha256:%x", sum)
	}
	sort.SliceStable(sorted, func(i, j int) bool {
		if sorted[i].Group != sorted[j].Group {
			return sorted[i].Group < sorted[j].Group
		}
		if sorted[i].Kind != sorted[j].Kind {
			return sorted[i].Kind < sorted[j].Kind
		}
		if sorted[i].Namespace != sorted[j].Namespace {
			return sorted[i].Namespace < sorted[j].Namespace
		}
		if sorted[i].Name != sorted[j].Name {
			return sorted[i].Name < sorted[j].Name
		}
		if sorted[i].Component != sorted[j].Component {
			return sorted[i].Component < sorted[j].Component
		}
		return sorted[i].Version < sorted[j].Version
	})

	b, err := json.Marshal(sorted)
	if err != nil {
		b = []byte(fmt.Sprintf("%v", sorted))
	}
	sum := sha256.Sum256(b)
	return fmt.Sprintf("sha256:%x", sum)
}
