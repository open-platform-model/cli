package inventory

import (
	"context"
	"testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

	k8sinventory "github.com/open-platform-model/library/opm/k8s/inventory"
	opmlabels "github.com/open-platform-model/library/opm/k8s/labels"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	apierrors "k8s.io/apimachinery/pkg/api/errors"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"
	k8stesting "k8s.io/client-go/testing"

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

// staleObject is a live object in "default" the instance still owns: it carries the OPM
// managed-by label, as every object an apply wrote does, so the delete
// verdict lets a prune delete it.
func staleObject(apiVersion, kind, name string) *unstructured.Unstructured {
	obj := liveObject(apiVersion, kind, "default", name)
	obj.SetLabels(map[string]string{opmlabels.ManagedBy: opmlabels.ManagedByCLI})
	return obj
}

func TestSplitProtected(t *testing.T) {
	stale := []k8sinventory.Entry{
		entry("", "ConfigMap", "default", "a"),
		entry("", "Namespace", "", "ns"),
		entry("apps", "Deployment", "default", "web"),
		entry("apiextensions.k8s.io", "CustomResourceDefinition", "", "widgets.example.io"),
		entry("example.io", "Namespace", "default", "not-core"),
	}

	prunable, protected := SplitProtected(stale)

	assert.Equal(t, []k8sinventory.Entry{stale[0], stale[2], stale[4]}, prunable)
	assert.Equal(t, []k8sinventory.Entry{stale[1], stale[3]}, protected)

	p, q := SplitProtected(nil)
	assert.Empty(t, p)
	assert.Empty(t, q)
}

func TestPruneStaleResources_SkipsProtectedKinds(t *testing.T) {
	ctx := context.Background()
	cm := staleObject("v1", "ConfigMap", "stale")
	ns := liveObject("v1", "Namespace", "", "apps")
	crd := liveObject("apiextensions.k8s.io/v1", "CustomResourceDefinition", "", "widgets.example.io")
	client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), cm, ns, crd)}

	stale := []k8sinventory.Entry{
		{Group: "", Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "stale"},
		{Group: "", Version: "v1", Kind: "Namespace", Name: "apps"},
		{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition", Name: "widgets.example.io"},
	}
	_, pruneErr := PruneStaleResources(ctx, client, stale, "")
	require.NoError(t, pruneErr)

	_, err := client.ResourceClient(schema.GroupVersionResource{Version: "v1", Resource: "configmaps"}, "default").Get(ctx, "stale", metav1.GetOptions{})
	assert.True(t, apierrors.IsNotFound(err), "the ConfigMap is pruned")

	_, err = client.ResourceClient(schema.GroupVersionResource{Version: "v1", Resource: "namespaces"}, "").Get(ctx, "apps", metav1.GetOptions{})
	assert.NoError(t, err, "the Namespace stays")

	_, err = client.ResourceClient(schema.GroupVersionResource{Group: "apiextensions.k8s.io", Version: "v1", Resource: "customresourcedefinitions"}, "").Get(ctx, "widgets.example.io", metav1.GetOptions{})
	assert.NoError(t, err, "the CRD stays")
}

// TestPruneStaleResources_DeletesInDescendingWeightOrder pins the direction
// of the stale prune: highest weight first (Deployment, then Service, then
// ConfigMap), equal weights in input order.
func TestPruneStaleResources_DeletesInDescendingWeightOrder(t *testing.T) {
	ctx := context.Background()
	stale := []k8sinventory.Entry{
		{Group: "", Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "cm-1"},
		{Group: "apps", Version: "v1", Kind: "Deployment", Namespace: "default", Name: "deploy"},
		{Group: "", Version: "v1", Kind: "Service", Namespace: "default", Name: "svc"},
		{Group: "", Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "cm-2"},
	}
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(),
		staleObject("v1", "ConfigMap", "cm-1"),
		staleObject("apps/v1", "Deployment", "deploy"),
		staleObject("v1", "Service", "svc"),
		staleObject("v1", "ConfigMap", "cm-2"),
	)
	client := &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: dyn}

	_, err := PruneStaleResources(ctx, client, stale, "")
	require.NoError(t, err)

	var deleted []string
	for _, a := range dyn.Actions() {
		if d, ok := a.(k8stesting.DeleteAction); ok {
			deleted = append(deleted, d.GetName())
		}
	}
	assert.Equal(t, []string{"deploy", "svc", "cm-1", "cm-2"}, deleted)
}
