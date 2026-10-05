package inventory

import (
	"strings"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/k8s/object"
)

// cueResource compiles a CUE manifest source into an *object.Resource.
func cueResource(t *testing.T, src string) *object.Resource {
	t.Helper()
	v := cuecontext.New().CompileString(src)
	require.NoError(t, v.Err())
	return &object.Resource{Value: v, Instance: "demo", Component: "web", Transformer: "t"}
}

// export runs the render's single object.Export over the resources.
func export(t *testing.T, resources ...*object.Resource) []object.Exported {
	t.Helper()
	out, err := object.Export(resources)
	require.NoError(t, err)
	return out
}

// renderResources returns a typical 3-resource compiled set, exported.
func renderResources(t *testing.T) []object.Exported {
	t.Helper()
	return export(t,
		cueResource(t, `apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "app", namespace: "ns"}, spec: replicas: 2`),
		cueResource(t, `apiVersion: "v1", kind: "Service", metadata: {name: "app", namespace: "ns"}, spec: port: 8080`),
		cueResource(t, `apiVersion: "v1", kind: "ConfigMap", metadata: {name: "config", namespace: "ns"}, data: key: "value"`),
	)
}

func TestComputeRenderDigest_Format(t *testing.T) {
	digest, err := ComputeRenderDigest(renderResources(t))
	require.NoError(t, err)
	require.True(t, strings.HasPrefix(digest, "sha256:"), "digest should start with sha256:")
	assert.Len(t, digest, len("sha256:")+64, "SHA256 hex should be 64 chars")
}

func TestComputeRenderDigest_Deterministic_InputOrder(t *testing.T) {
	r1 := renderResources(t)
	r2 := []object.Exported{r1[2], r1[0], r1[1]}
	r3 := []object.Exported{r1[1], r1[2], r1[0]}

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
	changed[0] = export(t, cueResource(t, `apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "app", namespace: "ns"}, spec: replicas: 5`))[0]
	modified, err := ComputeRenderDigest(changed)
	require.NoError(t, err)

	assert.NotEqual(t, original, modified)
}

func TestComputeRenderDigest_AddedResource(t *testing.T) {
	resources := renderResources(t)
	original, err := ComputeRenderDigest(resources)
	require.NoError(t, err)

	resources = append(resources, export(t, cueResource(t, `apiVersion: "v1", kind: "Secret", metadata: {name: "my-secret", namespace: "ns"}`))...)
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
	original := make([]object.Exported, len(resources))
	copy(original, resources)

	_, err := ComputeRenderDigest(resources)
	require.NoError(t, err)

	for i, r := range resources {
		assert.Same(t, original[i].Object, r.Object, "input order must not be mutated")
	}
}

// goldenRenderDigest is the digest of renderResources recorded before the
// digest moved from the retired pkg/core.Resource to the library's
// opm/k8s/object export. It must not move until the
// cli adopts the library's shared render digest with the operator, which
// changes every stored digest once, on purpose.
const goldenRenderDigest = "sha256:0404868537e1114e9ea6e7d02af68ba6a4693531727c4be713212753de6840ae"

func TestComputeRenderDigest_Golden(t *testing.T) {
	d, err := ComputeRenderDigest(renderResources(t))
	require.NoError(t, err)
	assert.Equal(t, goldenRenderDigest, d)
}

// multiSlashResources holds an object whose apiVersion has two slashes. Its
// group is the apiVersion up to the last slash ("a/b"), so it sorts after
// "" and "apps" but before "z". A sort key that parsed the apiVersion as a
// group-version would empty the key and move it, changing the digest.
func multiSlashResources(t *testing.T) []object.Exported {
	t.Helper()
	return export(t,
		cueResource(t, `apiVersion: "z/v1", kind: "Zed", metadata: {name: "z", namespace: "ns"}`),
		cueResource(t, `apiVersion: "a/b/c", kind: "Odd", metadata: {name: "odd", namespace: "ns"}`),
		cueResource(t, `apiVersion: "v1", kind: "ConfigMap", metadata: {name: "config", namespace: "ns"}`),
		cueResource(t, `apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "app", namespace: "ns"}`),
	)
}

// goldenMultiSlashDigest is the digest of multiSlashResources recorded
// before the same move; see goldenRenderDigest.
const goldenMultiSlashDigest = "sha256:43021f99cc1864b793c40493c4d0c196a8b7a7f58c528d728d8986d1388e378e"

func TestComputeRenderDigest_MultiSlashGolden(t *testing.T) {
	d, err := ComputeRenderDigest(multiSlashResources(t))
	require.NoError(t, err)
	assert.Equal(t, goldenMultiSlashDigest, d)
}

func TestComputeRenderDigest_RefusesObjectWithoutJSON(t *testing.T) {
	objs := renderResources(t)
	objs[1].JSON = nil
	_, err := ComputeRenderDigest(objs)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "has no exported JSON")
}
