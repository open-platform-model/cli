package inventory

import (
	"crypto/sha256"
	"fmt"
	"sort"

	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

// ComputeRenderDigest computes the render digest over the kernel-compiled
// resources, using EXACTLY the operator's algorithm and serialization
// (opm-operator internal/status.RenderDigest): sort by Group, Kind,
// Namespace, Name; hash the concatenation of each resource's CUE-value
// JSON (Resource.MarshalJSON — CUE field order, NOT sorted-key Go-map
// JSON). Byte-for-byte parity with the operator's digest is the D7.4
// handoff verification contract (enhancement 0006 D9/D30); do not change
// one side without the other.
func ComputeRenderDigest(resources []*pkgcore.Resource) (string, error) {
	sorted := make([]*pkgcore.Resource, len(resources))
	copy(sorted, resources)
	sort.SliceStable(sorted, func(i, j int) bool {
		gi, gj := sorted[i].GVK(), sorted[j].GVK()
		if gi.Group != gj.Group {
			return gi.Group < gj.Group
		}
		if gi.Kind != gj.Kind {
			return gi.Kind < gj.Kind
		}
		if sorted[i].Namespace() != sorted[j].Namespace() {
			return sorted[i].Namespace() < sorted[j].Namespace()
		}
		return sorted[i].Name() < sorted[j].Name()
	})

	h := sha256.New()
	for _, r := range sorted {
		b, err := r.MarshalJSON()
		if err != nil {
			return "", fmt.Errorf("render digest: %w", err)
		}
		h.Write(b)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}
