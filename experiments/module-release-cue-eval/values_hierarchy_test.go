package modulereleasecueeval

// ---------------------------------------------------------------------------
// Approach A: values hierarchy and filtering system
//
// These tests prove that module-release-cue-eval uses the Approach A loading
// strategy from values-load-isolation:
//
//   1. Module package files are loaded WITHOUT values*.cue → the module's
//      values field is abstract (just the #config constraint), with no
//      concrete values baked into the package.
//
//   2. values.cue is loaded separately via ctx.CompileBytes → provides the
//      module author's default values (Layer 1).
//
//   3. User-provided values form Layer 2. When user values are present,
//      they completely replace the module defaults — no partial merging.
//      When absent, module defaults are used instead.
//
// The hierarchy is therefore:
//   Layer 1 (lowest): module defaults from values.cue
//   Layer 2 (highest): user-provided values (replaces Layer 1 entirely)
//
// Reference: experiments/values-load-isolation/approach_a_test.go
// ---------------------------------------------------------------------------

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestHierarchy_FilteredLoadHasNoConflict proves that loading fake_module via
// Approach A (values*.cue excluded from load.Instances) produces a module value
// with no CUE unification error. This mirrors TestApproachA_FilteredLoadHasNoConflict
// from values-load-isolation and confirms the same technique applies here.
func TestHierarchy_FilteredLoadHasNoConflict(t *testing.T) {
	ctx, _ := buildCatalogValue(t)

	moduleVal := buildFakeModuleValue(t, ctx)

	assert.NoError(t, moduleVal.Err(), "Approach A filtered load must produce a non-errored module value")
	assert.True(t, moduleVal.Exists(), "filtered module value must exist")
}

// TestHierarchy_ModuleValuesUseConfigDefaultAfterFilter proves that after
// Approach A filtering, the module's values field resolves to the #config
// built-in * defaults — NOT to the concrete values from values.cue.
//
// fake_module distinguishes these two:
//   - #config.image built-in default: "nginx:1.0"  (the * marker in #config)
//   - values.cue default:             "nginx:latest" (the separately-loaded file)
//
// After filtering (values.cue excluded from load), values.image must resolve to
// "nginx:1.0" (from #config's own * default). If values.cue were still baked in,
// the value would be "nginx:latest" instead — proving filtering worked.
func TestHierarchy_ModuleValuesUseConfigDefaultAfterFilter(t *testing.T) {
	ctx, _ := buildCatalogValue(t)

	moduleVal := buildFakeModuleValue(t, ctx)

	valuesPath := moduleVal.LookupPath(cue.ParsePath("values"))
	assert.True(t, valuesPath.Exists(), "values path should still exist as a schema constraint")

	imagePath := moduleVal.LookupPath(cue.ParsePath("values.image"))
	assert.True(t, imagePath.Exists(), "values.image should exist in the filtered module")

	image, err := imagePath.String()
	require.NoError(t, err, "values.image should resolve to #config's built-in * default")
	assert.Equal(t, "nginx:1.0", image,
		"after filtering, values.image must be the #config default ('nginx:1.0'), "+
			"NOT the values.cue default ('nginx:latest') — proving values.cue was excluded")
}

// TestHierarchy_DefaultsFileLoadsCleanly proves that values.cue, loaded
// separately via ctx.CompileBytes (buildFakeModuleDefaults), compiles without
// error and contains the expected concrete module author defaults.
func TestHierarchy_DefaultsFileLoadsCleanly(t *testing.T) {
	ctx, _ := buildCatalogValue(t)

	defaults := buildFakeModuleDefaults(t, ctx)

	require.NoError(t, defaults.Err(), "values.cue should compile cleanly in isolation")
	require.True(t, defaults.Exists(), "module defaults value should exist")

	image, err := defaults.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err, "values.image should be a concrete string in the defaults file")
	assert.Equal(t, "nginx:latest", image, "module default image should be nginx:latest")

	replicas, err := defaults.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err, "values.replicas should be a concrete int in the defaults file")
	assert.Equal(t, int64(1), replicas, "module default replicas should be 1")
}

// TestHierarchy_NoUserValues_ModuleDefaultsApplied proves that when no user
// values are provided (userValuesCUE=""), fillReleaseWithHierarchy uses the
// module author defaults as the effective values for the release.
//
// This is Layer 1 of the hierarchy in action: the module author's values.cue
// drives the release when no user values override them.
//
// Uses _testModule from the catalog whose #config is: { replicaCount: int & >=1, image: string }.
// Module defaults are supplied inline to match that #config shape.
func TestHierarchy_NoUserValues_ModuleDefaultsApplied(t *testing.T) {
	ctx, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	// Module author defaults matching _testModule.#config (inline, like a values.cue).
	moduleDefaults := ctx.CompileString(`{ values: { replicaCount: 1, image: "nginx:latest" } }`)
	require.NoError(t, moduleDefaults.Err(), "inline module defaults should compile cleanly")

	// No user values — hierarchy falls back to module defaults entirely.
	result := fillReleaseWithHierarchy(schema, testModule, "my-release", "default",
		moduleDefaults, "" /* no user values */)

	require.NoError(t, result.Err(), "release with module defaults should not error")

	// The values field should reflect the module defaults.
	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err, "values.image should be concrete when module defaults are applied")
	assert.Equal(t, "nginx:latest", image,
		"values.image should come from the module defaults (Layer 1)")

	replicaCount, err := result.LookupPath(cue.ParsePath("values.replicaCount")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), replicaCount,
		"values.replicaCount should come from the module defaults (Layer 1)")
}

// TestHierarchy_UserValuesCompletelyReplaceDefaults proves that when user values
// are provided, they completely replace the module defaults — the module author's
// values are entirely ignored.
//
// This is Layer 2 of the hierarchy: user values win, and no field from the
// module defaults bleeds through.
//
// Uses _testModule from the catalog whose #config is: { replicaCount: int & >=1, image: string }.
func TestHierarchy_UserValuesCompletelyReplaceDefaults(t *testing.T) {
	ctx, catalogVal := buildCatalogValue(t)
	schema := releaseSchemaFromCatalog(t, catalogVal)
	testModule := testModuleFromCatalog(t, catalogVal)

	// Module author defaults (Layer 1): image="nginx:latest".
	moduleDefaults := ctx.CompileString(`{ values: { replicaCount: 1, image: "nginx:latest" } }`)
	require.NoError(t, moduleDefaults.Err())

	// User provides a different image — this must completely replace the defaults.
	result := fillReleaseWithHierarchy(schema, testModule, "my-release", "default",
		moduleDefaults, `{ replicaCount: 2, image: "nginx:1.28" }`)

	require.NoError(t, result.Err(), "release with user values should not error")

	// The values field must carry the user's values, not the module defaults.
	image, err := result.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.28", image,
		"user image must completely replace module default 'nginx:latest' (Layer 2 wins)")

	replicaCount, err := result.LookupPath(cue.ParsePath("values.replicaCount")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(2), replicaCount,
		"user replicaCount must completely replace module default '1' (Layer 2 wins)")
}

// TestHierarchy_DefaultsFilenamePattern proves that the isValuesFile predicate
// correctly identifies all values*.cue patterns used by Approach A filtering.
// This is the same contract as values-load-isolation.
func TestHierarchy_DefaultsFilenamePattern(t *testing.T) {
	cases := []struct {
		name     string
		isValues bool
	}{
		{"values.cue", true},
		{"values_forge.cue", true},
		{"values_testing.cue", true},
		{"values_prod.cue", true},
		{"module.cue", false},
		{"components.cue", false},
		{"config.cue", false},
		{"cue.mod", false},
	}

	for _, tc := range cases {
		assert.Equal(t, tc.isValues, isValuesFile(tc.name),
			"isValuesFile(%q) should be %v", tc.name, tc.isValues)
	}
}

// TestHierarchy_FilterDoesNotDropNonValueFiles proves that Approach A filtering
// only removes values*.cue files and leaves all other .cue files intact.
// Both module.cue and any components files must survive the filter.
func TestHierarchy_FilterDoesNotDropNonValueFiles(t *testing.T) {
	path := fakeModulePath(t)
	all := cueFilesInDir(&testing.T{}, path)

	var loaded, filtered []string
	for _, f := range all {
		base := filepath.Base(f)
		if isValuesFile(base) {
			filtered = append(filtered, base)
		} else {
			loaded = append(loaded, base)
		}
	}

	t.Logf("Loaded files:   %v", loaded)
	t.Logf("Filtered files: %v", filtered)

	assert.Contains(t, loaded, "module.cue",
		"module.cue must survive Approach A filtering")
	assert.Contains(t, filtered, "values.cue",
		"values.cue must be excluded by Approach A filtering")
	assert.NotContains(t, loaded, "values.cue",
		"values.cue must not appear in the loaded file list")
}
