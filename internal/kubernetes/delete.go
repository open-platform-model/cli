package kubernetes

import (
	"context"
	"fmt"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
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

	// InventoryLive is the list of live resources pre-fetched from the inventory
	// Secret. When non-nil, resource enumeration uses this list instead of a
	// full label-scan (inventory-first path). Pass nil to fall back to label-scan.
	InventoryLive []*unstructured.Unstructured

	// InventorySecretName is the name of the inventory Secret to delete last.
	// Only used when InventoryLive is non-nil. Empty means no inventory Secret to delete.
	InventorySecretName string

	// InventorySecretNamespace is the namespace of the inventory Secret.
	InventorySecretNamespace string
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
// Resources are discovered via OPM labels (or via inventory when InventoryLive is set)
// and deleted in reverse weight order. Returns noResourcesFoundError when no resources match.
//
// When opts.InventoryLive is non-nil, the inventory-first path is used: the provided
// live resources are deleted, then the inventory Secret is deleted last. This avoids
// derived resources (e.g., Endpoints) that were never applied by OPM being incorrectly
// discovered and deleted via label-scan.
func Delete(ctx context.Context, client *Client, opts DeleteOptions) (*DeleteResult, error) {
	result := &DeleteResult{}

	// Use release name for logging if available, otherwise use ReleaseID
	logName := opts.ReleaseName
	if logName == "" {
		logName = fmt.Sprintf("release-id:%s", opts.ReleaseID)
	}
	modLog := output.ModuleLogger(logName)

	var resources []*unstructured.Unstructured

	if opts.InventoryLive != nil {
		// Inventory-first: use pre-fetched live resources from the inventory.
		// Avoids label-scanning all API types and excludes derived resources.
		output.Debug("using inventory-first deletion",
			"release", logName,
			"namespace", opts.Namespace,
			"count", len(opts.InventoryLive),
		)
		resources = opts.InventoryLive
	} else {
		// Fallback: discover resources via label-scan for backward compatibility.
		output.Debug("discovering release resources via label-scan",
			"release", logName,
			"namespace", opts.Namespace,
		)
		var err error
		resources, err = DiscoverResources(ctx, client, DiscoveryOptions{
			ReleaseName:  opts.ReleaseName,
			Namespace:    opts.Namespace,
			ReleaseID:    opts.ReleaseID,
			ExcludeOwned: true,
		})
		if err != nil {
			return nil, fmt.Errorf("discovering release resources: %w", err)
		}
	}

	result.Resources = resources

	// Return error when no resources found (and no inventory Secret to delete)
	if len(resources) == 0 && opts.InventorySecretName == "" {
		return nil, &noResourcesFoundError{
			ReleaseName: opts.ReleaseName,
			ReleaseID:   opts.ReleaseID,
			Namespace:   opts.Namespace,
		}
	}

	output.Debug("resources to delete", "count", len(resources))

	// Sort in reverse weight order (highest weight first = delete webhooks before deployments)
	sortByWeightDescending(resources)

	// Delete each workload resource
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

	// Delete the inventory Secret last (after all workload resources are gone).
	// This ensures the inventory is only removed when the release is fully deleted.
	if opts.InventorySecretName != "" && !opts.DryRun {
		invSecretNS := opts.InventorySecretNamespace
		if invSecretNS == "" {
			invSecretNS = opts.Namespace
		}
		if err := client.Clientset.CoreV1().Secrets(invSecretNS).Delete(ctx, opts.InventorySecretName, metav1.DeleteOptions{}); err != nil {
			if !apierrors.IsNotFound(err) {
				output.Debug("could not delete inventory Secret", "name", opts.InventorySecretName, "err", err)
			}
		} else {
			output.Debug("deleted inventory Secret", "name", opts.InventorySecretName, "namespace", invSecretNS)
		}
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
