package valuesloadisolation

// ---------------------------------------------------------------------------
// Approach A: Explicit file list — filter values*.cue from load.Instances
//
// Instead of passing "." to load.Instances, enumerate all .cue files in the
// module directory, filter out any file matching values*.cue, and pass the
// remaining files explicitly. Load values.cue separately via ctx.CompileBytes.
//
// Key question: does load.Instances treat a list of explicit .cue filenames
// the same as "." (i.e. as a single package), or does it create multiple
// instances?
//
// Design:
//   mod.Raw   = BuildInstance(filtered files)  — no concrete values
//   mod.Values = ctx.CompileBytes(values.cue)  — default values, or zero if absent
//   user vals  = ctx.CompileBytes(--values file) — overrides mod.Values
// ---------------------------------------------------------------------------

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// approachALoad loads a module directory using the explicit file list approach.
// It:
//  1. Lists all .cue files in dir
//  2. Filters out values*.cue files
//  3. Passes the remaining files to load.Instances (relative to dir)
//  4. Loads values.cue separately if present
//
// Returns (baseValue, defaultValues, error).
// defaultValues is a zero cue.Value if values.cue does not exist.
func approachALoad(t *testing.T, dir string) (ctx *cue.Context, baseVal cue.Value, defaultVals cue.Value) {
	t.Helper()
	ctx = cuecontext.New()

	// Step 1: list all .cue files, filter out values*.cue
	all := cueFilesInDir(t, dir)
	var moduleFiles []string
	var valuesFile string
	for _, f := range all {
		base := filepath.Base(f)
		if isValuesFile(base) {
			if base == "values.cue" {
				valuesFile = f
			}
			// All values*.cue files are excluded from the package load
			continue
		}
		// Pass relative path to load.Instances (relative to Dir)
		rel, err := filepath.Rel(dir, f)
		require.NoError(t, err)
		moduleFiles = append(moduleFiles, "./"+rel)
	}

	// Step 2: load module package from filtered file list
	instances := load.Instances(moduleFiles, &load.Config{Dir: dir})
	require.Len(t, instances, 1, "filtered file list should produce exactly one package instance")
	require.NoError(t, instances[0].Err, "filtered load should not error")

	baseVal = ctx.BuildInstance(instances[0])
	require.NoError(t, baseVal.Err(), "BuildInstance on filtered files should not conflict")

	// Step 3: load values.cue separately if present
	if valuesFile != "" {
		content, err := os.ReadFile(valuesFile)
		require.NoError(t, err)
		defaultVals = ctx.CompileBytes(content, cue.Filename(valuesFile))
		require.NoError(t, defaultVals.Err(), "values.cue should compile cleanly in isolation")
	}

	return ctx, baseVal, defaultVals
}

// TestApproachA_FilteredLoadHasNoConflict proves that excluding values*.cue
// from load.Instances eliminates the unification conflict.
func TestApproachA_FilteredLoadHasNoConflict(t *testing.T) {
	_, baseVal, _ := approachALoad(t, modulePath(t))
	assert.NoError(t, baseVal.Err(), "base value must have no error after filtering values*.cue")
	assert.True(t, baseVal.Exists(), "base value must exist")
}

// TestApproachA_ExplicitFilesProduceSingleInstance verifies that passing a
// list of .cue filenames to load.Instances produces exactly one instance
// (not one per file), meaning they are treated as a single package.
func TestApproachA_ExplicitFilesProduceSingleInstance(t *testing.T) {
	dir := modulePath(t)
	all := cueFilesInDir(t, dir)

	var moduleFiles []string
	for _, f := range all {
		if isValuesFile(filepath.Base(f)) {
			continue
		}
		rel, err := filepath.Rel(dir, f)
		require.NoError(t, err)
		moduleFiles = append(moduleFiles, "./"+rel)
	}

	t.Logf("Loading explicit files: %v", moduleFiles)
	instances := load.Instances(moduleFiles, &load.Config{Dir: dir})
	t.Logf("Got %d instances", len(instances))
	assert.Len(t, instances, 1, "explicit .cue files in the same dir should produce one instance")
}

// TestApproachA_BaseValueHasNoConcreteValues confirms that after filtering,
// mod.Raw (baseVal) does not contain concrete values. The values path should
// be abstract (the #config constraint) rather than a concrete struct.
func TestApproachA_BaseValueHasNoConcreteValues(t *testing.T) {
	_, baseVal, _ := approachALoad(t, modulePath(t))

	valuesPath := baseVal.LookupPath(cue.ParsePath("values"))
	assert.True(t, valuesPath.Exists(), "values path should exist as a schema constraint")

	// serverType should be abstract (the enum constraint), not a concrete string
	serverType := baseVal.LookupPath(cue.ParsePath("values.serverType"))
	assert.True(t, serverType.Exists())

	concreteStr, err := serverType.String()
	if err == nil {
		t.Logf("WARNING: values.serverType is concrete: %q — expected abstract", concreteStr)
	} else {
		t.Logf("values.serverType is abstract (not concrete string): %v — correct", err)
	}
	assert.Error(t, err, "values.serverType should not be a concrete string in mod.Raw after filtering")
}

// TestApproachA_DefaultValuesLoadedSeparately proves that values.cue loaded
// via ctx.CompileBytes has the expected concrete values, accessible at values.serverType.
func TestApproachA_DefaultValuesLoadedSeparately(t *testing.T) {
	_, _, defaultVals := approachALoad(t, modulePath(t))

	require.True(t, defaultVals.Exists(), "defaultVals should exist (values.cue is present in testdata)")

	serverType := defaultVals.LookupPath(cue.ParsePath("values.serverType"))
	require.True(t, serverType.Exists(), "values.serverType should be present in separately loaded values.cue")

	str, err := serverType.String()
	require.NoError(t, err, "values.serverType should be a concrete string")
	assert.Equal(t, "PAPER", str, "default serverType from values.cue should be PAPER")
}

// TestApproachA_ExternalValuesInjectCleanly proves that a user-provided
// external values file (simulating --values flag) can be injected via
// FillPath without conflict, because baseVal has no concrete values baked in.
func TestApproachA_ExternalValuesInjectCleanly(t *testing.T) {
	ctx, baseVal, _ := approachALoad(t, modulePath(t))

	// Load external values file (simulates --values)
	extPath := externalValuesPath(t)
	content, err := os.ReadFile(extPath)
	require.NoError(t, err)
	extVals := ctx.CompileBytes(content, cue.Filename(extPath))
	require.NoError(t, extVals.Err(), "external values file should compile cleanly")

	// Extract the values field (mirrors builder/values.go selectValues logic)
	selectedValues := extVals.LookupPath(cue.ParsePath("values"))
	require.True(t, selectedValues.Exists(), "external values file must have a top-level 'values' field")

	// Inject via FillPath — this is what builder.Build does
	result := baseVal.FillPath(cue.ParsePath("values"), selectedValues)
	assert.NoError(t, result.Err(), "FillPath injection should not conflict when baseVal has no concrete values")

	// Confirm the injected value is correct
	serverType := result.LookupPath(cue.ParsePath("values.serverType"))
	str, err := serverType.String()
	require.NoError(t, err)
	assert.Equal(t, "FABRIC", str, "injected serverType should be FABRIC from external_values.cue")
	t.Logf("External values injected cleanly: serverType=%q", str)
}

// TestApproachA_DefaultValuesInjectCleanly proves that even the default
// values.cue (loaded separately) can be injected via FillPath without conflict.
func TestApproachA_DefaultValuesInjectCleanly(t *testing.T) {
	ctx, baseVal, defaultVals := approachALoad(t, modulePath(t))
	_ = ctx

	require.True(t, defaultVals.Exists())

	selectedValues := defaultVals.LookupPath(cue.ParsePath("values"))
	require.True(t, selectedValues.Exists())

	result := baseVal.FillPath(cue.ParsePath("values"), selectedValues)
	assert.NoError(t, result.Err(), "FillPath with default values.cue should not conflict")

	serverType := result.LookupPath(cue.ParsePath("values.serverType"))
	str, err := serverType.String()
	require.NoError(t, err)
	assert.Equal(t, "PAPER", str)
}

// TestApproachA_ModuleMetadataIntact confirms that filtering values*.cue does
// not affect module metadata — name, version, fqn are still correct in baseVal.
func TestApproachA_ModuleMetadataIntact(t *testing.T) {
	_, baseVal, _ := approachALoad(t, modulePath(t))

	name, err := baseVal.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-server", name)

	version, err := baseVal.LookupPath(cue.ParsePath("metadata.version")).String()
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)

	fqn, err := baseVal.LookupPath(cue.ParsePath("metadata.fqn")).String()
	require.NoError(t, err)
	assert.Equal(t, "example.com/test-server@v0", fqn)
}

// TestApproachA_AllNonValuesFilesLoaded proves that filtering values*.cue does
// not accidentally drop other package files. Both module.cue (metadata, #config)
// and components.cue (#components with server + proxy) must be present in baseVal.
// This is the critical regression check: we want isolation of values files only.
func TestApproachA_AllNonValuesFilesLoaded(t *testing.T) {
	_, baseVal, _ := approachALoad(t, modulePath(t))

	// From module.cue: metadata and #config
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("metadata.name")).Exists(),
		"metadata.name (from module.cue) must exist in baseVal",
	)
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("#config")).Exists(),
		"#config (from module.cue) must exist in baseVal",
	)

	// From components.cue: #components with both server and proxy components
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("#components")).Exists(),
		"#components (from components.cue) must exist in baseVal — file must not be filtered",
	)
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("#components.server")).Exists(),
		"#components.server (from components.cue) must exist",
	)
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("#components.proxy")).Exists(),
		"#components.proxy (from components.cue) must exist",
	)

	// Enumerate which files were loaded — should include both module.cue and components.cue
	all := cueFilesInDir(t, modulePath(t))
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
	assert.Contains(t, loaded, "module.cue")
	assert.Contains(t, loaded, "components.cue")
	assert.NotContains(t, loaded, "values.cue")
	assert.NotContains(t, loaded, "values_forge.cue")
	assert.NotContains(t, loaded, "values_testing.cue")
}
