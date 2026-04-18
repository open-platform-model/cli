package crd

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBuildCRD_FullAssembly(t *testing.T) {
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

	crd, err := BuildCRD(modVal, Options{Group: "opmodel.dev"})
	require.NoError(t, err)

	// Top-level fields.
	assert.Equal(t, "apiextensions.k8s.io/v1", crd.Object["apiVersion"])
	assert.Equal(t, "CustomResourceDefinition", crd.Object["kind"])

	meta := crd.Object["metadata"].(map[string]any)
	assert.Equal(t, "myservices.module.opmodel.dev", meta["name"])

	// Minimal provenance: managed-by + name/version labels are always present.
	labels := meta["labels"].(map[string]any)
	assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "my-service", labels["module.opmodel.dev/name"])
	assert.Equal(t, "0.1.0", labels["module.opmodel.dev/version"])

	// No annotations when no optional metadata was provided.
	_, hasAnnotations := meta["annotations"]
	assert.False(t, hasAnnotations)

	spec := crd.Object["spec"].(map[string]any)
	assert.Equal(t, "module.opmodel.dev", spec["group"])
	assert.Equal(t, "Namespaced", spec["scope"])

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
	assert.Equal(t, true, v0["storage"])
	_, hasSubresources := v0["subresources"]
	assert.False(t, hasSubresources, "status subresource must not be emitted in POC")

	schema := v0["schema"].(map[string]any)
	openAPI := schema["openAPIV3Schema"].(map[string]any)
	assert.Equal(t, "object", openAPI["type"])

	props := openAPI["properties"].(map[string]any)
	assert.Contains(t, props, "name")
	assert.Contains(t, props, "replicas")
	assert.Contains(t, props, "tag")
}

func TestBuildCRD_VersionMappingMajor2(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {name: "my-service", version: "2.3.1"}
#config: {name: string}
`)
	require.NoError(t, modVal.Err())

	crd, err := BuildCRD(modVal, Options{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	spec := crd.Object["spec"].(map[string]any)
	versions := spec["versions"].([]any)
	v0 := versions[0].(map[string]any)
	assert.Equal(t, "v2", v0["name"])
}

func TestBuildCRD_CustomGroup(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()
	modVal := ctx.CompileString(`
metadata: {name: "widget", version: "1.0.0"}
#config: {label: string}
`)
	require.NoError(t, modVal.Err())

	crd, err := BuildCRD(modVal, Options{Group: "example.com"})
	require.NoError(t, err)

	meta := crd.Object["metadata"].(map[string]any)
	assert.Equal(t, "widgets.example.com", meta["name"])

	spec := crd.Object["spec"].(map[string]any)
	assert.Equal(t, "example.com", spec["group"])
}

// TestBuildCRD_ProvenanceFullMetadata exercises the path where the module
// declares all optional metadata fields (description, modulePath, fqn, uuid,
// labels, annotations). Mirrors the zot_registry style.
func TestBuildCRD_ProvenanceFullMetadata(t *testing.T) {
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

	crd, err := BuildCRD(modVal, Options{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	meta := crd.Object["metadata"].(map[string]any)

	labels := meta["labels"].(map[string]any)
	// Module-declared label passes through.
	assert.Equal(t, "registry", labels["app.kubernetes.io/component"])
	// OPM-owned labels are always present.
	assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "zot-registry", labels["module.opmodel.dev/name"])
	assert.Equal(t, "0.1.0", labels["module.opmodel.dev/version"])

	annotations := meta["annotations"].(map[string]any)
	// Module-declared annotation passes through.
	assert.Equal(t, "PROJ-42", annotations["example.com/ticket"])
	// All optional provenance annotations are populated from metadata.*.
	assert.Equal(t, "opmodel.dev/modules", annotations["module.opmodel.dev/path"])
	assert.Equal(t, "opmodel.dev/modules/zot-registry:0.1.0", annotations["module.opmodel.dev/fqn"])
	assert.Equal(t, "Production-ready Zot OCI registry", annotations["module.opmodel.dev/description"])
	assert.Equal(t, "11111111-2222-3333-4444-555555555555", annotations["module.opmodel.dev/uuid"])
}

// TestBuildCRD_ProvenanceOPMKeysWinOnCollision guards the invariant that a
// module author cannot shadow OPM-owned keys by declaring them in
// metadata.labels or metadata.annotations — otherwise a module could, e.g.,
// claim to be managed by something other than OPM.
func TestBuildCRD_ProvenanceOPMKeysWinOnCollision(t *testing.T) {
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

	crd, err := BuildCRD(modVal, Options{Group: "module.opmodel.dev"})
	require.NoError(t, err)

	meta := crd.Object["metadata"].(map[string]any)

	labels := meta["labels"].(map[string]any)
	assert.Equal(t, "opm-cli", labels["app.kubernetes.io/managed-by"])
	assert.Equal(t, "svc", labels["module.opmodel.dev/name"])
	assert.Equal(t, "1.0.0", labels["module.opmodel.dev/version"])

	annotations := meta["annotations"].(map[string]any)
	assert.Equal(t, "opmodel.dev/modules", annotations["module.opmodel.dev/path"])
}

func TestBuildCRD_Errors(t *testing.T) {
	t.Parallel()

	ctx := cuecontext.New()

	tests := []struct {
		name    string
		src     string
		opts    Options
		wantErr string
	}{
		{
			name: "missing group",
			src: `
metadata: {name: "svc", version: "1.0.0"}
#config: {}
`,
			opts:    Options{},
			wantErr: "CRD group is required",
		},
		{
			name: "missing metadata.name",
			src: `
metadata: {version: "1.0.0"}
#config: {}
`,
			opts:    Options{Group: "module.opmodel.dev"},
			wantErr: "metadata.name is not set",
		},
		{
			name: "missing metadata.version",
			src: `
metadata: {name: "svc"}
#config: {}
`,
			opts:    Options{Group: "module.opmodel.dev"},
			wantErr: "metadata.version is not set",
		},
		{
			name:    "missing #config",
			src:     `metadata: {name: "svc", version: "1.0.0"}`,
			opts:    Options{Group: "module.opmodel.dev"},
			wantErr: "no #config definition",
		},
		{
			name: "invalid name (starts with digit)",
			src: `
metadata: {name: "123svc", version: "1.0.0"}
#config: {}
`,
			opts:    Options{Group: "module.opmodel.dev"},
			wantErr: "invalid CRD kind",
		},
		{
			name: "invalid version",
			src: `
metadata: {name: "svc", version: "not-a-version"}
#config: {}
`,
			opts:    Options{Group: "module.opmodel.dev"},
			wantErr: "non-numeric major",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			t.Parallel()

			modVal := ctx.CompileString(tt.src)
			require.NoError(t, modVal.Err())

			_, err := BuildCRD(modVal, tt.opts)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.wantErr)
		})
	}
}
