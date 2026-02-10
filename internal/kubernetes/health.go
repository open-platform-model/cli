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
)

// workloadKinds are resources that use the Ready condition for health.
var workloadKinds = map[string]bool{
	"Deployment":  true,
	"StatefulSet": true,
	"DaemonSet":   true,
}

// passiveKinds are resources that are healthy as soon as they exist.
var passiveKinds = map[string]bool{
	"ConfigMap":             true,
	"Secret":                true,
	"Service":               true,
	"PersistentVolumeClaim": true,
	"ServiceAccount":        true,
	"Namespace":             true,
	"ClusterRole":           true,
	"ClusterRoleBinding":    true,
	"Role":                  true,
	"RoleBinding":           true,
	"Ingress":               true,
	"NetworkPolicy":         true,
	"PodDisruptionBudget":   true,
	"ResourceQuota":         true,
	"LimitRange":            true,
	"StorageClass":          true,
	"PriorityClass":         true,
}

// EvaluateHealth determines the health status of a Kubernetes resource
// based on its kind and status conditions.
func EvaluateHealth(resource *unstructured.Unstructured) HealthStatus {
	kind := resource.GetKind()

	// Workloads: Deployment, StatefulSet, DaemonSet — check Ready condition
	if workloadKinds[kind] {
		return evaluateWorkloadHealth(resource)
	}

	// Jobs: check Complete condition
	if kind == "Job" {
		return evaluateJobHealth(resource)
	}

	// CronJobs: always healthy (scheduled)
	if kind == "CronJob" {
		return HealthReady
	}

	// Passive resources: healthy on creation
	if passiveKinds[kind] {
		return HealthReady
	}

	// Custom resources: check for Ready condition, fallback to passive
	return evaluateCustomHealth(resource)
}

// evaluateWorkloadHealth checks the Ready condition on workload resources.
func evaluateWorkloadHealth(resource *unstructured.Unstructured) HealthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Available" || c.Type == "Ready" {
			if c.Status == "True" {
				return HealthReady
			}
			return HealthNotReady
		}
	}
	return HealthNotReady
}

// evaluateJobHealth checks the Complete condition on Job resources.
func evaluateJobHealth(resource *unstructured.Unstructured) HealthStatus {
	conditions := getConditions(resource)
	for _, c := range conditions {
		if c.Type == "Complete" {
			if c.Status == "True" {
				return HealthComplete
			}
		}
		if c.Type == "Failed" {
			if c.Status == "True" {
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
			if c.Status == "True" {
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
