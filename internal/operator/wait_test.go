package operator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

func crdFixture(established bool) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apiextensions.k8s.io/v1",
		"kind":       "CustomResourceDefinition",
		"metadata": map[string]any{
			"name": "moduleinstances.opmodel.dev",
		},
	}}
	if established {
		_ = unstructured.SetNestedSlice(obj.Object, []any{
			map[string]any{"type": "Established", "status": "True"},
		}, "status", "conditions")
	}
	return obj
}

func deploymentFixture(ready bool) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name":      "opm-operator-controller-manager",
			"namespace": "opm-operator-system",
		},
	}}
	if ready {
		_ = unstructured.SetNestedSlice(obj.Object, []any{
			map[string]any{"type": "Available", "status": "True"},
		}, "status", "conditions")
	}
	return obj
}

func TestCRDEstablishedPredicate(t *testing.T) {
	assert.False(t, CRDEstablishedPredicate(crdFixture(false)))
	assert.True(t, CRDEstablishedPredicate(crdFixture(true)))
}

func TestWorkloadReadyPredicate(t *testing.T) {
	assert.False(t, WorkloadReadyPredicate(deploymentFixture(false)))
	assert.True(t, WorkloadReadyPredicate(deploymentFixture(true)))
}

func TestDefaultPredicate_DispatchesByKind(t *testing.T) {
	assert.True(t, DefaultPredicate(crdFixture(true)))
	assert.False(t, DefaultPredicate(crdFixture(false)))
	assert.True(t, DefaultPredicate(deploymentFixture(true)))
	assert.False(t, DefaultPredicate(deploymentFixture(false)))

	// Passive kinds are ready as soon as they exist.
	svc := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "v1",
		"kind":       "Service",
		"metadata":   map[string]any{"name": "opm-operator-metrics"},
	}}
	assert.True(t, DefaultPredicate(svc))
}

func fakeClientWith(objs ...*unstructured.Unstructured) *kubernetes.Client {
	scheme := runtime.NewScheme()
	runtimeObjs := make([]runtime.Object, len(objs))
	for i, o := range objs {
		runtimeObjs[i] = o
	}
	// Custom resources (e.g. ModuleInstance) need an explicit GVR->ListKind
	// mapping: the fake tracker only infers one from pre-seeded objects, so
	// an empty-fixture test (no objects of that kind) would otherwise panic
	// on List with "you must register resource to list kind".
	listKinds := map[schema.GroupVersionResource]string{
		moduleInstanceGVR: "ModuleInstanceList",
	}
	return &kubernetes.Client{Dynamic: fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, runtimeObjs...)}
}

func TestWait_ReturnsNilOnceObjectBecomesReady(t *testing.T) {
	notReady := crdFixture(false)
	client := fakeClientWith(notReady)

	// Flip the CRD to Established=True shortly after the wait starts.
	go func() {
		time.Sleep(20 * time.Millisecond)
		ready := crdFixture(true)
		_, err := client.ResourceClient(kubernetes.GVRFromUnstructured(ready), "").Update(context.Background(), ready, metav1.UpdateOptions{})
		assert.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{notReady}, DefaultPredicate, modeReady, time.Now(), 5*time.Millisecond)
	require.NoError(t, err)
}

func TestWait_TimesOutNamingTheUnreadyObject(t *testing.T) {
	notReady := crdFixture(false)
	client := fakeClientWith(notReady)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{notReady}, DefaultPredicate, modeReady, time.Now(), 5*time.Millisecond)
	require.Error(t, err)
	assert.ErrorContains(t, err, "moduleinstances.opmodel.dev")
	assert.ErrorContains(t, err, "timed out")
}

func TestWait_EmptyObjectsReturnsImmediately(t *testing.T) {
	client := fakeClientWith()
	err := Wait(context.Background(), client, nil, DefaultPredicate, time.Now())
	require.NoError(t, err)
}

func TestWait_ContextCancellationStopsWait(t *testing.T) {
	notReady := crdFixture(false)
	client := fakeClientWith(notReady)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := waitUntil(ctx, client, []*unstructured.Unstructured{notReady}, DefaultPredicate, modeReady, time.Now(), 5*time.Millisecond)
	require.ErrorIs(t, err, context.Canceled)
	assert.NotContains(t, err.Error(), "timed out", "a caller cancellation is not a timeout")
}

func TestWait_TimeoutReportsElapsedBudget(t *testing.T) {
	notReady := crdFixture(false)
	client := fakeClientWith(notReady)

	// The budget started well before this wait: the message must report
	// the consumed budget, not the remaining slice of it.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{notReady}, DefaultPredicate, modeReady, time.Now().Add(-90*time.Second), 5*time.Millisecond)
	require.Error(t, err)
	assert.ErrorContains(t, err, "timed out after 1m30s")
}

func TestWaitAbsent_ReturnsOnceObjectDisappears(t *testing.T) {
	doomed := deploymentFixture(false)
	client := fakeClientWith(doomed)

	go func() {
		time.Sleep(20 * time.Millisecond)
		err := client.ResourceClient(kubernetes.GVRFromUnstructured(doomed), doomed.GetNamespace()).Delete(context.Background(), doomed.GetName(), metav1.DeleteOptions{})
		assert.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{doomed}, AbsentPredicate, modeAbsent, time.Now(), 5*time.Millisecond)
	require.NoError(t, err)
}

func TestWaitAbsent_TimesOutNamingThePersistingObject(t *testing.T) {
	doomed := deploymentFixture(false)
	client := fakeClientWith(doomed)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{doomed}, AbsentPredicate, modeAbsent, time.Now(), 5*time.Millisecond)
	require.Error(t, err)
	assert.ErrorContains(t, err, "timed out")
	assert.ErrorContains(t, err, "finish terminating")
	assert.ErrorContains(t, err, "Deployment/opm-operator-controller-manager in opm-operator-system")
}

func TestWaitAbsent_AlreadyAbsentReturnsImmediately(t *testing.T) {
	client := fakeClientWith()
	err := WaitAbsent(context.Background(), client, []*unstructured.Unstructured{deploymentFixture(false)}, time.Now())
	require.NoError(t, err)
}

func TestWait_ReadinessFailsFastOnNotFound(t *testing.T) {
	// Never created on the cluster: the caller "applied" it and it is gone.
	client := fakeClientWith()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := Wait(ctx, client, []*unstructured.Unstructured{deploymentFixture(false)}, DefaultPredicate, time.Now())
	require.Error(t, err)
	assert.ErrorContains(t, err, "Deployment/opm-operator-controller-manager in opm-operator-system was applied and has since disappeared")
	assert.NotContains(t, err.Error(), "timed out")
	assert.Less(t, time.Since(start), time.Second, "must not wait out the timeout")
}

func TestWait_OtherGetErrorStaysPendingInBothModes(t *testing.T) {
	obj := deploymentFixture(true)
	client := fakeClientWith(obj)
	fake, ok := client.Dynamic.(*fakedynamic.FakeDynamicClient)
	require.True(t, ok)
	fake.PrependReactor("get", "deployments", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewServiceUnavailable("apiserver is restarting")
	})

	for _, tc := range []struct {
		name      string
		mode      waitMode
		predicate ReadyPredicate
	}{
		{"ready", modeReady, DefaultPredicate},
		{"absent", modeAbsent, AbsentPredicate},
	} {
		t.Run(tc.name, func(t *testing.T) {
			ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
			defer cancel()
			err := waitUntil(ctx, client, []*unstructured.Unstructured{obj}, tc.predicate, tc.mode, time.Now(), 5*time.Millisecond)
			require.Error(t, err)
			assert.ErrorContains(t, err, "timed out", "a transient Get error must keep polling, not fail fast")
		})
	}
}
