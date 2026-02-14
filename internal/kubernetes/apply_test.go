package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
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

func TestGvrFromObject(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")

	gvr := gvrFromObject(obj)
	assert.Equal(t, "apps", gvr.Group)
	assert.Equal(t, "v1", gvr.Version)
	assert.Equal(t, "deployments", gvr.Resource)
}

func TestGvrFromObject_CoreGroup(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("v1")
	obj.SetKind("Service")

	gvr := gvrFromObject(obj)
	assert.Equal(t, "", gvr.Group)
	assert.Equal(t, "v1", gvr.Version)
	assert.Equal(t, "services", gvr.Resource)
}
