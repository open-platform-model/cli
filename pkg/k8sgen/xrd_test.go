package k8sgen

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildXRD_FullAssembly(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {
	name: "my-service"
	version: "0.1.0"
}
debugValues: {name: "demo"}
#config: {
	name: string
	replicas: int | *3
	tag?: string
}
`)
	require.NoError(t, modVal.Err())

	xrd, err := BuildXRD(modVal, XRDOptions{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	// Top-level fields.
	assert.Equal(t, "apiextensions.crossplane.io/v2", xrd.Object["apiVersion"])
	assert.Equal(t, "CompositeResourceDefinition", xrd.Object["kind"])

	meta := xrd.Object["metadata"].(map[string]any)
	assert.Equal(t, "myservices.module.opmodel.dev", meta["name"])

	// Minimal provenance: managed-by + name/version labels always present.
	labels := meta["labels"].(map[string]any)
	assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-service", labels["module.opmodel.dev/name"])
	assert.Equal(t, "0.1.0", labels["module.opmodel.dev/version"])

	// No annotations when no optional metadata was provided.
	_, hasAnnotations := meta["annotations"]
	assert.False(t, hasAnnotations)

	spec := xrd.Object["spec"].(map[string]any)
	assert.Equal(t, "Namespaced", spec["scope"], "default scope is Namespaced in v2")
	assert.Equal(t, "module.opmodel.dev", spec["group"])

	names := spec["names"].(map[string]any)
	assert.Equal(t, "MyService", names["kind"])
	assert.Equal(t, "MyServiceList", names["listKind"])
	assert.Equal(t, "myservices", names["plural"])
	assert.Equal(t, "myservice", names["singular"])

	versions := spec["versions"].([]any)
	require.Len(t, versions, 1)

	v0 := versions[0].(map[string]any)
	assert.Equal(t, "v1alpha1", v0["name"])
	assert.Equal(t, true, v0["served"])
	assert.Equal(t, true, v0["referenceable"])
	assert.Equal(t, true, v0["storage"])

	schema := v0["schema"].(map[string]any)
	openAPI := schema["openAPIV3Schema"].(map[string]any)
	assert.Equal(t, "object", openAPI["type"])

	// #config is wrapped under properties.spec; the root required list names
	// "spec" so Crossplane accepts the schema.
	assert.Equal(t, []any{"spec"}, openAPI["required"])

	props := openAPI["properties"].(map[string]any)
	require.Contains(t, props, "spec")
	_, hasStatus := props["status"]
	assert.False(t, hasStatus, "status is not emitted in POC scope")

	specSchema := props["spec"].(map[string]any)
	assert.Equal(t, "object", specSchema["type"])
	specProps := specSchema["properties"].(map[string]any)
	assert.Contains(t, specProps, "name")
	assert.Contains(t, specProps, "replicas")
	assert.Contains(t, specProps, "tag")
}

// TestBuildXRD_SchemaWrappedUnderSpec pins the wrapping contract: the
// extracted #config schema is placed verbatim under properties.spec, and
// the #config's own required list travels with it under properties.spec.required.
func TestBuildXRD_SchemaWrappedUnderSpec(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {name: "svc", version: "1.0.0"}
#config: {
	mustHave: string
	optional?: string
}
`)
	require.NoError(t, modVal.Err())

	xrd, err := BuildXRD(modVal, XRDOptions{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	spec := xrd.Object["spec"].(map[string]any)
	v0 := spec["versions"].([]any)[0].(map[string]any)
	openAPI := v0["schema"].(map[string]any)["openAPIV3Schema"].(map[string]any)
	specSchema := openAPI["properties"].(map[string]any)["spec"].(map[string]any)

	required, ok := specSchema["required"].([]any)
	require.True(t, ok, "spec schema must propagate the #config required list")
	assert.ElementsMatch(t, []any{"mustHave"}, required)
}

func TestBuildXRD_ScopeExplicit(t *testing.T) {
	t.Parallel()

	tests := []struct {
		name  string
		scope XRDScope
		want  string
	}{
		{"cluster", XRDScopeCluster, "Cluster"},
		{"legacy-cluster", XRDScopeLegacyCluster, "LegacyCluster"},
		{"namespaced-explicit", XRDScopeNamespaced, "Namespaced"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			ctx := cuecontext.New()
			modVal := ctx.CompileString(`
metadata: {name: "svc", version: "1.0.0"}
#config: {}
`)
			require.NoError(t, modVal.Err())

			xrd, err := BuildXRD(modVal, XRDOptions{
				Group: "module.opmodel.dev",
				Scope: tt.scope,
			})
			require.NoError(t, err)

			spec := xrd.Object["spec"].(map[string]any)
			assert.Equal(t, tt.want, spec["scope"])
		})
	}
}

func TestBuildXRD_CustomGroup(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {name: "widget", version: "1.0.0"}
#config: {label: string}
`)
	require.NoError(t, modVal.Err())

	xrd, err := BuildXRD(modVal, XRDOptions{Group: "example.com"})
	require.NoError(t, err)

	meta := xrd.Object["metadata"].(map[string]any)
	assert.Equal(t, "widgets.example.com", meta["name"])

	spec := xrd.Object["spec"].(map[string]any)
	assert.Equal(t, "example.com", spec["group"])
}

// TestBuildXRD_ProvenanceFullMetadata mirrors the CRD provenance test.
// Labels and annotations stamped on the XRD must match the CRD behavior —
// they're produced by the same shared buildProvenance helper, and regressions
// there would also break CRD; pinning XRD independently makes the breakage
// localized on its way to the fix.
func TestBuildXRD_ProvenanceFullMetadata(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {
	modulePath:  "opmodel.dev/modules"
	name:        "zot-registry"
	version:     "0.1.0"
	description: "Production-ready Zot OCI registry"
	fqn:         "opmodel.dev/modules/zot-registry:0.1.0"
	uuid:        "11111111-2222-3333-4444-555555555555"
	labels: {
		"app.kubernetes.io/component": "registry"
	}
	annotations: {
		"example.com/ticket": "PROJ-42"
	}
}
#config: {name: string}
`)
	require.NoError(t, modVal.Err())

	xrd, err := BuildXRD(modVal, XRDOptions{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	meta := xrd.Object["metadata"].(map[string]any)

	labels := meta["labels"].(map[string]any)
	assert.Equal(t, "registry", labels["app.kubernetes.io/component"])
	assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "zot-registry", labels["module.opmodel.dev/name"])
	assert.Equal(t, "0.1.0", labels["module.opmodel.dev/version"])

	annotations := meta["annotations"].(map[string]any)
	assert.Equal(t, "PROJ-42", annotations["example.com/ticket"])
	assert.Equal(t, "opmodel.dev/modules", annotations["module.opmodel.dev/path"])
	assert.Equal(t, "opmodel.dev/modules/zot-registry:0.1.0", annotations["module.opmodel.dev/fqn"])
	assert.Equal(t, "Production-ready Zot OCI registry", annotations["module.opmodel.dev/description"])
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", annotations["module.opmodel.dev/uuid"])
}

// TestBuildXRD_ProvenanceOPMKeysWinOnCollision mirrors the CRD invariant:
// a module author cannot shadow OPM-owned keys by declaring them in
// metadata.labels or metadata.annotations.
func TestBuildXRD_ProvenanceOPMKeysWinOnCollision(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {
	name:    "svc"
	version: "1.0.0"
	modulePath: "opmodel.dev/modules"
	labels: {
		"app.kubernetes.io/managed-by": "malicious-tool"
		"module.opmodel.dev/name":      "imposter"
		"module.opmodel.dev/version":   "9.9.9"
	}
	annotations: {
		"module.opmodel.dev/path": "attacker.example/modules"
	}
}
#config: {}
`)
	require.NoError(t, modVal.Err())

	xrd, err := BuildXRD(modVal, XRDOptions{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	meta := xrd.Object["metadata"].(map[string]any)

	labels := meta["labels"].(map[string]any)
	assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "svc", labels["module.opmodel.dev/name"])
	assert.Equal(t, "1.0.0", labels["module.opmodel.dev/version"])

	annotations := meta["annotations"].(map[string]any)
	assert.Equal(t, "opmodel.dev/modules", annotations["module.opmodel.dev/path"])
}

// TestBuildXRD_ReservedCrossplaneField rejects a #config whose top-level
// properties include "crossplane" — the field would collide with Crossplane
// v2's own reserved spec.crossplane.*.
func TestBuildXRD_ReservedCrossplaneField(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {name: "svc", version: "1.0.0"}
#config: {
	crossplane: { foo: string }
	other: string
}
`)
	require.NoError(t, modVal.Err())

	_, err := BuildXRD(modVal, XRDOptions{Group: "module.opmodel.dev"})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "crossplane")
	assert.Contains(t, err.Error(), "reserves")
}

func TestBuildXRD_Errors(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()

	tests := []struct {
		name    string
		src     string
		opts    XRDOptions
		wantErr string
	}{
		{
			name: "missing group",
			src: `
metadata: {name: "svc", version: "1.0.0"}
#config: {}
`,
			opts:    XRDOptions{},
			wantErr: "XRD group is required",
		},
		{
			name: "invalid scope",
			src: `
metadata: {name: "svc", version: "1.0.0"}
#config: {}
`,
			opts:    XRDOptions{Group: "module.opmodel.dev", Scope: XRDScope("Bogus")},
			wantErr: "invalid XRD scope",
		},
		{
			name: "missing metadata.name",
			src: `
metadata: {version: "1.0.0"}
#config: {}
`,
			opts:    XRDOptions{Group: "module.opmodel.dev"},
			wantErr: "metadata.name is not set",
		},
		{
			name: "missing metadata.version",
			src: `
metadata: {name: "svc"}
#config: {}
`,
			opts:    XRDOptions{Group: "module.opmodel.dev"},
			wantErr: "metadata.version is not set",
		},
		{
			name:    "missing #config",
			src:     `metadata: {name: "svc", version: "1.0.0"}`,
			opts:    XRDOptions{Group: "module.opmodel.dev"},
			wantErr: "no #config definition",
		},
		{
			name: "invalid name (starts with digit)",
			src: `
metadata: {name: "123svc", version: "1.0.0"}
#config: {}
`,
			opts:    XRDOptions{Group: "module.opmodel.dev"},
			wantErr: "invalid CRD kind",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			modVal := ctx.CompileString(tt.src)
			require.NoError(t, modVal.Err())

			_, err := BuildXRD(modVal, tt.opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
