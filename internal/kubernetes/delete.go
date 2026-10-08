package kubernetes

import (
	"context"
	"errors"
	"fmt"

	"github.com/charmbracelet/log"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/k8s/lifecycle"
	"github.com/open-platform-model/library/opm/k8s/object"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

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

	// Entries are the entries of the instance's record, the list the
	// deletion plan is built from. When nil, the plan is built from the
	// objects of InventoryLive and Unreadable instead, for a caller that
	// holds live objects and no record.
	Entries []k8sinventory.Entry

	// InventoryLive is the list of live resources pre-fetched from the
	// ModuleInstance CR inventory by the caller: the claims Delete keeps are
	// reported from it, and a dry run lists it. When nil or empty (and
	// InventoryRecordExists is false, and nothing is unreadable), Delete
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

	// Run is the deletion plan as it was driven. The caller asks
	// lifecycle.MayReleaseHold with its plan and state before it deletes
	// the instance's record.
	Run DeletionRun
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

// Delete removes the resources belonging to an instance deployment, as the
// library's deletion plan orders and judges them (RunDeletion). The plan is
// built from opts.Entries, the entries of the instance's record, and judged
// with opts.InstanceUUID. The ModuleInstance CR itself is deleted last by the
// caller, after Delete returns, and only when the hold verdict of
// DeleteResult.Run releases it.
//
// A PersistentVolumeClaim is deleted only with opts.DeleteData; otherwise it
// stays out of the plan and is listed in DeleteResult.Kept. For every other
// entry the plan decides: a CRD or Namespace is never deleted and never
// read; any other object is read again just before its delete and deleted
// only when the library's delete verdict allows it for opts.InstanceUUID
// (still OPM-managed, not another instance's, not adopted by another
// instance); otherwise it is left behind with the verdict's message
// (DeleteResult.LeftBehind). An object that is already gone counts neither as
// deleted nor as an error; any other read error, and a delete refused because
// the object was replaced since the read (ErrReplaced), is a per-resource
// error, so the hold verdict keeps the ModuleInstance and a re-run retries. A
// resource the caller's discovery could not read (opts.Unreadable) is
// reported first and gets the same outcome without a second read.
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

	// Highest weight first, the order the plan deletes in.
	SortObjects(resources, object.Descending)

	// A kept claim never enters the plan: it is reported from the live read.
	if !opts.DeleteData {
		for _, res := range resources {
			if IsDataClaim(res.GroupVersionKind().Group, res.GetKind()) {
				result.Kept = append(result.Kept, LeftBehindResource{Kind: res.GetKind(), Namespace: res.GetNamespace(), Name: res.GetName(), Reason: KeptClaimReason})
			}
		}
	}

	// Reported above, before the run; the plan still holds them, so that a
	// failed one holds the record.
	unreadable := make(map[ownership.Object]error, len(opts.Unreadable))
	for _, u := range opts.Unreadable {
		unreadable[ownership.Object{Group: u.Group, Kind: u.Kind, Namespace: u.Namespace, Name: u.Name}] = u.Err
	}

	plan := lifecycle.NewDeletionPlan(planEntries(opts), lifecycle.Policy{Prune: true}, opts.InstanceUUID)
	run, err := RunDeletion(ctx, client, plan, DeletionOptions{
		DryRun:     opts.DryRun,
		Unreadable: unreadable,
		OnStep: func(step StepResult) {
			if _, reported := unreadable[entryObject(step.Entry)]; reported {
				return
			}
			recordStep(result, step, opts.DryRun, instanceLog)
		},
	})
	result.Run = run
	if err != nil {
		return nil, err
	}

	// The ModuleInstance CR is deleted last by the caller (after this returns),
	// so the inventory record is only removed once the instance is fully torn down.
	return result, nil
}

// planEntries are the entries Delete plans over: the record's entries, or
// else those of the live and unreadable objects the caller holds, without
// the PersistentVolumeClaims Delete keeps.
func planEntries(opts DeleteOptions) []k8sinventory.Entry {
	entries := opts.Entries
	if entries == nil {
		for _, res := range opts.InventoryLive {
			gvk := res.GroupVersionKind()
			entries = append(entries, k8sinventory.Entry{Group: gvk.Group, Version: gvk.Version, Kind: gvk.Kind, Namespace: res.GetNamespace(), Name: res.GetName()})
		}
		for _, u := range opts.Unreadable {
			entries = append(entries, k8sinventory.Entry{Group: u.Group, Kind: u.Kind, Namespace: u.Namespace, Name: u.Name})
		}
	}
	if opts.DeleteData {
		return entries
	}
	planned := make([]k8sinventory.Entry, 0, len(entries))
	for _, e := range entries {
		if !IsDataClaim(e.Group, e.Kind) {
			planned = append(planned, e)
		}
	}
	return planned
}

// recordStep adds one finished step of the plan to result and prints its
// line: a failed read or delete is a per-resource error, a skip other than
// "already gone" is left behind, and a delete counts.
func recordStep(result *DeleteResult, step StepResult, dryRun bool, instanceLog *log.Logger) {
	kind, ns, name := step.Entry.Kind, step.Entry.Namespace, step.Entry.Name
	switch {
	case step.Outcome.Result == lifecycle.ResultFailed:
		// A failed read and a failed delete each keep their own line.
		verb := "deleting"
		if step.Failed == lifecycle.ActionRead {
			verb = "reading"
		}
		instanceLog.Warn(fmt.Sprintf("%s %s/%s: %v", verb, kind, name, step.Err))
		result.Errors = append(result.Errors, resourceError{Kind: kind, Name: name, Namespace: ns, Err: step.Err})
	case step.Outcome.Skip == ownership.SkipAlreadyAbsent:
		instanceLog.Debug("resource already gone", "kind", kind, "namespace", ns, "name", name)
	case step.Outcome.Skip == ownership.SkipSafetyExcluded:
		result.LeftBehind = append(result.LeftBehind, LeftBehindResource{Kind: kind, Namespace: ns, Name: name, Reason: ProtectedKindReason})
	case step.Outcome.Skip != "":
		result.LeftBehind = append(result.LeftBehind, LeftBehindResource{Kind: kind, Namespace: ns, Name: name, Reason: step.Outcome.Message})
	case dryRun:
		instanceLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusUnchanged))
		result.Deleted++
	default:
		instanceLog.Info(output.FormatResourceLine(kind, ns, name, output.StatusDeleted))
		result.Deleted++
	}
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
