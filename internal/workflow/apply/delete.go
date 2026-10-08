package apply

import (
	"context"
	"fmt"

	"github.com/charmbracelet/log"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// DeleteDataOperatorManagedNote is the warning every command prints when
// --delete-data is set for an operator-managed instance. The flag steers only
// what opm itself deletes; there the operator deletes, and the instance's
// spec.dataPolicy is what it obeys.
const DeleteDataOperatorManagedNote = "--delete-data does not change what the operator does with an operator-managed instance: " +
	"spec.dataPolicy on the ModuleInstance decides whether the operator deletes PersistentVolumeClaims"

// DeleteRequest is a CLI-owned instance delete: the instance's recorded
// inventory, already read from the cluster, and its record.
type DeleteRequest struct {
	Client *kubernetes.Client
	// InstanceName or InstanceID selects the instance; Namespace is where.
	InstanceName string
	InstanceID   string
	Namespace    string
	// Record is the instance's ModuleInstance record; nil when none exists.
	Record *inventory.Record
	// Live are the live objects the record's inventory lists.
	Live []*unstructured.Unstructured
	// Unreadable are the recorded objects discovery could not read (an
	// error other than NotFound). Each is a per-object error, so the record
	// is kept and the caller reports failure; a protected kind is left
	// behind as it would be if read.
	Unreadable []kubernetes.UnreadableResource
	DryRun     bool
	// DeleteData deletes tracked PersistentVolumeClaims too. When false each
	// is kept and listed in the result's Kept.
	DeleteData bool
	Log        *log.Logger
}

// RecordDeleteError reports that every tracked object of an instance was
// deleted and the delete of its ModuleInstance record then failed. The record
// is still in the cluster and lists objects that are gone; a re-run finds them
// absent and tries the record again.
type RecordDeleteError struct {
	Namespace string
	Name      string
	Err       error
}

func (e *RecordDeleteError) Error() string {
	return fmt.Sprintf("the tracked resources of ModuleInstance %s/%s were deleted, but the record remains: %v; fix the cause and re-run, re-running is safe",
		e.Namespace, e.Name, e.Err)
}

func (e *RecordDeleteError) Unwrap() error { return e.Err }

// DeleteRecorded deletes a CLI-owned instance's tracked objects, in
// descending resource-weight order, leaving CRDs, Namespaces and any object
// that no longer carries the instance's identity behind, and keeping
// PersistentVolumeClaims unless req.DeleteData, then deletes the
// ModuleInstance record last: only on a real run with no per-object error,
// so a re-run can retry what failed. An object discovery could not read
// (req.Unreadable) is such an error. Already absent objects count as
// deleted. A record delete that fails is returned as a *RecordDeleteError
// and is not logged here: the caller reports it and must not report success.
// `opm instance delete` and `opm operator uninstall` share it. The caller
// reports the result.
func DeleteRecorded(ctx context.Context, req DeleteRequest) (*kubernetes.DeleteResult, error) {
	instanceLog := req.Log
	instanceLog.Info(fmt.Sprintf("deleting resources in namespace %q", req.Namespace))

	uuid := ""
	if req.Record != nil {
		uuid = req.Record.InstanceUUID
	}
	deleteResult, err := kubernetes.Delete(ctx, req.Client, kubernetes.DeleteOptions{
		InstanceName:          req.InstanceName,
		Namespace:             req.Namespace,
		InstanceID:            req.InstanceID,
		InstanceUUID:          uuid,
		DryRun:                req.DryRun,
		DeleteData:            req.DeleteData,
		InventoryLive:         req.Live,
		InventoryRecordExists: req.Record != nil,
		Unreadable:            req.Unreadable,
	})
	if err != nil {
		instanceLog.Error("delete failed", "error", err)
		return nil, err
	}

	for _, lb := range deleteResult.LeftBehind {
		instanceLog.Warn(output.FormatResourceLine(lb.Kind, lb.Namespace, lb.Name, output.StatusLeftBehind), "reason", lb.Reason)
	}

	// Kept on purpose, so an informational line and never a warning.
	for _, k := range deleteResult.Kept {
		instanceLog.Info(output.FormatResourceLine(k.Kind, k.Namespace, k.Name, output.StatusKept))
	}

	if len(deleteResult.Errors) > 0 {
		instanceLog.Warn(fmt.Sprintf("%d resource(s) had errors", len(deleteResult.Errors)))
		for _, e := range deleteResult.Errors {
			instanceLog.Error(e.Error())
		}
	}

	// Delete the ModuleInstance CR last — only after every tracked workload
	// resource is gone (0006:D1). Skipped on dry-run and on partial
	// failure (so a re-run can retry the remaining workloads).
	if !req.DryRun && req.Record != nil && len(deleteResult.Errors) == 0 {
		if err := inventory.DeleteCR(ctx, req.Client, req.Record.Name, req.Record.Namespace); err != nil {
			return nil, &RecordDeleteError{Namespace: req.Record.Namespace, Name: req.Record.Name, Err: err}
		}
	}
	return deleteResult, nil
}
