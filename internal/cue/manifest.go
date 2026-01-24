package cue

import (
	"sort"

	"github.com/opmodel/cli/pkg/weights"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// Manifest represents a rendered Kubernetes manifest.
type Manifest struct {
	// Object is the unstructured Kubernetes object.
	Object *unstructured.Unstructured

	// ComponentName is the OPM component this manifest belongs to.
	ComponentName string

	// Weight determines apply/delete order (lower = earlier apply).
	Weight int
}

// ManifestSet is an ordered collection of manifests.
type ManifestSet struct {
	// Manifests is the list of manifests.
	Manifests []*Manifest

	// Module metadata for labeling.
	Module ModuleMetadata

	// NamespaceOverride is the namespace from --namespace flag.
	NamespaceOverride string
}

// NewManifestSet creates a new ManifestSet with the given module metadata.
func NewManifestSet(module ModuleMetadata) *ManifestSet {
	return &ManifestSet{
		Manifests: make([]*Manifest, 0),
		Module:    module,
	}
}

// Add adds a manifest to the set, calculating its weight automatically.
func (ms *ManifestSet) Add(obj *unstructured.Unstructured, componentName string) {
	m := &Manifest{
		Object:        obj,
		ComponentName: componentName,
		Weight:        weights.GetWeight(obj.GetKind()),
	}
	ms.Manifests = append(ms.Manifests, m)
}

// AddWithWeight adds a manifest with an explicit weight.
func (ms *ManifestSet) AddWithWeight(obj *unstructured.Unstructured, componentName string, weight int) {
	m := &Manifest{
		Object:        obj,
		ComponentName: componentName,
		Weight:        weight,
	}
	ms.Manifests = append(ms.Manifests, m)
}

// SortForApply sorts manifests in ascending weight order (lower weights first).
func (ms *ManifestSet) SortForApply() {
	sort.SliceStable(ms.Manifests, func(i, j int) bool {
		return ms.Manifests[i].Weight < ms.Manifests[j].Weight
	})
}

// SortForDelete sorts manifests in descending weight order (higher weights first).
func (ms *ManifestSet) SortForDelete() {
	sort.SliceStable(ms.Manifests, func(i, j int) bool {
		return ms.Manifests[i].Weight > ms.Manifests[j].Weight
	})
}

// Len returns the number of manifests.
func (ms *ManifestSet) Len() int {
	return len(ms.Manifests)
}

// ByKind returns manifests filtered by kind.
func (ms *ManifestSet) ByKind(kind string) []*Manifest {
	var result []*Manifest
	for _, m := range ms.Manifests {
		if m.Object.GetKind() == kind {
			result = append(result, m)
		}
	}
	return result
}

// ByComponent returns manifests filtered by component name.
func (ms *ManifestSet) ByComponent(componentName string) []*Manifest {
	var result []*Manifest
	for _, m := range ms.Manifests {
		if m.ComponentName == componentName {
			result = append(result, m)
		}
	}
	return result
}

// Objects returns all unstructured objects.
func (ms *ManifestSet) Objects() []*unstructured.Unstructured {
	objects := make([]*unstructured.Unstructured, len(ms.Manifests))
	for i, m := range ms.Manifests {
		objects[i] = m.Object
	}
	return objects
}

// Clone creates a deep copy of the ManifestSet.
func (ms *ManifestSet) Clone() *ManifestSet {
	clone := &ManifestSet{
		Manifests:         make([]*Manifest, len(ms.Manifests)),
		Module:            ms.Module,
		NamespaceOverride: ms.NamespaceOverride,
	}
	for i, m := range ms.Manifests {
		clone.Manifests[i] = &Manifest{
			Object:        m.Object.DeepCopy(),
			ComponentName: m.ComponentName,
			Weight:        m.Weight,
		}
	}
	return clone
}
