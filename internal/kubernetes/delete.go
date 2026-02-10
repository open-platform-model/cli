package kubernetes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/internal/output"
)

// DeleteOptions configures a delete operation.
type DeleteOptions struct {
	// ModuleName is the module name to delete.
	// Mutually exclusive with ReleaseID.
	ModuleName string

	// Namespace is the namespace to search for resources.
	Namespace string

	// ReleaseID is the release identity UUID for discovery.
	// Mutually exclusive with ModuleName.
	ReleaseID string

	// DryRun previews resources to delete without removing them.
	DryRun bool

	// Wait waits for resources to be fully deleted.
	Wait bool
}

// DeleteResult contains the outcome of a delete operation.
type DeleteResult struct {
	// Deleted is the number of resources successfully deleted.
	Deleted int

	// Resources lists all discovered resources (for dry-run display).
	Resources []*unstructured.Unstructured

	// Errors contains per-resource errors (non-fatal).
	Errors []ResourceError
}

// Delete removes all resources belonging to a module deployment.
// Resources are discovered via OPM labels and deleted in reverse weight order.
// Returns NoResourcesFoundError when no resources match the selector.
func Delete(ctx context.Context, client *Client, opts DeleteOptions) (*DeleteResult, error) {
	result := &DeleteResult{}

	// Use module name for logging if available, otherwise use ReleaseID
	logName := opts.ModuleName
	if logName == "" {
		logName = fmt.Sprintf("release-id:%s", opts.ReleaseID)
	}
	modLog := output.ModuleLogger(logName)

	// Discover resources via labels
	resources, err := DiscoverResources(ctx, client, DiscoveryOptions{
		ModuleName: opts.ModuleName,
		Namespace:  opts.Namespace,
		ReleaseID:  opts.ReleaseID,
	})
	if err != nil {
		return nil, fmt.Errorf("discovering module resources: %w", err)
	}

	result.Resources = resources

	// Return error when no resources found
	if len(resources) == 0 {
		return nil, &NoResourcesFoundError{
			ModuleName: opts.ModuleName,
			ReleaseID:  opts.ReleaseID,
			Namespace:  opts.Namespace,
		}
	}

	// Sort in reverse weight order (highest weight first = delete webhooks before deployments)
	SortByWeightDescending(resources)

	// Delete each resource
	for _, res := range resources {
		kind := res.GetKind()
		name := res.GetName()
		ns := res.GetNamespace()

		if opts.DryRun {
			output.Println(output.FormatResourceLine(kind, ns, name, output.StatusUnchanged))
			result.Deleted++
			continue
		}

		if err := deleteResource(ctx, client, res); err != nil {
			modLog.Warn(fmt.Sprintf("deleting %s/%s: %v", kind, name, err))
			result.Errors = append(result.Errors, ResourceError{
				Kind:      kind,
				Name:      name,
				Namespace: ns,
				Err:       err,
			})
			continue
		}

		output.Println(output.FormatResourceLine(kind, ns, name, output.StatusDeleted))
		result.Deleted++
	}

	return result, nil
}

// deleteResource deletes a single resource with foreground propagation.
func deleteResource(ctx context.Context, client *Client, obj *unstructured.Unstructured) error {
	gvr := gvrFromUnstructured(obj)
	propagation := metav1.DeletePropagationForeground

	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}

	ns := obj.GetNamespace()
	if ns != "" {
		return client.Dynamic.Resource(gvr).Namespace(ns).Delete(ctx, obj.GetName(), deleteOpts)
	}
	return client.Dynamic.Resource(gvr).Delete(ctx, obj.GetName(), deleteOpts)
}

// gvrFromUnstructured derives GroupVersionResource from an unstructured object.
func gvrFromUnstructured(obj *unstructured.Unstructured) schema.GroupVersionResource {
	gvk := obj.GroupVersionKind()
	return schema.GroupVersionResource{
		Group:    gvk.Group,
		Version:  gvk.Version,
		Resource: kindToResource(gvk.Kind),
	}
}
