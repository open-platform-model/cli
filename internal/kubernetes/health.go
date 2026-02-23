package kubernetes

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// healthStatus represents the health state of a resource.
type healthStatus string

const (
	// healthReady means the resource is ready and healthy.
	healthReady healthStatus = "Ready"
	// healthNotReady means the resource exists but is not yet ready.
	healthNotReady healthStatus = "NotReady"
	// healthComplete means the resource has completed (e.g., a Job).
	healthComplete healthStatus = "Complete"
	// healthUnknown means the health state could not be determined.
	healthUnknown healthStatus = "Unknown"
	// healthMissing means the resource is tracked in the inventory but no longer
	// exists on the cluster (deleted outside of OPM).
	healthMissing healthStatus = "Missing"
	// healthBound means a PersistentVolumeClaim is bound to a PersistentVolume.
	healthBound healthStatus = "Bound"
)

// conditionStatusTrue is the Kubernetes condition status value representing "true".
const conditionStatusTrue = "True"

// workloadKinds are resources that use the Available/Ready condition for health.
// Note: StatefulSet is intentionally excluded — it does not emit conditions
// and must be evaluated via readyReplicas instead.
var workloadKinds = map[string]bool{
	kindDeployment: true,
	kindDaemonSet:  true,
}

// passiveKinds are resources that are healthy as soon as they exist.
// Note: PersistentVolumeClaim is intentionally excluded — it has a lifecycle
// phase (Pending → Bound → Lost) evaluated by evaluatePVCHealth.
var passiveKinds = map[string]bool{
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
func evaluateHealth(resource *unstructured.Unstructured) healthStatus {
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
		return healthReady
	}

	// PersistentVolumeClaim: has a lifecycle phase (Pending → Bound → Lost).
	if kind == "PersistentVolumeClaim" {
		return evaluatePVCHealth(resource)
	}

	// Passive resources: healthy on creation
	if passiveKinds[kind] {
		return healthReady
	}

	// Custom resources: check for Ready condition, fallback to passive
	return evaluateCustomHealth(resource)
}

// evaluatePVCHealth reads the PVC lifecycle phase from status.phase.
// Bound → healthBound (green), Pending/Lost → their raw phase (yellow).
// Falls back to healthReady for PVCs with no status yet (e.g. just created).
func evaluatePVCHealth(resource *unstructured.Unstructured) healthStatus {
	phase, _, _ := unstructured.NestedString(resource.Object, "status", "phase") //nolint:errcheck // best-effort PVC phase display
	if phase != "" {
		return healthStatus(phase)
	}
	return healthReady // fallback: PVC created but not yet provisioned
}

// evaluateWorkloadHealth checks the Ready condition on workload resources.
func evaluateWorkloadHealth(resource *unstructured.Unstructured) healthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Available" || c.Type == "Ready" {
			if c.Status == conditionStatusTrue {
				return healthReady
			}
			return healthNotReady
		}
	}
	return healthNotReady
}

// evaluateStatefulSetHealth checks readyReplicas for StatefulSet resources.
// StatefulSets do not emit Available/Ready status conditions; readiness is
// signalled via readyReplicas reaching the desired replica count.
func evaluateStatefulSetHealth(resource *unstructured.Unstructured) healthStatus {
	desired, found, _ := unstructured.NestedInt64(resource.Object, "spec", "replicas") //nolint:errcheck
	if !found {
		desired = 1 // spec.replicas defaults to 1 when omitted
	}
	if desired == 0 {
		return healthReady
	}
	ready, _, _ := unstructured.NestedInt64(resource.Object, "status", "readyReplicas") //nolint:errcheck
	if ready >= desired {
		return healthReady
	}
	return healthNotReady
}

// evaluateJobHealth checks the Complete condition on Job resources.
func evaluateJobHealth(resource *unstructured.Unstructured) healthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Complete" {
			if c.Status == conditionStatusTrue {
				return healthComplete
			}
		}
		if c.Type == "Failed" {
			if c.Status == conditionStatusTrue {
				return healthNotReady
			}
		}
	}
	return healthNotReady
}

// evaluateCustomHealth checks for a Ready condition on custom resources.
// If no Ready condition exists, treats the resource as passive (healthy).
func evaluateCustomHealth(resource *unstructured.Unstructured) healthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Ready" {
			if c.Status == conditionStatusTrue {
				return healthReady
			}
			return healthNotReady
		}
	}
	// No Ready condition — treat as passive
	return healthReady
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

		condType, _, _ := unstructured.NestedString(c, "type")
		condStatus, _, _ := unstructured.NestedString(c, "status")

		if condType != "" {
			conditions = append(conditions, condition{
				Type:   condType,
				Status: condStatus,
			})
		}
	}

	return conditions
}
