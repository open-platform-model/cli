package operator

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/apimachinery/pkg/util/sets"
	"k8s.io/client-go/util/csaupgrade"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// MigrationStoppedError reports a migration write that failed after the
// install's writes began. Every step is idempotent, so re-running install
// completes it.
type MigrationStoppedError struct {
	Step string
	Err  error
}

func (e *MigrationStoppedError) Error() string {
	return fmt.Sprintf("operator migration stopped after it began: %s: %v; re-run 'opm operator install' to complete it", e.Step, e.Err)
}

func (e *MigrationStoppedError) Unwrap() error { return e.Err }

// MoveOwnership hands the fields a client-side `kubectl apply` owns on each
// object of plan.MoveOwnership to the field manager opm-cli, rewriting
// managedFields only, so the instance apply's forced apply then drops what
// the module does not render, last-applied-configuration included. Each
// object is read again first: the CRD step has changed the CRDs since the
// plan read them. A missing object or one with no client-side manager left
// is skipped. Fields of every other manager stay where they are.
func MoveOwnership(ctx context.Context, client *kubernetes.Client, plan *MigrationPlan) error {
	if plan == nil {
		return nil
	}
	for _, planned := range plan.MoveOwnership {
		step := "moving the field ownership of " + objPath(planned.GetKind(), planned.GetNamespace(), planned.GetName())
		ri := client.ResourceClient(kubernetes.GVRFromUnstructured(planned), planned.GetNamespace())
		live, err := ri.Get(ctx, planned.GetName(), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return &MigrationStoppedError{Step: step, Err: err}
		}
		patch, err := csaupgrade.UpgradeManagedFieldsPatch(live, sets.New(clientSideApplyManager), kubernetes.FieldManager)
		if err != nil {
			return &MigrationStoppedError{Step: step, Err: err}
		}
		if patch == nil {
			continue
		}
		if _, err := ri.Patch(ctx, planned.GetName(), types.JSONPatchType, patch, metav1.PatchOptions{}); err != nil && !apierrors.IsNotFound(err) {
			return &MigrationStoppedError{Step: step, Err: err}
		}
	}
	return nil
}

// DeleteSuperseded deletes what the migration replaces (0012:D8:R7): first
// the earlier controller Deployment, with foreground propagation, waiting
// until it is gone under ctx's deadline (since is when that budget started),
// then the superseded role bindings. Each delete is preconditioned on the
// uid the plan proved, so an object replaced since then is never deleted. A
// missing object is done.
func DeleteSuperseded(ctx context.Context, client *kubernetes.Client, plan *MigrationPlan, since time.Time) error {
	if plan == nil {
		return nil
	}
	if d := plan.RecreateDeployment; d != nil {
		if err := deleteProven(ctx, client, d); err != nil {
			return err
		}
		if err := kubernetes.WaitAbsent(ctx, client, []*unstructured.Unstructured{d}, since); err != nil {
			return &MigrationStoppedError{Step: "waiting for " + objPath(d.GetKind(), d.GetNamespace(), d.GetName()) + " to be deleted", Err: err}
		}
	}
	for _, b := range plan.DeleteBindings {
		if err := deleteProven(ctx, client, b.Live); err != nil {
			return err
		}
	}
	return nil
}

func deleteProven(ctx context.Context, client *kubernetes.Client, obj *unstructured.Unstructured) error {
	uid := obj.GetUID()
	propagation := metav1.DeletePropagationForeground
	opts := metav1.DeleteOptions{PropagationPolicy: &propagation}
	if uid != "" {
		opts.Preconditions = &metav1.Preconditions{UID: &uid}
	}
	err := client.ResourceClient(kubernetes.GVRFromUnstructured(obj), obj.GetNamespace()).Delete(ctx, obj.GetName(), opts)
	if err != nil && !apierrors.IsNotFound(err) {
		return &MigrationStoppedError{Step: "deleting " + objPath(obj.GetKind(), obj.GetNamespace(), obj.GetName()), Err: err}
	}
	return nil
}
