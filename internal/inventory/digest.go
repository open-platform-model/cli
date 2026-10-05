package inventory

import (
	"crypto/sha256"
	"fmt"
	"sort"
	"strings"

	"github.com/open-platform-model/library/opm/k8s/object"
)

// ComputeRenderDigest computes the render digest over a render's exported
// objects, using EXACTLY the operator's algorithm and serialization
// (opm-operator internal/status.RenderDigest): sort by Group, Kind,
// Namespace, Name; hash the concatenation of each object's CUE-export JSON
// (object.Exported.JSON, the bytes of the render's single object.Export —
// CUE field order, NOT sorted-key Go-map JSON). Byte-for-byte parity with
// the operator's digest is kept so a future ownership transfer has a
// recorded value to verify against (0006:D9/D30); do not change one side
// without the other.
//
// The sort key reads the decoded object the way the CUE accessors of the
// retired pkg/core.Resource read the value: the group is the apiVersion up
// to its last "/", and kind, namespace and name are read as strings, "" when
// missing. It deliberately does not use Object.GroupVersionKind(), which
// returns an empty key, kind included, for an apiVersion with more than one
// "/", and would move such an object in the hashed order.
func ComputeRenderDigest(objs []object.Exported) (string, error) {
	keys := make([]digestKey, len(objs))
	for i := range objs {
		keys[i] = keyOf(&objs[i])
	}
	order := make([]int, len(objs))
	for i := range order {
		order[i] = i
	}
	sort.SliceStable(order, func(a, b int) bool {
		ka, kb := keys[order[a]], keys[order[b]]
		if ka.group != kb.group {
			return ka.group < kb.group
		}
		if ka.kind != kb.kind {
			return ka.kind < kb.kind
		}
		if ka.namespace != kb.namespace {
			return ka.namespace < kb.namespace
		}
		return ka.name < kb.name
	})

	h := sha256.New()
	for _, i := range order {
		if objs[i].JSON == nil {
			return "", fmt.Errorf("render digest: object %d (%s/%s) has no exported JSON", i, keys[i].kind, keys[i].name)
		}
		h.Write(objs[i].JSON)
	}
	return fmt.Sprintf("sha256:%x", h.Sum(nil)), nil
}

// digestKey is the four-field sort key of one exported object.
type digestKey struct{ group, kind, namespace, name string }

func keyOf(e *object.Exported) digestKey {
	if e.Object == nil {
		return digestKey{}
	}
	apiVersion := e.Object.GetAPIVersion()
	group := ""
	if idx := strings.LastIndex(apiVersion, "/"); idx >= 0 {
		group = apiVersion[:idx]
	}
	return digestKey{
		group:     group,
		kind:      e.Object.GetKind(),
		namespace: e.Object.GetNamespace(),
		name:      e.Object.GetName(),
	}
}
