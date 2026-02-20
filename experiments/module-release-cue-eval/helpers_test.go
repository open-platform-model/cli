package modulereleasecueeval

// ---------------------------------------------------------------------------
// Helpers for the module-release-cue-eval experiment.
//
// This experiment proves Approach C: injecting a loaded module cue.Value into
// #ModuleRelease via FillPath, then letting CUE evaluate UUID, labels, and
// components — eliminating the Go reimplementation in release.Builder.
//
// Two loading strategies are tested:
//
//   Strategy A (dual-load, no registry):
//     1. Load opmodel.dev/core@v0 from local catalog source
//     2. Load a fake module in the SAME context
//     3. Get #ModuleRelease schema from catalog value
//     4. FillPath to inject module + metadata + values
//     5. Read back uuid, labels, components
//
//   Strategy B (overlay, Approach C proper, requires OPM_REGISTRY):
//     1. Load a real module with an overlay that injects a synthetic file
//        exposing #ModuleRelease as _ReleaseType
//     2. FillPath to inject the module value itself as #module
//     3. Read back the same fields
//
// The catalog source is at catalog/v0/core/ in the monorepo root.
// Both the catalog and the module must be loaded with the same *cue.Context.
// ---------------------------------------------------------------------------

import (
	"os"
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/require"
)

// ---------------------------------------------------------------------------
// Strategy A: dual-load helpers
// ---------------------------------------------------------------------------

// buildCatalogValue loads opmodel.dev/core@v0 from the local catalog source
// directory. Returns the shared context and the evaluated catalog cue.Value.
//
// The catalog path is resolved relative to this test file, walking up to the
// monorepo root. Both the catalog and the module must use the SAME context for
// FillPath injection to work — callers must pass ctx to buildFakeModuleValue.
func buildCatalogValue(t *testing.T) (*cue.Context, cue.Value) {
	t.Helper()
	ctx := cuecontext.New()
	path := catalogPath(t)
	instances := load.Instances([]string{"."}, &load.Config{Dir: path})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err, "catalog load.Instances should not error")
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err(), "catalog BuildInstance should not error")
	return ctx, val
}

// releaseSchemaFromCatalog extracts #ModuleRelease from a catalog cue.Value.
// The returned value is the schema (constraint), not a concrete instance.
func releaseSchemaFromCatalog(t *testing.T, catalogVal cue.Value) cue.Value {
	t.Helper()
	v := catalogVal.LookupPath(cue.ParsePath("#ModuleRelease"))
	require.True(t, v.Exists(), "#ModuleRelease should exist in catalog value")
	require.NoError(t, v.Err(), "#ModuleRelease should not be errored")
	return v
}

// testModuleFromCatalog extracts _testModule from the catalog value.
// _testModule is defined as #Module & {...} in the catalog — guaranteed valid.
// Using it as the "module" input tests that a proper #Module value injects cleanly.
//
// _testModule is a hidden field (starts with _) — it must be accessed via cue.Hid
// with the module path "opmodel.dev/core@v0". Using cue.ParsePath("_testModule")
// fails with "hidden label not allowed".
func testModuleFromCatalog(t *testing.T, catalogVal cue.Value) cue.Value {
	t.Helper()
	v := catalogVal.LookupPath(cue.MakePath(cue.Hid("_testModule", "opmodel.dev/core@v0")))
	require.True(t, v.Exists(), "_testModule should exist in catalog value")
	require.NoError(t, v.Err(), "_testModule should not be errored")
	return v
}

// buildFakeModuleValue loads the fake_module test fixture using the provided
// context. The fake module does NOT import opmodel.dev/core@v0 — it is a bare
// CUE file that mimics the #Module shape without the type constraint applied.
//
// Decision 3 tests whether this value can be injected into #module!: #Module.
func buildFakeModuleValue(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	path := fakeModulePath(t)
	instances := load.Instances([]string{"."}, &load.Config{Dir: path})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err, "fake_module load.Instances should not error")
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err(), "fake_module BuildInstance should not error")
	return val
}

// fillRelease performs the full FillPath sequence to construct a #ModuleRelease
// from a schema value, a module value, release metadata, and user-supplied values.
//
// This is the proposed Go API for Approach C — the function under test for
// Decisions 2-7. It is intentionally NOT wrapped in require.NoError so that
// callers can inspect intermediate error states if needed.
//
// FillPath order matters: #module must be filled before values, because
// _#module: #module & {#config: values} depends on #module being present.
func fillRelease(
	schema cue.Value,
	moduleVal cue.Value,
	name, namespace string,
	userValues string, // CUE literal string for values field, e.g. `{image:"nginx:1.0"}`
) cue.Value {
	ctx := schema.Context()

	userVals := ctx.CompileString(userValues)

	return schema.
		FillPath(cue.MakePath(cue.Def("module")), moduleVal).
		FillPath(cue.ParsePath("metadata.name"), ctx.CompileString(`"`+name+`"`)).
		FillPath(cue.ParsePath("metadata.namespace"), ctx.CompileString(`"`+namespace+`"`)).
		FillPath(cue.ParsePath("values"), userVals)
}

// ---------------------------------------------------------------------------
// Strategy B: real-module helpers
// ---------------------------------------------------------------------------

// buildRealModuleWithSchema loads both the real_module test fixture AND
// opmodel.dev/core@v0 (from the module's resolved dependency cache) into the
// provided context. Returns both values.
//
// Strategy B key insight: because the module already imports opmodel.dev/core@v0
// in its cue.mod/module.cue, we can load the core package from within the module
// directory. CUE resolves it against the same pinned version — no separate catalog
// load is needed. Both values share the same *cue.Context so FillPath works.
//
// Requires OPM_REGISTRY to be set; callers should skip if absent.
func buildRealModuleWithSchema(t *testing.T, ctx *cue.Context) (moduleVal cue.Value, releaseSchema cue.Value) {
	t.Helper()
	path := realModulePath(t)

	// Load opmodel.dev/core@v0 using the module dir — CUE resolves the import
	// from the module's pinned deps (v0.1.28) without a separate registry lookup.
	coreInstances := load.Instances([]string{"opmodel.dev/core@v0"}, &load.Config{Dir: path})
	require.Len(t, coreInstances, 1)
	require.NoError(t, coreInstances[0].Err, "core package load should not error")
	coreVal := ctx.BuildInstance(coreInstances[0])
	require.NoError(t, coreVal.Err(), "core BuildInstance should not error")

	releaseSchema = coreVal.LookupPath(cue.ParsePath("#ModuleRelease"))
	require.True(t, releaseSchema.Exists(), "#ModuleRelease must exist in core value")
	require.NoError(t, releaseSchema.Err(), "#ModuleRelease must not error")

	// Load the module itself with the SAME context so FillPath injection works.
	modInstances := load.Instances([]string{"."}, &load.Config{Dir: path})
	require.Len(t, modInstances, 1)
	require.NoError(t, modInstances[0].Err, "real_module load should not error")
	moduleVal = ctx.BuildInstance(modInstances[0])
	require.NoError(t, moduleVal.Err(), "real_module BuildInstance should not error")

	return moduleVal, releaseSchema
}

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

// catalogPath returns the absolute path to catalog/v0/core in the monorepo.
// Resolved relative to this test file by walking up to the repo root.
func catalogPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	// experiments/module-release-cue-eval/ → cli/ → open-platform-model/
	repoRoot := filepath.Join(filepath.Dir(file), "..", "..", "..")
	return filepath.Join(repoRoot, "catalog", "v0", "core")
}

// fakeModulePath returns the path to the fake_module test fixture.
func fakeModulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "fake_module")
}

// realModulePath returns the path to the real_module test fixture.
func realModulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "real_module")
}

// requireRegistry skips the test if OPM_REGISTRY is not set and configures
// CUE_REGISTRY for the duration of the test.
func requireRegistry(t *testing.T) {
	t.Helper()
	registry := os.Getenv("OPM_REGISTRY")
	if registry == "" {
		t.Skip("OPM_REGISTRY not set — skipping registry-dependent Strategy B test")
	}
	t.Setenv("CUE_REGISTRY", registry)
}
