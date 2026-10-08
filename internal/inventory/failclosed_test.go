package inventory

import (
	"context"
	"errors"
	"testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"

	"github.com/open-platform-model/cli/internal/kubernetes"
)

// The first-install check refuses an object it cannot read, admitted or not;
// only a NotFound answer proves the name is free.
func TestPreApplyExistenceCheck_UnreadableObjectRefuses(t *testing.T) {
	ctx := context.Background()
	entry := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "taken"}
	forbidden := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "taken", errors.New("no access"))

	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())
	dyn.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, forbidden
	})
	client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn}

	err := PreApplyExistenceCheck(ctx, client, []k8sinventory.Entry{entry}, nil)
	require.Error(t, err, "an unreadable object is refused")
	assert.Contains(t, err.Error(), "ConfigMap/taken")
	assert.Contains(t, err.Error(), `"default"`)
	assert.True(t, apierrors.IsForbidden(err), "the read error stays in the chain")

	admit := AdmitSet{{Kind: "ConfigMap", Namespace: "default", Name: "taken"}: {}}
	require.Error(t, PreApplyExistenceCheck(ctx, client, []k8sinventory.Entry{entry}, admit), "admission does not pass an unreadable object")

	absent := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme())}
	require.NoError(t, PreApplyExistenceCheck(ctx, absent, []k8sinventory.Entry{entry}, nil), "an absent object passes")
}

// A prune goes on after a failed delete and reports exactly the entries that
// are still in the cluster, each with its delete error; a NotFound answer is
// not a failure.
func TestPruneStaleResources_ReportsTheEntriesItCouldNotDelete(t *testing.T) {
	ctx := context.Background()
	denied := apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "stuck", errors.New("no delete access"))
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(),
		liveObject("v1", "ConfigMap", "default", "stuck"),
		liveObject("v1", "ConfigMap", "default", "gone"),
	)
	dyn.PrependReactor("delete", "configmaps", func(action k8stesting.Action) (bool, runtime.Object, error) {
		if action.(k8stesting.DeleteAction).GetName() == "stuck" {
			return true, nil, denied
		}
		return false, nil, nil
	})
	client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn}

	stuck := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "stuck"}
	gone := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "gone"}
	absent := k8sinventory.Entry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "absent"}

	err := PruneStaleResources(ctx, client, []k8sinventory.Entry{stuck, gone, absent})

	var pruneErr *PruneError
	require.ErrorAs(t, err, &pruneErr)
	assert.Equal(t, []k8sinventory.Entry{stuck}, pruneErr.Failed, "only the entry whose delete failed is reported")
	require.Len(t, pruneErr.Errs, 1)
	assert.Contains(t, pruneErr.Errs[0].Error(), "ConfigMap/stuck")
	assert.True(t, apierrors.IsForbidden(err), "the delete error stays in the chain")

	_, getErr := client.ResourceClient(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "default").Get(ctx, "gone", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(getErr), "the delete after the failed one still ran")
}
