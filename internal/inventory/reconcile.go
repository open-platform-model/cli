package inventory

import (
	"context"
	"fmt"
	"time"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/output"
)

// reconcilePollInterval is how often the reconcile waits re-read the CR. Poll
// based rather than watch based (design LD1): the CLI's dynamic client usage
// stays get/list-only, and a one-shot wait does not justify informer caches.
const reconcilePollInterval = 2 * time.Second

// DefaultReconcileTimeout bounds a reconcile wait when the caller states no
// preference. It matches the operator-install readiness waits.
const DefaultReconcileTimeout = 5 * time.Minute

// ResolveTimeout reads a caller-supplied bound, falling back to
// DefaultReconcileTimeout when none was stated. Both waits below apply it to
// their own argument, so a zero or negative `--timeout` means "the default" on
// every command rather than an already-expired deadline on whichever one forgot
// to normalize.
func ResolveTimeout(timeout time.Duration) time.Duration {
	if timeout <= 0 {
		return DefaultReconcileTimeout
	}
	return timeout
}

// ReconcileOutcome is what a bounded reconcile wait observed. It is always
// returned — including on timeout — so callers can report the operator's last
// known verdict instead of a bare deadline error (design LD1).
type ReconcileOutcome struct {
	// Record is the last successfully read CR state, or nil when the CR was
	// never readable during the wait.
	Record *Record

	// Ready is the Ready condition attributed to the awaited generation, or
	// nil when the operator had not yet reported one.
	Ready *Condition

	// Generation is the generation the wait was watching for.
	Generation int64

	// TimedOut reports that the wait hit its deadline before the operator
	// reconciled the awaited generation.
	TimedOut bool
}

// Reconciled reports whether the operator observed the awaited generation and
// reported it Ready.
func (o *ReconcileOutcome) Reconciled() bool {
	return o != nil && !o.TimedOut && o.Ready.IsTrue()
}

// Describe renders the outcome for user-facing output.
func (o *ReconcileOutcome) Describe() string {
	switch {
	case o == nil:
		return "no reconcile observed"
	case o.TimedOut && o.Record == nil:
		return "timed out before the ModuleInstance became readable"
	case o.TimedOut && o.Ready == nil:
		return fmt.Sprintf(
			"timed out waiting for the operator to reconcile generation %d (last observed generation %d); %s",
			o.Generation, o.Record.ObservedGeneration, o.Ready.Describe())
	case o.TimedOut:
		return fmt.Sprintf("timed out waiting for generation %d; last %s condition: %s",
			o.Generation, ConditionTypeReady, o.Ready.Describe())
	default:
		return ConditionTypeReady + ": " + o.Ready.Describe()
	}
}

// WaitForReconcile polls the ModuleInstance CR until the operator has observed
// the given generation and reported a Ready condition for it, or timeout
// elapses. It never returns an error for a not-yet-reconciled or absent CR —
// those are ordinary intermediate states that resolve into the returned
// outcome. An error is returned only when the CR cannot be read at all and the
// wait has nothing left to observe.
func WaitForReconcile(ctx context.Context, client *kubernetes.Client, name, namespace string, generation int64, timeout time.Duration) (*ReconcileOutcome, error) {
	return waitForReconcile(ctx, client, name, namespace, generation, timeout, reconcilePollInterval)
}

func waitForReconcile(ctx context.Context, client *kubernetes.Client, name, namespace string, generation int64, timeout, pollInterval time.Duration) (*ReconcileOutcome, error) {
	ctx, cancel := context.WithTimeout(ctx, ResolveTimeout(timeout))
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	outcome := &ReconcileOutcome{Generation: generation}
	var lastReadErr error

	for {
		rec, err := GetRecord(ctx, client, name, namespace)
		switch {
		case err != nil:
			// Transient read failures do not abort the wait; the deadline does.
			lastReadErr = err
			output.Debug("reconcile wait: could not read ModuleInstance", "name", name, "error", err)
		case rec != nil:
			lastReadErr = nil
			outcome.Record = rec
			outcome.Ready = rec.ReadyFor(generation)
			if outcome.Ready != nil {
				return outcome, nil
			}
		}

		select {
		case <-ctx.Done():
			outcome.TimedOut = true
			if outcome.Record == nil && lastReadErr != nil {
				return outcome, fmt.Errorf("waiting for ModuleInstance %q to reconcile: %w", name, lastReadErr)
			}
			return outcome, nil
		case <-ticker.C:
		}
	}
}

// WaitForAbsence polls until the ModuleInstance CR no longer exists — the
// completion signal for an operator-finalized delete, whose cleanup finalizer
// keeps the CR present until every workload is pruned (design LD7).
func WaitForAbsence(ctx context.Context, client *kubernetes.Client, name, namespace string, timeout time.Duration) error {
	return waitForAbsence(ctx, client, name, namespace, timeout, reconcilePollInterval)
}

func waitForAbsence(ctx context.Context, client *kubernetes.Client, name, namespace string, timeout, pollInterval time.Duration) error {
	timeout = ResolveTimeout(timeout)
	ctx, cancel := context.WithTimeout(ctx, timeout)
	defer cancel()

	ticker := time.NewTicker(pollInterval)
	defer ticker.Stop()

	for {
		rec, err := GetRecord(ctx, client, name, namespace)
		if err == nil && rec == nil {
			return nil
		}
		if err != nil {
			output.Debug("absence wait: could not read ModuleInstance", "name", name, "error", err)
		}

		select {
		case <-ctx.Done():
			return fmt.Errorf(
				"timed out after %s waiting for ModuleInstance %q in namespace %q to be removed — the operator's %s finalizer may still be pruning workloads; check 'kubectl get moduleinstance %s -n %s -o yaml' and the operator logs",
				timeout, name, namespace, CleanupFinalizer, name, namespace)
		case <-ticker.C:
		}
	}
}
