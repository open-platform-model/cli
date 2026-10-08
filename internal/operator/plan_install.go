package operator

import (
	"context"
	"errors"
	"fmt"
	"regexp"
	"strings"
	"time"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	"github.com/open-platform-model/library/opm/kernel"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/modref"
	"github.com/open-platform-model/cli/internal/publish"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
	workflowrender "github.com/open-platform-model/cli/internal/workflow/render"
)

// ModuleRegistry is what install reads the operator module through: version
// resolution and the module source. modconfig's cached registry satisfies it.
type ModuleRegistry interface {
	modref.Source
	ModuleFetcher
}

// RenderFunc renders a resolved operator module version as the instance
// OperatorInstanceName in OperatorNamespace, from the one values source given,
// against the module's own dependency pins (never the cluster Platform).
type RenderFunc func(ctx context.Context, module *modref.Resolution, values kernel.Source) (*workflowrender.Result, error)

// InstallEnv is what install works through.
type InstallEnv struct {
	Client *kubernetes.Client
	Render RenderFunc
	// CLIVersion is this CLI's version, for the operator-version rule.
	CLIVersion string
}

// PlanOptions configures what PlanInstall decides.
type PlanOptions struct {
	// ValuesFiles are -f/--values files, layered over the recorded values.
	ValuesFiles []string
	// ResetValues is --reset-values.
	ResetValues bool
	// CRDsOnly is --crds-only: only the rendered CRDs are applied.
	CRDsOnly bool
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
	// Render is the instance's render.
	Render *workflowrender.Result
	// CRDs are the render's CustomResourceDefinitions.
	CRDs []*unstructured.Unstructured
	// PrevRecord is the operator instance's record; nil on a fresh cluster.
	PrevRecord *inventory.Record
	Values     *Values
	CRDsOnly   bool
	Extra      []*unstructured.Unstructured
	// Migration is the migration of an operator installed from an earlier
	// release manifest; it plans no write on a cluster with nothing to
	// migrate.
	Migration *MigrationPlan
	// BudgetStart and Timeout are the --timeout budget the writes share.
	BudgetStart time.Time
	Timeout     time.Duration
}

// operatorTagShape is the shape of an opm-operator release tag.
var operatorTagShape = regexp.MustCompile(`^v\d+\.\d+\.\d+(-[0-9A-Za-z.-]+)?$`)

// looksLikeOperatorTag reports whether a --version value is shaped like an
// opm-operator release tag rather than an operator module version: a
// "v"-prefixed release, or a release on a major above the module's v0.
func looksLikeOperatorTag(version string) bool {
	if operatorTagShape.MatchString(version) {
		return true
	}
	return operatorTagShape.MatchString("v"+version) && !strings.HasPrefix(version, "0.")
}

// operatorTagRefusal is the refusal for an old-style --version value.
func operatorTagRefusal(version string, cause error) error {
	r := publish.Refusal{
		Headline: fmt.Sprintf("no operator module version matches %q: --version now takes an operator module version (for example %s), not an opm-operator release tag",
			version, PinnedModuleVersion),
		Action: "Pass a version of " + OperatorModulePath + "; install prints the operator release it deploys.",
	}
	var mr *modref.RefusalError
	if errors.As(cause, &mr) {
		r.Evidence = mr.Refusal.Evidence
	}
	return &modref.RefusalError{Refusal: r}
}

// ResolveTarget resolves the module version install deploys, the pin when
// version is empty, and reads the operator release it states, all before any
// cluster call. A selector the registry cannot satisfy is a
// *modref.RefusalError (worded for an old operator tag when it looks like
// one), an unreadable operator version a *VersionError, a registry that
// cannot be reached a *publish.ConnectivityError.
func ResolveTarget(ctx context.Context, src ModuleRegistry, registry, version string) (*modref.Resolution, Target, error) {
	sel := modref.Selector{Exact: PinnedModuleVersion}
	if version != "" {
		var err error
		sel, err = modref.ParseSelector(version)
		if err != nil {
			if looksLikeOperatorTag(version) {
				return nil, Target{}, operatorTagRefusal(version, err)
			}
			return nil, Target{}, err
		}
	}
	res, err := modref.Resolve(ctx, src, modref.Request{Path: OperatorModulePath, Selector: sel, Registry: registry})
	if err != nil {
		var mr *modref.RefusalError
		if version != "" && errors.As(err, &mr) && looksLikeOperatorTag(version) {
			return nil, Target{}, operatorTagRefusal(version, err)
		}
		return nil, Target{}, err
	}
	opVersion, err := ReadOperatorVersion(ctx, src, res.Version)
	if err != nil {
		return nil, Target{}, err
	}
	return res, Target{ModuleVersion: res.Version, OperatorVersion: opVersion, Default: version == ""}, nil
}

// OwnedRecordError refuses an install over an operator instance record
// whose spec.owner is not cli (an absent owner resolves as operator-owned):
// the operator's own instance is CLI-owned, and the instance apply would
// otherwise only edit its spec.
type OwnedRecordError struct{}

func (e *OwnedRecordError) Error() string {
	return fmt.Sprintf(
		"refusing to install: ModuleInstance %s/%s is not spec.owner: cli, and the operator never reconciles the instance that deploys it; "+
			"set spec.owner to cli (kubectl patch moduleinstance %s -n %s --type=merge -p '{\"spec\":{\"owner\":\"cli\"}}'), then retry",
		OperatorNamespace, OperatorInstanceName, OperatorInstanceName, OperatorNamespace)
}

// RecordedValuesError wraps a render that rejected values recorded on the
// operator's instance, naming --reset-values.
type RecordedValuesError struct{ Err error }

func (e *RecordedValuesError) Error() string {
	return fmt.Sprintf("%v (the values recorded on %s/%s took part; pass --reset-values to start from the module's defaults)",
		e.Err, OperatorNamespace, OperatorInstanceName)
}

func (e *RecordedValuesError) Unwrap() error { return e.Err }

// GuardError is the apply guard's refusal: an object the plan applies
// exists and is not OPM's, belongs to or is adopted by another instance, is
// terminating, or could not be read by the guard (the read error is then in
// the chain). Nothing was written. The install command exits
// 2 on every GuardError, whatever it wraps. An object that is unreadable
// before the guard runs is refused earlier, by the terminating wait or the
// migration proof, with the exit code of the read error.
type GuardError struct{ Err error }

func (e *GuardError) Error() string {
	return "refusing to install: " + e.Err.Error() + "\nnothing was changed"
}
func (e *GuardError) Unwrap() error { return e.Err }

// PlanInstall runs every check that can refuse the install, in order, and
// writes nothing: the record read, the values merge, the render (which
// checks the values against the target's #config), the target rules
// (CheckTarget), the status-subresource permission check, the wait for
// terminating objects, the migration proof of an operator installed from an
// earlier release manifest and, last, the apply guard over every object the
// plan applies, with or without a record. The terminating wait starts the
// --timeout budget the writes then share.
func PlanInstall(ctx context.Context, env InstallEnv, res *modref.Resolution, target Target, opts PlanOptions) (*Plan, error) {
	rec, err := inventory.GetRecord(ctx, env.Client, OperatorInstanceName, OperatorNamespace)
	if err != nil {
		return nil, fmt.Errorf("reading the operator's instance record %s/%s: %w", OperatorNamespace, OperatorInstanceName, err)
	}
	if rec != nil && inventory.ResolveOwnership(rec) == inventory.ModeOperatorOwned {
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
		CRDsOnly:   opts.CRDsOnly,
		Extra:      opts.Extra,
		Timeout:    opts.Timeout,
	}

	if !opts.CRDsOnly {
		if err := inventory.GateStatusRBAC(ctx, env.Client, OperatorNamespace); err != nil {
			return nil, &opmexit.ExitError{Code: opmexit.ExitPermissionDenied, Err: err}
		}
	}

	plan.BudgetStart = time.Now()
	waitCtx, cancel := context.WithDeadline(ctx, plan.Deadline())
	defer cancel()
	if err := waitForTerminating(waitCtx, env.Client, append(plan.Objects(), plan.Extra...), plan.BudgetStart); err != nil {
		return nil, err
	}

	// The migration proof (0012:D8:R6): which existing objects came from an
	// earlier operator manifest. It runs before the guard, which would
	// otherwise refuse the first of them.
	plan.Migration, err = PlanMigration(ctx, env.Client, plan.Objects(), result.Instance.UUID, opts.CRDsOnly)
	if err != nil {
		return nil, err
	}

	// The apply guard, the last check, on every install: an object the plan
	// applies must be absent, in the record, this instance's, or admitted by
	// the migration's proof. Install needs every object it renders, so an
	// object another instance is adopting refuses here too, where any other
	// apply would leave it out and go on. Nothing has been written yet, so no
	// write of the install reaches an object the guard did not allow.
	var previous []k8sinventory.Entry
	if rec != nil {
		previous = rec.Inventory.Entries
	}
	if _, err := inventory.Guard(ctx, env.Client, inventory.GuardInput{
		Entries:      workflowapply.CurrentInventoryEntries(plan.Objects()),
		Previous:     previous,
		InstanceUUID: result.Instance.UUID,
		Admit:        plan.Migration.Admit(),
		RefuseLetGo:  true,
	}); err != nil {
		return nil, &GuardError{Err: err}
	}
	return plan, nil
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
// object, or only the CRDs for --crds-only.
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
	var crds []*unstructured.Unstructured
	for _, obj := range objs {
		if obj.GetKind() == kindCustomResourceDefinition {
			crds = append(crds, obj)
		}
	}
	return crds
}
