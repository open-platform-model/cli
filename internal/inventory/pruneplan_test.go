package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/runtime"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"
	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
)

// A stale resource counts as pruned when the API server accepts its delete,
// also when it still exists afterwards, as an object held by a finalizer
// does: it is neither a failed entry nor left behind.
func TestPruneStaleResources_AcceptedDeleteOfAnObjectThatStays(t *testing.T) {
	ctx := context.Background()
	live := staleObject("v1", "ConfigMap", "held")
	client := newDynamicClient(live)
	dyn := client.Dynamic.(*dynamicfake.FakeDynamicClient)
	dyn.PrependReactor("delete", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, nil // accepted; the object stays, as under a finalizer
	})

	leftBehind, err := PruneStaleResources(ctx, client, []k8sinventory.Entry{entry("", "ConfigMap", "default", "held")}, "")

	require.NoError(t, err)
	assert.Empty(t, leftBehind)
	_, getErr := client.Dynamic.Resource(kubetest.GVR(live)).Namespace("default").Get(ctx, "held", metav1.GetOptions{})
	assert.NoError(t, getErr, "the object is still there")
}
