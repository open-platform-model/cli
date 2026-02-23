package kubernetes

import (
	"fmt"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// extractWideInfo extracts workload-specific wide-format information from a resource.
// Returns nil for unsupported resource kinds.
// Never call this on MissingResource entries (no live object).
func extractWideInfo(resource *unstructured.Unstructured) *wideInfo {
	if resource == nil {
		return nil
	}
	kind := resource.GetKind()
	switch kind {
	case "Deployment", "StatefulSet":
		return extractWorkloadWideInfo(resource)
	case "DaemonSet":
		return extractDaemonSetWideInfo(resource)
	case "PersistentVolumeClaim":
		return extractPVCWideInfo(resource)
	case "Ingress":
		return extractIngressWideInfo(resource)
	default:
		return nil
	}
}

// extractWorkloadWideInfo extracts replicas and image for Deployment/StatefulSet.
// Field extraction is best-effort; missing fields produce zero values, not errors.
func extractWorkloadWideInfo(resource *unstructured.Unstructured) *wideInfo {
	ready, _, _ := unstructured.NestedInt64(resource.Object, "status", "readyReplicas") //nolint:errcheck // best-effort; missing field → 0
	desired, _, _ := unstructured.NestedInt64(resource.Object, "spec", "replicas")      //nolint:errcheck // best-effort; missing field → 0
	image := firstContainerImage(resource)

	wi := &wideInfo{}
	if desired > 0 || ready > 0 {
		wi.Replicas = fmt.Sprintf("%d/%d", ready, desired)
	}
	wi.Image = image
	return wi
}

// extractDaemonSetWideInfo extracts replicas and image for DaemonSet.
// Field extraction is best-effort; missing fields produce zero values, not errors.
func extractDaemonSetWideInfo(resource *unstructured.Unstructured) *wideInfo {
	ready, _, _ := unstructured.NestedInt64(resource.Object, "status", "numberReady")              //nolint:errcheck // best-effort; missing field → 0
	desired, _, _ := unstructured.NestedInt64(resource.Object, "status", "desiredNumberScheduled") //nolint:errcheck // best-effort; missing field → 0
	image := firstContainerImage(resource)

	wi := &wideInfo{}
	if desired > 0 || ready > 0 {
		wi.Replicas = fmt.Sprintf("%d/%d", ready, desired)
	}
	wi.Image = image
	return wi
}

// extractPVCWideInfo extracts storage capacity and phase for PVC.
// Field extraction is best-effort; missing fields produce empty strings.
func extractPVCWideInfo(resource *unstructured.Unstructured) *wideInfo {
	storage, _, _ := unstructured.NestedString(resource.Object, "status", "capacity", "storage") //nolint:errcheck // best-effort; missing field → ""
	phase, _, _ := unstructured.NestedString(resource.Object, "status", "phase")                 //nolint:errcheck // best-effort; missing field → ""

	wi := &wideInfo{}
	switch {
	case storage != "" && phase != "":
		wi.Replicas = fmt.Sprintf("%s (%s)", storage, phase)
	case storage != "":
		wi.Replicas = storage
	case phase != "":
		wi.Replicas = phase
	}
	return wi
}

// extractIngressWideInfo extracts the first rule host for Ingress.
func extractIngressWideInfo(resource *unstructured.Unstructured) *wideInfo {
	rules, _, err := unstructured.NestedSlice(resource.Object, "spec", "rules")
	if err != nil || len(rules) == 0 {
		return &wideInfo{}
	}
	rule, ok := rules[0].(map[string]interface{})
	if !ok {
		return &wideInfo{}
	}
	host, _, _ := unstructured.NestedString(rule, "host") //nolint:errcheck // best-effort; missing field → ""
	return &wideInfo{Image: host}
}

// firstContainerImage returns the image of the first container in the pod template spec.
// Returns "" if not available.
func firstContainerImage(resource *unstructured.Unstructured) string {
	containers, _, err := unstructured.NestedSlice(resource.Object, "spec", "template", "spec", "containers")
	if err != nil || len(containers) == 0 {
		return ""
	}
	c, ok := containers[0].(map[string]interface{})
	if !ok {
		return ""
	}
	image, _, _ := unstructured.NestedString(c, "image") //nolint:errcheck // best-effort; missing field → ""
	return image
}
