package handoff

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
)

func testRequest() Request {
	return Request{Name: "podinfo", Namespace: "demo", Log: output.InstanceLogger("test")}
}

func inv(revision int, kinds ...string) pkginventory.Inventory {
	entries := make([]inventory.InventoryEntry, 0, len(kinds))
	for _, k := range kinds {
		entries = append(entries, inventory.InventoryEntry{
			Group: "apps", Version: "v1", Kind: k, Namespace: "demo", Name: "podinfo", Component: "web",
		})
	}
	return pkginventory.Inventory{Revision: revision, Count: len(entries), Entries: entries}
}

func reconciled(revision int, kinds ...string) *inventory.ReconcileOutcome {
	return &inventory.ReconcileOutcome{
		Generation: 2,
		Ready:      &inventory.Condition{Type: inventory.ConditionTypeReady, Status: inventory.ConditionTrue, Reason: "Reconciled", ObservedGeneration: 2},
		Record:     &inventory.Record{Name: "podinfo", Namespace: "demo", Inventory: inv(revision, kinds...)},
	}
}

// The D40 success case: same resources, new revision, Ready. The managed-by
// relabel that the operator performs is invisible here by design — identity is
// what is compared, not content.
func TestReportVerdict_InventoryStableReconcileSucceeds(t *testing.T) {
	before := inv(3, "Deployment", "Service")
	outcome := reconciled(4, "Service", "Deployment") // order deliberately differs

	require.NoError(t, reportVerdict(testRequest(), before, outcome))
}

func TestReportVerdict_ChangedEntrySetFails(t *testing.T) {
	before := inv(3, "Deployment", "Service")
	outcome := reconciled(4, "Deployment")

	err := reportVerdict(testRequest(), before, outcome)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "changed the instance's resource set")
	assert.Contains(t, err.Error(), "no longer tracked")
}

func TestReportVerdict_UnincrementedRevisionFails(t *testing.T) {
	before := inv(4, "Deployment")
	outcome := reconciled(4, "Deployment")

	err := reportVerdict(testRequest(), before, outcome)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "new inventory revision")
}

func TestReportVerdict_FailedReconcileReportsTheOperatorCondition(t *testing.T) {
	outcome := &inventory.ReconcileOutcome{
		Generation: 2,
		Ready: &inventory.Condition{
			Type: inventory.ConditionTypeReady, Status: inventory.ConditionFalse,
			Reason: "TransformerMissing", Message: "no matching transformer", ObservedGeneration: 2,
		},
		Record: &inventory.Record{Name: "podinfo", Namespace: "demo", Inventory: inv(3, "Deployment")},
	}

	err := reportVerdict(testRequest(), inv(3, "Deployment"), outcome)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "no matching transformer")
}

func TestReportVerdict_TimeoutReportsTheLastCondition(t *testing.T) {
	outcome := &inventory.ReconcileOutcome{
		Generation: 2,
		TimedOut:   true,
		Record:     &inventory.Record{Name: "podinfo", Namespace: "demo", ObservedGeneration: 1},
	}

	err := reportVerdict(testRequest(), inv(3, "Deployment"), outcome)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "timed out")
}

// Every post-flip failure must say the flip stands. Auto-reverting would
// reintroduce the reverse-handoff surface that 0006 D16 excluded, so the error
// text carrying this statement is part of the contract, not decoration.
func TestHandoffPostFlipError_StatesOwnershipIsNotReverted(t *testing.T) {
	err := handoffPostFlipError(testRequest(), stubCauseError{})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "ownership remains with the operator")
	assert.Contains(t, err.Error(), "not reverted")
	assert.Contains(t, err.Error(), "forward-only")
	assert.Contains(t, err.Error(), "kubectl describe moduleinstance podinfo -n demo")
	assert.Contains(t, err.Error(), "kubectl logs")
}

type stubCauseError struct{}

func (stubCauseError) Error() string { return "cause" }

// The verification window is wide — gates 4-5 download and render a module —
// and ownership is still `cli` throughout it, so a concurrent apply can move
// the spec. Flipping the stale record would silently revert that apply, and the
// D40 verdict could not catch it because its baseline would be the same stale
// snapshot.
func TestEnsureUnchangedSinceVerification_RefusesAConcurrentSpecChange(t *testing.T) {
	verified := &inventory.Record{Name: "podinfo", Namespace: "demo", Generation: 4}
	current := &inventory.Record{Name: "podinfo", Namespace: "demo", Generation: 5}

	err := ensureUnchangedSinceVerification(testRequest(), verified, current)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "generation 4 -> 5")
	assert.Contains(t, err.Error(), "left with the CLI")
	assert.Contains(t, err.Error(), "opm instance handoff podinfo -n demo")
}

func TestEnsureUnchangedSinceVerification_RefusesADisappearedInstance(t *testing.T) {
	verified := &inventory.Record{Name: "podinfo", Namespace: "demo", Generation: 4}

	err := ensureUnchangedSinceVerification(testRequest(), verified, nil)

	require.Error(t, err)
	assert.Contains(t, err.Error(), "disappeared")
	assert.Contains(t, err.Error(), "unchanged")
}

// A no-op re-apply does not bump the generation (design LD4), so an unchanged
// instance must pass even though the record was read twice.
func TestEnsureUnchangedSinceVerification_AllowsAnUnchangedInstance(t *testing.T) {
	verified := &inventory.Record{Name: "podinfo", Namespace: "demo", Generation: 4}
	current := &inventory.Record{Name: "podinfo", Namespace: "demo", Generation: 4}

	require.NoError(t, ensureUnchangedSinceVerification(testRequest(), verified, current))
}
