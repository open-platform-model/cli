package kubernetes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// DiscoverOptions configures resource discovery.
type DiscoverOptions struct {
	// Namespace to search in. If empty, searches all namespaces.
	Namespace string

	// IncludeClusterScoped includes cluster-scoped resources.
	IncludeClusterScoped bool
}

// DiscoverResources finds all resources matching the given label selector.
func (c *Client) DiscoverResources(ctx context.Context, selector map[string]string, opts DiscoverOptions) ([]*unstructured.Unstructured, error) {
	var result []*unstructured.Unstructured

	// Get all API resources
	apiResourceLists, err := c.Discovery.ServerPreferredResources()
	if err != nil {
		// Discovery may return partial results with errors
		if len(apiResourceLists) == 0 {
			return nil, fmt.Errorf("discovering API resources: %w", err)
		}
	}

	// Build label selector string
	labelSelector := labels.SelectorFromSet(selector).String()

	// Search each API group/version
	for _, apiResourceList := range apiResourceLists {
		gv, err := schema.ParseGroupVersion(apiResourceList.GroupVersion)
		if err != nil {
			continue
		}

		for _, apiResource := range apiResourceList.APIResources {
			// Skip subresources
			if len(apiResource.Name) == 0 || containsSlash(apiResource.Name) {
				continue
			}

			// Skip if not listable
			if !containsVerb(apiResource.Verbs, "list") {
				continue
			}

			// Skip cluster-scoped if not requested
			if !apiResource.Namespaced && !opts.IncludeClusterScoped {
				continue
			}

			gvr := schema.GroupVersionResource{
				Group:    gv.Group,
				Version:  gv.Version,
				Resource: apiResource.Name,
			}

			// List resources
			resources, err := c.listResources(ctx, gvr, apiResource.Namespaced, opts.Namespace, labelSelector)
			if err != nil {
				continue // Skip resources we can't list (permissions, etc.)
			}

			result = append(result, resources...)
		}
	}

	return result, nil
}

// DiscoverModuleResources finds all resources belonging to a module.
func (c *Client) DiscoverModuleResources(ctx context.Context, moduleName, moduleNamespace string) ([]*unstructured.Unstructured, error) {
	return c.DiscoverResources(ctx, ModuleSelector(moduleName, moduleNamespace), DiscoverOptions{
		Namespace:            moduleNamespace,
		IncludeClusterScoped: true,
	})
}

// DiscoverBundleResources finds all resources belonging to a bundle.
func (c *Client) DiscoverBundleResources(ctx context.Context, bundleName, bundleNamespace string) ([]*unstructured.Unstructured, error) {
	return c.DiscoverResources(ctx, BundleSelector(bundleName, bundleNamespace), DiscoverOptions{
		Namespace:            bundleNamespace,
		IncludeClusterScoped: true,
	})
}

// listResources lists resources of a specific type.
func (c *Client) listResources(ctx context.Context, gvr schema.GroupVersionResource, namespaced bool, namespace, labelSelector string) ([]*unstructured.Unstructured, error) {
	listOpts := metav1.ListOptions{
		LabelSelector: labelSelector,
	}

	var list *unstructured.UnstructuredList
	var err error

	if namespaced && namespace != "" {
		list, err = c.Dynamic.Resource(gvr).Namespace(namespace).List(ctx, listOpts)
	} else if namespaced {
		// Search all namespaces
		list, err = c.Dynamic.Resource(gvr).List(ctx, listOpts)
	} else {
		// Cluster-scoped
		list, err = c.Dynamic.Resource(gvr).List(ctx, listOpts)
	}

	if err != nil {
		return nil, err
	}

	result := make([]*unstructured.Unstructured, len(list.Items))
	for i := range list.Items {
		result[i] = &list.Items[i]
	}
	return result, nil
}

// containsSlash checks if a string contains a slash.
func containsSlash(s string) bool {
	for _, c := range s {
		if c == '/' {
			return true
		}
	}
	return false
}

// containsVerb checks if a verb list contains a specific verb.
func containsVerb(verbs []string, verb string) bool {
	for _, v := range verbs {
		if v == verb {
			return true
		}
	}
	return false
}
