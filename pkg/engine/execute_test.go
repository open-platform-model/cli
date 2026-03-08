package engine

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// cueValue is a test helper that compiles a CUE expression into a cue.Value.
func cueValue(t *testing.T, expr string) interface{ Exists() bool } {
	t.Helper()
	ctx := cuecontext.New()
	v := ctx.CompileString(expr)
	require.NoError(t, v.Err(), "compiling CUE expression %q", expr)
	return v
}

// TestIsSingleResource tests the isSingleResource heuristic.
// A resource is "single" when it has both "apiVersion" AND "kind" at the top level.
func TestIsSingleResource(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name string
		expr string
		want bool
	}{
		{
			name: "full kubernetes resource",
			expr: `{apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "test"}}`,
			want: true,
		},
		{
			name: "only apiVersion — not a single resource (map form)",
			expr: `{apiVersion: "apps/v1"}`,
			want: false,
		},
		{
			name: "only kind — not a single resource",
			expr: `{kind: "Deployment"}`,
			want: false,
		},
		{
			name: "neither field — map of resources",
			expr: `{deploy: {apiVersion: "apps/v1", kind: "Deployment"}}`,
			want: false,
		},
		{
			name: "empty struct",
			expr: `{}`,
			want: false,
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			v := ctx.CompileString(tc.expr)
			require.NoError(t, v.Err())
			assert.Equal(t, tc.want, isSingleResource(v))
		})
	}
}

// TestCollectResourceList tests that collectResourceList wraps each list item as a Resource.
func TestCollectResourceList(t *testing.T) {
	ctx := cuecontext.New()

	expr := `[
		{apiVersion: "v1", kind: "ConfigMap", metadata: {name: "cm1"}},
		{apiVersion: "v1", kind: "Secret",    metadata: {name: "sec1"}},
	]`
	v := ctx.CompileString(expr)
	require.NoError(t, v.Err())

	resources, err := collectResourceList(v, "my-release", "my-comp", "tf/kubernetes")
	require.NoError(t, err)
	require.Len(t, resources, 2)

	assert.Equal(t, "my-release", resources[0].Release)
	assert.Equal(t, "my-comp", resources[0].Component)
	assert.Equal(t, "tf/kubernetes", resources[0].Transformer)

	assert.Equal(t, "my-release", resources[1].Release)
	assert.Equal(t, "ConfigMap", resources[0].Kind())
	assert.Equal(t, "Secret", resources[1].Kind())
}

// TestCollectResourceList_Empty verifies empty list returns empty slice without error.
func TestCollectResourceList_Empty(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`[]`)
	require.NoError(t, v.Err())

	resources, err := collectResourceList(v, "rel", "comp", "tf/x")
	require.NoError(t, err)
	assert.Empty(t, resources)
}

// TestCollectResourceMap tests that collectResourceMap wraps each struct field as a Resource.
func TestCollectResourceMap(t *testing.T) {
	ctx := cuecontext.New()

	expr := `{
		deploy: {apiVersion: "apps/v1", kind: "Deployment", metadata: {name: "app"}},
		svc:    {apiVersion: "v1",      kind: "Service",    metadata: {name: "app-svc"}},
	}`
	v := ctx.CompileString(expr)
	require.NoError(t, v.Err())

	resources, err := collectResourceMap(v, "my-release", "my-comp", "tf/kubernetes")
	require.NoError(t, err)
	require.Len(t, resources, 2)

	for _, r := range resources {
		assert.Equal(t, "my-release", r.Release)
		assert.Equal(t, "my-comp", r.Component)
		assert.Equal(t, "tf/kubernetes", r.Transformer)
	}

	// Collect kinds to check both are present (map order is non-deterministic).
	kinds := map[string]bool{}
	for _, r := range resources {
		kinds[r.Kind()] = true
	}
	assert.True(t, kinds["Deployment"])
	assert.True(t, kinds["Service"])
}

// TestCollectResourceMap_Empty verifies empty struct returns empty slice without error.
func TestCollectResourceMap_Empty(t *testing.T) {
	ctx := cuecontext.New()
	v := ctx.CompileString(`{}`)
	require.NoError(t, v.Err())

	resources, err := collectResourceMap(v, "rel", "comp", "tf/x")
	require.NoError(t, err)
	assert.Empty(t, resources)
}
