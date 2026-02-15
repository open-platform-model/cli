package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"sort"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/labels"
	"k8s.io/apimachinery/pkg/runtime/schema"
	"k8s.io/client-go/discovery"

	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/pkg/weights"
)

// OPM labels applied to all managed resources.
const (
	LabelManagedBy      = "app.kubernetes.io/managed-by"
	labelManagedByValue = "open-platform-model"
	LabelModuleName     = "module.opmodel.dev/name"
	labelModuleVersion  = "module.opmodel.dev/version"
	LabelComponentName  = "component.opmodel.dev/name"
	// labelReleaseID is the release identity UUID label for resource discovery.
	labelReleaseID = "module-release.opmodel.dev/uuid"
	// labelModuleID is the module identity UUID label for resource discovery.
	labelModuleID = "module.opmodel.dev/uuid"
)

// fieldManagerName is the field manager used for server-side apply.
const fieldManagerName = "opm"

// errNoResourcesFound is returned when no resources match the selector.
var errNoResourcesFound = errors.New("no resources found")

// noResourcesFoundError contains details about a failed resource discovery.
type noResourcesFoundError struct {
	// ModuleName is the module name that was searched (empty if using release-id).
	ModuleName string
	// ReleaseID is the release-id that was searched (empty if using name).
	ReleaseID string
	// Namespace is the namespace that was searched.
	Namespace string
}

// Error implements the error interface.
func (e *noResourcesFoundError) Error() string {
	if e.ModuleName != "" {
		return fmt.Sprintf("no resources found for module %s in namespace %s", e.ModuleName, e.Namespace)
	}
	return fmt.Sprintf("no resources found for release-id %s in namespace %s", e.ReleaseID, e.Namespace)
}

// Is implements errors.Is for noResourcesFoundError.
func (e *noResourcesFoundError) Is(target error) bool {
	return target == errNoResourcesFound
}

// IsNoResourcesFound reports whether err (or any error in its chain)
// indicates that no resources matched the discovery selector.
func IsNoResourcesFound(err error) bool {
	return errors.Is(err, errNoResourcesFound)
}

// discoveryOptions configures resource discovery.
type DiscoveryOptions struct {
	// ModuleName is the module name to search for (used with name+namespace selector).
	// Mutually exclusive with ReleaseID.
	ModuleName string
	// Namespace is the target namespace for resource lookup.
	Namespace string
	// ReleaseID is the release identity UUID (used with release-id selector).
	// Mutually exclusive with ModuleName.
	ReleaseID string
	// ExcludeOwned excludes resources with ownerReferences from discovery results.
	// Used by delete and diff to prevent attempting to manage controller-managed children.
	ExcludeOwned bool
}

// buildModuleSelector creates a label selector that matches all resources
// belonging to a specific module deployment.
// Namespace scoping is handled by the Kubernetes API call (Namespace().List()),
// so the selector only needs managed-by + module name.
func buildModuleSelector(moduleName string) labels.Selector {
	return labels.SelectorFromSet(labels.Set{
		LabelManagedBy:  labelManagedByValue,
		LabelModuleName: moduleName,
	})
}

// BuildReleaseIDSelector creates a label selector that matches all resources
// with a specific release identity UUID.
func buildReleaseIDSelector(releaseID string) labels.Selector {
	return labels.SelectorFromSet(labels.Set{
		LabelManagedBy: labelManagedByValue,
		labelReleaseID: releaseID,
	})
}

// DiscoverResources finds all resources belonging to a module deployment
// by querying all API resources with an OPM label selector.
//
// Exactly one of ModuleName or ReleaseID must be provided (mutually exclusive).
// Validation of mutual exclusivity should happen at the command layer.
func DiscoverResources(ctx context.Context, client *Client, opts DiscoveryOptions) ([]*unstructured.Unstructured, error) {
	// Build selector based on what's provided
	var selector labels.Selector

	if opts.ReleaseID != "" {
		selector = buildReleaseIDSelector(opts.ReleaseID)
	} else if opts.ModuleName != "" {
		selector = buildModuleSelector(opts.ModuleName)
	} else {
		return nil, fmt.Errorf("either ModuleName or ReleaseID must be provided")
	}

	// Get all API resources from the server
	apiResources, err := discoverAPIResources(client)
	if err != nil {
		return nil, fmt.Errorf("discovering API resources: %w", err)
	}

	// Discover resources with the selector
	resources := discoverWithSelector(ctx, client, apiResources, selector, opts.Namespace, opts.ExcludeOwned)

	return resources, nil
}

// discoverWithSelector finds resources matching a single label selector.
func discoverWithSelector(ctx context.Context, client *Client, apiResources []apiResourceInfo, selector labels.Selector, namespace string, excludeOwned bool) []*unstructured.Unstructured {
	var allResources []*unstructured.Unstructured

	for i := range apiResources {
		apiResource := &apiResources[i]
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
		var err error
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

		for j := range items.Items {
			item := items.Items[j]

			// Filter out resources with ownerReferences if ExcludeOwned is true
			if excludeOwned && len(item.GetOwnerReferences()) > 0 {
				continue
			}

			allResources = append(allResources, &item)
		}
	}

	return allResources
}

// SortByWeightDescending sorts resources by weight in descending order (for deletion).
func sortByWeightDescending(resources []*unstructured.Unstructured) {
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

// discoverAPIResources lists all preferred API resources from the server.
func discoverAPIResources(client *Client) ([]apiResourceInfo, error) {
	// Use ServerPreferredResources instead of ServerGroupsAndResources
	// to get only the preferred version of each resource type
	apiResourceLists, err := client.Clientset.Discovery().ServerPreferredResources()
	if err != nil {
		// discovery.ErrGroupDiscoveryFailed is non-fatal - some groups may be unavailable
		if !discovery.IsGroupDiscoveryFailedError(err) {
			return nil, err
		}
		output.Warn("some API groups unavailable during discovery, results may be incomplete", "err", err)
	}

	var result []apiResourceInfo
	for _, list := range apiResourceLists {
		gv, parseErr := schema.ParseGroupVersion(list.GroupVersion)
		if parseErr != nil {
			continue
		}
		for i := range list.APIResources {
			r := &list.APIResources[i]
			// Skip subresources (e.g., pods/log)
			if containsSlash(r.Name) {
				continue
			}
			result = append(result, apiResourceInfo{
				group:    gv.Group,
				version:  gv.Version,
				resource: *r,
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
