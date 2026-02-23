package moduleconstruction

// ---------------------------------------------------------------------------
// Approach D: Module.Encode() — Go-native struct with Encode method
//
// Strategy: instead of passing mod.Raw or decomposed cue.Values directly, the
// Module struct holds its constituent parts and exposes an Encode(ctx) method
// that reconstructs a cue.Value for the builder. The builder calls Encode()
// and passes the result to FillPath — no Raw field needed.
//
// The moduleForRelease struct defined here mirrors what module.Module would
// look like after removing the Raw field:
//
//   type moduleForRelease struct {
//       Metadata   moduleMetadata  // Go-native (from mod.Metadata)
//       Config     cue.Value       // #config constraints (from mod.Config)
//       Components cue.Value       // #components as ONE value (not decomposed)
//   }
//
// Key design choice for D: Components is held as a SINGLE cue.Value (the full
// #components struct), NOT as the current map[string]*component.Component.
// Decomposition into the Go map happens AFTER the release is built (from the
// concrete result), not at load time. This preserves internal structure and
// avoids the eager decomposition that currently discards cue.Value cross-refs.
//
// The Encode method tries two internal strategies:
//
//   D-schema:  Uses #Module schema as base (same as Approach A internally).
//   D-compile: Compiles metadata from Go text, injects config/components
//              (same as Approach C internally).
//
// Both strategies face the same cross-reference gate. The value of D is not
// a new assembly mechanism but a cleaner API: the Module struct owns its data,
// Encode() is the explicit conversion point, and the builder has full control
// over what it passes to FillPath(#module, ...).
//
// Additional D-specific test: D4 verifies that modifying a Go field (e.g.
// Metadata.Name) before calling Encode() produces a CUE value reflecting that
// change — demonstrating the "control" benefit over mod.Raw.
// ---------------------------------------------------------------------------

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// moduleForRelease: the experiment struct (no Raw field)
// ---------------------------------------------------------------------------

type moduleMetadata struct {
	APIVersion       string
	Name             string
	Version          string
	DefaultNamespace string
}

// moduleForRelease holds the constituent parts of a module without mod.Raw.
// This is what module.Module would look like after the redesign.
type moduleForRelease struct {
	Metadata   moduleMetadata
	Config     cue.Value // #config: constraints from the module definition
	Components cue.Value // #components: full value, NOT decomposed into Go map
}

// encodeViaSchema assembles a cue.Value by filling extracted parts into the
// #Module schema (Approach A strategy internally). Returns the assembled value.
func (m *moduleForRelease) encodeViaSchema(ctx *cue.Context, moduleSchema cue.Value) cue.Value {
	metaSrc := fmt.Sprintf(`{
		apiVersion:       %q
		name:             %q
		version:          %q
		defaultNamespace: %q
	}`, m.Metadata.APIVersion, m.Metadata.Name, m.Metadata.Version, m.Metadata.DefaultNamespace)

	meta := ctx.CompileString(metaSrc)

	return moduleSchema.
		FillPath(cue.ParsePath("metadata"), meta).
		FillPath(cue.ParsePath("#config"), m.Config).
		FillPath(cue.ParsePath("#components"), m.Components)
}

// encodeViaCompile assembles a cue.Value by compiling a minimal CUE struct
// (metadata only) and injecting config/components via FillPath. Does NOT use
// the #Module schema as base — starts from a compiled struct directly.
func (m *moduleForRelease) encodeViaCompile(ctx *cue.Context) cue.Value {
	src := fmt.Sprintf(`{
		apiVersion: "opmodel.dev/core/v0"
		kind:       "Module"
		metadata: {
			apiVersion:       %q
			name:             %q
			version:          %q
			defaultNamespace: %q
		}
	}`, m.Metadata.APIVersion, m.Metadata.Name, m.Metadata.Version, m.Metadata.DefaultNamespace)

	compiled := ctx.CompileString(src)

	return compiled.
		FillPath(cue.ParsePath("#config"), m.Config).
		FillPath(cue.ParsePath("#components"), m.Components)
}

// newModuleForRelease builds a moduleForRelease from loadModuleRaw output.
// This simulates what the loader would do after the redesign: instead of
// setting mod.Raw, it populates the struct fields from the evaluated value.
//
// Key: Components is the full #components cue.Value — NOT decomposed.
func newModuleForRelease(t *testing.T, rawModule cue.Value) moduleForRelease {
	t.Helper()

	nameVal := rawModule.LookupPath(cue.ParsePath("metadata.name"))
	name, _ := nameVal.String()

	versionVal := rawModule.LookupPath(cue.ParsePath("metadata.version"))
	version, _ := versionVal.String()

	apiVersionVal := rawModule.LookupPath(cue.ParsePath("metadata.apiVersion"))
	apiVersion, _ := apiVersionVal.String()

	defaultNSVal := rawModule.LookupPath(cue.ParsePath("metadata.defaultNamespace"))
	defaultNS, _ := defaultNSVal.String()

	return moduleForRelease{
		Metadata: moduleMetadata{
			APIVersion:       apiVersion,
			Name:             name,
			Version:          version,
			DefaultNamespace: defaultNS,
		},
		Config:     rawModule.LookupPath(cue.ParsePath("#config")),
		Components: rawModule.LookupPath(cue.ParsePath("#components")), // held as single value
	}
}

// ---------------------------------------------------------------------------
// Tests
// ---------------------------------------------------------------------------

// TestD_StructPopulatedFromRaw confirms that newModuleForRelease correctly
// populates the struct fields from the raw module evaluation.
func TestD_StructPopulatedFromRaw(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	rawModule := loadModuleRaw(t, ctx)

	mod := newModuleForRelease(t, rawModule)

	assert.Equal(t, "simple", mod.Metadata.Name)
	assert.Equal(t, "0.1.0", mod.Metadata.Version)
	assert.True(t, mod.Config.Exists(), "Config should be populated")
	assert.True(t, mod.Components.Exists(), "Components should be populated")

	// Components is the full #components value — app should be accessible.
	appComp := mod.Components.LookupPath(cue.ParsePath("app"))
	assert.True(t, appComp.Exists(), "#components.app should be accessible on the held value")
}

// TestD_EncodeViaSchemaProducesValue confirms that encodeViaSchema produces
// a non-errored cue.Value from the struct fields.
func TestD_EncodeViaSchemaProducesValue(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	mod := newModuleForRelease(t, rawModule)
	moduleSchema := moduleSchemaFrom(t, coreVal)

	encoded := mod.encodeViaSchema(ctx, moduleSchema)

	assert.NoError(t, encoded.Err(), "encodeViaSchema should not error")
	assert.True(t, encoded.Exists(), "encoded value should exist")
}

// TestD_EncodeViaCompileProducesValue confirms that encodeViaCompile produces
// a non-errored cue.Value from the struct fields.
func TestD_EncodeViaCompileProducesValue(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	rawModule := loadModuleRaw(t, ctx)

	mod := newModuleForRelease(t, rawModule)

	encoded := mod.encodeViaCompile(ctx)

	assert.NoError(t, encoded.Err(), "encodeViaCompile should not error")
	assert.True(t, encoded.Exists(), "encoded value should exist")
}

// TestD_CrossRefsSurviveEncodeViaSchema is the gate test for D-schema strategy.
// Same question as A3/C3 but via the Encode method wrapping the schema approach.
func TestD_CrossRefsSurviveEncodeViaSchema(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	mod := newModuleForRelease(t, rawModule)
	moduleSchema := moduleSchemaFrom(t, coreVal)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	encoded := mod.encodeViaSchema(ctx, moduleSchema)
	require.NoError(t, encoded.Err())

	result := fillRelease(releaseSchema, encoded, "d-schema-release", "default", `{
		image:    "nginx:d-schema"
		replicas: 2
		port:     8080
	}`)

	if err := result.Err(); err != nil {
		t.Logf("RESULT (D-schema): cross-refs BROKEN — FillPath errored: %v", err)
		t.FailNow()
	}

	imageField := result.LookupPath(cue.ParsePath("components.app.spec.container.image"))
	imageStr, err := imageField.String()
	if err != nil {
		t.Logf("RESULT (D-schema): cross-refs BROKEN — image not concrete: %v", err)
		t.FailNow()
	}

	t.Logf("RESULT (D-schema): cross-refs SURVIVED — image = %q", imageStr)
	assert.Equal(t, "nginx:d-schema", imageStr)
}

// TestD_CrossRefsSurviveEncodeViaCompile is the gate test for D-compile strategy.
func TestD_CrossRefsSurviveEncodeViaCompile(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	mod := newModuleForRelease(t, rawModule)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	encoded := mod.encodeViaCompile(ctx)
	require.NoError(t, encoded.Err())

	result := fillRelease(releaseSchema, encoded, "d-compile-release", "default", `{
		image:    "nginx:d-compile"
		replicas: 2
		port:     8080
	}`)

	if err := result.Err(); err != nil {
		t.Logf("RESULT (D-compile): cross-refs BROKEN — FillPath errored: %v", err)
		t.FailNow()
	}

	imageField := result.LookupPath(cue.ParsePath("components.app.spec.container.image"))
	imageStr, err := imageField.String()
	if err != nil {
		t.Logf("RESULT (D-compile): cross-refs BROKEN — image not concrete: %v", err)
		t.FailNow()
	}

	t.Logf("RESULT (D-compile): cross-refs SURVIVED — image = %q", imageStr)
	assert.Equal(t, "nginx:d-compile", imageStr)
}

// TestD_GoFieldModificationReflectsInEncoded demonstrates the key control
// benefit of the Encode() pattern: modifying a Go field before calling Encode
// produces a CUE value reflecting that change. With mod.Raw, you would need to
// FillPath onto the opaque blob; with Encode(), the Go struct is the source of truth.
func TestD_GoFieldModificationReflectsInEncoded(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	mod := newModuleForRelease(t, rawModule)
	moduleSchema := moduleSchemaFrom(t, coreVal)

	// Confirm original name.
	assert.Equal(t, "simple", mod.Metadata.Name)

	// Modify the Go field directly — no FillPath needed.
	mod.Metadata.Name = "simple-override"
	mod.Metadata.DefaultNamespace = "overridden-ns"

	encoded := mod.encodeViaSchema(ctx, moduleSchema)
	require.NoError(t, encoded.Err())

	// The encoded CUE value should reflect the Go-level modification.
	assertFieldString(t, encoded, "metadata.name", "simple-override")
	assertFieldString(t, encoded, "metadata.defaultNamespace", "overridden-ns")
}

// TestD_ComponentsAsUndecomposedValue confirms that holding Components as a
// single cue.Value (not as map[string]*component.Component) provides access to
// the full structure including the "app" component and its cross-ref fields.
//
// This tests the deferred-decomposition design: decompose components AFTER the
// release is built (when values are concrete), not eagerly at load time.
func TestD_ComponentsAsUndecomposedValue(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	rawModule := loadModuleRaw(t, ctx)

	mod := newModuleForRelease(t, rawModule)

	// Full #components value is accessible.
	require.True(t, mod.Components.Exists())

	// Can iterate components.
	iter, err := mod.Components.Fields()
	require.NoError(t, err, "should be able to iterate #components fields")

	found := false
	for iter.Next() {
		if iter.Selector().Unquoted() == "app" {
			found = true
			// The component's spec is still abstract (cross-refs not yet resolved).
			imageField := iter.Value().LookupPath(cue.ParsePath("spec.container.image"))
			if imageField.Exists() {
				assertFieldAbstract(t, iter.Value(), "spec.container.image")
			}
		}
	}
	assert.True(t, found, "app component should be found in undecomposed #components value")
}

// TestD_ReleaseEndToEndViaSchema documents the downstream consequence of
// D-schema cross-ref failure: same as A and C — not viable for end-to-end.
//
// FINDING: Approach D is NOT viable when using either encodeViaSchema or
// encodeViaCompile. Both internally separate #config and #components before
// injecting them into a new parent, which breaks the cross-references.
//
// The Go field modification benefit (TestD_GoFieldModificationReflectsInEncoded)
// IS real and works — the limitation is only in the CUE cross-ref preservation.
func TestD_ReleaseEndToEndViaSchema(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	mod := newModuleForRelease(t, rawModule)
	moduleSchema := moduleSchemaFrom(t, coreVal)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	encoded := mod.encodeViaSchema(ctx, moduleSchema)
	require.NoError(t, encoded.Err())

	result := fillRelease(releaseSchema, encoded, "d-e2e", "production", `{
		image:    "service:stable"
		replicas: 3
		port:     8443
	}`)

	concreteErr := result.Validate(cue.Concrete(true))
	assert.Error(t, concreteErr,
		"FINDING: Approach D (via schema) release is not concrete — same cross-ref breakage as A and C")
	t.Logf("Concrete validation error (expected): %v", concreteErr)
}
