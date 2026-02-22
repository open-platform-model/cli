package valuesflow

// ---------------------------------------------------------------------------
// Pattern B: inline_module — inline values, multi-file package
//
// inline_module represents a module where the author has written concrete
// values directly in module.cue (no separate values.cue). The package spans
// two files: module.cue (metadata + #config + inline values) and
// components.cue (#components with two components).
//
// Three invariants are tested:
//
//   1. Multi-file loading: Approach A enumerates both module.cue and
//      components.cue. With no values*.cue present, nothing is filtered,
//      but the enumeration must still handle multiple files correctly.
//
//   2. Inline values in moduleVal: unlike values_module (where values.cue is
//      excluded), inline_module's values field IS concrete in moduleVal because
//      it comes from module.cue itself (never filtered by Approach A).
//
//   3. Fallback selection: selectValues() falls back to
//      moduleVal.LookupPath("values") when no separate defaultVals exists,
//      making the inline values the effective defaults (Layer 1).
//
// Fixture: testdata/inline_module/
//   module.cue:     metadata + #config + values: { image: "nginx:stable", replicas: 2 }
//   components.cue: #components.web + #components.sidecar
//
// For the release build test, _testModule from the catalog is used (same
// reason as release_build_test.go: fixture components use a free-form spec).
// ---------------------------------------------------------------------------

import (
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestInlineModule_MultiFilePackageLoadsCorrectly proves that Approach A loads
// both module.cue and components.cue. With no values*.cue files present,
// nothing is filtered — but enumeration must handle the multi-file case.
func TestInlineModule_MultiFilePackageLoadsCorrectly(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "inline_module"))

	// No values.cue → defaultVals must be zero
	assert.False(t, defaultVals.Exists(),
		"inline_module has no values.cue — defaultVals must be a zero cue.Value")

	// metadata from module.cue
	assert.True(t, moduleVal.LookupPath(cue.ParsePath("metadata.name")).Exists(),
		"metadata.name (from module.cue) must be present in moduleVal")

	// #components from components.cue — proves both files were loaded
	assert.True(t, moduleVal.LookupPath(cue.ParsePath("#components.web")).Exists(),
		"#components.web (from components.cue) must be present in moduleVal")
	assert.True(t, moduleVal.LookupPath(cue.ParsePath("#components.sidecar")).Exists(),
		"#components.sidecar (from components.cue) must be present — proves both files loaded")
}

// TestInlineModule_InlineValuesAreConcreteInModuleVal proves that the values
// field is concrete in moduleVal for inline_module. Unlike values_module where
// values.cue is excluded from the package load, here the values come from
// module.cue itself and are never filtered by Approach A.
func TestInlineModule_InlineValuesAreConcreteInModuleVal(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, _ := loadModuleApproachA(t, ctx, fixturePath(t, "inline_module"))

	valuesPath := moduleVal.LookupPath(cue.ParsePath("values"))
	require.True(t, valuesPath.Exists(),
		"inline_module has 'values: {...}' in module.cue — must be present in moduleVal")

	image, err := valuesPath.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err,
		"values.image must be concrete in moduleVal — inline values are part of the package")
	assert.Equal(t, "nginx:stable", image,
		"inline values from module.cue must be intact in moduleVal")
}

// TestInlineModule_InlineValuesUsedAsDefaults proves that selectValues falls
// back to moduleVal.LookupPath("values") when no separate defaultVals exists,
// using the inline values as the effective Layer 1 defaults.
func TestInlineModule_InlineValuesUsedAsDefaults(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "inline_module"))

	selected, ok := selectValues(t, ctx, moduleVal, defaultVals, "" /* no user file */)

	require.True(t, ok,
		"selectValues must succeed via inline values fallback when no values.cue exists")

	image, err := selected.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:stable", image,
		"inline defaults ('nginx:stable') must be used as Layer 1 when no user values provided")

	replicas, err := selected.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(2), replicas)
}

// TestInlineModule_UserValuesReplaceInlineDefaults proves that user values
// completely replace inline module defaults. The Layer 2 rule applies regardless
// of whether defaults come from a separate values.cue or inline in module.cue.
func TestInlineModule_UserValuesReplaceInlineDefaults(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "inline_module"))
	userFile := fixturePath(t, "user_values.cue")

	selected, ok := selectValues(t, ctx, moduleVal, defaultVals, userFile)

	require.True(t, ok)

	image, err := selected.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err)
	assert.Equal(t, "custom:2.0", image,
		"Layer 2: user values must win over inline defaults ('nginx:stable')")

	replicas, err := selected.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas,
		"Layer 2: user values must win over inline defaults")
}

// TestInlineModule_SchemaValidation_Passes proves that the inline module
// defaults satisfy the module's own #config schema.
func TestInlineModule_SchemaValidation_Passes(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "inline_module"))

	selected, _ := selectValues(t, ctx, moduleVal, defaultVals, "")
	config := moduleVal.LookupPath(cue.ParsePath("#config"))

	assert.NoError(t, validateAgainstConfig(config, selected),
		"inline module defaults must satisfy #config")
}

// TestInlineModule_ReleaseIsConcreteAfterBuild proves the full build sequence
// works when using inline values (from module.cue) as the defaults.
//
// Uses _testModule from the catalog (a known-valid #Module with proper
// #resources and #traits) as the module for the release build, and extracts
// its inline values to simulate the inline values flow.
func TestInlineModule_ReleaseIsConcreteAfterBuild(t *testing.T) {
	_, catalogVal := loadCatalog(t)
	schema := releaseSchema(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	// Extract inline values from _testModule — mirrors the inline_module pattern
	// where selectValues() falls back to moduleVal.LookupPath("values").
	inlineValues := testModule.LookupPath(cue.ParsePath("values"))
	require.True(t, inlineValues.Exists(),
		"_testModule must have inline values: { replicaCount: 2, image: 'nginx:12' }")

	image, err := inlineValues.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err, "inline values must be concrete")
	assert.Equal(t, "nginx:12", image, "inline values from _testModule must be intact")

	result := buildRelease(schema, testModule, "inline-release", "staging", inlineValues)
	require.NoError(t, result.Err(),
		"buildRelease must not error when using inline values as the selected values")

	err = result.Validate(cue.Concrete(true))
	assert.NoError(t, err,
		"#ModuleRelease must be fully concrete when built with inline values")
}
