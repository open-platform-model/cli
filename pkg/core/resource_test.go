package core_test

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime/schema"

	"github.com/opmodel/cli/pkg/core"
)

// deploymentCUE is a minimal concrete Kubernetes Deployment manifest in CUE.
const deploymentCUE = `
{
	apiVersion: "apps/v1"
	kind:       "Deployment"
	metadata: {
		name:      "my-app"
		namespace: "default"
		labels: {
			"app.kubernetes.io/name": "my-app"
		}
		annotations: {
			"example.com/note": "test"
		}
	}
	spec: {
		replicas: 1
	}
}
`

func newTestResource(t *testing.T, cueSrc string) *core.Resource {
	t.Helper()
	ctx := cuecontext.New()
	v := ctx.CompileString(cueSrc)
	require.NoError(t, v.Err())
	return &core.Resource{
		Value:       v,
		Release:     "test-release",
		Component:   "test-component",
		Transformer: "kubernetes#deployment",
	}
}

func TestResource_Accessors(t *testing.T) {
	r := newTestResource(t, deploymentCUE)

	assert.Equal(t, "Deployment", r.Kind())
	assert.Equal(t, "my-app", r.Name())
	assert.Equal(t, "default", r.Namespace())
	assert.Equal(t, "apps/v1", r.APIVersion())
	assert.Equal(t, map[string]string{"app.kubernetes.io/name": "my-app"}, r.Labels())
	assert.Equal(t, map[string]string{"example.com/note": "test"}, r.Annotations())
}

func TestResource_GVK(t *testing.T) {
	r := newTestResource(t, deploymentCUE)
	gvk := r.GVK()
	assert.Equal(t, "apps", gvk.Group)
	assert.Equal(t, "v1", gvk.Version)
	assert.Equal(t, "Deployment", gvk.Kind)
}

func TestResource_GVK_CoreGroup(t *testing.T) {
	r := newTestResource(t, `{
		apiVersion: "v1"
		kind:       "Service"
		metadata: name: "my-svc"
	}`)
	gvk := r.GVK()
	assert.Equal(t, "", gvk.Group)
	assert.Equal(t, "v1", gvk.Version)
	assert.Equal(t, "Service", gvk.Kind)
}

func TestResource_MarshalJSON(t *testing.T) {
	r := newTestResource(t, deploymentCUE)
	b, err := r.MarshalJSON()
	require.NoError(t, err)
	assert.Contains(t, string(b), `"kind":"Deployment"`)
	assert.Contains(t, string(b), `"name":"my-app"`)
}

func TestResource_ToUnstructured(t *testing.T) {
	r := newTestResource(t, deploymentCUE)
	u, err := r.ToUnstructured()
	require.NoError(t, err)
	assert.Equal(t, "Deployment", u.GetKind())
	assert.Equal(t, "my-app", u.GetName())
	assert.Equal(t, "default", u.GetNamespace())
}

func TestResource_Namespace_Empty(t *testing.T) {
	r := newTestResource(t, `{
		apiVersion: "apiextensions.k8s.io/v1"
		kind:       "CustomResourceDefinition"
		metadata: name: "foos.example.com"
	}`)
	assert.Equal(t, "", r.Namespace())
}

func TestGetWeight_KnownGVK(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "apps", Version: "v1", Kind: "Deployment"}
	assert.Equal(t, core.WeightDeployment, core.GetWeight(gvk))
}

func TestGetWeight_CoreService(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "", Version: "v1", Kind: "Service"}
	assert.Equal(t, core.WeightService, core.GetWeight(gvk))
}

func TestGetWeight_CRD(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "apiextensions.k8s.io", Version: "v1", Kind: "CustomResourceDefinition"}
	assert.Equal(t, core.WeightCRD, core.GetWeight(gvk))
}

func TestGetWeight_Unknown(t *testing.T) {
	gvk := schema.GroupVersionKind{Group: "example.com", Version: "v1", Kind: "Foo"}
	assert.Equal(t, core.WeightDefault, core.GetWeight(gvk))
}

func TestGetWeight_KindFallback(t *testing.T) {
	// Kind-only fallback when group/version don't match exactly.
	gvk := schema.GroupVersionKind{Group: "unknown.io", Version: "v99", Kind: "Deployment"}
	assert.Equal(t, core.WeightDeployment, core.GetWeight(gvk))
}
