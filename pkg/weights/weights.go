// Package weights provides resource weight constants for deployment ordering.
// Weights determine the order in which resources are applied (lower = earlier)
// and deleted (higher = earlier deletion, reverse order).
//
// Based on spec Section 6.2 - Resource Ordering.
package weights

// ResourceWeight maps Kubernetes kinds to their apply/delete weights.
// Lower weights are applied first, deleted last.
var ResourceWeight = map[string]int{
	// CRDs must be applied before any custom resources
	"CustomResourceDefinition": -100,

	// Namespaces before namespaced resources
	"Namespace": 0,

	// RBAC resources early
	"ClusterRole":        5,
	"ClusterRoleBinding": 5,
	"ResourceQuota":      5,
	"LimitRange":         5,

	// Service accounts and namespace-scoped RBAC
	"ServiceAccount": 10,
	"Role":           10,
	"RoleBinding":    10,

	// Configuration resources before workloads that use them
	"Secret":    15,
	"ConfigMap": 15,

	// Storage resources
	"StorageClass":          20,
	"PersistentVolume":      20,
	"PersistentVolumeClaim": 20,

	// Services before workloads (for DNS resolution)
	"Service": 50,

	// Workloads
	"DaemonSet":   100,
	"Deployment":  100,
	"StatefulSet": 100,
	"ReplicaSet":  100,

	// Jobs after workloads
	"Job":     110,
	"CronJob": 110,

	// Network policies and ingress after workloads
	"Ingress":       150,
	"NetworkPolicy": 150,

	// Autoscaling after workloads exist
	"HorizontalPodAutoscaler": 200,

	// Webhooks last (they validate/mutate other resources)
	"ValidatingWebhookConfiguration": 500,
	"MutatingWebhookConfiguration":   500,
}

// DefaultWeight is used for unknown resource kinds.
const DefaultWeight = 100

// GetWeight returns the weight for a given resource kind.
// Returns DefaultWeight if the kind is not in the map.
func GetWeight(kind string) int {
	if weight, ok := ResourceWeight[kind]; ok {
		return weight
	}
	return DefaultWeight
}

// KindsByWeight returns all known kinds sorted by weight (ascending).
func KindsByWeight() []string {
	// Pre-sorted list of kinds by weight
	return []string{
		"CustomResourceDefinition",
		"Namespace",
		"ClusterRole",
		"ClusterRoleBinding",
		"ResourceQuota",
		"LimitRange",
		"ServiceAccount",
		"Role",
		"RoleBinding",
		"Secret",
		"ConfigMap",
		"StorageClass",
		"PersistentVolume",
		"PersistentVolumeClaim",
		"Service",
		"DaemonSet",
		"Deployment",
		"StatefulSet",
		"ReplicaSet",
		"Job",
		"CronJob",
		"Ingress",
		"NetworkPolicy",
		"HorizontalPodAutoscaler",
		"ValidatingWebhookConfiguration",
		"MutatingWebhookConfiguration",
	}
}
