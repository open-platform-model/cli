package operator

import (
	"context"
	"fmt"
	"time"

	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
)

const (
	kindCustomResourceDefinition = "CustomResourceDefinition"
	kindNamespace                = "Namespace"
)

// InstallResult reports what an install wrote.
type InstallResult struct {
	Target Target
	// CRDs is the number of CRDs the CRD step applied.
	CRDs int
	// Recorded reports that the instance apply ran and wrote the record.
	Recorded bool
	// Extra is the number of --rbac objects applied.
	Extra int
}

// RolloutError reports that the operator's controller Deployment did not
// finish its rollout within the --timeout budget. Nothing is rolled back.
type RolloutError struct{ Err error }

func (e *RolloutError) Error() string {
	return fmt.Sprintf("Deployment %s/%s did not complete its rollout: %v; every applied object and the instance record remain, and re-running 'opm operator install' completes it",
		OperatorNamespace, ControllerDeploymentName, e.Err)
}

func (e *RolloutError) Unwrap() error { return e.Err }

// Install performs a plan's writes, in order, within the plan's --timeout
// budget: the CRD step (server-side apply of the rendered CRDs as opm-cli,
// then wait for Established), then, unless the plan is CRDs-only, the
// migration's writes and the CLI-owned instance apply of the whole render (no namespace creation, the
// running-operator ceiling skipped, pruning on), then the --rbac objects,
// then the controller Deployment's rollout. A step that fails stops the
// install; nothing is rolled back, and re-running install completes it.
func Install(ctx context.Context, env InstallEnv, plan *Plan) (*InstallResult, error) {
	ctx, cancel := context.WithDeadline(ctx, plan.Deadline())
	defer cancel()
	result := &InstallResult{Target: plan.Target}

	// Write 1: the CRDs, so the record is written only once they are served.
	for _, crd := range plan.CRDs {
		status, err := kubernetes.ApplyOne(ctx, env.Client, crd, kubernetes.ApplyOptions{})
		if err != nil {
			return result, fmt.Errorf("applying %s/%s: %w", crd.GetKind(), crd.GetName(), err)
		}
		result.CRDs++
		output.Info(output.FormatResourceLine(crd.GetKind(), crd.GetNamespace(), crd.GetName(), status))
	}
	if err := kubernetes.Wait(ctx, env.Client, plan.CRDs, DefaultPredicate, plan.BudgetStart); err != nil {
		return result, err
	}

	// The migration's writes, after the CRDs are served and before the
	// instance apply (0012:D8:R7): the field-ownership moves, then the
	// earlier Deployment, then the superseded role bindings. A cluster with
	// nothing to migrate writes nothing here.
	if !plan.CRDsOnly {
		if err := MoveOwnership(ctx, env.Client, plan.Migration); err != nil {
			return result, err
		}
		if err := DeleteSuperseded(ctx, env.Client, plan.Migration, plan.BudgetStart); err != nil {
			return result, err
		}
		for _, line := range MigrationReport(plan.Migration) {
			output.Info(line)
		}
	}

	// Write 2: the instance apply records every rendered object, the CRDs
	// and the Namespace included.
	if !plan.CRDsOnly {
		err := workflowapply.Execute(ctx, workflowapply.Request{
			Result:    plan.Render,
			K8sClient: env.Client,
			Log:       output.InstanceLogger(OperatorInstanceName),
			Admit:     plan.Migration.Admit(),
			Options: workflowapply.Options{
				CreateNS:               false,
				SkipOperatorCeiling:    true,
				AfterCallerWrites:      true,
				Timeout:                remaining(plan.Deadline()),
				SuccessAppliedMessage:  fmt.Sprintf("ModuleInstance %s/%s applied", OperatorNamespace, OperatorInstanceName),
				SuccessUpToDateMessage: fmt.Sprintf("ModuleInstance %s/%s up to date", OperatorNamespace, OperatorInstanceName),
			},
		})
		if err != nil {
			return result, err
		}
		result.Recorded = true
	}

	// The opt-in user role is never part of the render or the record.
	for _, obj := range plan.Extra {
		status, err := kubernetes.ApplyOne(ctx, env.Client, obj, kubernetes.ApplyOptions{})
		if err != nil {
			return result, fmt.Errorf("applying %s/%s: %w", obj.GetKind(), obj.GetName(), err)
		}
		result.Extra++
		output.Info(output.FormatResourceLine(obj.GetKind(), obj.GetNamespace(), obj.GetName(), status))
	}

	if plan.CRDsOnly {
		return result, nil
	}
	controller := controllerDeployment(plan.Render.Resources)
	if controller == nil {
		return result, fmt.Errorf("the render has no Deployment %s/%s to wait for", OperatorNamespace, ControllerDeploymentName)
	}
	if err := kubernetes.Wait(ctx, env.Client, []*unstructured.Unstructured{controller}, DefaultPredicate, plan.BudgetStart); err != nil {
		return result, &RolloutError{Err: err}
	}
	return result, nil
}

// remaining is the time left before deadline, at least a millisecond, so a
// spent budget fails the next wait instead of falling back to a default.
func remaining(deadline time.Time) time.Duration {
	if d := time.Until(deadline); d > time.Millisecond {
		return d
	}
	return time.Millisecond
}

// controllerDeployment returns the rendered controller Deployment.
func controllerDeployment(objs []*unstructured.Unstructured) *unstructured.Unstructured {
	for _, obj := range objs {
		if obj.GetKind() == kindDeployment && obj.GetName() == ControllerDeploymentName && obj.GetNamespace() == OperatorNamespace {
			return obj
		}
	}
	return nil
}

// waitForTerminating is the pre-apply guard: it reads every planned object
// and waits for those that exist with a deletionTimestamp (typically left by
// a previous uninstall's foreground delete) to disappear, under the caller's
// ctx deadline. Applying onto a terminating object succeeds and is then undone
// by the garbage collector, so the install would wait out its timeout on a
// resource that no longer exists. Absent and live objects never delay apply.
func waitForTerminating(ctx context.Context, client *kubernetes.Client, plan []*unstructured.Unstructured, since time.Time) error {
	terminating, err := terminatingObjects(ctx, client, plan)
	if err != nil {
		return err
	}
	if len(terminating) == 0 {
		return nil
	}

	for _, obj := range terminating {
		output.Info(output.FormatResourceLine(obj.GetKind(), obj.GetNamespace(), obj.GetName(), "waiting to finish terminating"))
	}
	return kubernetes.WaitAbsent(ctx, client, terminating, since)
}

// terminatingObjects returns the planned objects that exist on the cluster
// with metadata.deletionTimestamp set. A NotFound read, or a kind the cluster
// does not serve yet, means nothing to wait on; any other read or discovery
// error is returned, since the guard cannot tell whether the object is
// terminating.
func terminatingObjects(ctx context.Context, client *kubernetes.Client, plan []*unstructured.Unstructured) ([]*unstructured.Unstructured, error) {
	var terminating []*unstructured.Unstructured
	for _, obj := range plan {
		live, err := getObject(ctx, client, obj)
		if apierrors.IsNotFound(err) || kubernetes.IsKindNotServed(err) {
			continue
		}
		if err != nil {
			return nil, fmt.Errorf("checking %s/%s before apply: %w", obj.GetKind(), obj.GetName(), err)
		}
		if live.GetDeletionTimestamp() != nil {
			terminating = append(terminating, obj)
		}
	}
	return terminating, nil
}
