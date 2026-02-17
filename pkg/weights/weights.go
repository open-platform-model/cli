// Package weights provides resource ordering weights for Kubernetes resources.
// Resources with lower weights are applied first.
package weights

import (
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Default weights for Kubernetes resources.
// Lower weights are applied first.
const (
	WeightCRD                = -100
	WeightNamespace          = 0
	WeightClusterRole        = 5
	WeightClusterRoleBinding = 5
	WeightServiceAccount     = 10
	WeightRole               = 10
	WeightRoleBinding        = 10
	WeightSecret             = 15
	WeightConfigMap          = 15
	WeightStorageClass       = 20
	WeightPersistentVolume   = 20
	WeightPVC                = 20
	WeightService            = 50
	WeightDeployment         = 100
	WeightStatefulSet        = 100
	WeightDaemonSet          = 100
	WeightJob                = 110
	WeightCronJob            = 110
	WeightIngress            = 150
	WeightNetworkPolicy      = 150
	WeightHPA                = 200
	WeightVPA                = 200
	WeightPDB                = 200
	WeightWebhook            = 500
	WeightDefault            = 1000
)

// gvkWeights maps GVK to weight.
var gvkWeights = map[schema.GroupVersionKind]int{
	// CRDs
	{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}: WeightCRD,

	// Core resources
	{Group: "", Version: "v1", Kind: "Namespace"}:             WeightNamespace,
	{Group: "", Version: "v1", Kind: "ServiceAccount"}:        WeightServiceAccount,
	{Group: "", Version: "v1", Kind: "Secret"}:                WeightSecret,
	{Group: "", Version: "v1", Kind: "ConfigMap"}:             WeightConfigMap,
	{Group: "", Version: "v1", Kind: "PersistentVolume"}:      WeightPersistentVolume,
	{Group: "", Version: "v1", Kind: "PersistentVolumeClaim"}: WeightPVC,
	{Group: "", Version: "v1", Kind: "Service"}:               WeightService,

	// RBAC
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRole"}:        WeightClusterRole,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "ClusterRoleBinding"}: WeightClusterRoleBinding,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "Role"}:               WeightRole,
	{Group: "rbac.authorization.k8s.io", Version: "v1", Kind: "RoleBinding"}:        WeightRoleBinding,

	// Storage
	{Group: "storage.k8s.io", Version: "v1", Kind: "StorageClass"}: WeightStorageClass,

	// Workloads
	{Group: "apps", Version: "v1", Kind: "Deployment"}:  WeightDeployment,
	{Group: "apps", Version: "v1", Kind: "StatefulSet"}: WeightStatefulSet,
	{Group: "apps", Version: "v1", Kind: "DaemonSet"}:   WeightDaemonSet,
	{Group: "apps", Version: "v1", Kind: "ReplicaSet"}:  WeightDeployment,

	// Batch
	{Group: "batch", Version: "v1", Kind: "Job"}:     WeightJob,
	{Group: "batch", Version: "v1", Kind: "CronJob"}: WeightCronJob,

	// Networking
	{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"}:       WeightIngress,
	{Group: "networking.k8s.io", Version: "v1", Kind: "NetworkPolicy"}: WeightNetworkPolicy,

	// Autoscaling
	{Group: "autoscaling", Version: "v2", Kind: "HorizontalPodAutoscaler"}:      WeightHPA,
	{Group: "autoscaling", Version: "v1", Kind: "HorizontalPodAutoscaler"}:      WeightHPA,
	{Group: "autoscaling.k8s.io", Version: "v1", Kind: "VerticalPodAutoscaler"}: WeightVPA,

	// Policy
	{Group: "policy", Version: "v1", Kind: "PodDisruptionBudget"}: WeightPDB,

	// Admission webhooks
	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "ValidatingWebhookConfiguration"}: WeightWebhook,
	{Group: "admissionregistration.k8s.io", Version: "v1", Kind: "MutatingWebhookConfiguration"}:   WeightWebhook,
}

// kindWeights maps Kind to weight (for unknown groups/versions).
var kindWeights = map[string]int{
	"Namespace":                      WeightNamespace,
	"ServiceAccount":                 WeightServiceAccount,
	"Secret":                         WeightSecret,
	"ConfigMap":                      WeightConfigMap,
	"PersistentVolume":               WeightPersistentVolume,
	"PersistentVolumeClaim":          WeightPVC,
	"Service":                        WeightService,
	"ClusterRole":                    WeightClusterRole,
	"ClusterRoleBinding":             WeightClusterRoleBinding,
	"Role":                           WeightRole,
	"RoleBinding":                    WeightRoleBinding,
	"StorageClass":                   WeightStorageClass,
	"Deployment":                     WeightDeployment,
	"StatefulSet":                    WeightStatefulSet,
	"DaemonSet":                      WeightDaemonSet,
	"ReplicaSet":                     WeightDeployment,
	"Job":                            WeightJob,
	"CronJob":                        WeightCronJob,
	"Ingress":                        WeightIngress,
	"NetworkPolicy":                  WeightNetworkPolicy,
	"HorizontalPodAutoscaler":        WeightHPA,
	"VerticalPodAutoscaler":          WeightVPA,
	"PodDisruptionBudget":            WeightPDB,
	"ValidatingWebhookConfiguration": WeightWebhook,
	"MutatingWebhookConfiguration":   WeightWebhook,
	"CustomResourceDefinition":       WeightCRD,
}

// GetWeight returns the weight for a GVK.
// Lower weights should be applied first.
func GetWeight(gvk schema.GroupVersionKind) int {
	// Try exact match first
	if weight, ok := gvkWeights[gvk]; ok {
		return weight
	}

	// Fall back to kind-only match
	if weight, ok := kindWeights[gvk.Kind]; ok {
		return weight
	}

	// Default weight for unknown resources
	return WeightDefault
}
