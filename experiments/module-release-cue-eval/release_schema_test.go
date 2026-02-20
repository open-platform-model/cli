package modulereleasecueeval

// ---------------------------------------------------------------------------
// Decision 1: #ModuleRelease is accessible from the locally-loaded catalog
//
// Strategy A begins by loading opmodel.dev/core@v0 from the local catalog
// source directory. The evaluated catalog value must expose #ModuleRelease as
// a non-error, non-concrete cue.Value (it's a schema/constraint, not a value).
//
// These tests prove:
//   - catalogVal.LookupPath("#ModuleRelease") returns an existing value
//   - The value is not errored
//   - The value is NOT concrete (it is a schema constraint)
//   - Key sub-paths (#module, metadata, components, values) exist in the schema
//   - The catalog's built-in _testModuleRelease evaluates successfully
//     (proving the whole evaluation pipeline works end-to-end in pure CUE)
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestSchema_ModuleReleaseAccessible proves that #ModuleRelease can be looked
// up from the locally-loaded catalog and is a non-errored cue.Value.
func TestSchema_ModuleReleaseAccessible(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)

	assert.True(t, schema.Exists(), "#ModuleRelease should exist")
	assert.NoError(t, schema.Err(), "#ModuleRelease should not be errored")
}

// TestSchema_ModuleReleaseIsNotConcrete proves the schema value is a constraint,
// not a concrete instance. Concreteness requires #module, metadata, and values
// to be filled in — before any FillPath, it must remain abstract.
func TestSchema_ModuleReleaseIsNotConcrete(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)

	err := schema.Validate(cue.Concrete(true))
	assert.Error(t, err, "#ModuleRelease schema should not be concrete — it is an unfilled constraint")
}

// TestSchema_RequiredSubPathsExist proves which paths are accessible on the
// #ModuleRelease schema before any FillPath injection.
//
// Key finding: required fields (name!, namespace!) and computed fields that depend
// on them (uuid, version, labels, components, values) are NOT accessible before fill
// because CUE evaluates them as errors when their required inputs are missing.
// Only "metadata" itself (the struct) is accessible on the raw schema.
func TestSchema_RequiredSubPathsExist(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)

	// These top-level fields exist on the schema even before fill.
	topLevelAccessible := []string{
		"metadata",
		"apiVersion",
		"kind",
	}
	for _, p := range topLevelAccessible {
		v := schema.LookupPath(cue.ParsePath(p))
		assert.True(t, v.Exists(), "schema path %q should exist before fill", p)
	}

	// These paths are NOT accessible before fill because they depend on required
	// inputs (#module!, metadata.name!, metadata.namespace!). This is expected CUE
	// behavior — required fields cause dependent computations to be in error state.
	//
	// Note: metadata.labels is accessible at schema level (optional field, defined in schema)
	// but its values are non-concrete. metadata.name and metadata.namespace don't exist
	// because they use the ! (required) marker with no default.
	errorOrMissingBeforeFill := []string{
		"metadata.name",      // name! — required, not set → field not found
		"metadata.namespace", // namespace! — required, not set → field not found
		"metadata.uuid",      // depends on fqn + name + namespace → error
		"metadata.version",   // depends on #module.metadata.version → error
		"components",         // depends on #module → error
		"values",             // depends on #module.#config → error
	}
	for _, p := range errorOrMissingBeforeFill {
		v := schema.LookupPath(cue.ParsePath(p))
		// These either don't exist or are in error state before fill — both are expected.
		hasErr := !v.Exists() || v.Err() != nil
		assert.True(t, hasErr, "schema path %q should be missing or errored before fill (required inputs missing)", p)
	}
	for _, p := range errorOrMissingBeforeFill {
		v := schema.LookupPath(cue.ParsePath(p))
		// These either don't exist or are in error state before fill — both are expected.
		hasErr := !v.Exists() || v.Err() != nil
		assert.True(t, hasErr, "schema path %q should not be concrete before fill (required inputs missing)", p)
	}
}

// TestSchema_ModuleDefPathViaBehavior documents the behavior of cue.Def("module")
// for reading vs. writing to the #module! field.
//
// Key finding: LookupPath(cue.Def("module")) returns "field not found" on the
// schema. This is because #module! is a package-internal definition field that
// is not exported for external LookupPath access.
//
// However, FillPath(cue.Def("module"), val) DOES work — injection succeeds even
// though lookup fails. This is a deliberate CUE Go API asymmetry: FillPath can
// write to definition fields that LookupPath cannot read externally.
func TestSchema_ModuleDefPathViaBehavior(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)

	// LookupPath does NOT find #module — it's a package-internal definition field.
	moduleField := schema.LookupPath(cue.MakePath(cue.Def("module")))
	assert.False(t, moduleField.Exists(),
		"LookupPath(cue.Def(\"module\")) should NOT find #module (it is package-internal)")

	// FillPath DOES work — it can inject into definition fields even if LookupPath can't read them.
	// Use the catalog's _testModule as the injected value.
	testModule := testModuleFromCatalog(t, catalogVal)
	withModule := schema.FillPath(cue.MakePath(cue.Def("module")), testModule)
	assert.NoError(t, withModule.Err(),
		"FillPath(cue.Def(\"module\"), validModule) should succeed even though LookupPath fails")
}

// TestSchema_TestModuleFromCatalogIsValid proves that _testModule from the catalog
// is a non-errored, exists value — it is a known-good input for Decision 2+ tests.
func TestSchema_TestModuleFromCatalogIsValid(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)
	testModule := testModuleFromCatalog(t, catalogVal)

	assert.True(t, testModule.Exists())
	assert.NoError(t, testModule.Err())
}

// TestSchema_CatalogBuiltinReleaseEvaluates proves that _testModuleRelease —
// the catalog's own test instance of #ModuleRelease — evaluates to a concrete
// value with no errors. This is the "control" for the experiment: if even the
// catalog's own test case doesn't work, there is a catalog or environment problem.
func TestSchema_CatalogBuiltinReleaseEvaluates(t *testing.T) {
	_, catalogVal := buildCatalogValue(t)

	// _testModuleRelease is a hidden field — must use cue.Hid with the module path.
	builtinRelease := catalogVal.LookupPath(cue.MakePath(cue.Hid("_testModuleRelease", "opmodel.dev/core@v0")))
	require.True(t, builtinRelease.Exists(), "_testModuleRelease should exist in catalog")
	require.NoError(t, builtinRelease.Err(), "_testModuleRelease should not be errored")

	// _testModuleRelease has concrete name, namespace, module, and values — should be concrete.
	err := builtinRelease.Validate(cue.Concrete(true))
	assert.NoError(t, err, "_testModuleRelease should be fully concrete (catalog control test)")

	// Spot-check a field that CUE computes.
	name, err := builtinRelease.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-release", name)

	ns, err := builtinRelease.LookupPath(cue.ParsePath("metadata.namespace")).String()
	require.NoError(t, err)
	assert.Equal(t, "default", ns)
}
