package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	fakedynamic "k8s.io/client-go/dynamic/fake"
)

func TestKindToResource(t *testing.T) {
	tests := []struct {
		kind     string
		expected string
	}{
		{"Deployment", "deployments"},
		{"Service", "services"},
		{"Ingress", "ingresses"},
		{"ConfigMap", "configmaps"},
		{"NetworkPolicy", "networkpolicies"},
		{"DaemonSet", "daemonsets"},
		{"Endpoints", "endpoints"},
		{"EndpointSlice", "endpointslices"},
		{"CustomResourceDefinition", "customresourcedefinitions"},
		{"StorageClass", "storageclasses"},
		{"PodDisruptionBudget", "poddisruptionbudgets"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			assert.Equal(t, tt.expected, kindToResource(tt.kind))
		})
	}
}

func TestHeuristicPluralize(t *testing.T) {
	tests := []struct {
		kind     string
		expected string
	}{
		// Already-plural kinds
		{"Endpoints", "endpoints"},
		// -ss suffix
		{"Ingress", "ingresses"},
		// -y suffix with consonant
		{"NetworkPolicy", "networkpolicies"},
		// Regular
		{"Deployment", "deployments"},
		// -y suffix with vowel
		{"Gateway", "gateways"},
	}

	for _, tt := range tests {
		t.Run(tt.kind, func(t *testing.T) {
			assert.Equal(t, tt.expected, heuristicPluralize(tt.kind))
		})
	}
}

func TestGvrFromUnstructured(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")

	gvr := gvrFromUnstructured(obj)
	assert.Equal(t, "apps", gvr.Group)
	assert.Equal(t, "v1", gvr.Version)
	assert.Equal(t, "deployments", gvr.Resource)
}

func TestGvrFromUnstructured_CoreGroup(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("Service")

	gvr := gvrFromUnstructured(obj)
	assert.Equal(t, "", gvr.Group)
	assert.Equal(t, "v1", gvr.Version)
	assert.Equal(t, "services", gvr.Resource)
}

func TestResourceClient(t *testing.T) {
	scheme := runtime.NewScheme()
	fakeDynamic := fakedynamic.NewSimpleDynamicClient(scheme)
	client := &Client{Dynamic: fakeDynamic}

	gvr := schema.GroupVersionResource{Group: "", Version: "v1", Resource: "configmaps"}

	t.Run("namespace-scoped", func(t *testing.T) {
		rc := client.ResourceClient(gvr, "production")
		// Verify we get a non-nil interface back
		assert.NotNil(t, rc)
	})

	t.Run("cluster-scoped", func(t *testing.T) {
		rc := client.ResourceClient(gvr, "")
		// Verify we get a non-nil interface back
		assert.NotNil(t, rc)
	})
}
