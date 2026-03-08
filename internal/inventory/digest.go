package inventory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	pkgcore "github.com/opmodel/cli/pkg/core"
)

// SortResources sorts a slice of *unstructured.Unstructured with a deterministic 5-key total ordering:
// weight (ascending), group (alpha), kind (alpha), namespace (alpha), name (alpha).
//
// This sort is shared between ComputeManifestDigest and the pipeline output sort,
// ensuring that opm mod build output and the inventory digest use the same ordering.
func SortResources(resources []*unstructured.Unstructured) {
	sort.SliceStable(resources, func(i, j int) bool {
		ri, rj := resources[i], resources[j]

		wi := pkgcore.GetWeight(ri.GroupVersionKind())
		wj := pkgcore.GetWeight(rj.GroupVersionKind())
		if wi != wj {
			return wi < wj
		}

		gi, gj := ri.GroupVersionKind().Group, rj.GroupVersionKind().Group
		if gi != gj {
			return gi < gj
		}

		ki, kj := ri.GroupVersionKind().Kind, rj.GroupVersionKind().Kind
		if ki != kj {
			return ki < kj
		}

		nsi, nsj := ri.GetNamespace(), rj.GetNamespace()
		if nsi != nsj {
			return nsi < nsj
		}

		return ri.GetName() < rj.GetName()
	})
}

// ComputeManifestDigest computes a deterministic SHA256 digest over a set of
// rendered resources. The digest is independent of input order.
//
// Algorithm:
//  1. Sort resources using 5-key total ordering (SortResources)
//  2. json.Marshal each resource's Object (Go sorts map keys alphabetically)
//  3. Concatenate serialized bytes with newline separators
//  4. SHA256 the result → "sha256:<hex>"
//
// The digest is computed from the rendered output (pre-apply), so server-generated
// fields are not included. OPM labels injected by CUE transformers ARE included.
func ComputeManifestDigest(resources []*unstructured.Unstructured) string {
	// Work on a copy to avoid mutating the caller's slice
	sorted := make([]*unstructured.Unstructured, len(resources))
	copy(sorted, resources)
	SortResources(sorted)

	h := sha256.New()
	for i, r := range sorted {
		b, err := json.Marshal(r.Object)
		if err != nil {
			// json.Marshal on map[string]interface{} should never fail in practice
			// but fall back to a stable string representation if it does
			b = []byte(fmt.Sprintf("%v", r.Object))
		}
		h.Write(b)
		if i < len(sorted)-1 {
			h.Write([]byte("\n"))
		}
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}
