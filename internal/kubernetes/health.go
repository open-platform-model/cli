package kubernetes

import (
	"context"
	"fmt"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// HealthEvaluator evaluates the health of Kubernetes resources.
type HealthEvaluator struct{}

// NewHealthEvaluator creates a new HealthEvaluator.
func NewHealthEvaluator() *HealthEvaluator {
	return &HealthEvaluator{}
}

// EvaluateHealth evaluates the health of a single resource.
func (h *HealthEvaluator) EvaluateHealth(resource *unstructured.Unstructured) ResourceStatus {
	status := ResourceStatus{
		Kind:       resource.GetKind(),
		APIVersion: resource.GetAPIVersion(),
		Name:       resource.GetName(),
		Namespace:  resource.GetNamespace(),
	}

	// Calculate age
	if created := resource.GetCreationTimestamp(); !created.IsZero() {
		status.Age = time.Since(created.Time)
	}

	// Get component label
	if labels := resource.GetLabels(); labels != nil {
		status.Component = labels[LabelComponentName]
	}

	// Evaluate health based on resource type
	switch resource.GetKind() {
	case "Deployment":
		status.Health, status.Message = h.evaluateDeployment(resource)
	case "StatefulSet":
		status.Health, status.Message = h.evaluateStatefulSet(resource)
	case "DaemonSet":
		status.Health, status.Message = h.evaluateDaemonSet(resource)
	case "ReplicaSet":
		status.Health, status.Message = h.evaluateReplicaSet(resource)
	case "Job":
		status.Health, status.Message = h.evaluateJob(resource)
	case "CronJob":
		// CronJobs are always considered healthy
		status.Health = HealthReady
		status.Message = "CronJob active"
	case "Pod":
		status.Health, status.Message = h.evaluatePod(resource)
	case "Service", "ConfigMap", "Secret", "Namespace", "ServiceAccount",
		"Role", "RoleBinding", "ClusterRole", "ClusterRoleBinding",
		"PersistentVolumeClaim", "PersistentVolume":
		// These are considered healthy if they exist
		status.Health = HealthReady
		status.Message = "Created"
	default:
		// For custom resources, check for Ready condition
		status.Health, status.Message = h.evaluateGeneric(resource)
	}

	return status
}

// EvaluateAll evaluates the health of multiple resources.
func (h *HealthEvaluator) EvaluateAll(resources []*unstructured.Unstructured) []ResourceStatus {
	result := make([]ResourceStatus, 0, len(resources))
	for _, r := range resources {
		result = append(result, h.EvaluateHealth(r))
	}
	return result
}

// evaluateDeployment evaluates deployment health.
func (h *HealthEvaluator) evaluateDeployment(obj *unstructured.Unstructured) (HealthStatus, string) {
	// Check conditions
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		return HealthProgressing, "Waiting for conditions"
	}

	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _ := cond["type"].(string)
		condStatus, _ := cond["status"].(string)
		message, _ := cond["message"].(string)

		switch condType {
		case "Available":
			if condStatus == "True" {
				return HealthReady, message
			}
		case "Progressing":
			if condStatus == "True" {
				reason, _ := cond["reason"].(string)
				if reason == "NewReplicaSetAvailable" {
					continue // This is fine
				}
				return HealthProgressing, message
			}
		case "ReplicaFailure":
			if condStatus == "True" {
				return HealthFailed, message
			}
		}
	}

	return HealthNotReady, "Not available"
}

// evaluateStatefulSet evaluates statefulset health.
func (h *HealthEvaluator) evaluateStatefulSet(obj *unstructured.Unstructured) (HealthStatus, string) {
	replicas, _, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	readyReplicas, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")
	currentReplicas, _, _ := unstructured.NestedInt64(obj.Object, "status", "currentReplicas")

	if replicas == 0 {
		return HealthReady, "Scaled to zero"
	}

	if readyReplicas == replicas && currentReplicas == replicas {
		return HealthReady, fmt.Sprintf("%d/%d replicas ready", readyReplicas, replicas)
	}

	if readyReplicas < replicas {
		return HealthProgressing, fmt.Sprintf("%d/%d replicas ready", readyReplicas, replicas)
	}

	return HealthNotReady, fmt.Sprintf("%d/%d replicas ready", readyReplicas, replicas)
}

// evaluateDaemonSet evaluates daemonset health.
func (h *HealthEvaluator) evaluateDaemonSet(obj *unstructured.Unstructured) (HealthStatus, string) {
	desired, _, _ := unstructured.NestedInt64(obj.Object, "status", "desiredNumberScheduled")
	ready, _, _ := unstructured.NestedInt64(obj.Object, "status", "numberReady")
	updated, _, _ := unstructured.NestedInt64(obj.Object, "status", "updatedNumberScheduled")

	if desired == 0 {
		return HealthReady, "No nodes scheduled"
	}

	if ready == desired && updated == desired {
		return HealthReady, fmt.Sprintf("%d/%d nodes ready", ready, desired)
	}

	if ready < desired || updated < desired {
		return HealthProgressing, fmt.Sprintf("%d/%d nodes ready, %d updated", ready, desired, updated)
	}

	return HealthNotReady, fmt.Sprintf("%d/%d nodes ready", ready, desired)
}

// evaluateReplicaSet evaluates replicaset health.
func (h *HealthEvaluator) evaluateReplicaSet(obj *unstructured.Unstructured) (HealthStatus, string) {
	replicas, _, _ := unstructured.NestedInt64(obj.Object, "spec", "replicas")
	readyReplicas, _, _ := unstructured.NestedInt64(obj.Object, "status", "readyReplicas")

	if replicas == 0 {
		return HealthReady, "Scaled to zero"
	}

	if readyReplicas == replicas {
		return HealthReady, fmt.Sprintf("%d/%d replicas ready", readyReplicas, replicas)
	}

	return HealthProgressing, fmt.Sprintf("%d/%d replicas ready", readyReplicas, replicas)
}

// evaluateJob evaluates job health.
func (h *HealthEvaluator) evaluateJob(obj *unstructured.Unstructured) (HealthStatus, string) {
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		return HealthProgressing, "Running"
	}

	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _ := cond["type"].(string)
		condStatus, _ := cond["status"].(string)
		message, _ := cond["message"].(string)

		switch condType {
		case "Complete":
			if condStatus == "True" {
				return HealthReady, "Completed"
			}
		case "Failed":
			if condStatus == "True" {
				return HealthFailed, message
			}
		}
	}

	return HealthProgressing, "Running"
}

// evaluatePod evaluates pod health.
func (h *HealthEvaluator) evaluatePod(obj *unstructured.Unstructured) (HealthStatus, string) {
	phase, _, _ := unstructured.NestedString(obj.Object, "status", "phase")

	switch phase {
	case "Running":
		// Check container statuses
		containerStatuses, _, _ := unstructured.NestedSlice(obj.Object, "status", "containerStatuses")
		ready := 0
		total := len(containerStatuses)
		for _, cs := range containerStatuses {
			status, ok := cs.(map[string]interface{})
			if !ok {
				continue
			}
			if isReady, _ := status["ready"].(bool); isReady {
				ready++
			}
		}
		if total > 0 && ready == total {
			return HealthReady, fmt.Sprintf("Running (%d/%d containers ready)", ready, total)
		}
		return HealthProgressing, fmt.Sprintf("Running (%d/%d containers ready)", ready, total)
	case "Succeeded":
		return HealthReady, "Completed"
	case "Failed":
		return HealthFailed, "Failed"
	case "Pending":
		return HealthProgressing, "Pending"
	default:
		return HealthUnknown, phase
	}
}

// evaluateGeneric evaluates a generic resource by looking for Ready condition.
func (h *HealthEvaluator) evaluateGeneric(obj *unstructured.Unstructured) (HealthStatus, string) {
	// Check for conditions
	conditions, found, _ := unstructured.NestedSlice(obj.Object, "status", "conditions")
	if !found {
		// No conditions - consider healthy if it exists
		return HealthReady, "Created"
	}

	for _, c := range conditions {
		cond, ok := c.(map[string]interface{})
		if !ok {
			continue
		}

		condType, _ := cond["type"].(string)
		condStatus, _ := cond["status"].(string)
		message, _ := cond["message"].(string)

		if condType == "Ready" {
			if condStatus == "True" {
				return HealthReady, message
			}
			return HealthNotReady, message
		}
	}

	return HealthReady, "Created"
}

// GetModuleStatus evaluates and returns the status of a module's resources.
func (c *Client) GetModuleStatus(ctx context.Context, moduleName, moduleNamespace, moduleVersion string) (*ModuleStatus, error) {
	resources, err := c.DiscoverModuleResources(ctx, moduleName, moduleNamespace)
	if err != nil {
		return nil, fmt.Errorf("discovering resources: %w", err)
	}

	evaluator := NewHealthEvaluator()
	resourceStatuses := evaluator.EvaluateAll(resources)

	status := &ModuleStatus{
		Name:      moduleName,
		Version:   moduleVersion,
		Namespace: moduleNamespace,
		Resources: resourceStatuses,
	}
	status.CalculateSummary()

	return status, nil
}

// GetBundleStatus evaluates and returns the status of a bundle's resources.
func (c *Client) GetBundleStatus(ctx context.Context, bundleName, bundleNamespace, bundleVersion string) (*BundleStatus, error) {
	resources, err := c.DiscoverBundleResources(ctx, bundleName, bundleNamespace)
	if err != nil {
		return nil, fmt.Errorf("discovering resources: %w", err)
	}

	evaluator := NewHealthEvaluator()
	resourceStatuses := evaluator.EvaluateAll(resources)

	status := &BundleStatus{
		Name:      bundleName,
		Version:   bundleVersion,
		Namespace: bundleNamespace,
		Resources: resourceStatuses,
	}
	status.CalculateSummary()

	return status, nil
}
