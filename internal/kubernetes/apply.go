package kubernetes

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
)

var (
	// ErrApplyFailed is returned when a resource fails to apply.
	ErrApplyFailed = errors.New("apply failed")
	// ErrPermissionDenied is returned when RBAC permissions are insufficient.
	ErrPermissionDenied = errors.New("permission denied")
)

// ApplyOptions configures the apply operation.
type ApplyOptions struct {
	// DryRun performs server-side dry run without changes.
	DryRun bool

	// Namespace override for namespaced resources.
	Namespace string

	// Labels to inject into all resources.
	Labels Labels

	// Wait blocks until resources are ready.
	Wait bool

	// Timeout for the operation.
	Timeout time.Duration
}

// ApplyResult contains the results of an apply operation.
type ApplyResult struct {
	// Created is the count of resources created.
	Created int

	// Updated is the count of resources updated.
	Updated int

	// Unchanged is the count of resources unchanged.
	Unchanged int

	// Errors contains any errors that occurred.
	Errors []error
}

// Apply applies resources using server-side apply.
func (c *Client) Apply(ctx context.Context, resources []*unstructured.Unstructured, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}

	// Sort for correct apply order
	SortForApply(resources)

	for _, resource := range resources {
		// Inject labels
		if len(opts.Labels) > 0 {
			InjectLabels(resource.Object, opts.Labels)
		}

		// Set namespace if override provided and resource is namespaced
		if opts.Namespace != "" && resource.GetNamespace() == "" {
			// Check if resource is namespaced
			if isNamespaced, err := c.isNamespaced(resource); err == nil && isNamespaced {
				resource.SetNamespace(opts.Namespace)
			}
		}

		// Default namespace for namespaced resources without namespace
		if resource.GetNamespace() == "" {
			if isNamespaced, err := c.isNamespaced(resource); err == nil && isNamespaced {
				resource.SetNamespace(c.DefaultNamespace)
			}
		}

		err := c.applyResource(ctx, resource, opts.DryRun)
		if err != nil {
			if apierrors.IsForbidden(err) || apierrors.IsUnauthorized(err) {
				return nil, fmt.Errorf("%w: %v", ErrPermissionDenied, err)
			}
			result.Errors = append(result.Errors, fmt.Errorf("applying %s/%s: %w",
				resource.GetKind(), resource.GetName(), err))
			continue
		}

		result.Created++ // SSA doesn't distinguish created vs updated
	}

	if len(result.Errors) > 0 {
		return result, fmt.Errorf("%w: %d errors occurred", ErrApplyFailed, len(result.Errors))
	}

	return result, nil
}

// applyResource applies a single resource using server-side apply.
func (c *Client) applyResource(ctx context.Context, resource *unstructured.Unstructured, dryRun bool) error {
	// Get the GVR for this resource
	gvk := resource.GroupVersionKind()
	mapping, err := c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return fmt.Errorf("getting REST mapping: %w", err)
	}

	// Serialize the resource
	data, err := json.Marshal(resource.Object)
	if err != nil {
		return fmt.Errorf("marshaling resource: %w", err)
	}

	// Build patch options
	patchOpts := metav1.PatchOptions{
		FieldManager: FieldManager,
		Force:        ptr(true), // Force to take ownership
	}

	if dryRun {
		patchOpts.DryRun = []string{metav1.DryRunAll}
	}

	// Apply using server-side apply
	var resourceClient = c.Dynamic.Resource(mapping.Resource)

	if resource.GetNamespace() != "" {
		_, err = resourceClient.Namespace(resource.GetNamespace()).
			Patch(ctx, resource.GetName(), types.ApplyPatchType, data, patchOpts)
	} else {
		_, err = resourceClient.
			Patch(ctx, resource.GetName(), types.ApplyPatchType, data, patchOpts)
	}

	return err
}

// isNamespaced checks if a resource is namespace-scoped.
func (c *Client) isNamespaced(resource *unstructured.Unstructured) (bool, error) {
	gvk := resource.GroupVersionKind()
	mapping, err := c.Mapper.RESTMapping(gvk.GroupKind(), gvk.Version)
	if err != nil {
		return false, err
	}
	return mapping.Scope.Name() == "namespace", nil
}

// ptr returns a pointer to the given value.
func ptr[T any](v T) *T {
	return &v
}
