package moduleconstruction

// ---------------------------------------------------------------------------
// Approach C: Compile + Inject Hybrid
//
// Strategy: compile Go-native metadata as a CUE text literal, then inject
// the CUE-native parts (#config, #components) from the original evaluation
// via FillPath. This avoids relying on mod.Raw entirely by building the
// module value from two sources:
//
//   1. Metadata: compiled from Go strings (ctx.CompileString)
//   2. #config + #components: injected as cue.Values (LookupPath from rawModule)
//
//   metaCUE  := ctx.CompileString(`{apiVersion: "...", name: "...", version: "..."}`)
//   assembled := moduleSchema.
//       FillPath("metadata",    metaCUE).
//       FillPath("#config",     rawModule.LookupPath("#config")).
//       FillPath("#components", rawModule.LookupPath("#components"))
//
// Approach C differs from A in the metadata source: A extracts metadata from
// mod.Raw (a cue.Value), while C compiles it fresh from Go data. This gives
// the builder explicit control over every metadata field.
//
// The critical question is C3 (same gate as A3): after assembling via compile
// + inject, do #config cross-references in #components resolve when values
// are injected into the release?
//
// Approach C also tests C4: whether CUE-computed fields (FQN interpolation,
// UUID via uid.SHA1, standard labels) evaluate correctly on a value assembled
// from compiled text + injected cue.Values — since these computations require
// the full #Module schema context to resolve.
// ---------------------------------------------------------------------------

import (
	"fmt"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// compiledMetadata builds a CUE struct literal from Go-native metadata fields.
// This simulates what the builder would do: take mod.Metadata (Go struct) and
// produce a cue.Value for injection into the module schema.
func compiledMetadata(ctx *cue.Context) cue.Value {
	// These values match the simple_module fixture.
	src := fmt.Sprintf(`{
		apiVersion:       %q
		name:             %q
		version:          %q
		defaultNamespace: %q
	}`,
		"test.dev/simple-module@v0",
		"simple",
		"0.1.0",
		"default",
	)
	return ctx.CompileString(src)
}

// TestC_MetadataCompilesFromGoStrings confirms that Go-native metadata can be
// expressed as a CUE struct literal and compiled without error.
func TestC_MetadataCompilesFromGoStrings(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()

	meta := compiledMetadata(ctx)
	require.NoError(t, meta.Err(), "metadata compiled from Go strings should not error")

	assertFieldString(t, meta, "apiVersion", "test.dev/simple-module@v0")
	assertFieldString(t, meta, "name", "simple")
	assertFieldString(t, meta, "version", "0.1.0")
}

// TestC_HybridAssemblyProducesValidValue tests that mixing compiled metadata
// (from Go strings) with FillPath-injected cue.Values (#config, #components)
// produces a non-errored cue.Value.
func TestC_HybridAssemblyProducesValidValue(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)
	meta := compiledMetadata(ctx)

	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), meta).
		FillPath(cue.ParsePath("#config"), rawModule.LookupPath(cue.ParsePath("#config"))).
		FillPath(cue.ParsePath("#components"), rawModule.LookupPath(cue.ParsePath("#components")))

	assert.NoError(t, assembled.Err(),
		"hybrid assembly (compiled metadata + injected CUE values) should not error")
	assert.True(t, assembled.Exists(), "assembled value should exist")
}

// TestC_CrossRefsSurviveHybridAssembly is the gate test for Approach C.
//
// Same question as A3, but via the hybrid assembly path. After building a module
// from compiled metadata + injected #config/#components, do the cross-references
// between #config and #components resolve when values are injected into the release?
//
// If cross-refs survived: components.app.spec.container.image == "nginx:hybrid"
// If cross-refs broke:    components.app.spec.container.image is still abstract
//
// Note: #config and #components both come from the SAME rawModule evaluation
// in this approach, so they share the same CUE evaluation context. The question
// is whether that shared context is preserved after they are injected into the
// new parent (moduleSchema).
func TestC_CrossRefsSurviveHybridAssembly(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)
	releaseSchema := releaseSchemaFrom(t, coreVal)
	meta := compiledMetadata(ctx)

	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), meta).
		FillPath(cue.ParsePath("#config"), rawModule.LookupPath(cue.ParsePath("#config"))).
		FillPath(cue.ParsePath("#components"), rawModule.LookupPath(cue.ParsePath("#components")))
	require.NoError(t, assembled.Err(), "hybrid assembly should not error")

	result := fillRelease(releaseSchema, assembled, "hybrid-release", "default", `{
		image:    "nginx:hybrid"
		replicas: 4
		port:     9000
	}`)

	if err := result.Err(); err != nil {
		t.Logf("RESULT: cross-refs appear BROKEN — FillPath errored: %v", err)
		t.FailNow()
	}

	imageField := result.LookupPath(cue.ParsePath("components.app.spec.container.image"))
	if !imageField.Exists() {
		t.Log("RESULT: components.app.spec.container.image does not exist")
		t.FailNow()
	}

	imageStr, err := imageField.String()
	if err != nil {
		t.Logf("RESULT: cross-refs BROKEN — spec.container.image is not concrete: %v", err)
		t.Logf("Approach C cannot preserve #config cross-references through hybrid assembly.")
		t.FailNow()
	}

	t.Logf("RESULT: cross-refs SURVIVED — spec.container.image = %q", imageStr)
	assert.Equal(t, "nginx:hybrid", imageStr)
}

// TestC_ComputedFieldsEvaluateOnAssembled tests whether CUE-computed fields in
// the #Module schema evaluate correctly on the hybrid-assembled value.
//
// The #Module schema computes:
//
//	metadata.fqn  = "\(metadata.apiVersion)#\(_definitionName)"
//	metadata.uuid = uid.SHA1(OPMNamespace, "\(fqn):\(version)")
//	metadata.labels["module.opmodel.dev/name"] = "\(name)"
//	etc.
//
// These computations use CUE interpolation and the uuid package import.
// When metadata is compiled from Go strings and injected, do CUE still
// evaluate these derived fields? Or do they require the full module loading
// context to resolve?
func TestC_ComputedFieldsEvaluateOnAssembled(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)
	meta := compiledMetadata(ctx)

	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), meta).
		FillPath(cue.ParsePath("#config"), rawModule.LookupPath(cue.ParsePath("#config"))).
		FillPath(cue.ParsePath("#components"), rawModule.LookupPath(cue.ParsePath("#components")))
	require.NoError(t, assembled.Err())

	// FQN should be computed from apiVersion + name (KebabToPascal).
	// "test.dev/simple-module@v0" + "simple" → "test.dev/simple-module@v0#Simple"
	fqnVal := assembled.LookupPath(cue.ParsePath("metadata.fqn"))
	if fqnVal.Exists() {
		if fqn, err := fqnVal.String(); err == nil {
			t.Logf("RESULT: metadata.fqn evaluated = %q", fqn)
		} else {
			t.Logf("RESULT: metadata.fqn exists but is not concrete: %v", err)
		}
	} else {
		t.Log("RESULT: metadata.fqn does not exist on assembled value")
	}

	// UUID should be computed via uid.SHA1.
	uuidVal := assembled.LookupPath(cue.ParsePath("metadata.uuid"))
	if uuidVal.Exists() {
		if uuid, err := uuidVal.String(); err == nil {
			t.Logf("RESULT: metadata.uuid evaluated = %q", uuid)
		} else {
			t.Logf("RESULT: metadata.uuid exists but is not concrete: %v", err)
		}
	} else {
		t.Log("RESULT: metadata.uuid does not exist on assembled value")
	}

	// Standard label: module.opmodel.dev/name should equal metadata.name.
	labelVal := assembled.LookupPath(cue.ParsePath(`metadata.labels."module.opmodel.dev/name"`))
	if labelVal.Exists() {
		if label, err := labelVal.String(); err == nil {
			t.Logf("RESULT: module name label evaluated = %q", label)
			assert.Equal(t, "simple", label, "name label should match module name")
		} else {
			t.Logf("RESULT: name label exists but is not concrete: %v", err)
		}
	} else {
		t.Log("RESULT: module name label does not exist on assembled value")
	}
}

// TestC_BuilderCanControlMetadataFields demonstrates the key benefit of C:
// the builder can construct metadata from Go data and override any field
// without needing to extract and patch mod.Raw. This is the "control" goal.
func TestC_BuilderCanControlMetadataFields(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	// Builder explicitly constructs metadata — can set any field precisely.
	customMeta := ctx.CompileString(`{
		apiVersion:       "test.dev/simple-module@v0"
		name:             "simple"
		version:          "0.1.0"
		defaultNamespace: "custom-namespace"
		description:      "Overridden by builder"
	}`)
	require.NoError(t, customMeta.Err())

	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), customMeta).
		FillPath(cue.ParsePath("#config"), rawModule.LookupPath(cue.ParsePath("#config"))).
		FillPath(cue.ParsePath("#components"), rawModule.LookupPath(cue.ParsePath("#components")))
	require.NoError(t, assembled.Err())

	result := fillRelease(releaseSchema, assembled, "c-control-test", "default", `{
		image:    "nginx:control"
		replicas: 1
		port:     80
	}`)

	assert.NoError(t, result.Err(), "release with builder-controlled metadata should not error")
}

// TestC_ReleaseEndToEnd documents the downstream consequence of C3 failing:
// same result as Approach A — cross-refs broken, release not concrete.
//
// Note: metadata, FQN, UUID, and labels DO resolve correctly (TestC_ComputedFieldsEvaluateOnAssembled
// passes). The failure is specifically in #components cross-refs to #config.
//
// FINDING: Approach C is NOT viable for end-to-end release construction.
// The hybrid assembly (compiled metadata + injected config/components) does not
// preserve the #config.image cross-reference in #components.
func TestC_ReleaseEndToEnd(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)
	releaseSchema := releaseSchemaFrom(t, coreVal)
	meta := compiledMetadata(ctx)

	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), meta).
		FillPath(cue.ParsePath("#config"), rawModule.LookupPath(cue.ParsePath("#config"))).
		FillPath(cue.ParsePath("#components"), rawModule.LookupPath(cue.ParsePath("#components")))
	require.NoError(t, assembled.Err())

	result := fillRelease(releaseSchema, assembled, "c-release", "production", `{
		image:    "service:v3.0.0"
		replicas: 5
		port:     443
	}`)

	concreteErr := result.Validate(cue.Concrete(true))
	assert.Error(t, concreteErr,
		"FINDING: Approach C release is not concrete — cross-refs broken same as Approach A")
	t.Logf("Concrete validation error (expected): %v", concreteErr)
}
