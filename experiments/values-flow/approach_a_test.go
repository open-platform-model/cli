package valuesflow

// ---------------------------------------------------------------------------
// Approach A: file filtering and load mechanics
//
// Approach A is the core loading strategy: instead of passing "." to
// load.Instances, the module directory is enumerated, values*.cue files are
// filtered out, and the remaining files are passed as an explicit list.
// values.cue is then loaded separately via ctx.CompileBytes.
//
// Key invariants proven here:
//   - Explicit file list → exactly ONE package instance (not one per file)
//   - moduleVal has NO concrete values after filtering (values.cue excluded)
//   - values.cue loaded separately contains the expected concrete defaults
//   - Non-values files (module.cue, components.cue) are all retained
//
// Fixture: testdata/values_module/ — separate values.cue pattern
// Reference: experiments/values-load-isolation/approach_a_test.go
// ---------------------------------------------------------------------------

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestApproachA_ExplicitFilesProduceSingleInstance proves that passing an
// explicit list of .cue filenames to load.Instances produces exactly one
// package instance — not one per file. The files are treated as a single package.
func TestApproachA_ExplicitFilesProduceSingleInstance(t *testing.T) {
	dir := fixturePath(t, "values_module")
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

	instances := load.Instances(moduleFiles, &load.Config{Dir: dir})

	assert.Len(t, instances, 1,
		"explicit .cue file list in the same directory must produce exactly one package instance")
}

// TestApproachA_ModuleRawHasNoConcreteValues proves that after Approach A load
// of values_module, moduleVal does not contain a concrete values field.
// values.cue was excluded from load.Instances so the package carries no
// concrete values — only the abstract #config schema.
func TestApproachA_ModuleRawHasNoConcreteValues(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, _ := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))

	valuesPath := moduleVal.LookupPath(cue.ParsePath("values"))

	assert.False(t, valuesPath.Exists(),
		"moduleVal must not have a values field after Approach A load: values.cue was excluded from load.Instances")
}

// TestApproachA_DefaultValuesLoadedSeparately proves that values.cue, loaded
// separately by Approach A via ctx.CompileBytes, contains the expected concrete
// defaults — distinct from any abstract #config constraint.
func TestApproachA_DefaultValuesLoadedSeparately(t *testing.T) {
	ctx, _ := loadCatalog(t)
	_, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))

	require.True(t, defaultVals.Exists(),
		"defaultVals must exist — values_module has a values.cue")

	image, err := defaultVals.LookupPath(cue.ParsePath("values.image")).String()
	require.NoError(t, err,
		"values.image must be a concrete string in the separately-loaded values.cue")
	assert.Equal(t, "nginx:latest", image)

	replicas, err := defaultVals.LookupPath(cue.ParsePath("values.replicas")).Int64()
	require.NoError(t, err,
		"values.replicas must be a concrete int in the separately-loaded values.cue")
	assert.Equal(t, int64(1), replicas)
}

// TestApproachA_PackageFilesAllRetained proves that filtering values*.cue does
// not accidentally drop other .cue files. All non-values package files must
// appear in moduleVal after the Approach A load.
func TestApproachA_PackageFilesAllRetained(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, _ := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))

	// module.cue defines metadata and #config
	assert.True(t, moduleVal.LookupPath(cue.ParsePath("metadata.name")).Exists(),
		"metadata.name (from module.cue) must be present in moduleVal")
	assert.True(t, moduleVal.LookupPath(cue.ParsePath("#config")).Exists(),
		"#config (from module.cue) must be present in moduleVal")

	// #components defined in module.cue
	assert.True(t, moduleVal.LookupPath(cue.ParsePath("#components.web")).Exists(),
		"#components.web (from module.cue) must be present in moduleVal")
}

// TestApproachA_FilteredLoadHasNoConflict proves that the Approach A load of
// values_module completes without a CUE unification error. The values.cue
// exclusion eliminates any potential conflict between concrete values files.
func TestApproachA_FilteredLoadHasNoConflict(t *testing.T) {
	ctx, _ := loadCatalog(t)
	moduleVal, defaultVals := loadModuleApproachA(t, ctx, fixturePath(t, "values_module"))

	assert.NoError(t, moduleVal.Err(),
		"moduleVal must have no error after filtering values*.cue")
	assert.True(t, moduleVal.Exists(),
		"moduleVal must exist after Approach A load")

	// defaultVals loads cleanly in isolation too
	assert.NoError(t, defaultVals.Err(),
		"defaultVals (values.cue loaded separately) must have no error")
}
