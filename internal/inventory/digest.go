package inventory

import (
	"crypto/sha256"
	"encoding/json"
	"fmt"
	"sort"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/pkg/weights"
)

// SortResources sorts a slice of *build.Resource with a deterministic 5-key total ordering:
// weight (ascending), group (alpha), kind (alpha), namespace (alpha), name (alpha).
//
// This sort is shared between ComputeManifestDigest and the pipeline output sort,
// ensuring that opm mod build output and the inventory digest use the same ordering.
func SortResources(resources []*build.Resource) {
	sort.SliceStable(resources, func(i, j int) bool {
		ri, rj := resources[i], resources[j]

		wi := weights.GetWeight(ri.GVK())
		wj := weights.GetWeight(rj.GVK())
		if wi != wj {
			return wi < wj
		}

		gi, gj := ri.GVK().Group, rj.GVK().Group
		if gi != gj {
			return gi < gj
		}

		ki, kj := ri.GVK().Kind, rj.GVK().Kind
		if ki != kj {
			return ki < kj
		}

		nsi, nsj := ri.Namespace(), rj.Namespace()
		if nsi != nsj {
			return nsi < nsj
		}

		return ri.Name() < rj.Name()
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
func ComputeManifestDigest(resources []*build.Resource) string {
	// Work on a copy to avoid mutating the caller's slice
	sorted := make([]*build.Resource, len(resources))
	copy(sorted, resources)
	SortResources(sorted)

	h := sha256.New()
	for i, r := range sorted {
		b, err := json.Marshal(r.Object.Object)
		if err != nil {
			// json.Marshal on map[string]interface{} should never fail in practice
			// but fall back to a stable string representation if it does
			b = []byte(fmt.Sprintf("%v", r.Object.Object))
		}
		h.Write(b)
		if i < len(sorted)-1 {
			h.Write([]byte("\n"))
		}
	}

	return fmt.Sprintf("sha256:%x", h.Sum(nil))
}
