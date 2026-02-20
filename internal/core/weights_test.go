package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

func TestGetWeight(t *testing.T) {
	tests := []struct {
		name string
		gvk  schema.GroupVersionKind
		want int
	}{
		{
			name: "Namespace",
			gvk:  schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Namespace"},
			want: WeightNamespace,
		},
		{
			name: "ConfigMap",
			gvk:  schema.GroupVersionKind{Group: "", Version: "v1", Kind: "ConfigMap"},
			want: WeightConfigMap,
		},
		{
			name: "Secret",
			gvk:  schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Secret"},
			want: WeightSecret,
		},
		{
			name: "Service",
			gvk:  schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"},
			want: WeightService,
		},
		{
			name: "Deployment",
			gvk:  schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"},
			want: WeightDeployment,
		},
		{
			name: "StatefulSet",
			gvk:  schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "StatefulSet"},
			want: WeightStatefulSet,
		},
		{
			name: "Ingress",
			gvk:  schema.GroupVersionKind{Group: "networking.k8s.io", Version: "v1", Kind: "Ingress"},
			want: WeightIngress,
		},
		{
			name: "CRD",
			gvk:  schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"},
			want: WeightCRD,
		},
		{
			name: "unknown resource",
			gvk:  schema.GroupVersionKind{Group: "custom.example.com", Version: "v1", Kind: "MyResource"},
			want: WeightDefault,
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := GetWeight(tt.gvk)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestWeightOrder(t *testing.T) {
	// Verify that weights are in the correct order for apply
	assert.Less(t, WeightCRD, WeightNamespace, "CRDs should come before Namespaces")
	assert.Less(t, WeightNamespace, WeightConfigMap, "Namespaces should come before ConfigMaps")
	assert.Less(t, WeightConfigMap, WeightService, "ConfigMaps should come before Services")
	assert.Less(t, WeightService, WeightDeployment, "Services should come before Deployments")
	assert.Less(t, WeightDeployment, WeightIngress, "Deployments should come before Ingresses")
	assert.Less(t, WeightIngress, WeightHPA, "Ingresses should come before HPAs")
}
