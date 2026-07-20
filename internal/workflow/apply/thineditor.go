package apply

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
)

// executeThinEditor is the apply path for an operator-owned instance
// (enhancement 0006 D18, design LD6). The CLI acts as a spec editor and
// nothing more: it writes spec.module and spec.values, then watches the
// operator reconcile them.
//
// Deliberately skipped here, because the operator does them: applying
// resources, pruning stale ones, writing status, and the status-RBAC
// pre-flight (D23 — the CLI writes no status in this mode, so proving it may
// would be theater). Still enforced by the caller before this point: the CRD
// gates and the version-skew ceiling (D24), since an old CLI writing spec for
// a newer operator is the unsafe skew direction.
func executeThinEditor(ctx context.Context, req Request, rec *inventory.Record) error {
	result := req.Result
	name := result.Instance.Name
	namespace := result.Instance.Namespace

	req.Log.Info("instance is operator-managed — editing its spec and waiting for the operator",
		"owner", inventory.DisplayOwner(rec.Owner))

	modulePath, moduleVersion, err := resolveThinEditRef(req, name, namespace)
	if err != nil {
		return err
	}

	// The edit carries the instance's CURRENT owner, not an empty one. Under
	// server-side apply this manager's document is its complete declared
	// intent, so omitting spec.owner would release the field and let the API
	// server prune the operator's ownership marker — the precise outcome the
	// thin editor exists to avoid. Restating the value it already holds is a
	// no-op (design LD6, corrected).
	generation, err := inventory.ApplySpec(ctx, req.K8sClient, inventory.SpecInput{
		Name:          name,
		Namespace:     namespace,
		Owner:         rec.Owner,
		ModulePath:    modulePath,
		ModuleVersion: moduleVersion,
		Values:        result.Values,
		// A local render was refused above, so the provenance annotation must
		// not be stamped; any stale one is correctly cleared with it.
		SourceLocal: false,
	})
	if err != nil {
		return &opmexit.ExitError{Code: exitCodeFromK8sError(err), Err: err}
	}
	output.Debug("thin-editor spec written", "generation", generation)

	req.Log.Info("waiting for the operator to reconcile", "generation", generation)
	outcome, err := inventory.WaitForReconcile(ctx, req.K8sClient, name, namespace, generation, req.Options.Timeout)
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	if !outcome.Reconciled() {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf(
			"the operator did not reconcile the updated spec: %s\nThe spec change is written — the operator will retry. Investigate with:\n  kubectl describe moduleinstance %s -n %s",
			outcome.Describe(), name, namespace)}
	}

	output.Println(output.FormatCheckmark("Instance updated — operator reconciled " + outcome.Ready.Describe()))
	return nil
}

// previewThinEditor is the dry-run counterpart of executeThinEditor. Without
// it a dry-run against an operator-owned instance would resolve to
// CLI-executor mode and preview a render-and-apply the CLI will never perform
// (design LD6). It runs the same refusals, so a dry-run surfaces the local-bytes
// rejection rather than deferring it to the real apply.
func previewThinEditor(req Request, rec *inventory.Record) error {
	name := req.Result.Instance.Name
	namespace := req.Result.Instance.Namespace

	req.Log.Info("instance is operator-managed — previewing the spec edit",
		"owner", inventory.DisplayOwner(rec.Owner))

	modulePath, moduleVersion, err := resolveThinEditRef(req, name, namespace)
	if err != nil {
		return err
	}

	output.Println(output.FormatCheckmark(fmt.Sprintf(
		"Dry run: would set spec.module to %s@%s and update spec.values, then wait for the operator to reconcile.\n"+
			"          The CLI applies and prunes nothing itself in this mode.",
		modulePath, moduleVersion)))
	return nil
}

// resolveThinEditRef runs the checks that must hold before the CLI writes a
// spec the operator will have to resolve, and returns the reference to write.
//
// The operator fetches modules from the registry, so a render whose bytes came
// from a local checkout or a local replacement describes something it cannot
// obtain — writing that spec would strand the instance on a reference the
// operator fails to resolve. Refused before any write.
func resolveThinEditRef(req Request, name, namespace string) (path, version string, err error) {
	if req.Result.SourceLocal {
		return "", "", &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf(
			"instance %q in namespace %q is operator-managed, but this apply resolves its module from local bytes — the operator can only fetch published modules; publish the module and re-apply",
			name, namespace)}
	}

	modulePath, moduleVersion := req.Result.Module.CanonicalModuleRef()
	if modulePath == "" || moduleVersion == "" {
		return "", "", &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf(
			"instance %q has no complete module reference to write (path %q, version %q)",
			name, modulePath, moduleVersion)}
	}
	return modulePath, moduleVersion, nil
}
