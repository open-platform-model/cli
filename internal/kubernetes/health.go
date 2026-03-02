package kubernetes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// HealthStatus represents the health state of a resource.
type HealthStatus string

const (
	// HealthReady means the resource is ready and healthy.
	HealthReady HealthStatus = "Ready"
	// HealthNotReady means the resource exists but is not yet ready.
	HealthNotReady HealthStatus = "NotReady"
	// HealthComplete means the resource has completed (e.g., a Job).
	HealthComplete HealthStatus = "Complete"
	// HealthUnknown means the health state could not be determined.
	HealthUnknown HealthStatus = "Unknown"
	// HealthMissing means the resource is tracked in the inventory but no longer
	// exists on the cluster (deleted outside of OPM).
	HealthMissing HealthStatus = "Missing"
	// HealthBound means a PersistentVolumeClaim is bound to a PersistentVolume.
	HealthBound HealthStatus = "Bound"
)

// conditionStatusTrue is the Kubernetes condition status value representing "true".
const conditionStatusTrue = "True"

// workloadKinds are resources that use the Available/Ready condition for health.
// Note: StatefulSet is intentionally excluded — it does not emit conditions
// and must be evaluated via readyReplicas instead.
// Note: DaemonSet is intentionally excluded — it does not reliably emit
// Available/Ready conditions. Its health is conveyed via pod count in the tree.
var workloadKinds = map[string]bool{
	kindDeployment: true,
}

// passiveKinds are resources that are healthy as soon as they exist.
// Note: PersistentVolumeClaim is intentionally excluded — it has a lifecycle
// phase (Pending → Bound → Lost) evaluated by evaluatePVCHealth.
// Note: DaemonSet is included here — its health is conveyed via pod count in
// the tree (like ReplicaSet), not a binary ready/not-ready label.
var passiveKinds = map[string]bool{
	kindDaemonSet:         true,
	"ConfigMap":           true,
	"Secret":              true,
	"Service":             true,
	"ServiceAccount":      true,
	"Namespace":           true,
	"ClusterRole":         true,
	"ClusterRoleBinding":  true,
	"Role":                true,
	"RoleBinding":         true,
	"Ingress":             true,
	"NetworkPolicy":       true,
	"PodDisruptionBudget": true,
	"ResourceQuota":       true,
	"LimitRange":          true,
	"StorageClass":        true,
	"PriorityClass":       true,
}

// EvaluateHealth determines the health status of a Kubernetes resource
// based on its kind and status conditions.
func EvaluateHealth(resource *unstructured.Unstructured) HealthStatus {
	kind := resource.GetKind()

	// Workloads: Deployment, DaemonSet — check Available/Ready condition
	if workloadKinds[kind] {
		return evaluateWorkloadHealth(resource)
	}

	// StatefulSet: does not emit status conditions; check readyReplicas instead
	if kind == kindStatefulSet {
		return evaluateStatefulSetHealth(resource)
	}

	// Jobs: check Complete condition
	if kind == kindJob {
		return evaluateJobHealth(resource)
	}

	// CronJobs: always healthy (scheduled)
	if kind == "CronJob" {
		return HealthReady
	}

	// PersistentVolumeClaim: has a lifecycle phase (Pending → Bound → Lost).
	if kind == "PersistentVolumeClaim" {
		return evaluatePVCHealth(resource)
	}

	// Passive resources: healthy on creation
	if passiveKinds[kind] {
		return HealthReady
	}

	// Custom resources: check for Ready condition, fallback to passive
	return evaluateCustomHealth(resource)
}

// IsHealthy returns true if the given health status represents a healthy state.
// Healthy statuses are: HealthReady, HealthComplete, HealthBound.
func IsHealthy(status HealthStatus) bool {
	return status == HealthReady || status == HealthComplete || status == HealthBound
}

// QuickReleaseHealth evaluates aggregate health from pre-fetched resources.
// It calls EvaluateHealth on each live resource and counts healthy vs total.
// missingCount is the number of inventory-tracked resources not found on the cluster.
// Returns the aggregate status, ready count, and total count.
func QuickReleaseHealth(resources []*unstructured.Unstructured, missingCount int) (status HealthStatus, readyCount, totalCount int) {
	total := len(resources) + missingCount
	if total == 0 {
		return HealthUnknown, 0, 0
	}

	ready := 0
	for _, res := range resources {
		if IsHealthy(EvaluateHealth(res)) {
			ready++
		}
	}

	if ready == total {
		return HealthReady, ready, total
	}
	return HealthNotReady, ready, total
}

// evaluatePVCHealth reads the PVC lifecycle phase from status.phase.
// Bound → HealthBound (green), Pending/Lost → their raw phase (yellow).
// Falls back to HealthReady for PVCs with no status yet (e.g. just created).
func evaluatePVCHealth(resource *unstructured.Unstructured) HealthStatus {
	phase, _, _ := unstructured.NestedString(resource.Object, "status", "phase") //nolint:errcheck // best-effort PVC phase display
	if phase != "" {
		return HealthStatus(phase)
	}
	return HealthReady // fallback: PVC created but not yet provisioned
}

// evaluateWorkloadHealth checks the Ready condition on workload resources.
func evaluateWorkloadHealth(resource *unstructured.Unstructured) HealthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Available" || c.Type == "Ready" {
			if c.Status == conditionStatusTrue {
				return HealthReady
			}
			return HealthNotReady
		}
	}
	return HealthNotReady
}

// evaluateStatefulSetHealth checks readyReplicas for StatefulSet resources.
// StatefulSets do not emit Available/Ready status conditions; readiness is
// signaled via readyReplicas reaching the desired replica count.
func evaluateStatefulSetHealth(resource *unstructured.Unstructured) HealthStatus {
	desired, found, _ := unstructured.NestedInt64(resource.Object, "spec", "replicas") //nolint:errcheck // best-effort replica count
	if !found {
		desired = 1 // spec.replicas defaults to 1 when omitted
	}
	if desired == 0 {
		return HealthReady
	}
	ready, _, _ := unstructured.NestedInt64(resource.Object, "status", "readyReplicas") //nolint:errcheck // best-effort ready count
	if ready >= desired {
		return HealthReady
	}
	return HealthNotReady
}

// evaluateJobHealth checks the Complete condition on Job resources.
func evaluateJobHealth(resource *unstructured.Unstructured) HealthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Complete" {
			if c.Status == conditionStatusTrue {
				return HealthComplete
			}
		}
		if c.Type == "Failed" {
			if c.Status == conditionStatusTrue {
				return HealthNotReady
			}
		}
	}
	return HealthNotReady
}

// evaluateCustomHealth checks for a Ready condition on custom resources.
// If no Ready condition exists, treats the resource as passive (healthy).
func evaluateCustomHealth(resource *unstructured.Unstructured) HealthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Ready" {
			if c.Status == conditionStatusTrue {
				return HealthReady
			}
			return HealthNotReady
		}
	}
	// No Ready condition — treat as passive
	return HealthReady
}

// condition represents a Kubernetes status condition.
type condition struct {
	Type   string
	Status string
}

// getConditions extracts status conditions from an unstructured resource.
func getConditions(resource *unstructured.Unstructured) []condition {
	status, found, err := unstructured.NestedMap(resource.Object, "status")
	if err != nil || !found {
		return nil
	}

	rawConditions, found, err := unstructured.NestedSlice(status, "conditions")
	if err != nil || !found {
		return nil
	}

	var conditions []condition
	for _, raw := range rawConditions {
		c, ok := raw.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _, _ := unstructured.NestedString(c, "type")     //nolint:errcheck // best-effort condition parsing
		condStatus, _, _ := unstructured.NestedString(c, "status") //nolint:errcheck // best-effort condition parsing

		if condType != "" {
			conditions = append(conditions, condition{
				Type:   condType,
				Status: condStatus,
			})
		}
	}

	return conditions
}
