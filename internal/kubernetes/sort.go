package kubernetes

import (
	"sort"

	"github.com/opmodel/cli/pkg/weights"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// SortForApply sorts resources in the order they should be applied.
// Lower weight resources are applied first.
func SortForApply(resources []*unstructured.Unstructured) {
	sort.SliceStable(resources, func(i, j int) bool {
		return weights.GetWeight(resources[i].GetKind()) < weights.GetWeight(resources[j].GetKind())
	})
}

// SortForDelete sorts resources in the order they should be deleted.
// Higher weight resources are deleted first (reverse of apply order).
func SortForDelete(resources []*unstructured.Unstructured) {
	sort.SliceStable(resources, func(i, j int) bool {
		return weights.GetWeight(resources[i].GetKind()) > weights.GetWeight(resources[j].GetKind())
	})
}

// GroupByKind groups resources by their Kind.
func GroupByKind(resources []*unstructured.Unstructured) map[string][]*unstructured.Unstructured {
	result := make(map[string][]*unstructured.Unstructured)
	for _, r := range resources {
		kind := r.GetKind()
		result[kind] = append(result[kind], r)
	}
	return result
}

// GroupByNamespace groups resources by their namespace.
func GroupByNamespace(resources []*unstructured.Unstructured) map[string][]*unstructured.Unstructured {
	result := make(map[string][]*unstructured.Unstructured)
	for _, r := range resources {
		ns := r.GetNamespace()
		if ns == "" {
			ns = "_cluster" // Cluster-scoped resources
		}
		result[ns] = append(result[ns], r)
	}
	return result
}
