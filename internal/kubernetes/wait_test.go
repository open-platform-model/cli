package kubernetes

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
	fakedynamic "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"
)

func waitCRDFixture(established bool) *unstructured.Unstructured {
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

func waitDeploymentFixture(ready bool) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata": map[string]any{
			"name":      "opm-operator-controller-manager",
			"namespace": "opm-operator-system",
		},
	}}
	if ready {
		// A finished rollout of the default single replica.
		_ = unstructured.SetNestedMap(obj.Object, map[string]any{
			"observedGeneration": int64(0),
			"replicas":           int64(1),
			"updatedReplicas":    int64(1),
			"availableReplicas":  int64(1),
		}, "status")
	}
	return obj
}

func waitFakeClient(objs ...*unstructured.Unstructured) *Client {
	runtimeObjs := make([]runtime.Object, len(objs))
	for i, o := range objs {
		runtimeObjs[i] = o
	}
	return &Client{Dynamic: fakedynamic.NewSimpleDynamicClient(runtime.NewScheme(), runtimeObjs...)}
}

func TestCRDEstablishedPredicate(t *testing.T) {
	assert.False(t, CRDEstablishedPredicate(waitCRDFixture(false)))
	assert.True(t, CRDEstablishedPredicate(waitCRDFixture(true)))
}

func TestHealthyPredicate(t *testing.T) {
	assert.False(t, HealthyPredicate(waitDeploymentFixture(false)))
	assert.True(t, HealthyPredicate(waitDeploymentFixture(true)))
}

func TestDescribeObjectList_NamespacedAndClusterScoped(t *testing.T) {
	objs := []*unstructured.Unstructured{
		{Object: map[string]any{"kind": "Deployment", "metadata": map[string]any{"name": "mgr", "namespace": "sys"}}},
		{Object: map[string]any{"kind": "CustomResourceDefinition", "metadata": map[string]any{"name": "moduleinstances.opmodel.dev"}}},
	}

	described := DescribeObjectList(objs)
	assert.Equal(t, []string{"Deployment/mgr in sys", "CustomResourceDefinition/moduleinstances.opmodel.dev"}, described)
}

func TestWait_ReturnsNilOnceObjectBecomesReady(t *testing.T) {
	notReady := waitCRDFixture(false)
	client := waitFakeClient(notReady)

	// Flip the CRD to Established=True shortly after the wait starts.
	go func() {
		time.Sleep(20 * time.Millisecond)
		ready := waitCRDFixture(true)
		_, err := client.ResourceClient(GVRFromUnstructured(ready), "").Update(context.Background(), ready, metav1.UpdateOptions{})
		assert.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{notReady}, CRDEstablishedPredicate, modeReady, time.Now(), 5*time.Millisecond)
	require.NoError(t, err)
}

func TestWait_TimesOutNamingTheUnreadyObject(t *testing.T) {
	notReady := waitCRDFixture(false)
	client := waitFakeClient(notReady)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{notReady}, CRDEstablishedPredicate, modeReady, time.Now(), 5*time.Millisecond)
	require.Error(t, err)
	assert.ErrorContains(t, err, "moduleinstances.opmodel.dev")
	assert.ErrorContains(t, err, "timed out")
}

func TestWait_EmptyObjectsReturnsImmediately(t *testing.T) {
	client := waitFakeClient()
	err := Wait(context.Background(), client, nil, HealthyPredicate, time.Now())
	require.NoError(t, err)
}

func TestWait_ContextCancellationStopsWait(t *testing.T) {
	notReady := waitCRDFixture(false)
	client := waitFakeClient(notReady)

	ctx, cancel := context.WithCancel(context.Background())
	go func() {
		time.Sleep(10 * time.Millisecond)
		cancel()
	}()

	err := waitUntil(ctx, client, []*unstructured.Unstructured{notReady}, CRDEstablishedPredicate, modeReady, time.Now(), 5*time.Millisecond)
	require.ErrorIs(t, err, context.Canceled)
	assert.NotContains(t, err.Error(), "timed out", "a caller cancellation is not a timeout")
}

func TestWait_TimeoutReportsElapsedBudget(t *testing.T) {
	notReady := waitCRDFixture(false)
	client := waitFakeClient(notReady)

	// The budget started well before this wait: the message must report
	// the consumed budget, not the remaining slice of it.
	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{notReady}, CRDEstablishedPredicate, modeReady, time.Now().Add(-90*time.Second), 5*time.Millisecond)
	require.Error(t, err)
	assert.ErrorContains(t, err, "timed out after 1m30s")
}

func TestWaitAbsent_ReturnsOnceObjectDisappears(t *testing.T) {
	doomed := waitDeploymentFixture(false)
	client := waitFakeClient(doomed)

	go func() {
		time.Sleep(20 * time.Millisecond)
		err := client.ResourceClient(GVRFromUnstructured(doomed), doomed.GetNamespace()).Delete(context.Background(), doomed.GetName(), metav1.DeleteOptions{})
		assert.NoError(t, err)
	}()

	ctx, cancel := context.WithTimeout(context.Background(), time.Second)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{doomed}, AbsentPredicate, modeAbsent, time.Now(), 5*time.Millisecond)
	require.NoError(t, err)
}

func TestWaitAbsent_TimesOutNamingThePersistingObject(t *testing.T) {
	doomed := waitDeploymentFixture(false)
	client := waitFakeClient(doomed)

	ctx, cancel := context.WithTimeout(context.Background(), 20*time.Millisecond)
	defer cancel()
	err := waitUntil(ctx, client, []*unstructured.Unstructured{doomed}, AbsentPredicate, modeAbsent, time.Now(), 5*time.Millisecond)
	require.Error(t, err)
	assert.ErrorContains(t, err, "timed out")
	assert.ErrorContains(t, err, "finish terminating")
	assert.ErrorContains(t, err, "Deployment/opm-operator-controller-manager in opm-operator-system")
}

func TestWaitAbsent_AlreadyAbsentReturnsImmediately(t *testing.T) {
	client := waitFakeClient()
	err := WaitAbsent(context.Background(), client, []*unstructured.Unstructured{waitDeploymentFixture(false)}, time.Now())
	require.NoError(t, err)
}

func TestWait_ReadinessFailsFastOnNotFound(t *testing.T) {
	// Never created on the cluster: the caller "applied" it and it is gone.
	client := waitFakeClient()

	start := time.Now()
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := Wait(ctx, client, []*unstructured.Unstructured{waitDeploymentFixture(false)}, HealthyPredicate, time.Now())
	require.Error(t, err)
	assert.ErrorContains(t, err, "Deployment/opm-operator-controller-manager in opm-operator-system was applied and has since disappeared")
	assert.NotContains(t, err.Error(), "timed out")
	assert.Less(t, time.Since(start), time.Second, "must not wait out the timeout")
}

func TestWait_OtherGetErrorStaysPendingInBothModes(t *testing.T) {
	obj := waitDeploymentFixture(true)
	client := waitFakeClient(obj)
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
		{"ready", modeReady, HealthyPredicate},
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
