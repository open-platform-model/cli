package operator

import (
	"testing"

	"github.com/open-platform-model/cli/internal/kubernetes/kubetest"

	"github.com/stretchr/testify/assert"
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
		// A finished rollout of the default single replica.
		_ = unstructured.SetNestedMap(obj.Object, map[string]any{
			"observedGeneration": int64(0),
			"replicas":           int64(1),
			"updatedReplicas":    int64(1),
			"availableReplicas":  int64(1),
		}, "status")
	}
	return obj
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
	return &kubernetes.Client{Resources: kubetest.Resources(), Dynamic: fakedynamic.NewSimpleDynamicClientWithCustomListKinds(scheme, listKinds, runtimeObjs...)}
}
