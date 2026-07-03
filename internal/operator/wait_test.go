package operator

import (
	"context"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"

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

	err := wait(context.Background(), client, []*unstructured.Unstructured{notReady}, DefaultPredicate, time.Second, 5*time.Millisecond)
	require.NoError(t, err)
}

func TestWait_TimesOutNamingTheUnreadyObject(t *testing.T) {
	notReady := crdFixture(false)
	client := fakeClientWith(notReady)

	err := wait(context.Background(), client, []*unstructured.Unstructured{notReady}, DefaultPredicate, 20*time.Millisecond, 5*time.Millisecond)
	require.Error(t, err)
	assert.ErrorContains(t, err, "moduleinstances.opmodel.dev")
	assert.ErrorContains(t, err, "timed out")
}

func TestWait_EmptyObjectsReturnsImmediately(t *testing.T) {
	client := fakeClientWith()
	err := wait(context.Background(), client, nil, DefaultPredicate, time.Millisecond, time.Millisecond)
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

	err := wait(ctx, client, []*unstructured.Unstructured{notReady}, DefaultPredicate, 5*time.Second, 5*time.Millisecond)
	require.Error(t, err)
}
