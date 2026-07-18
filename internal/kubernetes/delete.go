package kubernetes

import (
	"context"
	"fmt"
	"sort"

	"github.com/open-platform-model/cli/pkg/resourceorder"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/output"
)

// DeleteOptions configures a delete operation.
type DeleteOptions struct {
	// InstanceName is the instance name to delete.
	// Mutually exclusive with InstanceID.
	InstanceName string

	// Namespace is the namespace to search for resources.
	Namespace string

	// InstanceID is the instance identity UUID for discovery.
	// Mutually exclusive with InstanceName.
	InstanceID string

	// DryRun previews resources to delete without removing them.
	DryRun bool

	// InventoryLive is the list of live resources pre-fetched from the
	// ModuleInstance CR inventory by the caller. Resources are deleted from this
	// list. When nil or empty (and InventoryRecordExists is false), Delete
	// returns noResourcesFoundError.
	InventoryLive []*unstructured.Unstructured

	// InventoryRecordExists indicates a ModuleInstance CR is present for the
	// instance. When true, an empty InventoryLive is not treated as
	// "not found" — the caller deletes the CR itself (last) after Delete
	// returns.
	InventoryRecordExists bool
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

// Delete removes all resources belonging to an instance deployment.
// opts.InventoryLive must be pre-fetched from the ModuleInstance CR inventory by
// the caller. Resources are deleted in reverse weight order. The ModuleInstance
// CR itself is deleted last by the caller, after Delete returns.
func Delete(ctx context.Context, client *Client, opts DeleteOptions) (*DeleteResult, error) {
	result := &DeleteResult{}

	// Use instance name for logging if available, otherwise use InstanceID
	logName := opts.InstanceName
	if logName == "" {
		logName = fmt.Sprintf("instance-id:%s", opts.InstanceID)
	}
	instanceLog := output.InstanceLogger(logName)

	resources := opts.InventoryLive

	instanceLog.Debug("deleting instance resources from inventory",
		"instance", logName,
		"namespace", opts.Namespace,
		"count", len(resources),
	)

	result.Resources = resources

	// Return error when no resources found and no ModuleInstance CR to delete.
	if len(resources) == 0 && !opts.InventoryRecordExists {
		return nil, &noResourcesFoundError{
			InstanceName: opts.InstanceName,
			InstanceID:   opts.InstanceID,
			Namespace:    opts.Namespace,
		}
	}

	instanceLog.Debug("resources to delete", "count", len(resources))

	// Sort in reverse weight order (highest weight first = delete webhooks before deployments)
	sortByWeightDescending(resources)

	// Delete each workload resource
	for _, res := range resources {
		kind := res.GetKind()
		name := res.GetName()
		ns := res.GetNamespace()

		if opts.DryRun {
			instanceLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusUnchanged))
			result.Deleted++
			continue
		}

		if err := deleteResource(ctx, client, res); err != nil {
			instanceLog.Warn(fmt.Sprintf("deleting %s/%s: %v", kind, name, err))
			result.Errors = append(result.Errors, resourceError{
				Kind:      kind,
				Name:      name,
				Namespace: ns,
				Err:       err,
			})
			continue
		}

		instanceLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusDeleted))
		result.Deleted++
	}

	// The ModuleInstance CR is deleted last by the caller (after this returns),
	// so the inventory record is only removed once the instance is fully torn down.
	return result, nil
}

// deleteResource deletes a single resource with foreground propagation.
func deleteResource(ctx context.Context, client *Client, obj *unstructured.Unstructured) error {
	gvr := GVRFromUnstructured(obj)
	ns := obj.GetNamespace()
	propagation := metav1.DeletePropagationForeground

	deleteOpts := metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	}

	return client.ResourceClient(gvr, ns).Delete(ctx, obj.GetName(), deleteOpts)
}

// sortByWeightDescending sorts resources by weight in descending order (for deletion).
func sortByWeightDescending(resources []*unstructured.Unstructured) {
	sort.SliceStable(resources, func(i, j int) bool {
		wi := resourceorder.GetWeight(resources[i].GroupVersionKind())
		wj := resourceorder.GetWeight(resources[j].GroupVersionKind())
		return wi > wj
	})
}
