package valuesloadisolation

// ---------------------------------------------------------------------------
// Approach B: CUE Overlay — shadow values*.cue files with empty package stubs
//
// Load with load.Instances([]string{"."}, cfg) as normal, but populate
// load.Config.Overlay to replace every values*.cue file with a file that
// contains only the package declaration — contributing nothing to the package.
// Load values.cue separately via ctx.CompileBytes.
//
// Key questions:
//   1. Is a package-declaration-only overlay valid CUE (no parse/eval error)?
//   2. Does the overlay correctly silence the values*.cue files?
//   3. Can we reliably extract the package name for the overlay header?
//
// Design:
//   mod.Raw    = BuildInstance("." with overlay shadowing values*.cue)
//   mod.Values = ctx.CompileBytes(values.cue)  — loaded separately, same as A
//   user vals  = ctx.CompileBytes(--values file)
// ---------------------------------------------------------------------------

import (
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// approachBLoad loads a module directory using the CUE overlay approach.
// It:
//  1. Scans for values*.cue files in dir
//  2. Extracts the package name from values.cue (or any values*.cue found)
//  3. Builds an overlay that replaces every values*.cue with "package <name>\n"
//  4. Loads with load.Instances([]string{"."}, cfg) using the overlay
//  5. Loads values.cue separately via ctx.CompileBytes
//
// Returns (ctx, baseValue, defaultValues).
func approachBLoad(t *testing.T, dir string) (ctx *cue.Context, baseVal cue.Value, defaultVals cue.Value) {
	t.Helper()
	ctx = cuecontext.New()

	// Step 1: find all values*.cue files in the module dir
	all := cueFilesInDir(t, dir)
	var valuesFiles []string
	var valuesFilePath string
	for _, f := range all {
		base := filepath.Base(f)
		if isValuesFile(base) {
			valuesFiles = append(valuesFiles, f)
			if base == "values.cue" {
				valuesFilePath = f
			}
		}
	}

	// Step 2: extract package name (needed to make the overlay a valid CUE file)
	var pkgName string
	if valuesFilePath != "" {
		var err error
		pkgName, err = extractPackageName(valuesFilePath)
		require.NoError(t, err, "extractPackageName should work on values.cue")
	} else if len(valuesFiles) > 0 {
		var err error
		pkgName, err = extractPackageName(valuesFiles[0])
		require.NoError(t, err)
	}

	// Step 3: build overlay — shadow every values*.cue with a package stub
	overlay := make(map[string]load.Source, len(valuesFiles))
	for _, vf := range valuesFiles {
		absPath, err := filepath.Abs(vf)
		require.NoError(t, err)
		stub := fmt.Sprintf("package %s\n", pkgName)
		overlay[absPath] = load.FromBytes([]byte(stub))
	}

	// Step 4: load with "." and the overlay
	cfg := &load.Config{
		Dir:     dir,
		Overlay: overlay,
	}
	instances := load.Instances([]string{"."}, cfg)
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err, "overlayed load should not error")

	baseVal = ctx.BuildInstance(instances[0])
	require.NoError(t, baseVal.Err(), "BuildInstance with overlay should not conflict")

	// Step 5: load values.cue separately if present
	if valuesFilePath != "" {
		content, err := os.ReadFile(valuesFilePath)
		require.NoError(t, err)
		defaultVals = ctx.CompileBytes(content, cue.Filename(valuesFilePath))
		require.NoError(t, defaultVals.Err(), "values.cue should compile cleanly in isolation")
	}

	return ctx, baseVal, defaultVals
}

// TestApproachB_PackageNameExtraction proves the extractPackageName helper
// correctly reads "package main" from values.cue before any loading occurs.
// This is the prerequisite step for building a valid overlay stub.
func TestApproachB_PackageNameExtraction(t *testing.T) {
	valuesFile := filepath.Join(modulePath(t), "values.cue")
	pkgName, err := extractPackageName(valuesFile)
	require.NoError(t, err)
	assert.Equal(t, "main", pkgName, "package name should be extracted as 'main'")
	t.Logf("Extracted package name: %q", pkgName)
}

// TestApproachB_PackageStubIsValidCUE verifies that a stub file containing
// only "package <name>\n" is accepted by the CUE evaluator without errors.
// An invalid stub would itself cause a load error, defeating the approach.
func TestApproachB_PackageStubIsValidCUE(t *testing.T) {
	ctx := cuecontext.New()
	stub := []byte("package main\n")
	val := ctx.CompileBytes(stub, cue.Filename("stub.cue"))
	assert.NoError(t, val.Err(), "a package-only stub should be valid CUE")
	t.Logf("Stub compiles without error: exists=%v", val.Exists())
}

// TestApproachB_OverlayLoadHasNoConflict proves that shadowing values*.cue
// via overlay eliminates the unification conflict.
func TestApproachB_OverlayLoadHasNoConflict(t *testing.T) {
	_, baseVal, _ := approachBLoad(t, modulePath(t))
	assert.NoError(t, baseVal.Err(), "base value must have no error after overlay")
	assert.True(t, baseVal.Exists(), "base value must exist")
}

// TestApproachB_BaseValueHasNoConcreteValues mirrors the Approach A test:
// confirms values.serverType is abstract (enum constraint) not a concrete string.
func TestApproachB_BaseValueHasNoConcreteValues(t *testing.T) {
	_, baseVal, _ := approachBLoad(t, modulePath(t))

	serverType := baseVal.LookupPath(cue.ParsePath("values.serverType"))
	assert.True(t, serverType.Exists())

	concreteStr, err := serverType.String()
	if err == nil {
		t.Logf("WARNING: values.serverType is concrete: %q — expected abstract", concreteStr)
	} else {
		t.Logf("values.serverType is abstract after overlay: %v — correct", err)
	}
	assert.Error(t, err, "values.serverType should not be concrete in mod.Raw with overlay applied")
}

// TestApproachB_DefaultValuesLoadedSeparately mirrors Approach A's equivalent test.
func TestApproachB_DefaultValuesLoadedSeparately(t *testing.T) {
	_, _, defaultVals := approachBLoad(t, modulePath(t))

	require.True(t, defaultVals.Exists())
	str, err := defaultVals.LookupPath(cue.ParsePath("values.serverType")).String()
	require.NoError(t, err)
	assert.Equal(t, "PAPER", str)
}

// TestApproachB_ExternalValuesInjectCleanly mirrors Approach A's injection test.
func TestApproachB_ExternalValuesInjectCleanly(t *testing.T) {
	ctx, baseVal, _ := approachBLoad(t, modulePath(t))

	extPath := externalValuesPath(t)
	content, err := os.ReadFile(extPath)
	require.NoError(t, err)
	extVals := ctx.CompileBytes(content, cue.Filename(extPath))
	require.NoError(t, extVals.Err())

	selectedValues := extVals.LookupPath(cue.ParsePath("values"))
	require.True(t, selectedValues.Exists())

	result := baseVal.FillPath(cue.ParsePath("values"), selectedValues)
	assert.NoError(t, result.Err(), "FillPath should not conflict with overlay approach")

	str, err := result.LookupPath(cue.ParsePath("values.serverType")).String()
	require.NoError(t, err)
	assert.Equal(t, "FABRIC", str)
	t.Logf("External values injected via overlay approach: serverType=%q", str)
}

// TestApproachB_OverlayShadowsAllValuesFiles confirms that every values*.cue
// file in the module directory is suppressed — not just values_forge.cue.
// After overlay, none of their concrete values appear in baseVal.
func TestApproachB_OverlayShadowsAllValuesFiles(t *testing.T) {
	_, baseVal, _ := approachBLoad(t, modulePath(t))

	// All three concrete serverType values from the three values files should
	// be absent. The field should be the abstract enum constraint, not any
	// concrete string.
	serverType := baseVal.LookupPath(cue.ParsePath("values.serverType"))
	require.True(t, serverType.Exists())

	str, err := serverType.String()
	assert.Error(t, err, "serverType must not resolve to a concrete string — overlay must shadow all files")
	if err == nil {
		// If this test fails, it means one of the values*.cue files leaked through.
		t.Errorf("serverType resolved to %q — at least one values*.cue was not shadowed", str)
	}
}

// TestApproachB_ModuleMetadataIntact mirrors the Approach A metadata test.
func TestApproachB_ModuleMetadataIntact(t *testing.T) {
	_, baseVal, _ := approachBLoad(t, modulePath(t))

	name, err := baseVal.LookupPath(cue.ParsePath("metadata.name")).String()
	require.NoError(t, err)
	assert.Equal(t, "test-server", name)

	version, err := baseVal.LookupPath(cue.ParsePath("metadata.version")).String()
	require.NoError(t, err)
	assert.Equal(t, "1.0.0", version)
}

// TestApproachB_AllNonValuesFilesLoaded mirrors the Approach A equivalent.
// Overlay only shadows values*.cue — module.cue and components.cue must
// still be fully loaded and their definitions present in baseVal.
func TestApproachB_AllNonValuesFilesLoaded(t *testing.T) {
	_, baseVal, _ := approachBLoad(t, modulePath(t))

	// From module.cue
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("metadata.name")).Exists(),
		"metadata.name (from module.cue) must exist after overlay",
	)
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("#config")).Exists(),
		"#config (from module.cue) must exist after overlay",
	)

	// From components.cue — the overlay must not silence non-values files
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("#components")).Exists(),
		"#components (from components.cue) must exist — overlay must only silence values*.cue",
	)
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("#components.server")).Exists(),
		"#components.server must exist",
	)
	assert.True(t,
		baseVal.LookupPath(cue.ParsePath("#components.proxy")).Exists(),
		"#components.proxy must exist",
	)
}
