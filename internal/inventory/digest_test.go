package inventory

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

	pkgcore "github.com/open-platform-model/cli/pkg/core"
)

// makeResourceWithContent builds a *unstructured.Unstructured with the given object fields.
// The component label (component.opmodel.dev/name) is set to the component parameter.
func makeResourceWithContent(group, version, kind, namespace, name, component string, extra map[string]interface{}) *unstructured.Unstructured { //nolint:unparam // version param kept for test flexibility
	obj := &unstructured.Unstructured{Object: map[string]interface{}{
		"apiVersion": group + "/" + version,
		"kind":       kind,
		"metadata": map[string]interface{}{
			"name":      name,
			"namespace": namespace,
			"labels": map[string]interface{}{
				pkgcore.LabelComponentName: component,
			},
		},
	}}
	if group == "" {
		obj.Object["apiVersion"] = version
	}
	for k, v := range extra {
		obj.Object[k] = v
	}
	obj.SetGroupVersionKind(schema.GroupVersionKind{Group: group, Version: version, Kind: kind})
	obj.SetNamespace(namespace)
	obj.SetName(name)
	// Preserve component label after SetName/SetNamespace (they don't touch labels)
	labels := obj.GetLabels()
	if labels == nil {
		labels = map[string]string{}
	}
	labels[pkgcore.LabelComponentName] = component
	obj.SetLabels(labels)
	return obj
}

// standardResources returns a typical 3-resource set.
func standardResources() []*unstructured.Unstructured {
	return []*unstructured.Unstructured{
		makeResourceWithContent("apps", "v1", "Deployment", "ns", "app", "web", map[string]interface{}{
			"spec": map[string]interface{}{"replicas": int64(2)},
		}),
		makeResourceWithContent("", "v1", "Service", "ns", "app", "web", map[string]interface{}{
			"spec": map[string]interface{}{"port": int64(8080)},
		}),
		makeResourceWithContent("", "v1", "ConfigMap", "ns", "config", "web", map[string]interface{}{
			"data": map[string]interface{}{"key": "value"},
		}),
	}
}

// cueResource compiles a CUE manifest source into a *pkgcore.Resource.
func cueResource(t *testing.T, src string) *pkgcore.Resource {
	t.Helper()
	v := cuecontext.New().CompileString(src)
	require.NoError(t, v.Err())
	return &pkgcore.Resource{Value: v, Instance: "demo", Component: "web", Transformer: "t"}
}

// renderResources returns a typical 3-resource compiled set.
func renderResources(t *testing.T) []*pkgcore.Resource {
	t.Helper()
	return []*pkgcore.Resource{
		cueResource(t, `apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "app", namespace: "ns"}, spec: replicas: 2`),
		cueResource(t, `apiVersion: "v1", kind: "Service", metadata: {name: "app", namespace: "ns"}, spec: port: 8080`),
		cueResource(t, `apiVersion: "v1", kind: "ConfigMap", metadata: {name: "config", namespace: "ns"}, data: key: "value"`),
	}
}

func TestComputeRenderDigest_Format(t *testing.T) {
	digest, err := ComputeRenderDigest(renderResources(t))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(digest, "sha256:"), "digest should start with sha256:")
	assert.Len(t, digest, len("sha256:")+64, "SHA256 hex should be 64 chars")
}

func TestComputeRenderDigest_Deterministic_InputOrder(t *testing.T) {
	r1 := renderResources(t)
	r2 := []*pkgcore.Resource{r1[2], r1[0], r1[1]}
	r3 := []*pkgcore.Resource{r1[1], r1[2], r1[0]}

	d1, err := ComputeRenderDigest(r1)
	require.NoError(t, err)
	d2, err := ComputeRenderDigest(r2)
	require.NoError(t, err)
	d3, err := ComputeRenderDigest(r3)
	require.NoError(t, err)

	assert.Equal(t, d1, d2)
	assert.Equal(t, d1, d3)
}

func TestComputeRenderDigest_ContentChange(t *testing.T) {
	original, err := ComputeRenderDigest(renderResources(t))
	require.NoError(t, err)

	changed := renderResources(t)
	changed[0] = cueResource(t, `apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "app", namespace: "ns"}, spec: replicas: 5`)
	modified, err := ComputeRenderDigest(changed)
	require.NoError(t, err)

	assert.NotEqual(t, original, modified)
}

func TestComputeRenderDigest_AddedResource(t *testing.T) {
	resources := renderResources(t)
	original, err := ComputeRenderDigest(resources)
	require.NoError(t, err)

	resources = append(resources, cueResource(t, `apiVersion: "v1", kind: "Secret", metadata: {name: "my-secret", namespace: "ns"}`))
	withExtra, err := ComputeRenderDigest(resources)
	require.NoError(t, err)

	assert.NotEqual(t, original, withExtra)
}

func TestComputeRenderDigest_EmptySet(t *testing.T) {
	d, err := ComputeRenderDigest(nil)
	require.NoError(t, err)
	// SHA256 of empty bytes — matches the operator's RenderDigest of an
	// empty set exactly.
	assert.Equal(t, "sha256:e3b0c44298fc1c149afbf4c8996fb92427ae41e4649b934ca495991b7852b855", d)
}

func TestComputeRenderDigest_DoesNotMutateInput(t *testing.T) {
	resources := renderResources(t)
	original := make([]*pkgcore.Resource, len(resources))
	copy(original, resources)

	_, err := ComputeRenderDigest(resources)
	require.NoError(t, err)

	for i, r := range resources {
		assert.Same(t, original[i], r, "input order must not be mutated")
	}
}
