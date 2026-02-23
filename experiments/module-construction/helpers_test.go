// Package moduleconstruction experiments with alternative ways to construct the
// cue.Value passed as #module into #ModuleRelease, eliminating the reliance on
// mod.Raw in the build pipeline.
//
// The central question is: after decomposing a loaded module cue.Value into its
// constituent parts (metadata, #config, #components), can we reassemble a valid
// cue.Value that preserves the cross-references between #config and #components?
//
// Four approaches are tested:
//
//	A (FillPath Assembly):  Extract fields from mod.Raw via LookupPath, fill
//	                        them individually into the #Module schema.
//
//	B (Selective Raw):      Keep mod.Raw as the base but apply targeted
//	                        FillPath overrides before builder injection.
//
//	C (Compile + Inject):   Compile Go-native metadata as CUE text, inject
//	                        #config and #components as cue.Values via FillPath.
//
//	D (Encode method):      Module struct holds parts; an Encode() method
//	                        assembles from its own fields without Raw.
//
// All approaches require OPM_REGISTRY for dep resolution of opmodel.dev/core@v0.
// Tests are skipped if OPM_REGISTRY is not set.
package moduleconstruction

import (
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
// Registry + context
// ---------------------------------------------------------------------------

// requireRegistry skips the test if OPM_REGISTRY is not set, and configures
// CUE_REGISTRY for the duration of the test.
func requireRegistry(t *testing.T) {
	t.Helper()
	registry := os.Getenv("OPM_REGISTRY")
	if registry == "" {
		t.Skip("OPM_REGISTRY not set — skipping registry-dependent test")
	}
	t.Setenv("CUE_REGISTRY", registry)
}

// newCtx returns a fresh *cue.Context. All values used together in FillPath
// operations must share the same context.
func newCtx() *cue.Context {
	return cuecontext.New()
}

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

// simpleModulePath returns the absolute path to testdata/simple_module.
func simpleModulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "simple_module")
}

// ---------------------------------------------------------------------------
// CUE loading helpers
// ---------------------------------------------------------------------------

// loadCore loads opmodel.dev/core@v0 using the simple_module directory so that
// CUE resolves the import against the module's pinned dependency cache.
// Both the module and core must use the SAME *cue.Context for FillPath to work.
func loadCore(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	path := simpleModulePath(t)
	instances := load.Instances([]string{"opmodel.dev/core@v0"}, &load.Config{Dir: path})
	require.Len(t, instances, 1, "core load.Instances should return exactly one instance")
	require.NoError(t, instances[0].Err, "core load.Instances should not error")
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err(), "core BuildInstance should not error")
	return val
}

// releaseSchemaFrom extracts #ModuleRelease from a core cue.Value.
func releaseSchemaFrom(t *testing.T, coreVal cue.Value) cue.Value {
	t.Helper()
	v := coreVal.LookupPath(cue.ParsePath("#ModuleRelease"))
	require.True(t, v.Exists(), "#ModuleRelease should exist in core")
	require.NoError(t, v.Err(), "#ModuleRelease should not error")
	return v
}

// moduleSchemaFrom extracts #Module from a core cue.Value.
func moduleSchemaFrom(t *testing.T, coreVal cue.Value) cue.Value {
	t.Helper()
	v := coreVal.LookupPath(cue.ParsePath("#Module"))
	require.True(t, v.Exists(), "#Module should exist in core")
	require.NoError(t, v.Err(), "#Module should not error")
	return v
}

// loadModuleRaw loads the simple_module with Pattern A filtering: values*.cue
// files are excluded from load.Instances so that #config cross-references in
// #components remain abstract (not bound to concrete values at load time).
//
// The returned cue.Value is the fully evaluated module package without values.
func loadModuleRaw(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	path := simpleModulePath(t)

	all := cueFilesInDir(t, path)
	var moduleFiles []string
	for _, f := range all {
		if isValuesFile(filepath.Base(f)) {
			continue
		}
		rel, err := filepath.Rel(path, f)
		require.NoError(t, err)
		moduleFiles = append(moduleFiles, "./"+rel)
	}

	instances := load.Instances(moduleFiles, &load.Config{Dir: path})
	require.Len(t, instances, 1, "filtered module load should produce one instance")
	require.NoError(t, instances[0].Err, "module load.Instances should not error")
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err(), "module BuildInstance should not error")
	return val
}

// loadModuleDefaults loads values.cue separately via CompileBytes.
// The returned value has shape: { values: { image: ..., replicas: ..., port: ... } }.
func loadModuleDefaults(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	valuesFile := filepath.Join(simpleModulePath(t), "values.cue")
	content, err := os.ReadFile(valuesFile)
	require.NoError(t, err, "values.cue must be readable")
	v := ctx.CompileBytes(content, cue.Filename(valuesFile))
	require.NoError(t, v.Err(), "values.cue should compile cleanly")
	return v
}

// defaultValues extracts the concrete "values" field from values.cue.
func defaultValues(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	defaults := loadModuleDefaults(t, ctx)
	v := defaults.LookupPath(cue.ParsePath("values"))
	require.True(t, v.Exists(), `"values" field should exist in values.cue`)
	return v
}

// ---------------------------------------------------------------------------
// Release construction helpers
// ---------------------------------------------------------------------------

// fillRelease performs the full FillPath sequence into a #ModuleRelease schema.
// moduleVal is the value to inject as #module; valuesLiteral is a CUE literal
// for the values field (e.g. `{image: "nginx:1.0", replicas: 2, port: 80}`).
//
// FillPath order matters: #module before values, because the schema has
// _#module: #module & {#config: values} which requires #module to be present.
func fillRelease(
	schema cue.Value,
	moduleVal cue.Value,
	name, namespace string,
	valuesLiteral string,
) cue.Value {
	ctx := schema.Context()
	vals := ctx.CompileString(valuesLiteral)
	return schema.
		FillPath(cue.MakePath(cue.Def("module")), moduleVal).
		FillPath(cue.ParsePath("metadata.name"), ctx.CompileString(`"`+name+`"`)).
		FillPath(cue.ParsePath("metadata.namespace"), ctx.CompileString(`"`+namespace+`"`)).
		FillPath(cue.ParsePath("values"), vals)
}

// fillReleaseWithValue is like fillRelease but accepts a cue.Value for values
// instead of a CUE literal string. Use when values were loaded from a file.
func fillReleaseWithValue(
	schema cue.Value,
	moduleVal cue.Value,
	name, namespace string,
	vals cue.Value,
) cue.Value {
	ctx := schema.Context()
	return schema.
		FillPath(cue.MakePath(cue.Def("module")), moduleVal).
		FillPath(cue.ParsePath("metadata.name"), ctx.CompileString(`"`+name+`"`)).
		FillPath(cue.ParsePath("metadata.namespace"), ctx.CompileString(`"`+namespace+`"`)).
		FillPath(cue.ParsePath("values"), vals)
}

// ---------------------------------------------------------------------------
// Assertion helpers
// ---------------------------------------------------------------------------

// assertConcrete asserts that v is fully concrete (no unresolved constraints).
func assertConcrete(t *testing.T, v cue.Value, msgAndArgs ...interface{}) {
	t.Helper()
	err := v.Validate(cue.Concrete(true))
	if len(msgAndArgs) > 0 {
		require.NoError(t, err, msgAndArgs...)
	} else {
		require.NoError(t, err, "value should be fully concrete")
	}
}

// assertFieldString asserts that the string field at path equals expected.
func assertFieldString(t *testing.T, v cue.Value, path, expected string) {
	t.Helper()
	field := v.LookupPath(cue.ParsePath(path))
	require.True(t, field.Exists(), "field %q should exist", path)
	got, err := field.String()
	require.NoError(t, err, "field %q should be a concrete string", path)
	require.Equal(t, expected, got, "field %q value mismatch", path)
}

// assertFieldInt asserts that the int field at path equals expected.
func assertFieldInt(t *testing.T, v cue.Value, path string, expected int64) {
	t.Helper()
	field := v.LookupPath(cue.ParsePath(path))
	require.True(t, field.Exists(), "field %q should exist", path)
	got, err := field.Int64()
	require.NoError(t, err, "field %q should be a concrete int", path)
	require.Equal(t, expected, got, "field %q value mismatch", path)
}

// assertFieldAbstract asserts that the field at path is NOT concrete —
// it still has an unresolved constraint. Used to confirm cross-refs are broken.
func assertFieldAbstract(t *testing.T, v cue.Value, path string) {
	t.Helper()
	field := v.LookupPath(cue.ParsePath(path))
	require.True(t, field.Exists(), "field %q should exist", path)
	err := field.Validate(cue.Concrete(true))
	require.Error(t, err, "field %q should NOT be concrete (expected abstract constraint)", path)
}

// ---------------------------------------------------------------------------
// File utilities
// ---------------------------------------------------------------------------

// isValuesFile reports whether a filename matches the values*.cue pattern.
func isValuesFile(name string) bool {
	base := filepath.Base(name)
	return strings.HasPrefix(base, "values") && strings.HasSuffix(base, ".cue")
}

// cueFilesInDir returns all top-level .cue files in dir (non-recursive).
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
