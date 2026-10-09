package kubernetes

import (
	"context"
	"errors"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

// Prometheus is served as "prometheuses"; the guessed plural was "prometheus".
var (
	promGVK = schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "Prometheus"}
	promGVR = promGVK.GroupVersion().WithResource("prometheuses")
	fooGVK  = schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"}
)

// irregularCluster is a fake cluster that serves Prometheus as "prometheuses"
// and holds objs under that resource.
func irregularCluster(t *testing.T, objs ...*unstructured.Unstructured) (*Client, *dynamicfake.FakeDynamicClient) {
	t.Helper()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{promGVR: "PrometheusList"})
	for _, obj := range objs {
		require.NoError(t, dyn.Tracker().Create(promGVR, obj.DeepCopy(), obj.GetNamespace()))
	}
	resources := kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{promGVK: {Resource: promGVR.Resource}})
	return &Client{Dynamic: dyn, Resources: resources}, dyn
}

func ownedPrometheus(name string) *unstructured.Unstructured {
	return owned("monitoring.coreos.com/v1", "Prometheus", name, "default", opmlabels.ManagedByCLI, testInstanceUUID)
}

func deletePrometheus(t *testing.T, client *Client, obj *unstructured.Unstructured) *DeleteResult {
	t.Helper()
	result, err := Delete(context.Background(), client, DeleteOptions{
		InstanceName:          "demo",
		Namespace:             "default",
		InstanceUUID:          testInstanceUUID,
		InventoryLive:         []*unstructured.Unstructured{obj.DeepCopy()},
		InventoryRecordExists: true,
	})
	require.NoError(t, err)
	return result
}

// A kind whose plural no rule produces is deleted under the name the cluster
// gives it. With a guessed name the read answered 404 and the object stayed.
func TestDelete_UsesTheResourceTheClusterServes(t *testing.T) {
	obj := ownedPrometheus("main")
	client, dyn := irregularCluster(t, obj)

	result := deletePrometheus(t, client, obj)

	assert.Empty(t, result.Errors)
	assert.Equal(t, 1, result.Deleted)
	_, err := dyn.Tracker().Get(promGVR, "default", "main")
	assert.True(t, apierrors.IsNotFound(err), "the object must be gone from the cluster: %v", err)
}

// A recorded kind the cluster does not serve, and a discovery request that
// fails, are per-resource errors: never "already gone".
func TestDelete_UnresolvedKindIsAnError(t *testing.T) {
	forbidden := discoveryFailure(promGVK, apierrors.NewForbidden(schema.GroupResource{Group: promGVK.Group}, "", errors.New("no discovery")))
	tests := map[string]struct {
		outcome error
		check   func(t *testing.T, err error)
	}{
		"kind not served": {
			outcome: &KindNotServedError{GVK: promGVK},
			check: func(t *testing.T, err error) {
				assert.True(t, IsKindNotServed(err))
				assert.Contains(t, err.Error(), `kind "Prometheus" of monitoring.coreos.com/v1`)
			},
		},
		"discovery forbidden": {
			outcome: forbidden,
			check: func(t *testing.T, err error) {
				assert.True(t, apierrors.IsForbidden(err), "the API error stays in the chain")
			},
		},
	}
	for name, tt := range tests {
		t.Run(name, func(t *testing.T) {
			obj := ownedPrometheus("main")
			client, _ := irregularCluster(t, obj)
			client.Resources = kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{promGVK: {Err: tt.outcome}})

			result := deletePrometheus(t, client, obj)

			assert.Zero(t, result.Deleted)
			require.Len(t, result.Errors, 1, "the resource must not count as already gone")
			assert.Equal(t, "Prometheus", result.Errors[0].Kind)
			tt.check(t, result.Errors[0].Err)
		})
	}
}

func TestApplyOne_UsesTheResourceTheClusterServes(t *testing.T) {
	client, dyn := irregularCluster(t)
	obj := makeUnstructured("monitoring.coreos.com/v1", "Prometheus", "main", "default")

	// The fake tracker has no server-side apply: answer the patch and keep
	// the resource it was sent to.
	var patched []string
	dyn.PrependReactor("patch", "*", func(action k8stesting.Action) (bool, runtime.Object, error) {
		patched = append(patched, action.GetResource().Resource)
		return true, obj.DeepCopy(), nil
	})

	_, err := ApplyOne(context.Background(), client, obj, ApplyOptions{})
	require.NoError(t, err)
	assert.Equal(t, []string{"prometheuses"}, patched)
}

func TestApplyOne_UnservedKindIsNamed(t *testing.T) {
	client, dyn := irregularCluster(t)
	client.Resources = kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{promGVK: {Err: &KindNotServedError{GVK: promGVK}}})
	obj := makeUnstructured("monitoring.coreos.com/v1", "Prometheus", "main", "default")

	_, err := ApplyOne(context.Background(), client, obj, ApplyOptions{DryRun: true})

	require.Error(t, err)
	assert.True(t, IsKindNotServed(err))
	assert.Contains(t, err.Error(), `kind "Prometheus" of monitoring.coreos.com/v1`)
	assert.Empty(t, dyn.Actions(), "nothing is sent for a kind that is not served")
}

// Diff asks whether a rendered object exists. An object the cluster serves
// under an irregular name is found; an object of a kind that is not served
// does not exist; a failed discovery request is a diff error.
func TestDiff_ResolvesRenderedKinds(t *testing.T) {
	rendered := makeUnstructured("monitoring.coreos.com/v1", "Prometheus", "main", "default")

	t.Run("served under an irregular name", func(t *testing.T) {
		client, _ := irregularCluster(t, rendered)
		result, err := Diff(context.Background(), client, []*unstructured.Unstructured{rendered.DeepCopy()}, "demo", NewComparer())
		require.NoError(t, err)
		assert.Empty(t, result.Errors)
		assert.Zero(t, result.Added, "the live object exists, so the rendered one is not new")
		assert.Equal(t, 1, result.Unchanged)
	})

	t.Run("kind not served", func(t *testing.T) {
		client, _ := irregularCluster(t)
		client.Resources = kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{promGVK: {Err: &KindNotServedError{GVK: promGVK}}})
		result, err := Diff(context.Background(), client, []*unstructured.Unstructured{rendered.DeepCopy()}, "demo", NewComparer())
		require.NoError(t, err)
		assert.Empty(t, result.Errors)
		assert.Equal(t, 1, result.Added)
	})

	t.Run("discovery unavailable", func(t *testing.T) {
		client, _ := irregularCluster(t)
		unavailable := discoveryFailure(promGVK, apierrors.NewServiceUnavailable("discovery is down"))
		client.Resources = kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{promGVK: {Err: unavailable}})
		result, err := Diff(context.Background(), client, []*unstructured.Unstructured{rendered.DeepCopy()}, "demo", NewComparer())
		require.NoError(t, err)
		assert.Zero(t, result.Added, "a failed discovery request is not an absent object")
		require.Len(t, result.Errors, 1)
		assert.True(t, apierrors.IsServiceUnavailable(result.Errors[0]))
	})
}

// A wait never reads an unresolved kind as an object that is gone.
func TestWaitAbsent_UnservedKindStopsTheWait(t *testing.T) {
	shortWaitPoll(t)
	client, _ := irregularCluster(t)
	client.Resources = kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{promGVK: {Err: &KindNotServedError{GVK: promGVK}}})
	obj := makeUnstructured("monitoring.coreos.com/v1", "Prometheus", "main", "default")

	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	err := WaitAbsent(ctx, client, []*unstructured.Unstructured{obj}, time.Now())

	require.Error(t, err, "the object must not count as absent")
	assert.True(t, IsKindNotServed(err))
}

// The API server lists a new kind in discovery a moment after its
// CustomResourceDefinition is Established. The apply waits for it.
func TestApply_WaitsUntilDiscoveryServesTheNewKind(t *testing.T) {
	shortWaitPoll(t)
	cluster := &stagingCluster{established: true}
	client := cluster.client(t)
	resources := kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{fooGVK: {Err: &KindNotServedError{GVK: fooGVK}}})
	client.Resources = resources

	ctx, cancel := context.WithCancel(context.Background())
	t.Cleanup(cancel)
	go func() {
		// Serve the kind once the apply has asked for it and been refused.
		for resources.Calls(fooGVK) < 2 {
			select {
			case <-ctx.Done():
				return
			case <-time.After(time.Millisecond):
			}
		}
		resources.Set(fooGVK, kubetest.Outcome{Resource: "foos"})
	}()

	result, err := Apply(context.Background(), client, stagingInput(), "test", ApplyOptions{})
	require.NoError(t, err)
	assert.Empty(t, result.Errors)
	assert.Equal(t, 6, result.Applied)
	assert.Contains(t, cluster.patchOrder(), "Foo/my-foo")
	assert.GreaterOrEqual(t, resources.Calls(fooGVK), 3, "refused twice, then served")
}

func TestApply_DiscoveryNeverServesTheNewKind(t *testing.T) {
	shortWaitPoll(t)
	cluster := &stagingCluster{established: true}
	client := cluster.client(t)
	client.Resources = kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{fooGVK: {Err: &KindNotServedError{GVK: fooGVK}}})

	_, err := Apply(context.Background(), client, stagingInput(), "test", ApplyOptions{
		EstablishDeadline: time.Now().Add(200 * time.Millisecond),
	})

	require.Error(t, err)
	assert.Contains(t, err.Error(), "Foo")
	assert.Equal(t, []string{"CustomResourceDefinition/foos.example.com", "Namespace/demo"}, cluster.patchOrder(),
		"the second stage must not start")
}

func TestApply_DiscoveryFailureStopsTheWait(t *testing.T) {
	shortWaitPoll(t)
	cluster := &stagingCluster{established: true}
	client := cluster.client(t)
	forbidden := discoveryFailure(fooGVK, apierrors.NewForbidden(schema.GroupResource{Group: fooGVK.Group}, "", errors.New("no discovery")))
	client.Resources = kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{fooGVK: {Err: forbidden}})

	_, err := Apply(context.Background(), client, stagingInput(), "test", ApplyOptions{})

	require.Error(t, err)
	assert.True(t, apierrors.IsForbidden(err))
}

// discoveryFailure is the error the resolver returns when the discovery
// request for the group and version of gvk fails with cause.
func discoveryFailure(gvk schema.GroupVersionKind, cause error) error {
	return &DiscoveryError{GroupVersion: gvk.GroupVersion(), Err: cause}
}

// A failed discovery request is not one object's error: the apply stops at
// it, sends nothing more, and returns the API error for the exit code. It is
// the same in a dry run.
func TestApply_DiscoveryFailureStopsTheApply(t *testing.T) {
	serviceGVK := schema.GroupVersionKind{Version: "v1", Kind: "Service"}
	unavailable := discoveryFailure(serviceGVK, apierrors.NewServiceUnavailable("discovery is down"))

	for _, dryRun := range []bool{false, true} {
		cluster := &stagingCluster{}
		client := cluster.client(t)
		client.Resources = kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{serviceGVK: {Err: unavailable}})
		input := []*unstructured.Unstructured{
			stagingObject("v1", "ConfigMap", "cfg", "demo"),
			stagingObject("v1", "Service", "web", "demo"),
			stagingObject("apps/v1", "Deployment", "web", "demo"),
		}

		result, err := Apply(context.Background(), client, input, "test", ApplyOptions{DryRun: dryRun})

		require.Error(t, err, "dryRun=%v", dryRun)
		assert.True(t, IsDiscoveryFailure(err))
		assert.True(t, apierrors.IsServiceUnavailable(err), "the API error stays in the chain")
		assert.Contains(t, err.Error(), "Service/web")
		assert.Empty(t, result.Errors, "it is not a per-resource error")
		assert.Equal(t, []string{"ConfigMap/cfg"}, cluster.patchOrder(), "nothing is sent after the failure")
	}
}

// An object at a version its CustomResourceDefinition does not serve fails at
// once and alone: the apply does not wait for a kind that never comes, and
// the other objects are applied.
func TestApply_VersionTheDefinitionDoesNotServeFailsAtOnce(t *testing.T) {
	shortWaitPoll(t)
	for _, version := range []string{"v1alpha1", "v1beta1"} {
		t.Run(version, func(t *testing.T) {
			gvk := schema.GroupVersionKind{Group: "example.com", Version: version, Kind: "Foo"}
			cluster := &stagingCluster{established: true}
			client := cluster.client(t)
			resources := kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{gvk: {Err: &KindNotServedError{GVK: gvk}}})
			client.Resources = resources
			input := []*unstructured.Unstructured{
				stagingCRD(),
				stagingObject("example.com/"+version, "Foo", "my-foo", "demo"),
				stagingObject("v1", "ConfigMap", "cfg", "demo"),
			}

			start := time.Now()
			result, err := Apply(context.Background(), client, input, "test", ApplyOptions{
				EstablishDeadline: start.Add(30 * time.Second),
			})

			require.NoError(t, err)
			assert.Less(t, time.Since(start), 10*time.Second, "the apply must not wait out the deadline")
			assert.Equal(t, 1, resources.Calls(gvk), "asked once, by the apply of the object itself")
			require.Len(t, result.Errors, 1)
			assert.True(t, IsKindNotServed(result.Errors[0].Err))
			assert.Contains(t, cluster.patchOrder(), "ConfigMap/cfg")
		})
	}
}
