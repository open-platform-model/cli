package kubernetes

import (
	"context"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

var (
	// ErrDeleteFailed is returned when a resource fails to delete.
	ErrDeleteFailed = errors.New("delete failed")
	// ErrResourceNotFound is returned when a resource is not found.
	ErrResourceNotFound = errors.New("resource not found")
)

// DeleteOptions configures the delete operation.
type DeleteOptions struct {
	// DryRun shows what would be deleted without deleting.
	DryRun bool

	// Force removes finalizers and force-deletes stuck resources.
	Force bool

	// Timeout for the operation.
	Timeout time.Duration

	// PropagationPolicy determines how dependents are deleted.
	// Defaults to Background.
	PropagationPolicy metav1.DeletionPropagation
}

// DeleteResult contains the results of a delete operation.
type DeleteResult struct {
	// Deleted is the count of resources deleted.
	Deleted int

	// NotFound is the count of resources not found.
	NotFound int

	// Errors contains any errors that occurred.
	Errors []error
}

// Delete deletes resources in reverse weight order.
func (c *Client) Delete(ctx context.Context, resources []*unstructured.Unstructured, opts DeleteOptions) (*DeleteResult, error) {
	result := &DeleteResult{}

	// Sort for correct delete order (reverse of apply)
	SortForDelete(resources)

	// Set default propagation policy
	propagation := opts.PropagationPolicy
	if propagation == "" {
		propagation = metav1.DeletePropagationBackground
	}

	for _, resource := range resources {
		err := c.deleteResource(ctx, resource, opts.DryRun, opts.Force, propagation)
		if err != nil {
			if apierrors.IsNotFound(err) {
				result.NotFound++
				continue
			}
			if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
				return nil, fmt.Errorf("%w: %v", ErrPermissionDenied, err)
			}
			result.Errors = append(result.Errors, fmt.Errorf("deleting %s/%s: %w",
				resource.GetKind(), resource.GetName(), err))
			continue
		}
		result.Deleted++
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("%w: %d errors occurred", ErrDeleteFailed, len(result.Errors))
	}

	return result, nil
}

// DeleteModuleResources discovers and deletes all resources belonging to a module.
func (c *Client) DeleteModuleResources(ctx context.Context, moduleName, moduleNamespace string, opts DeleteOptions) (*DeleteResult, error) {
	resources, err := c.DiscoverModuleResources(ctx, moduleName, moduleNamespace)
	if err != nil {
		return nil, fmt.Errorf("discovering resources: %w", err)
	}

	if len(resources) == 0 {
		return &DeleteResult{}, nil
	}

	return c.Delete(ctx, resources, opts)
}

// DeleteBundleResources discovers and deletes all resources belonging to a bundle.
func (c *Client) DeleteBundleResources(ctx context.Context, bundleName, bundleNamespace string, opts DeleteOptions) (*DeleteResult, error) {
	resources, err := c.DiscoverBundleResources(ctx, bundleName, bundleNamespace)
	if err != nil {
		return nil, fmt.Errorf("discovering resources: %w", err)
	}

	if len(resources) == 0 {
		return &DeleteResult{}, nil
	}

	return c.Delete(ctx, resources, opts)
}

// deleteResource deletes a single resource.
func (c *Client) deleteResource(ctx context.Context, resource *unstructured.Unstructured, dryRun, force bool, propagation metav1.DeletionPropagation) error {
	// Get the GVR for this resource
	gvk := resource.GroupVersionKind()
	mapping, err := c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("getting REST mapping: %w", err)
	}

	// If force is enabled, remove finalizers first
	if force && len(resource.GetFinalizers()) > 0 {
		if err := c.removeFinalizers(ctx, resource, dryRun); err != nil {
			return fmt.Errorf("removing finalizers: %w", err)
		}
	}

	// Build delete options
	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}

	if dryRun {
		deleteOpts.DryRun = []string{metav1.DryRunAll}
	}

	// Delete the resource
	resourceClient := c.Dynamic.Resource(mapping.Resource)

	if resource.GetNamespace() != "" {
		return resourceClient.Namespace(resource.GetNamespace()).
			Delete(ctx, resource.GetName(), deleteOpts)
	}

	return resourceClient.Delete(ctx, resource.GetName(), deleteOpts)
}

// removeFinalizers removes all finalizers from a resource.
func (c *Client) removeFinalizers(ctx context.Context, resource *unstructured.Unstructured, dryRun bool) error {
	// Get the GVR for this resource
	gvk := resource.GroupVersionKind()
	mapping, err := c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return err
	}

	// Create a patch to remove finalizers
	patch := []byte(`{"metadata":{"finalizers":null}}`)

	patchOpts := metav1.PatchOptions{}
	if dryRun {
		patchOpts.DryRun = []string{metav1.DryRunAll}
	}

	resourceClient := c.Dynamic.Resource(mapping.Resource)

	if resource.GetNamespace() != "" {
		_, err = resourceClient.Namespace(resource.GetNamespace()).
			Patch(ctx, resource.GetName(), "application/merge-patch+json", patch, patchOpts)
	} else {
		_, err = resourceClient.
			Patch(ctx, resource.GetName(), "application/merge-patch+json", patch, patchOpts)
	}

	return err
}
