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
	Live   []*unstructured.Unstructured
	DryRun bool
	Log    *log.Logger
}

// DeleteRecorded deletes a CLI-owned instance's tracked objects, in
// descending resource-weight order, leaving CRDs, Namespaces and any object
// that no longer carries the instance's identity behind, then deletes the
// ModuleInstance record last: only on a real run with no per-object error,
// so a re-run can retry what failed. Already absent objects count as
// deleted. `opm instance delete` and `opm operator uninstall` share it. The
// caller reports the result.
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
		InventoryLive:         req.Live,
		InventoryRecordExists: req.Record != nil,
	})
	if err != nil {
		instanceLog.Error("delete failed", "error", err)
		return nil, err
	}

	for _, lb := range deleteResult.LeftBehind {
		instanceLog.Warn(output.FormatResourceLine(lb.Kind, lb.Namespace, lb.Name, output.StatusLeftBehind), "reason", lb.Reason)
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
			instanceLog.Warn("could not delete ModuleInstance CR", "error", err)
		}
	}
	return deleteResult, nil
}
