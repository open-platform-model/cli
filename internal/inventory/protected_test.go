package inventory

import (
	"context"
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
)

func liveObject(apiVersion, kind, ns, name string) *unstructured.Unstructured {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion(apiVersion)
	obj.SetKind(kind)
	obj.SetName(name)
	if ns != "" {
		obj.SetNamespace(ns)
	}
	return obj
}

func TestSplitProtected(t *testing.T) {
	stale := []InventoryEntry{
		entry("", "ConfigMap", "default", "a", "app"),
		entry("", "Namespace", "", "ns", "app"),
		entry("apps", "Deployment", "default", "web", "app"),
		entry("apiextensions.k8s.io", "CustomResourceDefinition", "", "widgets.example.io", "app"),
		entry("example.io", "Namespace", "default", "not-core", "app"),
	}

	prunable, protected := SplitProtected(stale)

	assert.Equal(t, []InventoryEntry{stale[0], stale[2], stale[4]}, prunable)
	assert.Equal(t, []InventoryEntry{stale[1], stale[3]}, protected)

	p, q := SplitProtected(nil)
	assert.Empty(t, p)
	assert.Empty(t, q)
}

func TestPruneStaleResources_SkipsProtectedKinds(t *testing.T) {
	ctx := context.Background()
	cm := liveObject("v1", "ConfigMap", "default", "stale")
	ns := liveObject("v1", "Namespace", "", "apps")
	crd := liveObject("apiextensions.k8s.io/v1", "CustomResourceDefinition", "", "widgets.example.io")
	client := &kubernetes.Client{Dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), cm, ns, crd)}

	stale := []InventoryEntry{
		{Group: "", Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "stale"},
		{Group: "", Version: "v1", Kind: "Namespace", Name: "apps"},
		{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition", Name: "widgets.example.io"},
	}
	require.NoError(t, PruneStaleResources(ctx, client, stale))

	_, err := client.ResourceClient(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "default").Get(ctx, "stale", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "the ConfigMap is pruned")

	_, err = client.ResourceClient(schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}, "").Get(ctx, "apps", metav1.GetOptions{})
	assert.NoError(t, err, "the Namespace stays")

	_, err = client.ResourceClient(schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}, "").Get(ctx, "widgets.example.io", metav1.GetOptions{})
	assert.NoError(t, err, "the CRD stays")
}

// The first-install refusal names no bypass flag, since none bypasses it.
func TestPreApplyExistenceCheck_UntrackedNamesNoFlag(t *testing.T) {
	cm := liveObject("v1", "ConfigMap", "default", "taken")
	client := &kubernetes.Client{Dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), cm)}

	err := PreApplyExistenceCheck(context.Background(), client, []InventoryEntry{
		{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "taken"},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists and is not managed by OPM")
	assert.Contains(t, err.Error(), "remove or rename it")
	assert.NotContains(t, err.Error(), "--force")
}
