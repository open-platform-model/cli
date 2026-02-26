package kubernetes

import (
	"context"
	"strings"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	appsv1 "k8s.io/api/apps/v1"
	corev1 "k8s.io/api/core/v1"
	metav1 "k8s.io/apimachinery/pkg/apis/meta/v1"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/types"
	"k8s.io/client-go/kubernetes/fake"
)

// ─────────────────────────────────────────────────────────────────────────────
// Helpers
// ─────────────────────────────────────────────────────────────────────────────

func makeRes(kind, ns, name string) *unstructured.Unstructured {
	u := &unstructured.Unstructured{}
	u.SetKind(kind)
	u.SetAPIVersion("v1")
	u.SetNamespace(ns)
	u.SetName(name)
	u.SetUID(types.UID("uid-" + name))
	return u
}

func makeTreeClient(objects ...runtime.Object) *Client {
	return &Client{Clientset: fake.NewSimpleClientset(objects...)} //nolint:staticcheck // fake.NewClientset requires generated apply configs
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 2: Component grouping
// ─────────────────────────────────────────────────────────────────────────────

func TestGroupByComponent_NormalMapping(t *testing.T) {
	res := makeRes("Deployment", "default", "web")
	componentMap := map[string]string{
		"Deployment/default/web": "server",
	}

	groups := groupByComponent([]*unstructured.Unstructured{res}, componentMap)
	assert.Len(t, groups, 1)
	assert.Contains(t, groups, "server")
	assert.Len(t, groups["server"], 1)
}

func TestGroupByComponent_MissingKey_GoesToNoComponent(t *testing.T) {
	res := makeRes("ConfigMap", "default", "cfg")
	groups := groupByComponent([]*unstructured.Unstructured{res}, map[string]string{})
	assert.Contains(t, groups, noComponentLabel)
	assert.Len(t, groups[noComponentLabel], 1)
}

func TestGroupByComponent_EmptyStringValue_GoesToNoComponent(t *testing.T) {
	res := makeRes("Service", "default", "svc")
	componentMap := map[string]string{
		"Service/default/svc": "",
	}
	groups := groupByComponent([]*unstructured.Unstructured{res}, componentMap)
	assert.Contains(t, groups, noComponentLabel)
}

func TestGroupByComponent_MultipleComponents(t *testing.T) {
	r1 := makeRes("Deployment", "ns", "web")
	r2 := makeRes("StatefulSet", "ns", "db")
	r3 := makeRes("ConfigMap", "ns", "cfg")
	componentMap := map[string]string{
		"Deployment/ns/web": "server",
		"StatefulSet/ns/db": "database",
		"ConfigMap/ns/cfg":  "server",
	}

	groups := groupByComponent([]*unstructured.Unstructured{r1, r2, r3}, componentMap)
	assert.Len(t, groups["server"], 2)
	assert.Len(t, groups["database"], 1)
}

func TestSortedComponentNames_AlphabeticalWithNoComponentLast(t *testing.T) {
	groups := map[string][]*unstructured.Unstructured{
		"zebra":          {},
		noComponentLabel: {},
		"alpha":          {},
		"middle":         {},
	}
	names := sortedComponentNames(groups)
	assert.Equal(t, []string{"alpha", "middle", "zebra", noComponentLabel}, names)
}

func TestSortedComponentNames_NoNoComponent(t *testing.T) {
	groups := map[string][]*unstructured.Unstructured{
		"server":   {},
		"database": {},
	}
	names := sortedComponentNames(groups)
	assert.Equal(t, []string{"database", "server"}, names)
}

func TestSortedComponentNames_OnlyNoComponent(t *testing.T) {
	groups := map[string][]*unstructured.Unstructured{
		noComponentLabel: {},
	}
	names := sortedComponentNames(groups)
	assert.Equal(t, []string{noComponentLabel}, names)
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 4: Replica count extraction
// ─────────────────────────────────────────────────────────────────────────────

func TestGetReplicaCount_Deployment(t *testing.T) {
	res := makeRes("Deployment", "ns", "web")
	_ = unstructured.SetNestedField(res.Object, int64(3), "spec", "replicas")
	_ = unstructured.SetNestedField(res.Object, int64(2), "status", "readyReplicas")
	assert.Equal(t, "2/3", getReplicaCount(res))
}

func TestGetReplicaCount_Deployment_DefaultReplicas(t *testing.T) {
	// spec.replicas omitted — Kubernetes defaults to 1
	res := makeRes("Deployment", "ns", "web")
	_ = unstructured.SetNestedField(res.Object, int64(1), "status", "readyReplicas")
	assert.Equal(t, "1/1", getReplicaCount(res))
}

func TestGetReplicaCount_StatefulSet(t *testing.T) {
	res := makeRes("StatefulSet", "ns", "db")
	_ = unstructured.SetNestedField(res.Object, int64(2), "spec", "replicas")
	_ = unstructured.SetNestedField(res.Object, int64(1), "status", "readyReplicas")
	assert.Equal(t, "1/2", getReplicaCount(res))
}

func TestGetReplicaCount_DaemonSet(t *testing.T) {
	res := makeRes("DaemonSet", "ns", "agent")
	_ = unstructured.SetNestedField(res.Object, int64(3), "status", "desiredNumberScheduled")
	_ = unstructured.SetNestedField(res.Object, int64(3), "status", "numberReady")
	assert.Equal(t, "3/3", getReplicaCount(res))
}

func TestGetReplicaCount_Job(t *testing.T) {
	res := makeRes("Job", "ns", "migrate")
	_ = unstructured.SetNestedField(res.Object, int64(1), "spec", "completions")
	_ = unstructured.SetNestedField(res.Object, int64(1), "status", "succeeded")
	assert.Equal(t, "1/1", getReplicaCount(res))
}

func TestGetReplicaCount_PassiveResource(t *testing.T) {
	for _, kind := range []string{"Service", "ConfigMap", "Secret", "PersistentVolumeClaim", "Ingress"} {
		res := makeRes(kind, "ns", "r")
		assert.Empty(t, getReplicaCount(res), "kind %s should have empty replica count", kind)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 3: Ownership walking
// ─────────────────────────────────────────────────────────────────────────────

func TestWalkDeployment_ReturnsRSAndPods(t *testing.T) {
	ctx := context.Background()

	deployUID := types.UID("deploy-uid")
	rsUID := types.UID("rs-uid")

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-rs-abc",
			Namespace: "default",
			UID:       rsUID,
			OwnerReferences: []metav1.OwnerReference{
				{UID: deployUID},
			},
		},
		Status: appsv1.ReplicaSetStatus{Replicas: 2, ReadyReplicas: 2},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-rs-abc-x1",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{UID: rsUID},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}

	client := makeTreeClient(rs, pod)
	deploy := makeRes("Deployment", "default", "web")
	deploy.SetUID(deployUID)

	children := walkDeployment(ctx, client, deploy)
	require.Len(t, children, 1)
	assert.Equal(t, "ReplicaSet", children[0].Kind)
	assert.Equal(t, "web-rs-abc", children[0].Name)
	assert.Equal(t, "2 pods", children[0].Replicas)
	assert.Equal(t, HealthReady, children[0].Status)

	require.Len(t, children[0].Children, 1)
	assert.Equal(t, "Pod", children[0].Children[0].Kind)
	assert.Equal(t, "web-rs-abc-x1", children[0].Children[0].Name)
	// Pod status is raw K8s phase ("Running"), not a HealthStatus enum value.
	assert.Equal(t, HealthStatus("Running"), children[0].Children[0].Status)
	assert.True(t, children[0].Children[0].Ready, "pod with Ready condition=True should have Ready=true")
}

func TestWalkDeployment_OldRSWithZeroPods(t *testing.T) {
	ctx := context.Background()
	deployUID := types.UID("deploy-uid")

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "web-rs-old",
			Namespace: "default",
			UID:       "rs-old",
			OwnerReferences: []metav1.OwnerReference{
				{UID: deployUID},
			},
		},
		Status: appsv1.ReplicaSetStatus{Replicas: 0},
	}

	client := makeTreeClient(rs)
	deploy := makeRes("Deployment", "default", "web")
	deploy.SetUID(deployUID)

	children := walkDeployment(ctx, client, deploy)
	require.Len(t, children, 1)
	assert.Equal(t, HealthReady, children[0].Status, "RS with 0 replicas should be Ready")
	assert.Equal(t, "0 pods", children[0].Replicas)
}

func TestWalkStatefulSet_ReturnsPods(t *testing.T) {
	ctx := context.Background()
	ssUID := types.UID("ss-uid")

	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "db-0",
			Namespace: "default",
			OwnerReferences: []metav1.OwnerReference{
				{UID: ssUID},
			},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}

	client := makeTreeClient(pod)
	ss := makeRes("StatefulSet", "default", "db")
	ss.SetUID(ssUID)

	children := walkStatefulSet(ctx, client, ss)
	require.Len(t, children, 1)
	assert.Equal(t, "Pod", children[0].Kind)
	assert.Equal(t, "db-0", children[0].Name)
	// Pod status is raw K8s phase, not HealthStatus enum.
	assert.Equal(t, HealthStatus("Running"), children[0].Status)
	assert.True(t, children[0].Ready)
	assert.Empty(t, children[0].Children, "StatefulSet pods should have no RS layer")
}

func TestWalkOwnership_IgnoresUnownedPods(t *testing.T) {
	ctx := context.Background()
	deployUID := types.UID("deploy-uid")

	// Pod NOT owned by this deployment
	unownedPod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-pod",
			Namespace: "default",
		},
	}
	// RS NOT owned by this deployment
	unownedRS := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:      "other-rs",
			Namespace: "default",
		},
	}

	client := makeTreeClient(unownedPod, unownedRS)
	deploy := makeRes("Deployment", "default", "web")
	deploy.SetUID(deployUID)

	children := walkDeployment(ctx, client, deploy)
	assert.Empty(t, children)
}

func TestWalkOwnership_PassiveResourceReturnsNil(t *testing.T) {
	ctx := context.Background()
	client := makeTreeClient()

	for _, kind := range []string{"Service", "ConfigMap", "Ingress", "PersistentVolumeClaim"} {
		res := makeRes(kind, "ns", "r")
		children := walkOwnership(ctx, client, res)
		assert.Nil(t, children, "passive resource %s should have no children", kind)
	}
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 5: Tree building
// ─────────────────────────────────────────────────────────────────────────────

func makeReadyDeployment(ns, name string) *unstructured.Unstructured {
	res := makeRes("Deployment", ns, name)
	_ = unstructured.SetNestedField(res.Object, int64(1), "spec", "replicas")
	_ = unstructured.SetNestedField(res.Object, int64(1), "status", "readyReplicas")
	// Deployment Available condition = true
	_ = unstructured.SetNestedSlice(res.Object, []interface{}{
		map[string]interface{}{"type": "Available", "status": "True"},
	}, "status", "conditions")
	return res
}

func TestBuildTree_Depth0_ComponentSummaryOnly(t *testing.T) {
	ctx := context.Background()
	client := makeTreeClient()

	r1 := makeReadyDeployment("ns", "web")
	r2 := makeRes("ConfigMap", "ns", "cfg")
	_ = unstructured.SetNestedField(r2.Object, nil, "status") // passive

	opts := TreeOptions{
		ReleaseInfo:   ReleaseInfo{Name: "my-app", Namespace: "ns"},
		InventoryLive: []*unstructured.Unstructured{r1, r2},
		ComponentMap: map[string]string{
			"Deployment/ns/web": "server",
			"ConfigMap/ns/cfg":  "server",
		},
		Depth: 0,
	}

	result := BuildTree(ctx, client, opts)
	require.Len(t, result.Components, 1)
	comp := result.Components[0]
	assert.Equal(t, "server", comp.Name)
	assert.Equal(t, 2, comp.ResourceCount)
	assert.Empty(t, comp.Resources, "depth=0 should not populate Resources")
}

func TestBuildTree_Depth1_ResourcesNoChildren(t *testing.T) {
	ctx := context.Background()
	client := makeTreeClient() // no K8s objects — walkOwnership should not be called

	res := makeReadyDeployment("ns", "web")
	opts := TreeOptions{
		ReleaseInfo:   ReleaseInfo{Name: "my-app", Namespace: "ns"},
		InventoryLive: []*unstructured.Unstructured{res},
		ComponentMap:  map[string]string{"Deployment/ns/web": "server"},
		Depth:         1,
	}

	result := BuildTree(ctx, client, opts)
	require.Len(t, result.Components, 1)
	require.Len(t, result.Components[0].Resources, 1)
	node := result.Components[0].Resources[0]
	assert.Equal(t, "Deployment", node.Kind)
	assert.Equal(t, "web", node.Name)
	assert.Equal(t, "1/1", node.Replicas)
	assert.Empty(t, node.Children, "depth=1 should not walk ownership")
}

func TestBuildTree_Depth2_FullTree(t *testing.T) {
	ctx := context.Background()

	deployUID := types.UID("deploy-uid-web")
	rsUID := types.UID("rs-uid")

	rs := &appsv1.ReplicaSet{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "web-rs",
			Namespace:       "ns",
			UID:             rsUID,
			OwnerReferences: []metav1.OwnerReference{{UID: deployUID}},
		},
		Status: appsv1.ReplicaSetStatus{Replicas: 1, ReadyReplicas: 1},
	}
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{
			Name:            "web-rs-x1",
			Namespace:       "ns",
			OwnerReferences: []metav1.OwnerReference{{UID: rsUID}},
		},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}
	client := makeTreeClient(rs, pod)

	deploy := makeReadyDeployment("ns", "web")
	deploy.SetUID(deployUID)

	opts := TreeOptions{
		ReleaseInfo:   ReleaseInfo{Name: "my-app", Namespace: "ns"},
		InventoryLive: []*unstructured.Unstructured{deploy},
		ComponentMap:  map[string]string{"Deployment/ns/web": "server"},
		Depth:         2,
	}

	result := BuildTree(ctx, client, opts)
	require.Len(t, result.Components, 1)
	require.Len(t, result.Components[0].Resources, 1)
	deployNode := result.Components[0].Resources[0]
	require.Len(t, deployNode.Children, 1, "should have ReplicaSet child")
	rsNode := deployNode.Children[0]
	assert.Equal(t, "ReplicaSet", rsNode.Kind)
	require.Len(t, rsNode.Children, 1, "should have Pod grandchild")
	assert.Equal(t, "Pod", rsNode.Children[0].Kind)
}

func TestBuildTree_ComponentSorting(t *testing.T) {
	ctx := context.Background()
	client := makeTreeClient()

	r1 := makeRes("ConfigMap", "ns", "c1")
	r2 := makeRes("ConfigMap", "ns", "c2")
	r3 := makeRes("ConfigMap", "ns", "c3")

	opts := TreeOptions{
		ReleaseInfo:   ReleaseInfo{Name: "app", Namespace: "ns"},
		InventoryLive: []*unstructured.Unstructured{r1, r2, r3},
		ComponentMap: map[string]string{
			"ConfigMap/ns/c1": "zebra",
			"ConfigMap/ns/c2": "alpha",
			"ConfigMap/ns/c3": noComponentLabel,
		},
		Depth: 1,
	}

	result := BuildTree(ctx, client, opts)
	require.Len(t, result.Components, 3)
	assert.Equal(t, "alpha", result.Components[0].Name)
	assert.Equal(t, "zebra", result.Components[1].Name)
	assert.Equal(t, noComponentLabel, result.Components[2].Name)
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 9: GetModuleTree entry point
// ─────────────────────────────────────────────────────────────────────────────

func TestGetModuleTree_EmptyInventory_ReturnsError(t *testing.T) {
	ctx := context.Background()
	client := makeTreeClient()

	opts := TreeOptions{
		ReleaseInfo:   ReleaseInfo{Name: "my-app", Namespace: "default"},
		InventoryLive: nil,
	}

	_, err := GetModuleTree(ctx, client, opts)
	require.Error(t, err)
	assert.True(t, IsNoResourcesFound(err))
	assert.Contains(t, err.Error(), "my-app")
}

func TestGetModuleTree_WithResources_ReturnsTree(t *testing.T) {
	ctx := context.Background()
	client := makeTreeClient()

	res := makeRes("ConfigMap", "ns", "cfg")
	opts := TreeOptions{
		ReleaseInfo:   ReleaseInfo{Name: "my-app", Namespace: "ns", Module: "example.com/app", Version: "1.0.0"},
		InventoryLive: []*unstructured.Unstructured{res},
		ComponentMap:  map[string]string{"ConfigMap/ns/cfg": "core"},
		Depth:         1,
	}

	result, err := GetModuleTree(ctx, client, opts)
	require.NoError(t, err)
	assert.Equal(t, "my-app", result.Release.Name)
	assert.Equal(t, "example.com/app", result.Release.Module)
	assert.Equal(t, "1.0.0", result.Release.Version)
	assert.Len(t, result.Components, 1)
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 6/7: Tree rendering
// ─────────────────────────────────────────────────────────────────────────────

func makeSimpleResult() *TreeResult {
	return &TreeResult{
		Release: ReleaseInfo{Name: "my-app", Namespace: "ns", Module: "example.com/app", Version: "1.0.0"},
		Components: []Component{
			{
				Name:          "server",
				ResourceCount: 2,
				Status:        HealthReady,
				Resources: []ResourceNode{
					{Kind: "Deployment", Name: "web", Namespace: "ns", Status: HealthReady, Replicas: "3/3"},
					{Kind: "Service", Name: "web-svc", Namespace: "ns", Status: HealthReady},
				},
			},
			{
				Name:          "database",
				ResourceCount: 1,
				Status:        HealthReady,
				Resources: []ResourceNode{
					{Kind: "StatefulSet", Name: "db", Namespace: "ns", Status: HealthReady, Replicas: "1/1"},
				},
			},
		},
	}
}

func TestFormatTreeTable_Colored_ContainsKeyElements(t *testing.T) {
	result := makeSimpleResult()
	out := formatTreeTable(result, true)

	assert.Contains(t, out, "my-app")
	assert.Contains(t, out, "example.com/app@1.0.0")
	assert.Contains(t, out, "server")
	assert.Contains(t, out, "database")
	assert.Contains(t, out, "Deployment/web")
	assert.Contains(t, out, "Service/web-svc")
	assert.Contains(t, out, "StatefulSet/db")
	assert.Contains(t, out, "3/3")
	assert.Contains(t, out, "1/1")
	// Tree chrome
	assert.Contains(t, out, "├──")
	assert.Contains(t, out, "└──")
	assert.Contains(t, out, "│")
}

func TestFormatPlainTree_NoANSICodes(t *testing.T) {
	result := makeSimpleResult()
	out := formatPlainTree(result)

	// No ANSI escape sequences
	assert.NotContains(t, out, "\x1b[")
	// Structure is preserved
	assert.Contains(t, out, "my-app")
	assert.Contains(t, out, "server")
	assert.Contains(t, out, "Deployment/web")
	assert.Contains(t, out, "├──")
	assert.Contains(t, out, "└──")
}

func TestFormatTreeTable_Depth0_SummaryLines(t *testing.T) {
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{Name: "server", ResourceCount: 3, Status: HealthReady},
			{Name: "database", ResourceCount: 1, Status: HealthNotReady},
		},
	}
	out := formatPlainTree(result)

	assert.Contains(t, out, "server")
	assert.Contains(t, out, "3 resources")
	assert.Contains(t, out, "database")
	assert.Contains(t, out, "1 resource")
	// No "│" separator — depth 0 has no sub-items
	assert.NotContains(t, out, "│\n")
}

func TestFormatTreeTable_NoModule_HeaderSimple(t *testing.T) {
	result := &TreeResult{
		Release:    ReleaseInfo{Name: "local-app", Namespace: "ns"},
		Components: []Component{{Name: "core", ResourceCount: 1, Status: HealthReady}},
	}
	out := formatPlainTree(result)
	// Header should just be the release name, no parenthesised module
	assert.True(t, strings.HasPrefix(out, "local-app\n"))
}

func TestFormatTreeTable_Children_Rendered(t *testing.T) {
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{
				Name:          "server",
				ResourceCount: 1,
				Status:        HealthReady,
				Resources: []ResourceNode{
					{
						Kind: "Deployment", Name: "web", Namespace: "ns", Status: HealthReady, Replicas: "2/2",
						Children: []ResourceNode{
							{
								Kind: "ReplicaSet", Name: "web-rs", Namespace: "ns", Status: HealthReady, Replicas: "2 pods",
								Children: []ResourceNode{
									{Kind: "Pod", Name: "web-rs-p1", Namespace: "ns", Status: "Running", Ready: true},
									{Kind: "Pod", Name: "web-rs-p2", Namespace: "ns", Status: "Running", Ready: true},
								},
							},
						},
					},
				},
			},
		},
	}
	out := formatPlainTree(result)
	assert.Contains(t, out, "ReplicaSet/web-rs")
	assert.Contains(t, out, "Pod/web-rs-p1")
	assert.Contains(t, out, "Pod/web-rs-p2")
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 8: JSON / YAML output
// ─────────────────────────────────────────────────────────────────────────────

func TestFormatTreeJSON_Schema(t *testing.T) {
	result := makeSimpleResult()
	out, err := FormatTreeJSON(result)
	require.NoError(t, err)

	assert.Contains(t, out, `"name": "my-app"`)
	assert.Contains(t, out, `"module": "example.com/app"`)
	assert.Contains(t, out, `"version": "1.0.0"`)
	assert.Contains(t, out, `"components"`)
	assert.Contains(t, out, `"name": "server"`)
	assert.Contains(t, out, `"kind": "Deployment"`)
	assert.Contains(t, out, `"replicas": "3/3"`)
}

func TestFormatTreeYAML_Schema(t *testing.T) {
	result := makeSimpleResult()
	out, err := FormatTreeYAML(result)
	require.NoError(t, err)

	assert.Contains(t, out, "name: my-app")
	assert.Contains(t, out, "module: example.com/app")
	assert.Contains(t, out, "components:")
	assert.Contains(t, out, "kind: Deployment")
	assert.Contains(t, out, "replicas: 3/3")
}

func TestFormatTreeJSON_NestedChildren(t *testing.T) {
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{
				Name:          "server",
				ResourceCount: 1,
				Status:        HealthReady,
				Resources: []ResourceNode{
					{
						Kind: "Deployment", Name: "web", Namespace: "ns", Status: HealthReady,
						Children: []ResourceNode{
							{Kind: "ReplicaSet", Name: "web-rs", Namespace: "ns", Status: HealthReady,
								Children: []ResourceNode{
									{Kind: "Pod", Name: "web-rs-p1", Namespace: "ns", Status: "Running", Ready: true},
								},
							},
						},
					},
				},
			},
		},
	}

	out, err := FormatTreeJSON(result)
	require.NoError(t, err)
	assert.Contains(t, out, `"children"`)
	assert.Contains(t, out, `"kind": "ReplicaSet"`)
	assert.Contains(t, out, `"kind": "Pod"`)
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 9 extras: pod phase and CrashLoop display (spec scenarios)
// ─────────────────────────────────────────────────────────────────────────────

// TestPodToNode_RunningReadyPod verifies that a Running+Ready pod is represented
// with Status="Running" and Ready=true — matching the spec "Pod shows phase status"
// scenario and the display convention of `mod status`.
func TestPodToNode_RunningReadyPod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-0", Namespace: "ns"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			Conditions: []corev1.PodCondition{
				{Type: corev1.PodReady, Status: corev1.ConditionTrue},
			},
		},
	}
	node := podToNode(pod)
	assert.Equal(t, HealthStatus("Running"), node.Status, "Running pod should preserve phase, not use HealthStatus enum")
	assert.True(t, node.Ready)
}

// TestPodToNode_CrashLoopPod verifies that a pod with CrashLoopBackOff waiting
// reason is represented with Status="CrashLoop" — matching the spec
// "Pod shows detailed container status" scenario.
func TestPodToNode_CrashLoopPod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-0", Namespace: "ns"},
		Status: corev1.PodStatus{
			Phase: corev1.PodRunning,
			ContainerStatuses: []corev1.ContainerStatus{
				{
					State: corev1.ContainerState{
						Waiting: &corev1.ContainerStateWaiting{
							Reason: "CrashLoopBackOff",
						},
					},
				},
			},
		},
	}
	node := podToNode(pod)
	assert.Equal(t, HealthStatus("CrashLoop"), node.Status,
		"CrashLoopBackOff should be shortened to CrashLoop per mapWaitingReason")
	assert.False(t, node.Ready)
}

// TestPodToNode_PendingPod verifies a Pending pod shows phase "Pending".
func TestPodToNode_PendingPod(t *testing.T) {
	pod := &corev1.Pod{
		ObjectMeta: metav1.ObjectMeta{Name: "web-0", Namespace: "ns"},
		Status:     corev1.PodStatus{Phase: corev1.PodPending},
	}
	node := podToNode(pod)
	assert.Equal(t, HealthStatus("Pending"), node.Status)
	assert.False(t, node.Ready)
}

// ─────────────────────────────────────────────────────────────────────────────
// Section 10: Output refinements — displayKind, PVC capacity, alignment
// ─────────────────────────────────────────────────────────────────────────────

func TestDisplayKind_PVC(t *testing.T) {
	assert.Equal(t, "PVC", displayKind("PersistentVolumeClaim"))
}

func TestDisplayKind_OtherKindsUnchanged(t *testing.T) {
	for _, kind := range []string{"Deployment", "StatefulSet", "Service", "ConfigMap", "Ingress", "Pod"} {
		assert.Equal(t, kind, displayKind(kind), "kind %s should not be abbreviated", kind)
	}
}

func TestGetReplicaCount_PVC_ActualCapacity(t *testing.T) {
	pvc := makeRes("PersistentVolumeClaim", "ns", "data")
	_ = unstructured.SetNestedField(pvc.Object, "10Gi", "status", "capacity", "storage")
	assert.Equal(t, "10Gi", getReplicaCount(pvc))
}

func TestGetReplicaCount_PVC_FallbackToRequest(t *testing.T) {
	// No status.capacity — falls back to spec.resources.requests.storage.
	pvc := makeRes("PersistentVolumeClaim", "ns", "data")
	_ = unstructured.SetNestedField(pvc.Object, "5Gi", "spec", "resources", "requests", "storage")
	assert.Equal(t, "5Gi", getReplicaCount(pvc))
}

func TestGetReplicaCount_PVC_NoCapacity(t *testing.T) {
	// Neither status nor spec capacity → empty string.
	pvc := makeRes("PersistentVolumeClaim", "ns", "data")
	assert.Empty(t, getReplicaCount(pvc))
}

func TestMeasureTreeWidths_SingleResource(t *testing.T) {
	// "Service/web-svc" at depth=0, prefix=4: width = 4 + 4 + 15 = 23.
	// colWidths[0] = 23 + 6 = 29.
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{
				Name:          "server",
				ResourceCount: 1,
				Resources:     []ResourceNode{{Kind: "Service", Name: "web-svc", Status: HealthReady}},
			},
		},
	}
	assert.Equal(t, map[int]int{0: 29}, measureTreeWidths(result))
}

func TestMeasureTreeWidths_PerDepthAlignment(t *testing.T) {
	// Deployment/web at depth 0 (prefix=4): 4+4+14 = 22 → colWidths[0] = 22+6 = 28
	// ReplicaSet/web-rs-abc at depth 1 (prefix=8): 8+4+21 = 33 → colWidths[1] = 33+6 = 39
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{
				Name:          "server",
				ResourceCount: 1,
				Resources: []ResourceNode{
					{
						Kind: "Deployment", Name: "web", Status: HealthReady,
						Children: []ResourceNode{
							{Kind: "ReplicaSet", Name: "web-rs-abc", Status: HealthReady, Replicas: "2 pods"},
						},
					},
				},
			},
		},
	}
	assert.Equal(t, map[int]int{0: 28, 1: 39}, measureTreeWidths(result))
}

func TestMeasureTreeWidths_PVCAbbreviated(t *testing.T) {
	// PVC/data at depth 0 (prefix=4): 4+4+8 = 16 → colWidths[0] = 16+6 = 22.
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{
				Name:          "storage",
				ResourceCount: 1,
				Resources:     []ResourceNode{{Kind: "PersistentVolumeClaim", Name: "data", Status: HealthBound, Replicas: "10Gi"}},
			},
		},
	}
	assert.Equal(t, map[int]int{0: 22}, measureTreeWidths(result))
}

// TestFormatPlainTree_StatusBeforeReplicas verifies that status appears before replicas.
func TestFormatPlainTree_StatusBeforeReplicas(t *testing.T) {
	result := makeSimpleResult() // has Deployment with Replicas="3/3" and Status=HealthReady
	out := formatPlainTree(result)

	// Find the Deployment line and check "Ready" comes before "3/3".
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Deployment/web") {
			readyIdx := strings.Index(line, "Ready")
			replicasIdx := strings.Index(line, "3/3")
			require.True(t, readyIdx >= 0, "Ready not found in line: %q", line)
			require.True(t, replicasIdx >= 0, "3/3 not found in line: %q", line)
			assert.Less(t, readyIdx, replicasIdx, "Ready should appear before 3/3 in: %q", line)
			return
		}
	}
	t.Fatal("Deployment/web line not found in output")
}

// TestFormatPlainTree_AlignedColumns verifies all status tokens start at the same column.
func TestFormatPlainTree_AlignedColumns(t *testing.T) {
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{
				Name:          "server",
				ResourceCount: 2,
				Status:        HealthReady,
				Resources: []ResourceNode{
					// Longer name drives the column width.
					{Kind: "Deployment", Name: "web-server", Status: HealthReady, Replicas: "3/3"},
					{Kind: "Service", Name: "svc", Status: HealthReady},
				},
			},
		},
	}
	out := formatPlainTree(result)

	// Collect the column index of "Ready" on each resource line.
	var readyCols []int
	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "Deployment/web-server") || strings.Contains(line, "Service/svc") {
			idx := strings.Index(line, "Ready")
			require.True(t, idx >= 0, "Ready not found in line: %q", line)
			readyCols = append(readyCols, idx)
		}
	}

	require.Len(t, readyCols, 2, "expected two resource lines")
	assert.Equal(t, readyCols[0], readyCols[1], "status columns should be aligned")
}

// TestFormatPlainTree_RSStatusSuppressed verifies ReplicaSet nodes omit the status column.
func TestFormatPlainTree_RSStatusSuppressed(t *testing.T) {
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{
				Name:          "server",
				ResourceCount: 1,
				Status:        HealthReady,
				Resources: []ResourceNode{
					{
						Kind: "Deployment", Name: "web", Status: HealthReady, Replicas: "2/2",
						Children: []ResourceNode{
							{Kind: "ReplicaSet", Name: "web-rs", Status: HealthReady, Replicas: "2 pods"},
						},
					},
				},
			},
		},
	}
	out := formatPlainTree(result)

	for _, line := range strings.Split(out, "\n") {
		if strings.Contains(line, "ReplicaSet/web-rs") {
			// Should contain "2 pods" (the replica count)...
			assert.Contains(t, line, "2 pods")
			// ...but NOT "Ready" (the health status).
			assert.NotContains(t, line, "Ready", "ReplicaSet line should not show health status: %q", line)
			return
		}
	}
	t.Fatal("ReplicaSet/web-rs line not found in output")
}

// TestFormatPlainTree_PVCAbbreviated verifies PersistentVolumeClaim is displayed as PVC.
func TestFormatPlainTree_PVCAbbreviated(t *testing.T) {
	result := &TreeResult{
		Release: ReleaseInfo{Name: "app", Namespace: "ns"},
		Components: []Component{
			{
				Name:          "storage",
				ResourceCount: 1,
				Status:        HealthBound,
				Resources: []ResourceNode{
					{Kind: "PersistentVolumeClaim", Name: "data", Status: HealthBound, Replicas: "10Gi"},
				},
			},
		},
	}
	out := formatPlainTree(result)

	assert.Contains(t, out, "PVC/data", "display kind should be abbreviated")
	assert.NotContains(t, out, "PersistentVolumeClaim/data", "full kind should not appear in terminal output")
	assert.Contains(t, out, "Bound")
	assert.Contains(t, out, "10Gi")
}
