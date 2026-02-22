// Package valuesflow proves the complete data flow for values processing
// that the builder will implement.
//
// The flow under test:
//
//	Approach A load (values*.cue excluded from load.Instances)
//	  → moduleVal  (mod.Raw: no concrete values baked in)
//	  → defaultVals (from values.cue, loaded separately via ctx.CompileBytes)
//	                 OR inline values directly from moduleVal
//
//	validateFileList() [Approach C]
//	  → error if any values*.cue other than values.cue is present
//
//	selectValues(moduleVal, defaultVals, userFile)
//	  → if userFile provided: load it (Layer 2 — completely replaces defaults)
//	  → else if separate defaultVals: use defaultVals.LookupPath("values")
//	  → else: fall back to moduleVal.LookupPath("values") (inline values)
//
//	validateAgainstConfig(config, selectedValues)
//	  → error if selectedValues violates #config constraints
//
//	buildRelease(schema, moduleVal, name, ns, selectedValues)
//	  → FillPath chain → concrete #ModuleRelease
//	  → release.values is the final, concrete, validated values
//
// Three module fixtures are used:
//
//	values_module  — pattern A: separate values.cue, single module.cue
//	inline_module  — pattern B: inline values, multi-file (module.cue + components.cue)
//	rogue_module   — pattern C: contains values_forge.cue → error path
//
// No registry required. Catalog loaded from local source (catalog/v0/core).
// No production code (loader.Load, builder.Build) is called — the same
// low-level CUE primitives are used as in values-load-isolation and
// module-release-cue-eval.
//
// Test files:
//
//	helpers_test.go          — shared helpers and fixtures (this file)
//	approach_c_test.go       — rogue file validation (Approach C)
//	approach_a_test.go       — file filtering and load mechanics (Approach A)
//	select_values_test.go    — values selection logic (Layer 1 / Layer 2)
//	schema_validation_test.go — schema validation against #config
//	release_build_test.go    — release build and concreteness
//	inline_module_test.go    — inline values / multi-file package (pattern B)
package valuesflow

import (
	"fmt"
	"os"
	"path/filepath"
	"runtime"
	"strings"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

// catalogPath returns the absolute path to catalog/v0/core in the monorepo.
func catalogPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	// experiments/values-flow/ → experiments/ → cli/ → open-platform-model/
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..", "..")
	return filepath.Join(repoRoot, "catalog", "v0", "core")
}

// fixturePath returns the absolute path to a named fixture under testdata/.
func fixturePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// ---------------------------------------------------------------------------
// Catalog helpers
// ---------------------------------------------------------------------------

// loadCatalog loads catalog/v0/core from local source.
// Returns the shared context and the evaluated catalog cue.Value.
// Both the catalog and any module must use the SAME context for FillPath to work.
func loadCatalog(t *testing.T) (*cue.Context, cue.Value) {
	t.Helper()
	ctx := cuecontext.New()
	instances := load.Instances([]string{"."}, &load.Config{Dir: catalogPath(t)})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err, "catalog load should not error")
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err(), "catalog BuildInstance should not error")
	return ctx, val
}

// releaseSchema extracts #ModuleRelease from a catalog value.
func releaseSchema(t *testing.T, catalogVal cue.Value) cue.Value {
	t.Helper()
	v := catalogVal.LookupPath(cue.ParsePath("#ModuleRelease"))
	require.True(t, v.Exists(), "#ModuleRelease must exist in catalog")
	require.NoError(t, v.Err(), "#ModuleRelease must not error")
	return v
}

// testModuleFromCatalog extracts _testModule from the catalog value.
// _testModule is a known-valid #Module with:
//
//	#config: { replicaCount: int & >=1, image: string }
//	values:  { replicaCount: 2, image: "nginx:12" }  (inline defaults)
//
// Used for buildRelease tests because it satisfies the strict closed #Component
// schema (has proper #resources and #traits — not a free-form spec).
// Fixture modules use a free-form spec sufficient for loading/filtering tests
// but not for injection into #ModuleRelease.
//
// _testModule is a hidden field; accessed via cue.Hid.
func testModuleFromCatalog(t *testing.T, catalogVal cue.Value) cue.Value {
	t.Helper()
	v := catalogVal.LookupPath(cue.MakePath(cue.Hid("_testModule", "opmodel.dev/core@v0")))
	require.True(t, v.Exists(), "_testModule must exist in catalog")
	require.NoError(t, v.Err(), "_testModule must not error")
	return v
}

// ---------------------------------------------------------------------------
// File system helpers
// ---------------------------------------------------------------------------

// isValuesFile reports whether a filename matches the values*.cue pattern:
// any .cue file whose base name starts with "values" and ends with ".cue".
func isValuesFile(name string) bool {
	base := filepath.Base(name)
	return strings.HasPrefix(base, "values") && strings.HasSuffix(base, ".cue")
}

// cueFilesInDir returns all .cue files in dir, non-recursive, excluding cue.mod/.
func cueFilesInDir(t *testing.T, dir string) []string {
	t.Helper()
	entries, err := os.ReadDir(dir)
	require.NoError(t, err)
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".cue") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files
}

// ---------------------------------------------------------------------------
// Core logic helpers — mirrors what loader.LoadModule() and builder.Build() will do
// ---------------------------------------------------------------------------

// validateFileList implements Approach C: scans files for rogue values*.cue
// entries (any values*.cue that is NOT "values.cue").
// Returns an error naming all rogue files if any are found.
// This mirrors the validation that will live in loader.LoadModule().
func validateFileList(files []string) error {
	var rogue []string
	for _, f := range files {
		base := filepath.Base(f)
		if isValuesFile(base) && base != "values.cue" {
			rogue = append(rogue, base)
		}
	}
	if len(rogue) > 0 {
		return fmt.Errorf(
			"module directory contains %d unexpected values file(s): %s\n"+
				"Only values.cue is allowed inside the module directory.\n"+
				"Move environment-specific files outside and use --values.",
			len(rogue), strings.Join(rogue, ", "),
		)
	}
	return nil
}

// loadModuleApproachA performs an Approach A + C load of the module at dir:
//  1. Enumerate all .cue files (Approach C: error on rogue values*.cue)
//  2. Separate into moduleFiles (non-values) and valuesFile (values.cue if present)
//  3. Load the package from the explicit moduleFiles list via load.Instances
//  4. Load values.cue separately via ctx.CompileBytes (zero Value if absent)
//
// Returns (moduleVal, defaultVals).
// defaultVals is a zero cue.Value when no values.cue is present in dir.
func loadModuleApproachA(t *testing.T, ctx *cue.Context, dir string) (moduleVal cue.Value, defaultVals cue.Value) {
	t.Helper()

	all := cueFilesInDir(t, dir)
	require.NoError(t, validateFileList(all), "module must not have rogue values*.cue files")

	var moduleFiles []string
	var valuesFilePath string

	for _, f := range all {
		base := filepath.Base(f)
		if isValuesFile(base) {
			if base == "values.cue" {
				valuesFilePath = f
			}
			continue // all values*.cue excluded from package load
		}
		rel, err := filepath.Rel(dir, f)
		require.NoError(t, err)
		moduleFiles = append(moduleFiles, "./"+rel)
	}

	require.NotEmpty(t, moduleFiles, "module must have at least one non-values .cue file")

	instances := load.Instances(moduleFiles, &load.Config{Dir: dir})
	require.Len(t, instances, 1, "explicit file list must produce exactly one package instance")
	require.NoError(t, instances[0].Err, "module load must not error")

	moduleVal = ctx.BuildInstance(instances[0])
	require.NoError(t, moduleVal.Err(), "BuildInstance must not conflict after Approach A filtering")

	if valuesFilePath != "" {
		content, err := os.ReadFile(valuesFilePath)
		require.NoError(t, err)
		defaultVals = ctx.CompileBytes(content, cue.Filename(valuesFilePath))
		require.NoError(t, defaultVals.Err(), "values.cue must compile cleanly in isolation")
	}

	return moduleVal, defaultVals
}

// loadValuesFile loads an external values file and extracts the "values" field.
// Mirrors what the builder does when processing a --values flag argument.
func loadValuesFile(t *testing.T, ctx *cue.Context, path string) cue.Value {
	t.Helper()
	content, err := os.ReadFile(path)
	require.NoError(t, err, "values file must be readable: %s", path)
	compiled := ctx.CompileBytes(content, cue.Filename(path))
	require.NoError(t, compiled.Err(), "values file must compile cleanly: %s", path)
	v := compiled.LookupPath(cue.ParsePath("values"))
	require.True(t, v.Exists(), "values file must contain a top-level 'values' field: %s", path)
	return v
}

// selectValues mirrors the builder's values selection logic:
//   - userValuesPath non-empty → load that file and use it (Layer 2 — replaces defaults entirely)
//   - defaultVals has "values" field → use it (Layer 1 — from separate values.cue)
//   - fallback → moduleVal.LookupPath("values") (inline values in module.cue)
//
// Returns the selected cue.Value and whether a selection was made.
func selectValues(t *testing.T, ctx *cue.Context, moduleVal, defaultVals cue.Value, userValuesPath string) (cue.Value, bool) {
	t.Helper()

	if userValuesPath != "" {
		v := loadValuesFile(t, ctx, userValuesPath)
		return v, true
	}

	if defaultVals.Exists() {
		v := defaultVals.LookupPath(cue.ParsePath("values"))
		if v.Exists() {
			return v, true
		}
	}

	// Fallback: inline values in module.cue
	v := moduleVal.LookupPath(cue.ParsePath("values"))
	return v, v.Exists()
}

// validateAgainstConfig validates selectedValues against the module's #config schema.
// Returns an error if the values violate any constraint.
// This mirrors builder Step 4: mod.Config.Unify(selectedValues).
func validateAgainstConfig(config, selectedValues cue.Value) error {
	if !config.Exists() || !selectedValues.Exists() {
		return nil
	}
	return config.Unify(selectedValues).Err()
}

// buildRelease performs the full FillPath sequence to construct a #ModuleRelease.
// Mirrors the builder's FillPath chain.
// FillPath order matters: #module must be filled before values because
// _#module: #module & {#config: values} depends on #module being present.
func buildRelease(schema, moduleVal cue.Value, name, namespace string, selectedValues cue.Value) cue.Value {
	ctx := schema.Context()
	return schema.
		FillPath(cue.MakePath(cue.Def("module")), moduleVal).
		FillPath(cue.ParsePath("metadata.name"), ctx.CompileString(`"`+name+`"`)).
		FillPath(cue.ParsePath("metadata.namespace"), ctx.CompileString(`"`+namespace+`"`)).
		FillPath(cue.ParsePath("values"), selectedValues)
}
