package inventory

import (
	"context"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

	pkginventory "github.com/open-platform-model/cli/pkg/inventory"
)

func TestDiscoverResourcesFromInventory_SortsEveryEntry(t *testing.T) {
	deploy := &unstructured.Unstructured{Object: map[string]any{
		"apiVersion": "apps/v1",
		"kind":       "Deployment",
		"metadata":   map[string]any{"name": "web", "namespace": "apps"},
	}}
	client := newDynamicClient(deploy)
	fake, ok := client.Dynamic.(*dynamicfake.FakeDynamicClient)
	require.True(t, ok)
	fake.PrependReactor("get", "configmaps", func(k8stesting.Action) (bool, runtime.Object, error) {
		return true, nil, apierrors.NewForbidden(schema.GroupResource{Resource: "configmaps"}, "settings", nil)
	})

	entries := []InventoryEntry{
		{Kind: "ConfigMap", Namespace: "apps", Name: "settings", Version: "v1"},
		{Group: "apps", Kind: "Deployment", Namespace: "apps", Name: "web", Version: "v1"},
		{Kind: "Secret", Namespace: "apps", Name: "gone", Version: "v1"},
	}
	rec := &Record{Name: "demo", Namespace: "apps", Inventory: pkginventory.Inventory{Entries: entries}}

	live, missing, unreadable, err := DiscoverResourcesFromInventory(context.Background(), client, rec)
	require.NoError(t, err)

	require.Len(t, live, 1)
	assert.Equal(t, "web", live[0].GetName())
	assert.Equal(t, []InventoryEntry{entries[2]}, missing)
	require.Len(t, unreadable, 1)
	assert.Equal(t, entries[0], unreadable[0].Entry)
	assert.True(t, apierrors.IsForbidden(unreadable[0].Err))
	assert.Equal(t, len(entries), len(live)+len(missing)+len(unreadable), "every entry lands in exactly one group")

	res := unreadable[0].Resource()
	assert.Equal(t, "ConfigMap", res.Kind)
	assert.Equal(t, "apps", res.Namespace)
	assert.Equal(t, "settings", res.Name)
	assert.Equal(t, "", res.Group)
	assert.True(t, apierrors.IsForbidden(res.Err))
}
