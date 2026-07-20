// Package handoff implements `opm instance handoff`: the verified, forward-only
// transfer of an instance from CLI management to operator management
// (enhancement 0006 slice C3, decisions D6/D7/D16/D38/D40).
//
// The shape is a precondition chain that aborts before touching the CR, a
// single-field ownership flip, and a bounded wait that judges the operator's
// first reconcile by D40's inventory-stable criterion. There is no reverse
// mode and no cross-actor digest comparison — both are designed out, not
// merely unimplemented.
package handoff

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/charmbracelet/log"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/operator"
	"github.com/open-platform-model/cli/internal/output"
	workflowapply "github.com/open-platform-model/cli/internal/workflow/apply"
	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
)

// Request is one handoff invocation.
type Request struct {
	// Name and Namespace identify the ModuleInstance to hand off.
	Name      string
	Namespace string

	K8sClient *kubernetes.Client
	Config    *config.GlobalConfig
	Log       *log.Logger

	// Timeout bounds the post-flip reconcile wait.
	Timeout time.Duration

	// Force bypasses the digest gate only (design LD2). It does not relax the
	// operator-readiness, ownership, provenance, or resolvability gates: an
	// unresolvable module or a local-provenance render makes the flip unsafe by
	// construction, not merely unverified.
	Force bool
}

// Execute runs the full handoff: gates, flip, verdict.
func Execute(ctx context.Context, req Request) error {
	rec, err := runPreconditions(ctx, req)
	if err != nil {
		return err
	}

	// Gates 4-5 downloaded and rendered a module, which takes real wall-clock
	// time, and ownership was still `cli` for all of it — so a concurrent
	// `opm instance apply` could legitimately have moved the spec since the
	// record was read. Re-read and refuse rather than flip.
	//
	// Refusing is the point: adopting the *fresh* module and values instead
	// would flip a document the digest gate never verified, since gate 5 proved
	// parity against the spec as it stood at the start. This manager is the sole
	// SSA writer for these fields with Force, so proceeding on the stale record
	// would instead silently revert the concurrent apply — and the D40 verdict
	// could not catch it, because the baseline would be that same stale
	// snapshot.
	fresh, err := inventory.GetRecord(ctx, req.K8sClient, req.Name, req.Namespace)
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf(
			"re-reading %q before the flip: %w", req.Name, err)}
	}
	if err := ensureUnchangedSinceVerification(req, rec, fresh); err != nil {
		return err
	}
	rec = fresh

	// The pre-flip inventory snapshot is the baseline the D40 verdict compares
	// against. Taken from the re-read above, so it reflects the state the flip
	// actually acts on.
	before := rec.Inventory

	req.Log.Info("handing off to the operator", "instance", req.Name, "namespace", req.Namespace)

	// The flip re-states the instance's current module and values alongside the
	// new owner. That is not redundancy: under server-side apply this manager's
	// document is its complete declared intent, so a document carrying only
	// spec.owner would release — and the API server would prune — the module
	// reference and values that opm-cli already owns. Restating an unchanged
	// value changes nothing and does not bump the generation; omitting it
	// deletes it (design LD4, corrected).
	generation, err := inventory.ApplySpec(ctx, req.K8sClient, inventory.SpecInput{
		Name:          req.Name,
		Namespace:     req.Namespace,
		Owner:         inventory.OwnerOperator,
		ModulePath:    rec.ModulePath,
		ModuleVersion: rec.ModuleVersion,
		Values:        rec.SpecValues,
		// Gate 3 refused a local-provenance instance, so the annotation is
		// already absent and omitting it is a no-op.
		SourceLocal: false,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	output.Debug("ownership flipped", "generation", generation)

	outcome, err := inventory.WaitForReconcile(ctx, req.K8sClient, req.Name, req.Namespace, generation, req.Timeout)
	if err != nil {
		return handoffPostFlipError(req, fmt.Errorf("observing the operator's reconcile: %w", err))
	}

	return reportVerdict(req, before, outcome)
}

// ensureUnchangedSinceVerification refuses the flip when the instance moved
// between the gate chain's read and the write, comparing metadata.generation —
// which the API server bumps on spec changes only, exactly the writes that
// would invalidate the verification render.
//
// It reports rather than repairs. Re-reading the spec and flipping the fresh
// values would be worse than the race it closes: gate 5 proved digest parity
// against the spec as it stood when verification began, so adopting a newer one
// flips a document nothing verified.
func ensureUnchangedSinceVerification(req Request, verified, current *inventory.Record) error {
	if current == nil {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf(
			"instance %q in namespace %q disappeared while handoff was verifying it; ownership was left unchanged",
			req.Name, req.Namespace)}
	}
	if current.Generation != verified.Generation {
		return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf(
			"instance %q changed while handoff was verifying it (generation %d -> %d) — the verification "+
				"render no longer describes what is deployed, so ownership was left with the CLI.\n"+
				"Re-run handoff to verify the current spec:\n  opm instance handoff %s -n %s",
			req.Name, verified.Generation, current.Generation, req.Name, req.Namespace)}
	}
	return nil
}

// runPreconditions executes the gate chain in cheapest-first order (design
// LD2), aborting on the first failure with the CR unmodified. It returns the
// record the gates validated.
func runPreconditions(ctx context.Context, req Request) (*inventory.Record, error) {
	// Gate 0: version-skew ceiling (D24) plus the CRD gates — the same
	// cluster-write preconditions every mutating path runs.
	if err := workflowapply.RunClusterGates(ctx, req.K8sClient); err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	// Gate 1: the operator must be installed and serving. Handing off to an
	// absent operator would leave the instance owned by nobody.
	if err := operator.CheckReady(ctx, req.K8sClient); err != nil {
		var notReady *operator.NotReadyError
		if errors.As(err, &notReady) {
			notReady.Hint = "handing off would leave this instance with no active manager"
		}
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	// Gate 2: the CR exists and the CLI owns it.
	rec, err := inventory.GetRecord(ctx, req.K8sClient, req.Name, req.Namespace)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	if rec == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitNotFound, Err: fmt.Errorf(
			"no ModuleInstance %q in namespace %q — apply the instance with the CLI before handing it off",
			req.Name, req.Namespace)}
	}
	if inventory.ResolveOwnership(rec) == inventory.ModeOperatorOwned {
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf(
			"instance %q in namespace %q is already operator-managed (spec.owner: %s) — handoff is forward-only and there is no reverse mode",
			req.Name, req.Namespace, inventory.DisplayOwner(rec.Owner))}
	}

	// Gate 3: the local-provenance annotation (D38). A metadata read, so it
	// runs before any registry or render work.
	if rec.SourceLocal {
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf(
			"instance %q was last applied from local module bytes (%s: %s) — the operator resolves modules from the registry only and cannot reproduce this render; publish the module, re-apply with the CLI, then hand off",
			req.Name, inventory.AnnotationSource, inventory.SourceLocal)}
	}

	if rec.ModulePath == "" || rec.ModuleVersion == "" {
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf(
			"instance %q has no complete spec.module reference (path %q, version %q) — re-apply with the CLI before handing off",
			req.Name, rec.ModulePath, rec.ModuleVersion)}
	}

	// Gates 4 and 5: strict-registry resolution and the digest self-comparison.
	if err := verify(ctx, req, rec); err != nil {
		return nil, err
	}

	return rec, nil
}

// verify runs the strict-registry verification render (gate 4) and the digest
// self-comparison against the CR's own status.lastAppliedRenderDigest (gate 5).
func verify(ctx context.Context, req Request, rec *inventory.Record) error {
	// Nothing to verify against is a metadata read, so it settles before the
	// module download — the chain is cheapest-first (design LD2), and paying
	// for a full registry render only to discover there is no recorded digest
	// to compare it with would invert that.
	if rec.LastAppliedRenderDigest == "" {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf(
			"instance %q records no status.lastAppliedRenderDigest to verify against — re-apply with the CLI before handing off",
			req.Name)}
	}

	// The path already carries its major-version tag (…/podinfo@v0), so the
	// pinned version is reported as its own field rather than concatenated —
	// "…podinfo@v0@v0.1.3" reads like a malformed reference.
	req.Log.Info("verifying the published module reproduces the deployed state",
		"module", rec.ModulePath, "version", rec.ModuleVersion)

	digest, err := VerificationDigest(ctx, VerificationInput{
		Client:        req.K8sClient,
		Config:        req.Config,
		Name:          rec.Name,
		Namespace:     rec.Namespace,
		ModulePath:    rec.ModulePath,
		ModuleVersion: rec.ModuleVersion,
		SpecValues:    rec.SpecValues,
	})
	if err != nil {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err}
	}

	if digest == rec.LastAppliedRenderDigest {
		output.Println(output.FormatCheckmark("verification render matches the deployed state"))
		return nil
	}

	mismatch := fmt.Errorf(
		"the published module does not reproduce the deployed state:\n"+
			"  deployed  (status.lastAppliedRenderDigest): %s\n"+
			"  published (%s %s): %s\n"+
			"the cluster is running something the registry no longer describes — "+
			"re-apply with the CLI to reconcile them, or pass --force to hand off anyway",
		rec.LastAppliedRenderDigest, rec.ModulePath, rec.ModuleVersion, digest)

	if !req.Force {
		return &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: mismatch}
	}
	output.Warn("--force: proceeding despite a verification digest mismatch")
	output.Details(mismatch.Error())
	return nil
}

// reportVerdict applies D40's inventory-stable criterion to the post-flip
// observation and renders the result (design LD5). Success requires the Ready
// condition True for the flipped generation, the inventory entry set unchanged
// as a set, and the revision incremented.
func reportVerdict(req Request, before pkginventory.Inventory, outcome *inventory.ReconcileOutcome) error {
	if outcome.TimedOut || !outcome.Reconciled() {
		return handoffPostFlipError(req, fmt.Errorf("the operator did not complete a successful reconcile: %s", outcome.Describe()))
	}

	after := outcome.Record.Inventory

	if drift := inventory.DescribeEntrySetDrift(before.Entries, after.Entries); drift != "" {
		return handoffPostFlipError(req, fmt.Errorf(
			"the operator's first reconcile changed the instance's resource set, which a handoff must never do: %s", drift))
	}

	if after.Revision <= before.Revision {
		return handoffPostFlipError(req, fmt.Errorf(
			"the operator did not record a new inventory revision (was %d, now %d) — it may not have taken ownership",
			before.Revision, after.Revision))
	}

	output.Println(output.FormatCheckmark(fmt.Sprintf(
		"operator adopted %d resources — managed-by relabel only, no workload changes", len(after.Entries))))
	req.Log.Info("handoff complete", "instance", req.Name, "namespace", req.Namespace, "owner", inventory.OwnerOperator)
	return nil
}

// handoffPostFlipError wraps a failure that happened after the ownership flip.
// The flip is not reverted: reverse handoff does not exist (D16), and
// auto-reverting would reintroduce exactly the design surface that decision
// excluded. The message says so, and names the investigation path.
func handoffPostFlipError(req Request, cause error) error {
	return &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf(
		"%w\n\nownership remains with the operator — the flip is not reverted, and handoff is forward-only with no reverse mode.\nInvestigate with:\n  kubectl describe moduleinstance %s -n %s\n  kubectl logs -n opm-operator-system deploy/opm-operator-controller-manager",
		cause, req.Name, req.Namespace)}
}
