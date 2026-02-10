package kubernetes

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/build"
)

func TestInjectLabels(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")
	obj.SetName("test-deploy")
	obj.SetNamespace("default")

	res := &build.Resource{
		Object:      obj,
		Component:   "web-server",
		Transformer: "opmodel.dev/transformers/kubernetes@v0#DeploymentTransformer",
	}

	meta := build.ModuleMetadata{
		Name:      "my-app",
		Namespace: "production",
		Version:   "1.0.0",
	}

	injectLabels(res, meta)

	labels := obj.GetLabels()
	assert.Equal(t, "open-platform-model", labels[LabelManagedBy])
	assert.Equal(t, "my-app", labels[LabelModuleName])
	assert.Equal(t, "production", labels[LabelModuleNamespace])
	assert.Equal(t, "1.0.0", labels[LabelModuleVersion])
	assert.Equal(t, "web-server", labels[LabelComponentName])
}

func TestInjectLabels_PreservesExisting(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetLabels(map[string]string{
		"existing-label": "existing-value",
	})

	res := &build.Resource{
		Object:    obj,
		Component: "api",
	}

	meta := build.ModuleMetadata{
		Name:      "svc",
		Namespace: "ns",
	}

	injectLabels(res, meta)

	labels := obj.GetLabels()
	assert.Equal(t, "existing-value", labels["existing-label"])
	assert.Equal(t, "open-platform-model", labels[LabelManagedBy])
	assert.Equal(t, "svc", labels[LabelModuleName])
}

func TestInjectLabels_NoVersionIfEmpty(t *testing.T) {
	obj := &unstructured.Unstructured{}
	res := &build.Resource{Object: obj}
	meta := build.ModuleMetadata{Name: "app", Namespace: "ns"}

	injectLabels(res, meta)

	labels := obj.GetLabels()
	_, hasVersion := labels[LabelModuleVersion]
	assert.False(t, hasVersion, "should not have version label when version is empty")
}

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

func TestInjectLabels_WithIdentity(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")
	obj.SetName("test-deploy")
	obj.SetNamespace("default")

	res := &build.Resource{
		Object:      obj,
		Component:   "web-server",
		Transformer: "opmodel.dev/transformers/kubernetes@v0#DeploymentTransformer",
	}

	meta := build.ModuleMetadata{
		Name:            "my-app",
		Namespace:       "production",
		Version:         "1.0.0",
		Identity:        "module-uuid-1234",
		ReleaseIdentity: "release-uuid-5678",
	}

	injectLabels(res, meta)

	labels := obj.GetLabels()
	assert.Equal(t, "module-uuid-1234", labels[LabelModuleID])
	assert.Equal(t, "release-uuid-5678", labels[LabelReleaseID])
}

func TestInjectLabels_WithoutIdentity(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetAPIVersion("apps/v1")
	obj.SetKind("Deployment")
	obj.SetName("test-deploy")

	res := &build.Resource{Object: obj}

	meta := build.ModuleMetadata{
		Name:            "my-app",
		Namespace:       "production",
		Identity:        "", // Empty identity
		ReleaseIdentity: "", // Empty release identity
	}

	injectLabels(res, meta)

	labels := obj.GetLabels()
	_, hasModuleID := labels[LabelModuleID]
	_, hasReleaseID := labels[LabelReleaseID]
	assert.False(t, hasModuleID, "should not have module-id label when identity is empty")
	assert.False(t, hasReleaseID, "should not have release-id label when release identity is empty")
}

func TestInjectLabels_PreservesExistingWithIdentity(t *testing.T) {
	obj := &unstructured.Unstructured{}
	obj.SetLabels(map[string]string{
		"existing-label": "existing-value",
		"user-label":     "user-value",
	})

	res := &build.Resource{
		Object:    obj,
		Component: "api",
	}

	meta := build.ModuleMetadata{
		Name:            "svc",
		Namespace:       "ns",
		Identity:        "mod-id",
		ReleaseIdentity: "rel-id",
	}

	injectLabels(res, meta)

	labels := obj.GetLabels()
	// User labels preserved
	assert.Equal(t, "existing-value", labels["existing-label"])
	assert.Equal(t, "user-value", labels["user-label"])
	// OPM labels added
	assert.Equal(t, "open-platform-model", labels[LabelManagedBy])
	assert.Equal(t, "svc", labels[LabelModuleName])
	// Identity labels added
	assert.Equal(t, "mod-id", labels[LabelModuleID])
	assert.Equal(t, "rel-id", labels[LabelReleaseID])
}
