package moduleconstruction

// ---------------------------------------------------------------------------
// Approach A: FillPath Assembly
//
// Strategy: extract #config, #components, and metadata individually from
// mod.Raw via LookupPath, then fill them one-by-one into the #Module schema
// to reconstruct a module cue.Value without relying on mod.Raw directly.
//
//   rawModule := loadModuleRaw(...)
//
//   assembled := moduleSchema.
//       FillPath("metadata",     rawModule.LookupPath("metadata")).
//       FillPath("#config",      rawModule.LookupPath("#config")).
//       FillPath("#components",  rawModule.LookupPath("#components"))
//
// The critical question is A3: do the cross-references between #config and
// #components survive the extract-and-reinject cycle? Specifically, the
// components.cue file contains:
//
//   spec: {
//       container: image: #config.image     // ← cross-reference
//       scaling: count:   #config.replicas  // ← cross-reference
//   }
//
// When #components is extracted via LookupPath and reinjected into a new
// parent, does CUE maintain the binding of #config.image to the new parent's
// #config, or is the reference broken (leaving image: string — still abstract)?
//
// Tests A1-A2 confirm extraction works.
// Test A3 is the gate: cross-refs must survive for this approach to be viable.
// Tests A4-A6 verify end-to-end release construction if A3 passes.
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestA_ExtractConfigViaLookupPath confirms that #config can be extracted from
// mod.Raw without error and contains the expected constraint fields.
func TestA_ExtractConfigViaLookupPath(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	rawModule := loadModuleRaw(t, ctx)

	configVal := rawModule.LookupPath(cue.ParsePath("#config"))

	require.True(t, configVal.Exists(), "#config should exist in mod.Raw")
	require.NoError(t, configVal.Err(), "#config should not be errored")

	// The fields should exist but remain abstract (constraints, not concrete values).
	imageField := configVal.LookupPath(cue.ParsePath("image"))
	require.True(t, imageField.Exists(), "#config.image should exist")
	assert.Error(t, imageField.Validate(cue.Concrete(true)),
		"#config.image should be abstract (constraint: string)")

	replicasField := configVal.LookupPath(cue.ParsePath("replicas"))
	require.True(t, replicasField.Exists(), "#config.replicas should exist")
	assert.Error(t, replicasField.Validate(cue.Concrete(true)),
		"#config.replicas should be abstract (constraint: int & >=1)")
}

// TestA_ExtractComponentsViaLookupPath confirms that #components can be
// extracted from mod.Raw and contains the expected "app" component.
func TestA_ExtractComponentsViaLookupPath(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	rawModule := loadModuleRaw(t, ctx)

	compsVal := rawModule.LookupPath(cue.ParsePath("#components"))

	require.True(t, compsVal.Exists(), "#components should exist in mod.Raw")
	require.NoError(t, compsVal.Err(), "#components should not be errored")

	appComp := compsVal.LookupPath(cue.ParsePath("app"))
	require.True(t, appComp.Exists(), "#components.app should exist")

	// The component's spec.container.image should exist but be abstract
	// (it references #config.image which is not yet concrete).
	imageField := appComp.LookupPath(cue.ParsePath("spec.container.image"))
	require.True(t, imageField.Exists(), "spec.container.image should exist in app component")
	assert.Error(t, imageField.Validate(cue.Concrete(true)),
		"spec.container.image should be abstract before values are injected")
}

// TestA_FillExtractedPartsIntoModuleSchema is the assembly test: extract
// metadata, #config, and #components from mod.Raw, then fill them individually
// into the #Module schema. This confirms the mechanical assembly works without
// CUE errors — independent of whether cross-refs are preserved.
func TestA_FillExtractedPartsIntoModuleSchema(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)

	metadataVal := rawModule.LookupPath(cue.ParsePath("metadata"))
	configVal := rawModule.LookupPath(cue.ParsePath("#config"))
	compsVal := rawModule.LookupPath(cue.ParsePath("#components"))

	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), metadataVal).
		FillPath(cue.ParsePath("#config"), configVal).
		FillPath(cue.ParsePath("#components"), compsVal)

	assert.NoError(t, assembled.Err(),
		"assembling parts into #Module schema via FillPath should not error")
	assert.True(t, assembled.Exists(), "assembled value should exist")
}

// TestA_CrossRefsSurviveReassembly is the gate test for Approach A.
//
// After assembling a module from extracted parts, we inject it into
// #ModuleRelease with concrete values and check whether the component's
// spec.container.image resolves to the concrete injected value.
//
// If cross-refs survived: components.app.spec.container.image == "nginx:test"
// If cross-refs broke:    components.app.spec.container.image is still abstract
//
// This is the key question: does CUE maintain the #config.image binding when
// #components is extracted from one parent and injected into a new parent?
func TestA_CrossRefsSurviveReassembly(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	// Assemble module from extracted parts.
	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), rawModule.LookupPath(cue.ParsePath("metadata"))).
		FillPath(cue.ParsePath("#config"), rawModule.LookupPath(cue.ParsePath("#config"))).
		FillPath(cue.ParsePath("#components"), rawModule.LookupPath(cue.ParsePath("#components")))
	require.NoError(t, assembled.Err(), "assembly should not error")

	// Inject assembled module into #ModuleRelease with concrete values.
	result := fillRelease(releaseSchema, assembled, "test-release", "default", `{
		image:    "nginx:test"
		replicas: 3
		port:     9090
	}`)

	if err := result.Err(); err != nil {
		t.Logf("RESULT: cross-refs appear BROKEN — FillPath errored: %v", err)
		t.Logf("This means #config.image in #components lost its binding after extract+reinject.")
		t.FailNow()
	}

	// Check whether components resolved to concrete values.
	imageField := result.LookupPath(cue.ParsePath("components.app.spec.container.image"))
	if !imageField.Exists() {
		t.Log("RESULT: components.app.spec.container.image does not exist in release result")
		t.FailNow()
	}

	imageStr, err := imageField.String()
	if err != nil {
		t.Logf("RESULT: cross-refs BROKEN — spec.container.image is not concrete: %v", err)
		t.Logf("The #config.image reference in #components did not bind to the injected values.")
		t.Logf("Approach A cannot preserve cross-references through extract-and-reinject.")
		t.FailNow()
	}

	t.Logf("RESULT: cross-refs SURVIVED — spec.container.image = %q", imageStr)
	assert.Equal(t, "nginx:test", imageStr,
		"spec.container.image should resolve to the injected value")
}

// TestA_ReleaseEndToEnd documents the downstream consequence of A3 failing:
// because cross-refs are broken after extract-and-reinject, the release is NOT
// fully concrete — components.app.spec.container.image remains "string" (abstract).
//
// FINDING: Approach A is NOT viable for end-to-end release construction.
func TestA_ReleaseEndToEnd(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), rawModule.LookupPath(cue.ParsePath("metadata"))).
		FillPath(cue.ParsePath("#config"), rawModule.LookupPath(cue.ParsePath("#config"))).
		FillPath(cue.ParsePath("#components"), rawModule.LookupPath(cue.ParsePath("#components")))
	require.NoError(t, assembled.Err())

	result := fillRelease(releaseSchema, assembled, "myapp", "production", `{
		image:    "myapp:v1.2.3"
		replicas: 2
		port:     8080
	}`)

	// The release will not be concrete because cross-refs in components are broken.
	// This confirms Approach A is not viable.
	concreteErr := result.Validate(cue.Concrete(true))
	assert.Error(t, concreteErr,
		"FINDING: Approach A release is not concrete — cross-refs broken, component image is still 'string'")
	t.Logf("Concrete validation error (expected): %v", concreteErr)
}

// TestA_AssembledModuleSatisfiesSchema verifies that the assembled value
// satisfies the #Module constraint (not just structurally — it must unify
// with #Module without error).
func TestA_AssembledModuleSatisfiesSchema(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)

	moduleSchema := moduleSchemaFrom(t, coreVal)

	assembled := moduleSchema.
		FillPath(cue.ParsePath("metadata"), rawModule.LookupPath(cue.ParsePath("metadata"))).
		FillPath(cue.ParsePath("#config"), rawModule.LookupPath(cue.ParsePath("#config"))).
		FillPath(cue.ParsePath("#components"), rawModule.LookupPath(cue.ParsePath("#components")))

	require.NoError(t, assembled.Err(), "assembled value should unify with #Module without error")

	// Verify key metadata fields resolved (they are CUE-computed: FQN, UUID, labels).
	fqn := assembled.LookupPath(cue.ParsePath("metadata.fqn"))
	assert.True(t, fqn.Exists(), "metadata.fqn should exist in assembled module")

	uuid := assembled.LookupPath(cue.ParsePath("metadata.uuid"))
	assert.True(t, uuid.Exists(), "metadata.uuid should exist in assembled module")
}
