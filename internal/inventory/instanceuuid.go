package inventory

import "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

// ExtractInstanceUUID returns the first non-empty
// module-instance.opmodel.dev/uuid label found across the rendered resources —
// the same mechanism the operator uses to populate status.instanceUUID. The
// UUID is a deterministic UUIDv5 computed in core CUE; the CLI never generates
// it. Returns "" when no rendered resource carries the label.
func ExtractInstanceUUID(resources []*unstructured.Unstructured) string {
	for _, r := range resources {
		if r == nil {
			continue
		}
		if v := r.GetLabels()[LabelInstanceUUID]; v != "" {
			return v
		}
	}
	return ""
}
