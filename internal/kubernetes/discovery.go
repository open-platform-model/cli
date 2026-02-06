package kubernetes

import (
	"context"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"github.com/opmodel/cli/pkg/weights"
)

// OPM labels applied to all managed resources.
const (
	LabelManagedBy       = "app.kubernetes.io/managed-by"
	LabelManagedByValue  = "open-platform-model"
	LabelModuleName      = "module.opmodel.dev/name"
	LabelModuleNamespace = "module.opmodel.dev/namespace"
	LabelModuleVersion   = "module.opmodel.dev/version"
	LabelComponentName   = "component.opmodel.dev/name"
)

// FieldManagerName is the field manager used for server-side apply.
const FieldManagerName = "opm"

// BuildModuleSelector creates a label selector that matches all resources
// belonging to a specific module deployment.
func BuildModuleSelector(moduleName, namespace string) labels.Selector {
	return labels.SelectorFromSet(labels.Set{
		LabelManagedBy:       LabelManagedByValue,
		LabelModuleName:      moduleName,
		LabelModuleNamespace: namespace,
	})
}

// DiscoverResources finds all resources belonging to a module deployment
// by querying all API resources with the OPM label selector.
func DiscoverResources(ctx context.Context, client *Client, moduleName, namespace string) ([]*unstructured.Unstructured, error) {
	selector := BuildModuleSelector(moduleName, namespace)

	// Get all API resources from the server
	apiResources, err := discoverAPIResources(client)
	if err != nil {
		return nil, fmt.Errorf("discovering API resources: %w", err)
	}

	var allResources []*unstructured.Unstructured

	for _, apiResource := range apiResources {
		// Skip resources that don't support list
		if !supportsVerb(apiResource.resource, "list") {
			continue
		}

		gvr := schema.GroupVersionResource{
			Group:    apiResource.group,
			Version:  apiResource.version,
			Resource: apiResource.resource.Name,
		}

		var items *unstructured.UnstructuredList
		listOpts := metav1.ListOptions{
			LabelSelector: selector.String(),
		}

		if apiResource.resource.Namespaced {
			// For namespaced resources, search in the target namespace
			items, err = client.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
		} else {
			// For cluster-scoped resources, search globally
			items, err = client.Dynamic.Resource(gvr).List(ctx, listOpts)
		}

		if err != nil {
			// Skip resources we can't list (RBAC, etc.) - log but don't fail
			continue
		}

		for i := range items.Items {
			item := items.Items[i]
			allResources = append(allResources, &item)
		}
	}

	return allResources, nil
}

// SortByWeightDescending sorts resources by weight in descending order (for deletion).
func SortByWeightDescending(resources []*unstructured.Unstructured) {
	sort.SliceStable(resources, func(i, j int) bool {
		wi := weights.GetWeight(resources[i].GroupVersionKind())
		wj := weights.GetWeight(resources[j].GroupVersionKind())
		return wi > wj
	})
}

// apiResourceInfo wraps an API resource with its group/version.
type apiResourceInfo struct {
	group    string
	version  string
	resource metav1.APIResource
}

// discoverAPIResources lists all available API resources from the server.
func discoverAPIResources(client *Client) ([]apiResourceInfo, error) {
	_, apiResourceLists, err := client.Clientset.Discovery().ServerGroupsAndResources()
	if err != nil {
		// discovery.ErrGroupDiscoveryFailed is non-fatal - some groups may be unavailable
		if !discovery.IsGroupDiscoveryFailedError(err) {
			return nil, err
		}
	}

	var result []apiResourceInfo
	for _, list := range apiResourceLists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		for _, r := range list.APIResources {
			// Skip subresources (e.g., pods/log)
			if containsSlash(r.Name) {
				continue
			}
			result = append(result, apiResourceInfo{
				group:    gv.Group,
				version:  gv.Version,
				resource: r,
			})
		}
	}

	return result, nil
}

// supportsVerb checks if an API resource supports a specific verb.
func supportsVerb(r metav1.APIResource, verb string) bool {
	for _, v := range r.Verbs {
		if v == verb {
			return true
		}
	}
	return false
}

// containsSlash checks if a string contains a slash (for subresource detection).
func containsSlash(s string) bool {
	for _, c := range s {
		if c == '/' {
			return true
		}
	}
	return false
}
