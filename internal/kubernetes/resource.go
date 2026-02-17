package kubernetes

import (
	"strings"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/dynamic"
)

// gvrFromUnstructured derives GroupVersionResource from an unstructured object.
func gvrFromUnstructured(obj *unstructured.Unstructured) schema.GroupVersionResource {
	gvk := obj.GroupVersionKind()
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: kindToResource(gvk.Kind),
	}
}

// knownKindResources maps Kind to its plural resource name for well-known types.
// This avoids incorrect heuristic pluralization (e.g., Endpoints -> endpointses).
var knownKindResources = map[string]string{
	"Namespace":                        "namespaces",
	"ServiceAccount":                   "serviceaccounts",
	"Secret":                           "secrets",
	"ConfigMap":                        "configmaps",
	"PersistentVolume":                 "persistentvolumes",
	"PersistentVolumeClaim":            "persistentvolumeclaims",
	"Service":                          "services",
	"Endpoints":                        "endpoints",
	"EndpointSlice":                    "endpointslices",
	"ClusterRole":                      "clusterroles",
	"ClusterRoleBinding":               "clusterrolebindings",
	"Role":                             "roles",
	"RoleBinding":                      "rolebindings",
	"StorageClass":                     "storageclasses",
	"Deployment":                       "deployments",
	"StatefulSet":                      "statefulsets",
	"DaemonSet":                        "daemonsets",
	"ReplicaSet":                       "replicasets",
	"Job":                              "jobs",
	"CronJob":                          "cronjobs",
	"Ingress":                          "ingresses",
	"IngressClass":                     "ingressclasses",
	"NetworkPolicy":                    "networkpolicies",
	"HorizontalPodAutoscaler":          "horizontalpodautoscalers",
	"VerticalPodAutoscaler":            "verticalpodautoscalers",
	"PodDisruptionBudget":              "poddisruptionbudgets",
	"ValidatingWebhookConfiguration":   "validatingwebhookconfigurations",
	"MutatingWebhookConfiguration":     "mutatingwebhookconfigurations",
	"CustomResourceDefinition":         "customresourcedefinitions",
	"ResourceQuota":                    "resourcequotas",
	"LimitRange":                       "limitranges",
	"Pod":                              "pods",
	"Node":                             "nodes",
	"Event":                            "events",
	"PriorityClass":                    "priorityclasses",
	"ValidatingAdmissionPolicy":        "validatingadmissionpolicies",
	"ValidatingAdmissionPolicyBinding": "validatingadmissionpolicybindings",
}

// kindToResource converts a Kind to its plural resource name.
// Uses a known lookup table for common types, falls back to heuristic.
func kindToResource(kind string) string {
	if resource, ok := knownKindResources[kind]; ok {
		return resource
	}
	return heuristicPluralize(kind)
}

// heuristicPluralize applies simple English pluralization rules.
func heuristicPluralize(kind string) string {
	lower := strings.ToLower(kind)
	switch {
	case strings.HasSuffix(lower, "ss") || strings.HasSuffix(lower, "sh") || strings.HasSuffix(lower, "ch") || strings.HasSuffix(lower, "x"):
		return lower + "es"
	case strings.HasSuffix(lower, "s"):
		// Already plural (e.g., Endpoints)
		return lower
	case strings.HasSuffix(lower, "y") && !isVowel(lower[len(lower)-2]):
		return lower[:len(lower)-1] + "ies"
	default:
		return lower + "s"
	}
}

func isVowel(b byte) bool {
	return b == 'a' || b == 'e' || b == 'i' || b == 'o' || b == 'u'
}

// ResourceClient returns the appropriate dynamic resource client for the given
// GVR and namespace. If namespace is empty, returns a cluster-scoped client.
func (c *Client) ResourceClient(gvr schema.GroupVersionResource, ns string) dynamic.ResourceInterface {
	if ns != "" {
		return c.Dynamic.Resource(gvr).Namespace(ns)
	}
	return c.Dynamic.Resource(gvr)
}
