package operator

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"strings"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// The ModuleInstance CRD coordinates are defined once in internal/inventory
// (enhancement 0006 D1/D13); these package-local aliases keep the existing
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

// FinalizerGuardError reports that uninstall was refused because one or more
// ModuleInstances still carry the operator's cleanup finalizer.
type FinalizerGuardError struct {
	Armed []ArmedInstance
}

func (e *FinalizerGuardError) Error() string {
	names := make([]string, len(e.Armed))
	for i, a := range e.Armed {
		names[i] = a.String()
	}
	return fmt.Sprintf(
		"refusing to uninstall: %d instance(s) still carry the %s finalizer: %s (use --remove-finalizers to proceed; this orphans their workloads)",
		len(e.Armed), cleanupFinalizer, strings.Join(names, ", "),
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

	// Errors contains per-resource delete errors. Uninstall is fire-and-report:
	// one resource failing to delete does not stop the rest.
	Errors []error
}

// Uninstall deletes everything the embedded manifest installed except its
// CRDs and Namespace, in descending resource-weight order (matching
// delete.go's teardown convention). Before deleting anything it checks for
// ModuleInstances still carrying the operator's cleanup finalizer and refuses
// (or, with opts.RemoveFinalizers, strips the finalizer and proceeds).
// Deletion does not wait for objects to fully disappear (fire-and-report):
// nothing downstream depends on "fully gone", and waiting only adds failure
// modes (stuck pod termination) to a command whose job is done at
// delete-issuance.
func Uninstall(ctx context.Context, client *kubernetes.Client, opts UninstallOptions) (*UninstallResult, error) {
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

	manifest, err := EmbeddedManifest()
	if err != nil {
		return nil, err
	}

	result := &UninstallResult{}
	for _, obj := range UninstallPlan(manifest) {
		if err := deleteOne(ctx, client, obj); err != nil {
			output.Warn(fmt.Sprintf("deleting %s/%s: %v", obj.GetKind(), obj.GetName(), err))
			result.Errors = append(result.Errors, fmt.Errorf("%s/%s: %w", obj.GetKind(), obj.GetName(), err))
			continue
		}
		result.Deleted++
		output.Info(output.FormatResourceLine(obj.GetKind(), obj.GetNamespace(), obj.GetName(), output.StatusDeleted))
	}

	return result, nil
}

// deleteOne deletes a single resource with foreground propagation, matching
// kubernetes.Delete's per-resource behavior. A NotFound response counts as
// success: uninstall is idempotent, and re-running it after a prior
// successful (or partial) run is a natural user action.
func deleteOne(ctx context.Context, client *kubernetes.Client, obj *unstructured.Unstructured) error {
	propagation := metav1.DeletePropagationForeground
	err := client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).Delete(ctx, obj.GetName(), metav1.DeleteOptions{
		PropagationPolicy: &propagation,
	})
	if apierrors.IsNotFound(err) {
		return nil
	}
	return err
}
