package controller

import (
	"context"
	"errors"
	"fmt"
	"time"

	"golang.org/x/mod/semver"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/controller/crds"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/modref"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

// ModuleRegistry is what install reads the controller module through: version
// resolution and the module source. modconfig's cached registry satisfies it.
type ModuleRegistry interface {
	modref.Source
	ModuleFetcher
}

// RenderFunc renders a resolved controller module version as the instance
// ControllerInstanceName in ControllerNamespace, from the one values source given,
// against the module's own dependency pins (never the cluster Platform).
type RenderFunc func(ctx context.Context, module *modref.Resolution, values kernel.Source) (*workflowrender.Result, error)

// InstallEnv is what install works through.
type InstallEnv struct {
	Client *kubernetes.Client
	Render RenderFunc
	// CLIVersion is this CLI's version, for the controller-version rule.
	CLIVersion string
}

// PlanOptions configures what PlanInstall decides.
type PlanOptions struct {
	// ValuesFiles are -f/--values files, layered over the recorded values.
	ValuesFiles []string
	// ResetValues is --reset-values.
	ResetValues bool
	// Timeout is the one --timeout budget, started by the terminating wait.
	Timeout time.Duration
	// Extra are objects install applies beside the render (the --rbac user
	// role): waited out like the render's, never guarded or recorded.
	Extra []*unstructured.Unstructured
}

// Plan is everything install decided before its first write.
type Plan struct {
	Target Target
	Module *modref.Resolution
	// Render is the instance's render; nil for a CRDs-only plan.
	Render *workflowrender.Result
	// CRDs are the render's CustomResourceDefinitions, or the embedded ones
	// for a CRDs-only plan.
	CRDs []*unstructured.Unstructured
	// PrevRecord is the controller instance's record; nil on a fresh cluster.
	PrevRecord *inventory.Record
	Values     *Values
	// CRDsOnly marks a PlanCRDsOnly plan: only the CRDs (and Extra) are
	// applied.
	CRDsOnly bool
	Extra    []*unstructured.Unstructured
	// BudgetStart and Timeout are the --timeout budget the writes share.
	BudgetStart time.Time
	Timeout     time.Duration
}

// ResolveTarget resolves the module version install deploys, the pin when
// version is empty, and reads the controller release it states, all before any
// cluster call. A selector the registry cannot satisfy is a
// *modref.RefusalError, an unreadable controller version a *VersionError, a
// registry that cannot be reached a *publish.ConnectivityError.
func ResolveTarget(ctx context.Context, src ModuleRegistry, registry, version string) (*modref.Resolution, Target, error) {
	sel := modref.Selector{Exact: PinnedModuleVersion}
	if version != "" {
		var err error
		sel, err = modref.ParseSelector(version)
		if err != nil {
			return nil, Target{}, err
		}
	}
	res, err := modref.Resolve(ctx, src, modref.Request{Path: ControllerModulePath, Selector: sel, Registry: registry})
	if err != nil {
		return nil, Target{}, err
	}
	ctrlVersion, err := ReadControllerVersion(ctx, src, res.Version)
	if err != nil {
		return nil, Target{}, err
	}
	return res, Target{ModuleVersion: res.Version, ControllerVersion: ctrlVersion, Default: version == ""}, nil
}

// OwnedRecordError refuses an install over a controller instance record
// whose spec.owner is not cli (an absent owner resolves as controller-owned):
// the controller's own instance is CLI-owned, and the instance apply would
// otherwise only edit its spec.
type OwnedRecordError struct{}

func (e *OwnedRecordError) Error() string {
	return fmt.Sprintf(
		"refusing to install: ModuleInstance %s/%s is not spec.owner: cli, and the controller never reconciles the instance that deploys it; "+
			"set spec.owner to cli (kubectl patch moduleinstance %s -n %s --type=merge -p '{\"spec\":{\"owner\":\"cli\"}}'), then retry",
		ControllerNamespace, ControllerInstanceName, ControllerInstanceName, ControllerNamespace)
}

// RecordedValuesError wraps a render that rejected values recorded on the
// controller's instance, naming --reset-values.
type RecordedValuesError struct{ Err error }

func (e *RecordedValuesError) Error() string {
	return fmt.Sprintf("%v (the values recorded on %s/%s took part; pass --reset-values to start from the module's defaults)",
		e.Err, ControllerNamespace, ControllerInstanceName)
}

func (e *RecordedValuesError) Unwrap() error { return e.Err }

// GuardError is the apply guard's refusal: an object the plan applies
// exists and is not OPM's, belongs to or is adopted by another instance, is
// terminating, or could not be read by the guard (the read error is then in
// the chain). Nothing was written. The install command exits
// 2 on every GuardError, whatever it wraps. An object that is unreadable
// before the guard runs is refused earlier, by the terminating wait, with the
// exit code of the read error.
type GuardError struct{ Err error }

func (e *GuardError) Error() string {
	return "refusing to install: " + e.Err.Error() + "\nnothing was changed"
}
func (e *GuardError) Unwrap() error { return e.Err }

// PlanInstall runs every check that can refuse the install, in order, and
// writes nothing: the record read, the values merge, the render (which
// checks the values against the target's #config), the target rules
// (CheckTarget), the status-subresource permission check, the wait for
// terminating objects and, last, the apply guard over every object the plan
// applies, with or without a record. No object is admitted by a proof of
// where it came from: an existing object passes when the record lists it,
// when it is annotated for adoption by this instance, or when it carries
// OPM's managed-by label and no other instance's UUID. The terminating wait starts the
// --timeout budget the writes then share.
func PlanInstall(ctx context.Context, env InstallEnv, res *modref.Resolution, target Target, opts PlanOptions) (*Plan, error) {
	rec, err := inventory.GetRecord(ctx, env.Client, ControllerInstanceName, ControllerNamespace)
	if err != nil {
		return nil, fmt.Errorf("reading the controller's instance record %s/%s: %w", ControllerNamespace, ControllerInstanceName, err)
	}
	if rec != nil && inventory.ResolveOwnership(rec) == inventory.ModeControllerOwned {
		return nil, &OwnedRecordError{}
	}

	values, result, err := renderWithValues(ctx, env, res, rec, opts)
	if err != nil {
		return nil, err
	}

	if err := CheckTarget(target, result.Resources, env.CLIVersion); err != nil {
		return nil, err
	}

	plan := &Plan{
		Target:     target,
		Module:     res,
		Render:     result,
		CRDs:       renderedCRDs(result.Resources),
		PrevRecord: rec,
		Values:     values,
		Extra:      opts.Extra,
		Timeout:    opts.Timeout,
	}

	if err := inventory.GateStatusRBAC(ctx, env.Client, ControllerNamespace); err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitPermissionDenied, Err: err}
	}

	if err := waitForTerminatingObjects(ctx, env, plan); err != nil {
		return nil, err
	}

	// The apply guard, the last check, on every install: an object the plan
	// applies must be absent, in the record, this instance's, or annotated
	// for adoption by it. Install needs every object it renders, so an
	// object another instance is adopting refuses here too, where any other
	// apply would leave it out and go on. Nothing has been written yet, so no
	// write of the install reaches an object the guard did not allow.
	var previous []k8sinventory.Entry
	if rec != nil {
		previous = rec.Inventory.Entries
	}
	if err := applyGuard(ctx, env, plan, previous, result.Instance.UUID); err != nil {
		return nil, err
	}
	return plan, nil
}

// embeddedCRDs are the CRDs a CRDs-only install applies; tests replace it.
var embeddedCRDs = crds.Objects

// PinnedTarget is the module version this cli pins and the controller it
// deploys.
func PinnedTarget() Target {
	return Target{ModuleVersion: "v" + PinnedModuleVersion, ControllerVersion: PinnedControllerVersion, Default: true}
}

// PlanCRDsOnly plans `opm controller install --crds-only` from the CRDs this
// cli embeds, which task controller:pin rendered from the pinned module
// version. It works offline: it resolves, fetches and renders nothing, and
// reads no ModuleInstance, so it cannot know whether the controller's
// instance is recorded. Its checks are the wait for terminating objects, the
// check of the existing CRDs (one a newer module version installed is
// refused), and the apply guard over the CRDs, asked as for a first install
// of the controller's instance: the embedded CRDs carry that instance's
// UUID, and no record is read. A CRD that exists and is not OPM's, or that
// carries another instance's identity, refuses the install with the guard's
// own message, which names the adopt annotation; nothing is adopted without
// it. The target rules already held when the version was pinned.
func PlanCRDsOnly(ctx context.Context, env InstallEnv, opts PlanOptions) (*Plan, error) {
	objs, err := embeddedCRDs()
	if err != nil {
		return nil, fmt.Errorf("reading the CRDs this cli embeds: %w", err)
	}
	plan := &Plan{
		Target:   PinnedTarget(),
		CRDs:     objs,
		CRDsOnly: true,
		Extra:    opts.Extra,
		Timeout:  opts.Timeout,
	}
	if err := waitForTerminatingObjects(ctx, env, plan); err != nil {
		return nil, err
	}
	if err := checkExistingCRDs(ctx, env.Client, objs); err != nil {
		return nil, err
	}
	if err := applyGuard(ctx, env, plan, nil, embeddedInstanceUUID(objs)); err != nil {
		return nil, err
	}
	return plan, nil
}

// embeddedInstanceUUID is the UUID of the controller's instance as the
// embedded CRDs carry it: the render that produced them stamped it on each.
func embeddedInstanceUUID(objs []*unstructured.Unstructured) string {
	for _, obj := range objs {
		if uuid := obj.GetLabels()[opmlabels.ModuleInstanceUUID]; uuid != "" {
			return uuid
		}
	}
	return ""
}

// moduleVersionLabel is the label the kernel stamps with the module version
// that rendered an object.
const moduleVersionLabel = "module.opmodel.dev/version"

// moduleNameLabel is the label the kernel stamps with the name of the module
// that rendered an object.
const moduleNameLabel = "module.opmodel.dev/name"

// checkExistingCRDs reads the CRDs a CRDs-only install would apply from the
// API server only, so the form stays offline, and refuses, before any write,
// a CRD the embedded CRD's module (module.opmodel.dev/name) rendered whose
// module.opmodel.dev/version is newer than this cli's pin: applying the
// embedded CRDs would replace that version's schemas (a *TargetError, exit
// 2). Versions of another module are not compared.
//
// A CRD that is absent, without a semver module version, or another module's
// passes here; the apply guard then judges whose it is.
func checkExistingCRDs(ctx context.Context, client *kubernetes.Client, objs []*unstructured.Unstructured) error {
	pinned := "v" + PinnedModuleVersion
	for _, obj := range objs {
		resource, err := client.ResourceClientFor(ctx, obj.GroupVersionKind(), "")
		if err != nil {
			return fmt.Errorf("reading %s/%s before apply: %w", obj.GetKind(), obj.GetName(), err)
		}
		live, err := resource.Get(ctx, obj.GetName(), metav1.GetOptions{})
		if apierrors.IsNotFound(err) {
			continue
		}
		if err != nil {
			return fmt.Errorf("reading %s/%s before apply: %w", obj.GetKind(), obj.GetName(), err)
		}
		labels := live.GetLabels()
		if labels[moduleNameLabel] != obj.GetLabels()[moduleNameLabel] {
			continue
		}
		recorded := labels[moduleVersionLabel]
		if !semver.IsValid("v"+recorded) || semver.Compare("v"+recorded, pinned) <= 0 {
			continue
		}
		return &TargetError{ModuleVersion: pinned, Rule: fmt.Sprintf(
			"CustomResourceDefinition %s was installed by module version %s, newer than this CLI's pin, and --crds-only would replace it with older CRDs; "+
				"run --crds-only from a CLI that pins module version %s or newer, or install that version in full (opm controller install --version %s)",
			obj.GetName(), recorded, recorded, recorded)}
	}
	return nil
}

// waitForTerminatingObjects starts the plan's --timeout budget and waits out
// the objects it applies that are terminating.
func waitForTerminatingObjects(ctx context.Context, env InstallEnv, plan *Plan) error {
	plan.BudgetStart = time.Now()
	waitCtx, cancel := context.WithDeadline(ctx, plan.Deadline())
	defer cancel()
	return waitForTerminating(waitCtx, env.Client, append(plan.Objects(), plan.Extra...), plan.BudgetStart)
}

// applyGuard asks the shared ownership guard (inventory.Guard) about every
// render object the plan applies, as instanceUUID, with previous as the
// record's inventory. Install needs every object it applies, so an object
// adopted by another instance is a refusal too.
func applyGuard(ctx context.Context, env InstallEnv, plan *Plan, previous []k8sinventory.Entry, instanceUUID string) error {
	if _, err := inventory.Guard(ctx, env.Client, inventory.GuardInput{
		Entries:      workflowapply.CurrentInventoryEntries(plan.Objects()),
		Previous:     previous,
		InstanceUUID: instanceUUID,
		RefuseLetGo:  true,
	}); err != nil {
		return &GuardError{Err: err}
	}
	return nil
}

// renderWithValues merges the values over the record's and renders the
// module with them; a rejection of recorded values names --reset-values.
func renderWithValues(ctx context.Context, env InstallEnv, res *modref.Resolution, rec *inventory.Record, opts PlanOptions) (*Values, *workflowrender.Result, error) {
	var recorded map[string]any
	if rec != nil {
		recorded = rec.SpecValues
	}
	values, err := MergeValues(ValuesInput{Recorded: recorded, Files: opts.ValuesFiles, Reset: opts.ResetValues})
	if err != nil {
		return nil, nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	result, err := env.Render(ctx, res, values.Source)
	if err != nil {
		var exitErr *opmexit.ExitError
		if values.FromRecord && errors.As(err, &exitErr) && exitErr.Code == opmexit.ExitValidationError {
			return nil, nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: &RecordedValuesError{Err: err}}
		}
		return nil, nil, err
	}
	return values, result, nil
}

// Objects returns the render objects the plan applies: every rendered
// object, or only the CRDs for a CRDs-only plan.
func (p *Plan) Objects() []*unstructured.Unstructured {
	if p.CRDsOnly {
		return p.CRDs
	}
	return p.Render.Resources
}

// Deadline is the end of the plan's --timeout budget.
func (p *Plan) Deadline() time.Time {
	return p.BudgetStart.Add(p.Timeout)
}

// renderedCRDs returns the render's CustomResourceDefinitions in render
// order.
func renderedCRDs(objs []*unstructured.Unstructured) []*unstructured.Unstructured {
	var out []*unstructured.Unstructured
	for _, obj := range objs {
		if obj.GetKind() == kindCustomResourceDefinition {
			out = append(out, obj)
		}
	}
	return out
}
