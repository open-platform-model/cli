package k8sgen

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildComposition_FullAssembly(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {
	modulePath: "example.com/modules"
	name:       "my-service"
	version:    "0.1.0"
}
#config: {
	replicas: *1 | int
}
`)
	require.NoError(t, modVal.Err())

	comp, err := BuildComposition(modVal, CompositionOptions{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	// Top-level fields: Composition is v1 even under Crossplane v2.
	assert.Equal(t, "apiextensions.crossplane.io/v1", comp.Object["apiVersion"])
	assert.Equal(t, "Composition", comp.Object["kind"])

	// Name is metadata.name verbatim; compositeTypeRef handles the structural binding.
	meta := comp.Object["metadata"].(map[string]any)
	assert.Equal(t, "my-service", meta["name"])

	// Provenance labels are always present (minimal module metadata case).
	labels := meta["labels"].(map[string]any)
	assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-service", labels["module.opmodel.dev/name"])
	assert.Equal(t, "0.1.0", labels["module.opmodel.dev/version"])

	spec := comp.Object["spec"].(map[string]any)
	assert.Equal(t, "Pipeline", spec["mode"])

	ref := spec["compositeTypeRef"].(map[string]any)
	// apiVersion must match what BuildXRD emits for the same module so the
	// Composition binds to the right XR kind.
	assert.Equal(t, "module.opmodel.dev/v1alpha1", ref["apiVersion"])
	assert.Equal(t, "MyService", ref["kind"])

	pipeline := spec["pipeline"].([]any)
	require.Len(t, pipeline, 1)

	step := pipeline[0].(map[string]any)
	assert.Equal(t, "render-opm-module", step["step"])
	assert.Equal(t, map[string]any{"name": "function-opm"}, step["functionRef"])

	input := step["input"].(map[string]any)
	assert.Equal(t, "template.fn.crossplane.io/v1beta1", input["apiVersion"])
	assert.Equal(t, "Input", input["kind"])

	module := input["module"].(map[string]any)
	assert.Equal(t, "example.com/modules/my-service", module["path"])
	assert.Equal(t, "0.1.0", module["version"])
}

// TestBuildComposition_PairsWithXRD pins the invariant that the Composition's
// compositeTypeRef exactly matches the apiVersion/kind of the XRD built from
// the same module. This is the only structural link between the two
// manifests; if it drifts, `kubectl apply` accepts both but Crossplane won't
// reconcile.
func TestBuildComposition_PairsWithXRD(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {
	modulePath: "example.com/modules"
	name:       "widget"
	version:    "2.3.1"
}
#config: {}
`)
	require.NoError(t, modVal.Err())

	group := "example.com"

	xrd, err := BuildXRD(modVal, XRDOptions{Group: group})
	require.NoError(t, err)
	comp, err := BuildComposition(modVal, CompositionOptions{Group: group})
	require.NoError(t, err)

	xrdSpec := xrd.Object["spec"].(map[string]any)
	xrdGroup := xrdSpec["group"].(string)
	xrdKind := xrdSpec["names"].(map[string]any)["kind"].(string)
	xrdVersion := xrdSpec["versions"].([]any)[0].(map[string]any)["name"].(string)

	ref := comp.Object["spec"].(map[string]any)["compositeTypeRef"].(map[string]any)
	assert.Equal(t, xrdGroup+"/"+xrdVersion, ref["apiVersion"])
	assert.Equal(t, xrdKind, ref["kind"])
}

func TestBuildComposition_FlagDefaults(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {
	modulePath: "example.com/modules"
	name:       "svc"
	version:    "1.0.0"
}
#config: {}
`)
	require.NoError(t, modVal.Err())

	// All three optional fields empty — defaults should kick in.
	comp, err := BuildComposition(modVal, CompositionOptions{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	step := comp.Object["spec"].(map[string]any)["pipeline"].([]any)[0].(map[string]any)
	assert.Equal(t, "render-opm-module", step["step"])
	assert.Equal(t, "function-opm", step["functionRef"].(map[string]any)["name"])
	assert.Equal(t, "template.fn.crossplane.io/v1beta1", step["input"].(map[string]any)["apiVersion"])
}

func TestBuildComposition_FlagOverrides(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {
	modulePath: "example.com/modules"
	name:       "svc"
	version:    "1.0.0"
}
#config: {}
`)
	require.NoError(t, modVal.Err())

	comp, err := BuildComposition(modVal, CompositionOptions{
		Group:           "module.opmodel.dev",
		FunctionName:    "function-opm-fork",
		StepName:        "render-custom",
		InputAPIVersion: "opm.fn.crossplane.io/v1alpha1",
	})
	require.NoError(t, err)

	step := comp.Object["spec"].(map[string]any)["pipeline"].([]any)[0].(map[string]any)
	assert.Equal(t, "render-custom", step["step"])
	assert.Equal(t, "function-opm-fork", step["functionRef"].(map[string]any)["name"])
	assert.Equal(t, "opm.fn.crossplane.io/v1alpha1", step["input"].(map[string]any)["apiVersion"])
}

func TestBuildComposition_ModulePathAssembly(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name       string
		modulePath string
		moduleName string
		wantPath   string
	}{
		{
			name:       "registry with single segment",
			modulePath: "example.com/modules",
			moduleName: "my-service",
			wantPath:   "example.com/modules/my-service",
		},
		{
			name:       "deep registry path",
			modulePath: "opmodel.dev/org/team/modules",
			moduleName: "widget",
			wantPath:   "opmodel.dev/org/team/modules/widget",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := cuecontext.New()
			src := `
metadata: {
	modulePath: "` + tt.modulePath + `"
	name:       "` + tt.moduleName + `"
	version:    "1.0.0"
}
#config: {}
`
			modVal := ctx.CompileString(src)
			require.NoError(t, modVal.Err())

			comp, err := BuildComposition(modVal, CompositionOptions{Group: "module.opmodel.dev"})
			require.NoError(t, err)

			module := comp.Object["spec"].(map[string]any)["pipeline"].([]any)[0].(map[string]any)["input"].(map[string]any)["module"].(map[string]any)
			assert.Equal(t, tt.wantPath, module["path"])
		})
	}
}

// TestBuildComposition_ProvenanceOPMKeysWinOnCollision mirrors the invariant
// held for CRD and XRD: module-declared labels/annotations cannot shadow
// OPM-owned keys.
func TestBuildComposition_ProvenanceOPMKeysWinOnCollision(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {
	modulePath: "example.com/modules"
	name:       "svc"
	version:    "1.0.0"
	labels: {
		"app.kubernetes.io/managed-by": "malicious-tool"
		"module.opmodel.dev/name":      "imposter"
	}
}
#config: {}
`)
	require.NoError(t, modVal.Err())

	comp, err := BuildComposition(modVal, CompositionOptions{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	labels := comp.Object["metadata"].(map[string]any)["labels"].(map[string]any)
	assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "svc", labels["module.opmodel.dev/name"])
}

func TestBuildComposition_Errors(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()

	tests := []struct {
		name    string
		src     string
		opts    CompositionOptions
		wantErr string
	}{
		{
			name: "missing group",
			src: `
metadata: {modulePath: "example.com/modules", name: "svc", version: "1.0.0"}
#config: {}
`,
			opts:    CompositionOptions{},
			wantErr: "composition group is required",
		},
		{
			name: "missing metadata.name",
			src: `
metadata: {modulePath: "example.com/modules", version: "1.0.0"}
#config: {}
`,
			opts:    CompositionOptions{Group: "module.opmodel.dev"},
			wantErr: "metadata.name is not set",
		},
		{
			name: "missing metadata.version",
			src: `
metadata: {modulePath: "example.com/modules", name: "svc"}
#config: {}
`,
			opts:    CompositionOptions{Group: "module.opmodel.dev"},
			wantErr: "metadata.version is not set",
		},
		{
			name: "missing metadata.modulePath",
			src: `
metadata: {name: "svc", version: "1.0.0"}
#config: {}
`,
			opts:    CompositionOptions{Group: "module.opmodel.dev"},
			wantErr: "metadata.modulePath",
		},
		{
			name: "invalid name (starts with digit)",
			src: `
metadata: {modulePath: "example.com/modules", name: "123svc", version: "1.0.0"}
#config: {}
`,
			opts:    CompositionOptions{Group: "module.opmodel.dev"},
			wantErr: "invalid CRD kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			modVal := ctx.CompileString(tt.src)
			require.NoError(t, modVal.Err())

			_, err := BuildComposition(modVal, tt.opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
