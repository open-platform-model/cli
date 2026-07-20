package inventory

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
)

// makeInstanceCR builds a ModuleInstance CR with a generation, an
// observedGeneration, and an optional Ready condition.
func makeInstanceCR(generation, observedGeneration int64, ready *Condition) *unstructured.Unstructured {
	const name, namespace = "app", "ns"

	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": APIVersionModuleInstance,
		"kind":       KindModuleInstance,
		"metadata": map[string]any{
			"name":       name,
			"namespace":  namespace,
			"generation": generation,
		},
		"spec": map[string]any{"owner": OwnerCLI},
	}}

	status := map[string]any{"observedGeneration": observedGeneration}
	if ready != nil {
		cond := map[string]any{
			"type":    ready.Type,
			"status":  ready.Status,
			"reason":  ready.Reason,
			"message": ready.Message,
		}
		if ready.ObservedGeneration > 0 {
			cond["observedGeneration"] = ready.ObservedGeneration
		}
		status["conditions"] = []any{cond}
	}
	obj.Object["status"] = status
	return obj
}

func readyCond(status, reason string, gen int64) *Condition {
	return &Condition{Type: ConditionTypeReady, Status: status, Reason: reason, Message: "reconcile " + reason, ObservedGeneration: gen}
}

func TestRecordReadyFor_GenerationAttribution(t *testing.T) {
	tests := []struct {
		name       string
		observed   int64
		cond       *Condition
		awaited    int64
		wantFound  bool
		wantStatus string
	}{
		{
			name:      "no condition reported yet",
			observed:  3,
			cond:      nil,
			awaited:   3,
			wantFound: false,
		},
		{
			name:      "condition is stale by its own observedGeneration",
			observed:  3,
			cond:      readyCond(ConditionTrue, "Reconciled", 2),
			awaited:   3,
			wantFound: false,
		},
		{
			name:       "condition reports the awaited generation",
			observed:   3,
			cond:       readyCond(ConditionTrue, "Reconciled", 3),
			awaited:    3,
			wantFound:  true,
			wantStatus: ConditionTrue,
		},
		{
			name:       "condition reports a newer generation",
			observed:   4,
			cond:       readyCond(ConditionFalse, "ReconcileFailed", 4),
			awaited:    3,
			wantFound:  true,
			wantStatus: ConditionFalse,
		},
		{
			name:      "unstated condition generation falls back to a stale status.observedGeneration",
			observed:  2,
			cond:      readyCond(ConditionTrue, "Reconciled", 0),
			awaited:   3,
			wantFound: false,
		},
		{
			name:       "unstated condition generation falls back to a caught-up status.observedGeneration",
			observed:   3,
			cond:       readyCond(ConditionTrue, "Reconciled", 0),
			awaited:    3,
			wantFound:  true,
			wantStatus: ConditionTrue,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			rec := recordFromUnstructured(makeInstanceCR(3, tt.observed, tt.cond))
			got := rec.ReadyFor(tt.awaited)
			if !tt.wantFound {
				assert.Nil(t, got)
				return
			}
			require.NotNil(t, got)
			assert.Equal(t, tt.wantStatus, got.Status)
		})
	}
}

func TestRecordFromUnstructured_ReadsGenerationAndConditions(t *testing.T) {
	rec := recordFromUnstructured(makeInstanceCR(7, 6, readyCond(ConditionFalse, "ReconcileFailed", 6)))

	assert.Equal(t, int64(7), rec.Generation)
	assert.Equal(t, int64(6), rec.ObservedGeneration)
	require.Len(t, rec.Conditions, 1)
	assert.Equal(t, ConditionTypeReady, rec.Conditions[0].Type)
	assert.Equal(t, "ReconcileFailed", rec.Conditions[0].Reason)
	assert.False(t, rec.ReadyCondition().IsTrue())
}

func TestConditionsFromUnstructured_MalformedIsEmpty(t *testing.T) {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": APIVersionModuleInstance,
		"kind":       KindModuleInstance,
		"metadata":   map[string]any{"name": "app", "namespace": "ns"},
		"status":     map[string]any{"conditions": "not-a-list"},
	}}
	assert.Empty(t, conditionsFromUnstructured(obj))
}

func TestWaitForReconcile_ReturnsOnReadyForGeneration(t *testing.T) {
	client := newDynamicClient(makeInstanceCR(4, 4, readyCond(ConditionTrue, "Reconciled", 4)))

	outcome, err := waitForReconcile(context.Background(), client, "app", "ns", 4, time.Second, time.Millisecond)
	require.NoError(t, err)
	require.NotNil(t, outcome)
	assert.True(t, outcome.Reconciled())
	assert.False(t, outcome.TimedOut)
	assert.Contains(t, outcome.Describe(), ConditionTrue)
}

func TestWaitForReconcile_FailedReconcileIsObservedNotAnError(t *testing.T) {
	client := newDynamicClient(makeInstanceCR(4, 4, readyCond(ConditionFalse, "ReconcileFailed", 4)))

	outcome, err := waitForReconcile(context.Background(), client, "app", "ns", 4, time.Second, time.Millisecond)
	require.NoError(t, err)
	assert.False(t, outcome.Reconciled())
	assert.False(t, outcome.TimedOut)
	require.NotNil(t, outcome.Ready)
	assert.Equal(t, "ReconcileFailed", outcome.Ready.Reason)
}

func TestWaitForReconcile_TimeoutReportsLastObservedCondition(t *testing.T) {
	// The operator is still reporting on the previous generation.
	client := newDynamicClient(makeInstanceCR(5, 4, readyCond(ConditionTrue, "Reconciled", 4)))

	outcome, err := waitForReconcile(context.Background(), client, "app", "ns", 5, 20*time.Millisecond, time.Millisecond)
	require.NoError(t, err)
	assert.True(t, outcome.TimedOut)
	assert.False(t, outcome.Reconciled())
	require.NotNil(t, outcome.Record)
	assert.Equal(t, int64(4), outcome.Record.ObservedGeneration)
	assert.Contains(t, outcome.Describe(), "timed out")
}

func TestWaitForReconcile_AbsentCRTimesOutWithoutError(t *testing.T) {
	client := newDynamicClient()

	outcome, err := waitForReconcile(context.Background(), client, "app", "ns", 1, 20*time.Millisecond, time.Millisecond)
	require.NoError(t, err)
	assert.True(t, outcome.TimedOut)
	assert.Nil(t, outcome.Record)
	assert.Contains(t, outcome.Describe(), "readable")
}

func TestWaitForAbsence_ReturnsWhenCRIsGone(t *testing.T) {
	client := newDynamicClient()

	require.NoError(t, waitForAbsence(context.Background(), client, "app", "ns", time.Second, time.Millisecond))
}

func TestWaitForAbsence_TimesOutWhileCRRemains(t *testing.T) {
	client := newDynamicClient(makeInstanceCR(1, 1, nil))

	err := waitForAbsence(context.Background(), client, "app", "ns", 20*time.Millisecond, time.Millisecond)
	require.Error(t, err)
	assert.Contains(t, err.Error(), CleanupFinalizer)
}

// An unstated bound must mean the shared default on every command, not an
// already-expired deadline on whichever one forgot to normalize.
func TestResolveTimeout_FallsBackToTheSharedDefault(t *testing.T) {
	assert.Equal(t, DefaultReconcileTimeout, ResolveTimeout(0))
	assert.Equal(t, DefaultReconcileTimeout, ResolveTimeout(-time.Second))
	assert.Equal(t, 30*time.Second, ResolveTimeout(30*time.Second))
}
