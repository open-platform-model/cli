package kubernetes

import (
	"context"
	"fmt"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/output"
)

// DeleteOptions configures a delete operation.
type DeleteOptions struct {
	// ReleaseName is the release name to delete.
	// Mutually exclusive with ReleaseID.
	ReleaseName string

	// Namespace is the namespace to search for resources.
	Namespace string

	// ReleaseID is the release identity UUID for discovery.
	// Mutually exclusive with ReleaseName.
	ReleaseID string

	// DryRun previews resources to delete without removing them.
	DryRun bool
}

// DeleteResult contains the outcome of a delete operation.
type DeleteResult struct {
	// Deleted is the number of resources successfully deleted.
	Deleted int

	// Resources lists all discovered resources (for dry-run display).
	Resources []*unstructured.Unstructured

	// Errors contains per-resource errors (non-fatal).
	Errors []resourceError
}

// Delete removes all resources belonging to a release deployment.
// Resources are discovered via OPM labels and deleted in reverse weight order.
// Returns noResourcesFoundError when no resources match the selector.
func Delete(ctx context.Context, client *Client, opts DeleteOptions) (*DeleteResult, error) {
	result := &DeleteResult{}

	// Use release name for logging if available, otherwise use ReleaseID
	logName := opts.ReleaseName
	if logName == "" {
		logName = fmt.Sprintf("release-id:%s", opts.ReleaseID)
	}
	modLog := output.ModuleLogger(logName)

	// Discover resources via labels
	output.Debug("discovering release resources",
		"release", logName,
		"namespace", opts.Namespace,
	)

	resources, err := DiscoverResources(ctx, client, DiscoveryOptions{
		ReleaseName:  opts.ReleaseName,
		Namespace:    opts.Namespace,
		ReleaseID:    opts.ReleaseID,
		ExcludeOwned: true,
	})
	if err != nil {
		return nil, fmt.Errorf("discovering release resources: %w", err)
	}

	result.Resources = resources

	// Return error when no resources found
	if len(resources) == 0 {
		return nil, &noResourcesFoundError{
			ReleaseName: opts.ReleaseName,
			ReleaseID:   opts.ReleaseID,
			Namespace:   opts.Namespace,
		}
	}

	output.Debug("discovered resources", "count", len(resources))

	// Sort in reverse weight order (highest weight first = delete webhooks before deployments)
	sortByWeightDescending(resources)

	// Delete each resource
	for _, res := range resources {
		kind := res.GetKind()
		name := res.GetName()
		ns := res.GetNamespace()

		if opts.DryRun {
			modLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusUnchanged))
			result.Deleted++
			continue
		}

		if err := deleteResource(ctx, client, res); err != nil {
			modLog.Warn(fmt.Sprintf("deleting %s/%s: %v", kind, name, err))
			result.Errors = append(result.Errors, resourceError{
				Kind:      kind,
				Name:      name,
				Namespace: ns,
				Err:       err,
			})
			continue
		}

		modLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusDeleted))
		result.Deleted++
	}

	return result, nil
}

// deleteResource deletes a single resource with foreground propagation.
func deleteResource(ctx context.Context, client *Client, obj *unstructured.Unstructured) error {
	gvr := gvrFromUnstructured(obj)
	ns := obj.GetNamespace()
	propagation := metav1.DeletePropagationForeground

	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}

	return client.ResourceClient(gvr, ns).Delete(ctx, obj.GetName(), deleteOpts)
}
