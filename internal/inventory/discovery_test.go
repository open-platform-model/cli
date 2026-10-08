package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/open-platform-model/cli/internal/kubernetes"
	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"
)

// Prometheus is served as "prometheuses"; the guessed plural was "prometheus".
var (
	promGVK   = schema.GroupVersionKind{Group: "monitoring.coreos.com", Version: "v1", Kind: "Prometheus"}
	promGVR   = promGVK.GroupVersion().WithResource("prometheuses")
	promEntry = k8sinventory.Entry{Group: promGVK.Group, Version: "v1", Kind: "Prometheus", Namespace: "default", Name: "main"}
)

// promCluster is a fake cluster that holds objs as Prometheus objects under
// "prometheuses" and resolves the kind as outcome says.
func promCluster(t *testing.T, outcome kubetest.Outcome, objs ...*unstructured.Unstructured) (*kubernetes.Client, *dynamicfake.FakeDynamicClient) {
	t.Helper()
	dyn := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{promGVR: "PrometheusList"})
	for _, obj := range objs {
		require.NoError(t, dyn.Tracker().Create(promGVR, obj.DeepCopy(), obj.GetNamespace()))
	}
	resources := kubetest.ResourcesWith(map[schema.GroupVersionKind]kubetest.Outcome{promGVK: outcome})
	return &kubernetes.Client{Dynamic: dyn, Resources: resources}, dyn
}

var (
	promServed    = kubetest.Outcome{Resource: "prometheuses"}
	promNotServed = kubetest.Outcome{Err: &kubernetes.KindNotServedError{GVK: promGVK}}
	promForbidden = kubetest.Outcome{Err: &kubernetes.DiscoveryError{GroupVersion: promGVK.GroupVersion(),
		Err: apierrors.NewForbidden(schema.GroupResource{Group: promGVK.Group}, "", errors.New("no discovery"))}}
	promDown = kubetest.Outcome{Err: &kubernetes.DiscoveryError{GroupVersion: promGVK.GroupVersion(),
		Err: apierrors.NewServiceUnavailable("discovery is down")}}
)

func promRecord() *Record {
	rec := &Record{}
	rec.Inventory.Entries = []k8sinventory.Entry{promEntry}
	return rec
}

func livePrometheus() *unstructured.Unstructured {
	return liveObject("monitoring.coreos.com/v1", "Prometheus", "default", "main")
}

// A recorded object is read under the resource the cluster serves. With a
// guessed name the read answered 404 and the object counted as missing.
func TestDiscoverResourcesFromInventory_ResolvesByDiscovery(t *testing.T) {
	ctx := context.Background()

	t.Run("served under an irregular name is live", func(t *testing.T) {
		client, _ := promCluster(t, promServed, livePrometheus())
		live, missing, unreadable, err := DiscoverResourcesFromInventory(ctx, client, promRecord())
		require.NoError(t, err)
		assert.Len(t, live, 1)
		assert.Empty(t, missing)
		assert.Empty(t, unreadable)
	})

	t.Run("kind not served is unreadable, not missing", func(t *testing.T) {
		client, _ := promCluster(t, promNotServed, livePrometheus())
		live, missing, unreadable, err := DiscoverResourcesFromInventory(ctx, client, promRecord())
		require.NoError(t, err)
		assert.Empty(t, live)
		assert.Empty(t, missing, "an unserved kind must never read as gone")
		require.Len(t, unreadable, 1)
		assert.Equal(t, promEntry, unreadable[0].Entry)
		assert.Contains(t, unreadable[0].Err.Error(), `kind "Prometheus" of monitoring.coreos.com/v1`)
	})

	t.Run("failed discovery is unreadable with the API error", func(t *testing.T) {
		client, _ := promCluster(t, promForbidden, livePrometheus())
		_, missing, unreadable, err := DiscoverResourcesFromInventory(ctx, client, promRecord())
		require.NoError(t, err)
		assert.Empty(t, missing)
		require.Len(t, unreadable, 1)
		assert.True(t, apierrors.IsForbidden(unreadable[0].Err))
	})
}

// A stale object is deleted under the resource the cluster serves. With a
// guessed name the delete answered 404, the prune reported success, and the
// object stayed in the cluster with no record of it.
func TestPruneStaleResources_ResolvesByDiscovery(t *testing.T) {
	ctx := context.Background()

	t.Run("served under an irregular name is deleted", func(t *testing.T) {
		owned := livePrometheus()
		owned.SetLabels(map[string]string{opmlabels.ManagedBy: opmlabels.ManagedByCLI})
		client, dyn := promCluster(t, promServed, owned)
		_, pruneErr := PruneStaleResources(ctx, client, []k8sinventory.Entry{promEntry}, "")
		require.NoError(t, pruneErr)
		_, err := dyn.Tracker().Get(promGVR, "default", "main")
		assert.True(t, apierrors.IsNotFound(err), "the stale object must be gone: %v", err)
	})

	for name, outcome := range map[string]kubetest.Outcome{"kind not served": promNotServed, "discovery forbidden": promForbidden} {
		t.Run(name+" keeps the entry", func(t *testing.T) {
			client, _ := promCluster(t, outcome, livePrometheus())
			// A ConfigMap beside it is still pruned.
			cm := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "absent"}

			_, err := PruneStaleResources(ctx, client, []k8sinventory.Entry{promEntry, cm}, "")

			var pruneErr *PruneError
			require.ErrorAs(t, err, &pruneErr, "the entry must not count as pruned")
			want := []k8sinventory.Entry{promEntry}
			if outcome == promForbidden {
				// A failed discovery request stops the prune: the untried
				// entry is reported too.
				want = append(want, cm)
			}
			assert.Equal(t, want, pruneErr.Failed)
			assert.Contains(t, err.Error(), "Prometheus/main")
			assert.Equal(t, outcome == promForbidden, apierrors.IsForbidden(err))
			assert.Equal(t, outcome == promNotServed, kubernetes.IsKindNotServed(err))
		})
	}
}

// The first-install check asks whether a rendered object exists. A kind the
// cluster does not serve has no objects; a failed discovery request leaves
// the question open and stops the apply.
func TestFirstInstallCheck_ResolvesByDiscovery(t *testing.T) {
	ctx := context.Background()
	entries := []k8sinventory.Entry{promEntry}

	t.Run("an untracked object under an irregular name is refused", func(t *testing.T) {
		client, _ := promCluster(t, promServed, livePrometheus())
		err := PreApplyExistenceCheck(ctx, client, entries, nil)
		require.Error(t, err, "the object exists and is not managed by OPM")
		assert.Contains(t, err.Error(), "Prometheus/main")
	})

	t.Run("a kind that is not served has no object", func(t *testing.T) {
		client, _ := promCluster(t, promNotServed)
		require.NoError(t, PreApplyExistenceCheck(ctx, client, entries, nil))
	})

	t.Run("a failed discovery request stops the check", func(t *testing.T) {
		client, _ := promCluster(t, promDown)
		err := PreApplyExistenceCheck(ctx, client, entries, nil)
		require.Error(t, err)
		assert.True(t, apierrors.IsServiceUnavailable(err), "the API error stays in the chain")
		assert.Contains(t, err.Error(), "Prometheus/main")
	})
}

// A failed discovery request stops the prune: the entry it hit and every
// entry not yet tried are reported as still in the cluster, and nothing more
// is deleted. An entry of a kind that is not served does not stop it.
func TestPruneStaleResources_DiscoveryFailureStopsThePrune(t *testing.T) {
	ctx := context.Background()
	cm := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "stale"}
	ns := k8sinventory.Entry{Version: "v1", Kind: "Namespace", Name: "apps"}
	liveCM := staleObject("v1", "ConfigMap", "stale")

	// Prune order is highest weight first: the Prometheus before the ConfigMap.
	client, _ := promCluster(t, promForbidden, livePrometheus())
	_, err := client.Dynamic.Resource(kubetest.GVR(liveCM)).Namespace("default").Create(ctx, liveCM, metav1.CreateOptions{})
	require.NoError(t, err)

	_, err = PruneStaleResources(ctx, client, []k8sinventory.Entry{cm, promEntry, ns}, "")

	var pruneErr *PruneError
	require.ErrorAs(t, err, &pruneErr)
	assert.Equal(t, []k8sinventory.Entry{promEntry, cm}, pruneErr.Failed, "the failed entry and the untried one, never the protected one")
	for _, e := range pruneErr.Errs {
		assert.True(t, kubernetes.IsDiscoveryFailure(e))
		assert.True(t, apierrors.IsForbidden(e))
	}
	_, err = client.Dynamic.Resource(kubetest.GVR(liveCM)).Namespace("default").Get(ctx, "stale", metav1.GetOptions{})
	require.NoError(t, err, "nothing is deleted after the failure")
}
