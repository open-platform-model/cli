package inventory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	pkgcore "github.com/open-platform-model/cli/pkg/core"
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

// K8sIdentity is the comparable key K8sIdentityEqual compares: an object's
// group, kind, namespace and name, without its component or version.
type K8sIdentity struct{ Group, Kind, Namespace, Name string }

// IdentityOf is an entry's K8sIdentity.
func IdentityOf(e InventoryEntry) K8sIdentity {
	return K8sIdentity{Group: e.Group, Kind: e.Kind, Namespace: e.Namespace, Name: e.Name}
}

// AdmitSet is an explicit set of objects the first-install existence check
// lets pass its untracked-object test.
type AdmitSet map[K8sIdentity]struct{}

// Has reports whether the set admits the entry; a nil set admits nothing.
func (s AdmitSet) Has(e InventoryEntry) bool {
	_, ok := s[IdentityOf(e)]
	return ok
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
