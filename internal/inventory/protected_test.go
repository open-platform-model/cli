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
	}, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "already exists and is not managed by OPM")
	assert.Contains(t, err.Error(), "remove or rename it")
	assert.NotContains(t, err.Error(), "--force")
}

// The admission set passes an admitted object's untracked test only: an
// unadmitted untracked object and an admitted terminating one still fail,
// and a nil set keeps the check as it is for every other apply.
func TestPreApplyExistenceCheck_AdmitSet(t *testing.T) {
	kustomized := liveObject("v1", "Namespace", "", "opm-operator-system")
	kustomized.SetLabels(map[string]string{"app.kubernetes.io/managed-by": "kustomize"})
	foreign := liveObject("v1", "ConfigMap", "default", "taken")
	doomed := liveObject("v1", "ServiceAccount", "default", "doomed")
	now := metav1.Now()
	doomed.SetDeletionTimestamp(&now)
	doomed.SetFinalizers([]string{"foregroundDeletion"})
	client := &kubernetes.Client{Dynamic: dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(), kustomized, foreign, doomed)}

	nsEntry := InventoryEntry{Version: "v1", Kind: "Namespace", Name: "opm-operator-system"}
	cmEntry := InventoryEntry{Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "taken"}
	saEntry := InventoryEntry{Version: "v1", Kind: "ServiceAccount", Namespace: "default", Name: "doomed"}
	admit := AdmitSet{
		{Kind: "Namespace", Name: "opm-operator-system"}:               {},
		{Kind: "ServiceAccount", Namespace: "default", Name: "doomed"}: {},
	}
	ctx := context.Background()

	require.NoError(t, PreApplyExistenceCheck(ctx, client, []InventoryEntry{nsEntry}, admit), "an admitted kustomize-labeled object passes")

	err := PreApplyExistenceCheck(ctx, client, []InventoryEntry{nsEntry, cmEntry}, admit)
	require.Error(t, err, "an unadmitted untracked object is refused")
	assert.Contains(t, err.Error(), "ConfigMap/taken")

	err = PreApplyExistenceCheck(ctx, client, []InventoryEntry{saEntry}, admit)
	require.Error(t, err, "an admitted terminating object is refused")
	assert.Contains(t, err.Error(), "is terminating")

	err = PreApplyExistenceCheck(ctx, client, []InventoryEntry{nsEntry}, nil)
	require.Error(t, err, "a nil set admits nothing")
	assert.Contains(t, err.Error(), "Namespace/opm-operator-system")
}

// TestPruneStaleResources_DeletesInDescendingWeightOrder pins the direction
// of the stale prune: highest weight first (Deployment, then Service, then
// ConfigMap), equal weights in input order.
func TestPruneStaleResources_DeletesInDescendingWeightOrder(t *testing.T) {
	ctx := context.Background()
	stale := []InventoryEntry{
		{Group: "", Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "cm-1"},
		{Group: "apps", Version: "v1", Kind: "Deployment", Namespace: "default", Name: "deploy"},
		{Group: "", Version: "v1", Kind: "Service", Namespace: "default", Name: "svc"},
		{Group: "", Version: "v1", Kind: "ConfigMap", Namespace: "default", Name: "cm-2"},
	}
	dyn := dynamicfake.NewSimpleDynamicClient(runtime.NewScheme(),
		liveObject("v1", "ConfigMap", "default", "cm-1"),
		liveObject("apps/v1", "Deployment", "default", "deploy"),
		liveObject("v1", "Service", "default", "svc"),
		liveObject("v1", "ConfigMap", "default", "cm-2"),
	)
	client := &kubernetes.Client{Dynamic: dyn}

	require.NoError(t, PruneStaleResources(ctx, client, stale))

	var deleted []string
	for _, a := range dyn.Actions() {
		if d, ok := a.(k8stesting.DeleteAction); ok {
			deleted = append(deleted, d.GetName())
		}
	}
	assert.Equal(t, []string{"deploy", "svc", "cm-1", "cm-2"}, deleted)
}
