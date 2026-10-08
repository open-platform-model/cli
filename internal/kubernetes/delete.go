package kubernetes

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	"github.com/open-platform-model/library/opm/k8s/object"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/library/opm/k8s/ownership"
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
	// status.instanceUUID), the identity the delete verdict is asked with. A
	// live object whose UUID label or adopt annotation names another instance
	// is left behind. Empty disables the UUID label comparison.
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
	// which is left behind as it would be if read, and a claim Delete keeps
	// (DeleteData false), which is kept.
	Unreadable []UnreadableResource

	// DeleteData is the command's --delete-data: delete tracked
	// PersistentVolumeClaims (IsDataClaim) like any other resource. When
	// false, each is kept and listed in DeleteResult.Kept.
	DeleteData bool
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
	// kind (IsProtectedKind), or an object the delete verdict skipped because
	// it is no longer OPM-managed, belongs to another instance or is adopted
	// by another instance. They are not errors.
	LeftBehind []LeftBehindResource

	// Kept lists the PersistentVolumeClaims Delete kept because
	// DeleteOptions.DeleteData was false. They are kept on purpose: not
	// errors, and reported apart from LeftBehind.
	Kept []LeftBehindResource

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

// KeptClaimReason is the reason of every entry in DeleteResult.Kept.
const KeptClaimReason = "PersistentVolumeClaims are kept unless --delete-data is set"

// Delete removes the resources belonging to an instance deployment.
// opts.InventoryLive must be pre-fetched from the ModuleInstance CR inventory by
// the caller. Resources are deleted in reverse weight order. The ModuleInstance
// CR itself is deleted last by the caller, after Delete returns.
//
// A CRD or Namespace is never deleted, and a PersistentVolumeClaim only with
// opts.DeleteData (DeleteResult.Kept otherwise). Every other object goes
// through JudgedDelete: it is read again just before its delete and deleted
// only when the library's delete verdict allows it for opts.InstanceUUID
// (still OPM-managed, not another instance's, not adopted by another
// instance); otherwise it is left behind with the verdict's message
// (DeleteResult.LeftBehind). An object that is already gone counts neither as
// deleted nor as an error; any other read error, and a delete refused because
// the object was replaced since the read (ErrReplaced), is a per-resource
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

	recordUnreadable(result, opts.Unreadable, opts.DeleteData, instanceLog)

	// Sort in reverse weight order (highest weight first = delete webhooks before deployments)
	SortObjects(resources, object.Descending)

	// Delete each workload resource
	for _, res := range resources {
		kind := res.GetKind()
		name := res.GetName()
		ns := res.GetNamespace()

		if !opts.DeleteData && IsDataClaim(res.GroupVersionKind().Group, kind) {
			result.Kept = append(result.Kept, LeftBehindResource{Kind: kind, Namespace: ns, Name: name, Reason: KeptClaimReason})
			continue
		}

		gvk := res.GroupVersionKind()
		if IsProtectedKind(gvk.Group, kind) {
			result.LeftBehind = append(result.LeftBehind, LeftBehindResource{Kind: kind, Namespace: ns, Name: name, Reason: ProtectedKindReason})
			continue
		}

		outcome, err := JudgedDelete(ctx, client, objectOf(res), gvk.Version, opts.InstanceUUID, opts.DryRun)
		switch {
		case err != nil:
			// Worded as before the verdict moved to the library: a failed
			// read and a failed delete each keep their own line.
			verb := "deleting"
			if IsLiveReadFailure(err) {
				verb = "reading"
			}
			instanceLog.Warn(fmt.Sprintf("%s %s/%s: %v", verb, kind, name, err))
			result.Errors = append(result.Errors, resourceError{Kind: kind, Name: name, Namespace: ns, Err: err})
		case outcome.Skip == ownership.SkipAlreadyAbsent:
			instanceLog.Debug("resource already gone", "kind", kind, "namespace", ns, "name", name)
		case outcome.Skip != "":
			result.LeftBehind = append(result.LeftBehind, LeftBehindResource{Kind: kind, Namespace: ns, Name: name, Reason: outcome.Message})
		case opts.DryRun:
			instanceLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusUnchanged))
			result.Deleted++
		default:
			instanceLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusDeleted))
			result.Deleted++
		}
	}

	// The ModuleInstance CR is deleted last by the caller (after this returns),
	// so the inventory record is only removed once the instance is fully torn down.
	return result, nil
}

// recordUnreadable adds the resources discovery could not read to result: a
// protected kind is left behind, as Delete leaves it when it is readable, a
// claim is kept unless deleteData, and any other is a per-resource error
// worded like a failed re-read.
func recordUnreadable(result *DeleteResult, unreadable []UnreadableResource, deleteData bool, instanceLog *log.Logger) {
	for _, u := range unreadable {
		if !deleteData && IsDataClaim(u.Group, u.Kind) {
			result.Kept = append(result.Kept, LeftBehindResource{Kind: u.Kind, Namespace: u.Namespace, Name: u.Name, Reason: KeptClaimReason})
			continue
		}
		if IsProtectedKind(u.Group, u.Kind) {
			result.LeftBehind = append(result.LeftBehind, LeftBehindResource{Kind: u.Kind, Namespace: u.Namespace, Name: u.Name, Reason: ProtectedKindReason})
			continue
		}
		instanceLog.Warn(fmt.Sprintf("reading %s/%s: %v", u.Kind, u.Name, u.Err))
		result.Errors = append(result.Errors, resourceError{Kind: u.Kind, Name: u.Name, Namespace: u.Namespace, Err: u.Err})
	}
}

// ErrReplaced reports a DELETE the API server refused on its UID
// precondition: the object that was read and judged is gone, and another
// holds its name. It is never reported as deleted; the next run reads the
// new object and judges it.
var ErrReplaced = errors.New("the object was replaced after it was read, so it was not deleted")

// liveReadError marks an error of JudgedDelete as a failure of the live
// read, kind resolution included, as opposed to a failure of the DELETE. It
// adds no text of its own.
type liveReadError struct{ err error }

func (e *liveReadError) Error() string { return e.err.Error() }
func (e *liveReadError) Unwrap() error { return e.err }

// IsLiveReadFailure reports whether an error of JudgedDelete came from
// reading the live object, so that no DELETE was sent.
func IsLiveReadFailure(err error) bool {
	var readErr *liveReadError
	return errors.As(err, &readErr)
}

// DeleteOutcome is what JudgedDelete did with one object.
type DeleteOutcome struct {
	// Deleted reports that the API server accepted the DELETE.
	Deleted bool
	// Skip is why the object was left in place; empty when the delete
	// verdict allowed the delete. An object that is already gone, at the
	// read or at the DELETE, is ownership.SkipAlreadyAbsent.
	Skip ownership.SkipReason
	// Message is the library's wording of the skip, for the user.
	Message string
}

// JudgedDelete is the one way the CLI deletes an object of an instance. It
// reads the live object under the resource the cluster serves its kind as,
// asks the library's delete verdict with instanceUUID, the identity that
// applied the object, and only on a proceed verdict sends the DELETE, with
// foreground propagation and a precondition on the UID of the object it
// read. A dry run stops after the verdict.
//
// A kind OPM never deletes is skipped without a read. A read that fails with
// anything but NotFound, a kind that cannot be resolved included, is returned
// as the error (IsLiveReadFailure): the object may still exist. A DELETE
// refused on the UID precondition is ErrReplaced.
func JudgedDelete(ctx context.Context, client *Client, obj ownership.Object, version, instanceUUID string, dryRun bool) (DeleteOutcome, error) {
	if ownership.SafetyExcluded(obj.Group, obj.Kind) {
		return outcomeOf(ownership.CanDelete(ownership.DeleteInput{Object: obj, InstanceUUID: instanceUUID})), nil
	}

	gvk := schema.GroupVersionKind{Group: obj.Group, Version: version, Kind: obj.Kind}
	resource, err := client.ResourceClientFor(ctx, gvk, obj.Namespace)
	if err != nil {
		return DeleteOutcome{}, &liveReadError{err: err}
	}
	live, err := resource.Get(ctx, obj.Name, metav1.GetOptions{})
	if err != nil {
		if !apierrors.IsNotFound(err) {
			return DeleteOutcome{}, &liveReadError{err: err}
		}
		live = nil
	}

	verdict := ownership.CanDelete(ownership.DeleteInput{Object: obj, Live: live, InstanceUUID: instanceUUID})
	if !verdict.Proceed() || dryRun {
		return outcomeOf(verdict), nil
	}

	propagation := metav1.DeletePropagationForeground
	err = resource.Delete(ctx, obj.Name, metav1.DeleteOptions{
		PropagationPolicy: &propagation,
		Preconditions:     verdict.Preconditions(),
	})
	switch {
	case err == nil:
		return DeleteOutcome{Deleted: true}, nil
	case apierrors.IsNotFound(err):
		// Gone between the read and the delete: already done.
		return DeleteOutcome{Skip: ownership.SkipAlreadyAbsent, Message: obj.String() + " no longer exists"}, nil
	case apierrors.IsConflict(err) && verdict.Preconditions() != nil:
		return DeleteOutcome{}, fmt.Errorf("%w: %w", ErrReplaced, err)
	default:
		return DeleteOutcome{}, err
	}
}

func outcomeOf(v ownership.DeleteVerdict) DeleteOutcome {
	return DeleteOutcome{Skip: v.Skip, Message: v.Message}
}

// objectOf is the ownership identity of a live or rendered object.
func objectOf(obj *unstructured.Unstructured) ownership.Object {
	gvk := obj.GroupVersionKind()
	return ownership.Object{Group: gvk.Group, Kind: gvk.Kind, Namespace: obj.GetNamespace(), Name: obj.GetName()}
}
