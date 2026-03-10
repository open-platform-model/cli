package kubernetes

import (
	"context"
	"encoding/json"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/opmodel/cli/internal/output"
)

// ApplyOptions configures an apply operation.
type ApplyOptions struct {
	// DryRun performs a server-side dry run without persisting changes.
	DryRun bool
}

// ApplyResult contains the outcome of an apply operation.
type ApplyResult struct {
	// Applied is the number of resources successfully applied.
	Applied int

	// Created is the number of resources that were newly created.
	Created int

	// Configured is the number of resources that were modified.
	Configured int

	// Unchanged is the number of resources that had no changes.
	Unchanged int

	// Errors contains per-resource errors (non-fatal).
	Errors []resourceError
}

// resourceError captures an error for a specific resource.
type resourceError struct {
	// Resource identifies the resource.
	Kind      string
	Name      string
	Namespace string

	// Err is the error.
	Err error
}

func (e *resourceError) Error() string {
	if e.Namespace != "" {
		return fmt.Sprintf("%s/%s in %s: %v", e.Kind, e.Name, e.Namespace, e.Err)
	}
	return fmt.Sprintf("%s/%s: %v", e.Kind, e.Name, e.Err)
}

// Apply performs server-side apply for a set of rendered resources.
// Resources are assumed to be already ordered by weight (from RenderResult).
// releaseName is used for logging only.
func Apply(ctx context.Context, client *Client, resources []*unstructured.Unstructured, releaseName string, opts ApplyOptions) (*ApplyResult, error) {
	result := &ApplyResult{}
	releaseLog := output.ReleaseLogger(releaseName)

	for _, res := range resources {
		kind := res.GetKind()
		name := res.GetName()
		ns := res.GetNamespace()

		// Apply the resource
		status, err := applyResource(ctx, client, res, opts)
		if err != nil {
			releaseLog.Warn(fmt.Sprintf("applying %s/%s: %v", kind, name, err))
			result.Errors = append(result.Errors, resourceError{
				Kind:      kind,
				Name:      name,
				Namespace: ns,
				Err:       err,
			})
			continue
		}

		result.Applied++
		switch status {
		case output.StatusCreated:
			result.Created++
		case output.StatusConfigured:
			result.Configured++
		case output.StatusUnchanged:
			result.Unchanged++
		}
		releaseLog.Info(output.FormatResourceLine(kind, ns, name, status))
	}

	return result, nil
}

// applyResource performs server-side apply for a single resource.
// Returns the status of the operation (created, configured, or unchanged).
func applyResource(ctx context.Context, client *Client, obj *unstructured.Unstructured, opts ApplyOptions) (string, error) {
	gvr := gvrFromUnstructured(obj)
	ns := obj.GetNamespace()

	// Check if resource already exists to determine status after apply.
	var existingVersion string
	existing, err := client.ResourceClient(gvr, ns).Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err == nil {
		existingVersion = existing.GetResourceVersion()
	}
	// If GET fails (NotFound or other), existingVersion stays empty -> "created"

	data, err := json.Marshal(obj)
	if err != nil {
		return "", fmt.Errorf("marshaling resource: %w", err)
	}

	patchOpts := metav1.PatchOptions{
		FieldManager: fieldManagerName,
		Force:        output.BoolPtr(true),
	}

	if opts.DryRun {
		patchOpts.DryRun = []string{metav1.DryRunAll}
	}

	result, patchErr := client.ResourceClient(gvr, ns).Patch(
		ctx, obj.GetName(), types.ApplyPatchType, data, patchOpts,
	)

	if patchErr != nil {
		return "", patchErr
	}

	// Determine status from before/after comparison.
	if existingVersion == "" {
		return output.StatusCreated, nil
	}
	if result != nil && result.GetResourceVersion() == existingVersion {
		return output.StatusUnchanged, nil
	}
	return output.StatusConfigured, nil
}
