package kubernetes

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/open-platform-model/cli/pkg/resourceorder"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/output"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
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

	// InstanceUUID is the instance's recorded UUID (the ModuleInstance's
	// status.instanceUUID). A live object whose UUID label differs is left
	// behind. Empty disables the UUID comparison, as in the operator's prune:
	// every object is then judged on its managed-by label alone.
	InstanceUUID string

	// DryRun previews resources to delete without removing them. The
	// protected-kind and ownership checks run in a dry run too, so the
	// preview lists the same left-behind set a real run would.
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

	// Unreadable lists tracked resources the caller's discovery could not
	// read. Each is a per-resource error (DeleteResult.Errors), so the caller
	// keeps the ModuleInstance and a re-run retries, except a protected kind,
	// which is left behind as it would be if read.
	Unreadable []UnreadableResource
}

// UnreadableResource is a tracked resource whose discovery read failed with an
// error other than NotFound. Delete treats it as DeleteOptions.Unreadable
// describes; GetInstanceStatus lists it with health Unknown.
type UnreadableResource struct {
	Group     string
	Kind      string
	Namespace string
	Name      string
	Err       error
}

// DeleteResult contains the outcome of a delete operation.
type DeleteResult struct {
	// Deleted is the number of resources successfully deleted.
	Deleted int

	// Resources lists all discovered resources (for dry-run display).
	Resources []*unstructured.Unstructured

	// LeftBehind lists the resources Delete declined to remove: a protected
	// kind (IsProtectedKind), or an object whose live labels show it is no
	// longer OPM-managed or belongs to another instance. They are not errors.
	LeftBehind []LeftBehindResource

	// Errors contains per-resource errors (non-fatal).
	Errors []resourceError
}

// LeftBehindResource is a tracked resource Delete did not remove, with the
// reason shown to the user.
type LeftBehindResource struct {
	Kind      string
	Namespace string
	Name      string
	Reason    string
}

// Reasons a tracked resource is left behind by Delete, besides
// ProtectedKindReason.
const (
	reasonNotManaged    = "no longer managed by OPM"
	reasonOtherInstance = "owned by another instance"
)

// Delete removes the resources belonging to an instance deployment.
// opts.InventoryLive must be pre-fetched from the ModuleInstance CR inventory by
// the caller. Resources are deleted in reverse weight order. The ModuleInstance
// CR itself is deleted last by the caller, after Delete returns.
//
// A CRD or Namespace is never deleted. Every other object is read again just
// before its delete and deleted only while it is still OPM-managed and, when
// both sides carry one, still has this instance's UUID; otherwise it is left
// behind (DeleteResult.LeftBehind). An object that is already gone counts
// neither as deleted nor as an error; any other read error is a per-resource
// error, so the caller keeps the ModuleInstance and a re-run retries. A
// resource the caller's discovery could not read (opts.Unreadable) gets the
// same outcome without a second read.
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
	if len(resources) == 0 && len(opts.Unreadable) == 0 && !opts.InventoryRecordExists {
		return nil, &noResourcesFoundError{
			InstanceName: opts.InstanceName,
			InstanceID:   opts.InstanceID,
			Namespace:    opts.Namespace,
		}
	}

	instanceLog.Debug("resources to delete", "count", len(resources))

	recordUnreadable(result, opts.Unreadable, instanceLog)

	// Sort in reverse weight order (highest weight first = delete webhooks before deployments)
	SortObjects(resources, resourceorder.Descending)

	// Delete each workload resource
	for _, res := range resources {
		kind := res.GetKind()
		name := res.GetName()
		ns := res.GetNamespace()

		reason, gone, err := checkDeletable(ctx, client, res, opts.InstanceUUID)
		switch {
		case err != nil:
			instanceLog.Warn(fmt.Sprintf("reading %s/%s: %v", kind, name, err))
			result.Errors = append(result.Errors, resourceError{Kind: kind, Name: name, Namespace: ns, Err: err})
			continue
		case gone:
			instanceLog.Debug("resource already gone", "kind", kind, "namespace", ns, "name", name)
			continue
		case reason != "":
			result.LeftBehind = append(result.LeftBehind, LeftBehindResource{Kind: kind, Namespace: ns, Name: name, Reason: reason})
			continue
		}

		if opts.DryRun {
			instanceLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusUnchanged))
			result.Deleted++
			continue
		}

		if err := deleteResource(ctx, client, res); err != nil {
			if apierrors.IsNotFound(err) {
				// Gone between the re-read and the delete: already done.
				instanceLog.Debug("resource already gone", "kind", kind, "namespace", ns, "name", name)
				continue
			}
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

// recordUnreadable adds the resources discovery could not read to result: a
// protected kind is left behind, as checkDeletable would leave it if read, and
// any other is a per-resource error worded like a failed re-read.
func recordUnreadable(result *DeleteResult, unreadable []UnreadableResource, instanceLog *log.Logger) {
	for _, u := range unreadable {
		if IsProtectedKind(u.Group, u.Kind) {
			result.LeftBehind = append(result.LeftBehind, LeftBehindResource{Kind: u.Kind, Namespace: u.Namespace, Name: u.Name, Reason: ProtectedKindReason})
			continue
		}
		instanceLog.Warn(fmt.Sprintf("reading %s/%s: %v", u.Kind, u.Name, u.Err))
		result.Errors = append(result.Errors, resourceError{Kind: u.Kind, Name: u.Name, Namespace: u.Namespace, Err: u.Err})
	}
}

// checkDeletable decides whether Delete may remove obj. It returns a non-empty
// reason when the object is left behind, gone when the live object no longer
// exists, and an error when the live read fails for any other reason.
func checkDeletable(ctx context.Context, client *Client, obj *unstructured.Unstructured, instanceUUID string) (reason string, gone bool, err error) {
	gvk := obj.GroupVersionKind()
	if IsProtectedKind(gvk.Group, gvk.Kind) {
		return ProtectedKindReason, false, nil
	}

	live, err := client.ResourceClient(GVRFromUnstructured(obj), obj.GetNamespace()).Get(ctx, obj.GetName(), metav1.GetOptions{})
	if err != nil {
		if apierrors.IsNotFound(err) {
			return "", true, nil
		}
		return "", false, err
	}

	labels := live.GetLabels()
	if !pkgcore.IsOPMManagedBy(labels[pkgcore.LabelManagedBy]) {
		return reasonNotManaged, false, nil
	}
	// The operator's tolerance: an object without a UUID label predates UUID
	// stamping, and an instance without a recorded UUID has nothing to compare.
	if liveUUID := labels[pkgcore.LabelModuleInstanceUUID]; instanceUUID != "" && liveUUID != "" && liveUUID != instanceUUID {
		return reasonOtherInstance, false, nil
	}
	return "", false, nil
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
