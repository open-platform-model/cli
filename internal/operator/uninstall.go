package operator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
)

// The ModuleInstance CRD coordinates are defined once in internal/inventory
// (0006:D1/D13); these package-local aliases keep the existing
// call sites (RBAC rule construction, finalizer name) readable.
const (
	opmodelAPIGroup         = inventory.GroupOpmodel
	moduleInstancesResource = inventory.ResourceModuleInstances
)

// moduleInstanceGVR is the ModuleInstance CRD's GroupVersionResource.
var moduleInstanceGVR = inventory.ModuleInstanceGVR

// cleanupFinalizer is the operator's finalizer that blocks uninstall until
// removed or the ModuleInstance is otherwise cleaned up. Defined once in
// internal/inventory alongside the rest of the CR coordinates.
const cleanupFinalizer = inventory.CleanupFinalizer

// ArmedInstance identifies a ModuleInstance still carrying the operator's
// cleanup finalizer.
type ArmedInstance struct {
	Namespace string
	Name      string
}

func (a ArmedInstance) String() string {
	return fmt.Sprintf("%s/%s", a.Namespace, a.Name)
}

// FinalizerGuardError reports that an action was refused because one or more
// ModuleInstances still carry the operator's cleanup finalizer. Uninstall sets
// only Armed; the other fields let another command that removes the operator
// name its own action and remedy.
type FinalizerGuardError struct {
	Armed []ArmedInstance

	// Action is what was refused, completing "refusing to <Action>". Empty
	// means "uninstall".
	Action string

	// Target, when set, is the instance the refused action would delete; it is
	// marked in the list when it is armed itself.
	Target ArmedInstance

	// Remedy is the closing parenthetical. Empty means uninstall's own flag.
	Remedy string
}

func (e *FinalizerGuardError) Error() string {
	action := e.Action
	if action == "" {
		action = "uninstall"
	}
	remedy := e.Remedy
	if remedy == "" {
		remedy = "use --remove-finalizers to proceed; this orphans their workloads"
	}
	names := make([]string, len(e.Armed))
	for i, a := range e.Armed {
		names[i] = a.String()
		if e.Target != (ArmedInstance{}) && a == e.Target {
			names[i] += " (the instance being deleted)"
		}
	}
	return fmt.Sprintf(
		"refusing to %s: %d instance(s) still carry the %s finalizer: %s (%s)",
		action, len(e.Armed), cleanupFinalizer, strings.Join(names, ", "), remedy,
	)
}

// OwnInstanceOwnerError reports that a delete was refused because the target
// deploys the operator and is operator-owned. The operator never reconciles,
// finalizes or prunes the instance that deploys it, so the operator-owned
// delete would wait on, and report, a cleanup that does not happen. The remedy
// is the one the operator itself names for such an instance.
type OwnInstanceOwnerError struct {
	Namespace string
	Name      string
	// Signal is the DeploysOperator signal that matched.
	Signal string
}

func (e *OwnInstanceOwnerError) Error() string {
	return fmt.Sprintf(
		"refusing to delete %s/%s: it deploys the operator (matched by %s) and is operator-owned, "+
			"but the operator never reconciles or prunes the instance that deploys it; "+
			"set spec.owner to cli (kubectl patch moduleinstance %s -n %s --type=merge -p '{\"spec\":{\"owner\":\"cli\"}}'), then retry",
		e.Namespace, e.Name, e.Signal, e.Name, e.Namespace,
	)
}

// CheckFinalizerGuard lists ModuleInstances cluster-wide and returns every
// instance that still carries the operator's cleanup finalizer. A list
// failure (including RBAC denial) is returned as-is so the caller fails
// closed without deleting anything.
func CheckFinalizerGuard(ctx context.Context, client *kubernetes.Client) ([]ArmedInstance, error) {
	list, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace("").List(ctx, metav1.ListOptions{})
	if err != nil {
		return nil, fmt.Errorf("listing moduleinstances: %w", err)
	}

	var armed []ArmedInstance
	for _, item := range list.Items {
		for _, f := range item.GetFinalizers() {
			if f == cleanupFinalizer {
				armed = append(armed, ArmedInstance{Namespace: item.GetNamespace(), Name: item.GetName()})
				break
			}
		}
	}
	return armed, nil
}

// RemoveCleanupFinalizer strips exactly the operator's cleanup finalizer from
// every armed instance via a targeted JSON patch (test the finalizer is still
// at the observed index, then remove it), leaving any other finalizers
// intact. Reports the orphaning consequence for each instance it patches.
// Every instance is attempted even if an earlier one fails — partial
// failures are collected and returned together (mirroring kubernetes.Apply's
// and kubernetes.Delete's per-resource error handling) so a failure on
// instance N doesn't leave instances 1..N-1 already stripped with no
// visibility into what happened.
func RemoveCleanupFinalizer(ctx context.Context, client *kubernetes.Client, armed []ArmedInstance) error {
	var errs []error
	for _, a := range armed {
		if err := removeOneCleanupFinalizer(ctx, client, a); err != nil {
			errs = append(errs, fmt.Errorf("%s: %w", a, err))
		}
	}
	return errors.Join(errs...)
}

func removeOneCleanupFinalizer(ctx context.Context, client *kubernetes.Client, a ArmedInstance) error {
	obj, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace(a.Namespace).Get(ctx, a.Name, metav1.GetOptions{})
	if err != nil {
		return fmt.Errorf("reading instance: %w", err)
	}

	idx := -1
	for i, f := range obj.GetFinalizers() {
		if f == cleanupFinalizer {
			idx = i
			break
		}
	}
	if idx == -1 {
		// Already gone — a previous run or the operator itself removed it.
		return nil
	}

	patch, err := json.Marshal([]map[string]any{
		{"op": "test", "path": fmt.Sprintf("/metadata/finalizers/%d", idx), "value": cleanupFinalizer},
		{"op": "remove", "path": fmt.Sprintf("/metadata/finalizers/%d", idx)},
	})
	if err != nil {
		return fmt.Errorf("building finalizer patch: %w", err)
	}

	if _, err := client.Dynamic.Resource(moduleInstanceGVR).Namespace(a.Namespace).Patch(
		ctx, a.Name, types.JSONPatchType, patch, metav1.PatchOptions{},
	); err != nil {
		return fmt.Errorf("removing finalizer: %w", err)
	}

	output.Warn(fmt.Sprintf(
		"removed %s finalizer from %s — its workload is now orphaned (no longer cleaned up by the operator)",
		cleanupFinalizer, a,
	))
	return nil
}

// UninstallOptions configures an uninstall run.
type UninstallOptions struct {
	// RemoveFinalizers strips the operator's cleanup finalizer from any
	// armed ModuleInstance before proceeding, orphaning its workload.
	RemoveFinalizers bool
}

// UninstallResult reports the outcome of an uninstall run.
type UninstallResult struct {
	// Deleted is the number of resources deleted.
	Deleted int

	// LeftBehind is the number of recorded objects left in place: the CRDs,
	// the Namespace and any object that no longer carries the instance's
	// identity.
	LeftBehind int

	// Errors contains per-resource delete errors. Uninstall is fire-and-report:
	// one resource failing to delete does not stop the rest, and the record is
	// then kept so a re-run retries.
	Errors []error
}

// NoRecordError refuses an uninstall on a cluster whose operator has no
// instance record: the CLI cannot prove what it would delete.
type NoRecordError struct{}

func (e *NoRecordError) Error() string {
	return fmt.Sprintf("no operator instance record %s/%s; run 'opm operator install' to record the running operator, then uninstall",
		OperatorNamespace, OperatorInstanceName)
}

// Uninstall deletes what the operator instance's record lists, except CRDs
// and the Namespace, then the record (0021:D11:R11). With no record it
// deletes nothing and returns a *NoRecordError. Before deleting anything it
// checks for ModuleInstances still carrying the operator's cleanup
// finalizer and refuses (or, with opts.RemoveFinalizers, strips the
// finalizer and proceeds). Objects are deleted in descending resource-weight
// order, an object that no longer carries the instance's identity is left
// behind, an already absent object counts as deleted, and deletion is not
// waited for (fire-and-report).
func Uninstall(ctx context.Context, client *kubernetes.Client, opts UninstallOptions) (*UninstallResult, error) {
	rec, err := inventory.GetRecord(ctx, client, OperatorInstanceName, OperatorNamespace)
	if err != nil {
		return nil, fmt.Errorf("reading the operator's instance record: %w", err)
	}
	if rec == nil {
		return nil, &NoRecordError{}
	}

	armed, err := CheckFinalizerGuard(ctx, client)
	if err != nil {
		return nil, err
	}
	if len(armed) > 0 {
		if !opts.RemoveFinalizers {
			return nil, &FinalizerGuardError{Armed: armed}
		}
		if err := RemoveCleanupFinalizer(ctx, client, armed); err != nil {
			return nil, err
		}
	}

	live, _, err := inventory.DiscoverResourcesFromInventory(ctx, client, rec)
	if err != nil {
		return nil, fmt.Errorf("reading the recorded objects: %w", err)
	}
	deleted, err := workflowapply.DeleteRecorded(ctx, workflowapply.DeleteRequest{
		Client:       client,
		InstanceName: OperatorInstanceName,
		Namespace:    OperatorNamespace,
		Record:       rec,
		Live:         live,
		Log:          output.InstanceLogger(OperatorInstanceName),
	})
	if err != nil {
		return nil, err
	}
	result := &UninstallResult{Deleted: deleted.Deleted, LeftBehind: len(deleted.LeftBehind)}
	for i := range deleted.Errors {
		result.Errors = append(result.Errors, &deleted.Errors[i])
	}
	return result, nil
}
