package moduleconstruction

// ---------------------------------------------------------------------------
// Approach B: Selective Raw Transform
//
// Strategy: keep mod.Raw as the base — never decompose it. Instead, apply
// targeted FillPath overrides before passing to the builder. The builder
// constructs a "prepared" module value and passes it to #ModuleRelease.
//
//   rawModule := loadModuleRaw(...)                  // mod.Raw equivalent
//   defaults  := defaultValues(...)                  // from values.cue
//
//   // Builder selects values (user override or module defaults)
//   selectedValues := defaults  // or user-provided values
//
//   // Optionally pre-transform Raw before injection
//   prepared := rawModule
//       // FillPath can be used here to override specific fields if needed
//
//   releaseSchema.FillPath(cue.Def("module"), prepared).
//       FillPath("values", selectedValues)...
//
// Approach B sidesteps the cross-reference question entirely because mod.Raw
// is never decomposed — the #config <-> #components linkage in the original
// CUE evaluation is always intact.
//
// The key questions for B are:
//   B1: Can you FillPath values onto Raw (values: _ → values: concrete)?
//   B2: Can targeted metadata overrides be applied to Raw via FillPath?
//   B3: Does pre-filling values on the module conflict with the release schema?
//       (#ModuleRelease has its own "values" field — will double-filling error?)
//   B4: Does the builder retain full control over which values are injected?
//   B5: Full end-to-end release with Raw-based approach.
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestB_RawModuleHasAbstractValues confirms that mod.Raw (loaded with Pattern A
// filtering) does NOT have concrete values — values.cue was excluded from the
// load, so the "values" field is open/abstract as defined by core.#Module.
func TestB_RawModuleHasAbstractValues(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	rawModule := loadModuleRaw(t, ctx)

	valuesField := rawModule.LookupPath(cue.ParsePath("values"))

	// "values" exists (it is _ in #Module schema) but is not concrete.
	if valuesField.Exists() {
		err := valuesField.Validate(cue.Concrete(true))
		assert.Error(t, err, "values field in mod.Raw should be abstract (values.cue was excluded from load)")
	} else {
		// values field may simply not exist if _ is treated as missing — both are fine.
		t.Log("values field does not exist in mod.Raw (values.cue was excluded from load)")
	}

	// Components spec fields should also be abstract.
	assertFieldAbstract(t, rawModule, "#components.app.spec.container.image")
}

// TestB_FillConcreteValuesOntoRaw tests whether FillPath can inject concrete
// values onto mod.Raw's abstract "values" field. With values: _ in #Module,
// FillPath should narrow it to the provided concrete struct.
func TestB_FillConcreteValuesOntoRaw(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	rawModule := loadModuleRaw(t, ctx)

	vals := ctx.CompileString(`{image: "nginx:test", replicas: 2, port: 8080}`)
	require.NoError(t, vals.Err())

	prepared := rawModule.FillPath(cue.ParsePath("values"), vals)

	require.NoError(t, prepared.Err(), "FillPath values onto Raw should not error")

	// The values field should now be concrete.
	assertFieldString(t, prepared, "values.image", "nginx:test")
	assertFieldInt(t, prepared, "values.replicas", 2)
}

// TestB_MetadataOverrideOnRaw documents a key limitation of Approach B:
// metadata fields that are already CONCRETE in mod.Raw (e.g. defaultNamespace: "default"
// set explicitly in module.cue) CANNOT be overridden via FillPath.
//
// CUE unification of two different concrete strings always conflicts:
//
//	"default" & "production" → error: conflicting values
//
// This means Approach B can only FillPath onto open/abstract fields in Raw.
// Concrete metadata fields require a different strategy (Approach C or D)
// where metadata is constructed fresh from Go data.
//
// FINDING: Concrete fields in mod.Raw are immutable via FillPath.
func TestB_MetadataOverrideOnRaw(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	rawModule := loadModuleRaw(t, ctx)

	// Confirm defaultNamespace is concrete in Raw.
	origNS := rawModule.LookupPath(cue.ParsePath("metadata.defaultNamespace"))
	if origNS.Exists() {
		ns, err := origNS.String()
		require.NoError(t, err)
		assert.Equal(t, "default", ns, "defaultNamespace should be concrete 'default' in Raw")
	}

	// Attempting to override a concrete field via FillPath produces a conflict.
	overridden := rawModule.FillPath(
		cue.ParsePath("metadata.defaultNamespace"),
		ctx.CompileString(`"production"`),
	)
	assert.Error(t, overridden.Err(),
		"FINDING: FillPath cannot override a concrete metadata field — "+
			"'default' & 'production' conflict in CUE unification. "+
			"Approach B cannot mutate concrete Raw fields; use Approach C or D for metadata control.")
}

// TestB_RawInjectionIntoReleasePreservesComponentRefs is the core validation
// for Approach B: because we never decompose mod.Raw, the #config <-> #components
// cross-references are always intact. This test confirms that injecting mod.Raw
// directly into #ModuleRelease produces concrete component values.
//
// This is the "it just works" baseline — mod.Raw is what the current builder uses.
// If this test fails, something is fundamentally broken in the test setup itself.
func TestB_RawInjectionIntoReleasePreservesComponentRefs(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	result := fillRelease(releaseSchema, rawModule, "test-release", "default", `{
		image:    "nginx:baseline"
		replicas: 1
		port:     8080
	}`)

	require.NoError(t, result.Err(), "raw injection into release should not error")
	assertConcrete(t, result, "release with raw module should be fully concrete")
	assertFieldString(t, result, "components.app.spec.container.image", "nginx:baseline")
	assertFieldInt(t, result, "components.app.spec.scaling.count", 1)
}

// TestB_PreFilledValuesOnRawDoNotConflictWithRelease tests whether pre-filling
// values on mod.Raw before injection causes a conflict with the release schema's
// own "values" field injection.
//
// The #ModuleRelease schema has:
//
//	values: close(#module.#config)
//
// And the builder injects:
//
//	.FillPath("values", selectedValues)
//
// If the module already has values filled in, does FillPath("values", ...) on
// the release schema conflict? This would only be a problem if the release schema
// unifies the module's values field with its own — which it does NOT (they are
// separate paths: #module.values vs release-level values).
func TestB_PreFilledValuesOnRawDoNotConflictWithRelease(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	// Pre-fill values on mod.Raw (as if the builder did this before injection).
	defaultVals := defaultValues(t, ctx)
	preparedModule := rawModule.FillPath(cue.ParsePath("values"), defaultVals)
	require.NoError(t, preparedModule.Err(), "pre-filling values on Raw should not error")

	// Now inject into release AND also provide values at the release level.
	// These are different fields: #module.values vs release.values.
	result := fillReleaseWithValue(releaseSchema, preparedModule, "test-release", "default", defaultVals)

	assert.NoError(t, result.Err(),
		"pre-filled module values should not conflict with release-level values injection")
}

// TestB_BuilderControlsWhichValuesToInject demonstrates the core value of B:
// the builder has full control over which values end up in the release, regardless
// of what is stored in mod.Raw. Module defaults and user-provided values are
// both tested — the builder selects one.
func TestB_BuilderControlsWhichValuesToInject(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	// Case 1: builder selects module defaults.
	defaultVals := defaultValues(t, ctx)
	releaseWithDefaults := fillReleaseWithValue(
		releaseSchema, rawModule, "default-release", "staging", defaultVals,
	)
	require.NoError(t, releaseWithDefaults.Err())
	assertFieldString(t, releaseWithDefaults, "components.app.spec.container.image", "nginx:latest")
	assertFieldInt(t, releaseWithDefaults, "components.app.spec.scaling.count", 1)

	// Case 2: builder selects user-provided values (override).
	releaseWithUserVals := fillRelease(
		releaseSchema, rawModule, "user-release", "staging", `{
			image:    "myapp:v2.0.0"
			replicas: 5
			port:     3000
		}`,
	)
	require.NoError(t, releaseWithUserVals.Err())
	assertFieldString(t, releaseWithUserVals, "components.app.spec.container.image", "myapp:v2.0.0")
	assertFieldInt(t, releaseWithUserVals, "components.app.spec.scaling.count", 5)
}

// TestB_ReleaseEndToEnd verifies the full Approach B pipeline:
// mod.Raw → no decomposition → #ModuleRelease FillPath → concrete release.
func TestB_ReleaseEndToEnd(t *testing.T) {
	requireRegistry(t)
	ctx := newCtx()
	coreVal := loadCore(t, ctx)
	rawModule := loadModuleRaw(t, ctx)
	releaseSchema := releaseSchemaFrom(t, coreVal)

	result := fillRelease(releaseSchema, rawModule, "myapp-prod", "production", `{
		image:    "myapp:v1.0.0"
		replicas: 3
		port:     8443
	}`)

	require.NoError(t, result.Err(), "end-to-end release construction should not error")
	assertConcrete(t, result, "release should be fully concrete")

	// Release metadata.
	assertFieldString(t, result, "metadata.name", "myapp-prod")
	assertFieldString(t, result, "metadata.namespace", "production")
	assertFieldString(t, result, "metadata.version", "0.1.0")

	// Component values resolved correctly.
	assertFieldString(t, result, "components.app.spec.container.image", "myapp:v1.0.0")
	assertFieldInt(t, result, "components.app.spec.scaling.count", 3)

	// Values at release level.
	assertFieldString(t, result, "values.image", "myapp:v1.0.0")
	assertFieldInt(t, result, "values.replicas", 3)
}
