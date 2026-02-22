package valuesoverlay

import (
	"encoding/json"
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

// modulePath returns the absolute path to the test module fixture.
func modulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "module")
}

// valuesFilePath returns the absolute path to a named file inside testdata/.
func valuesFilePath(t *testing.T, name string) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", name)
}

// isValuesFile reports whether a filename matches the values*.cue pattern.
func isValuesFile(name string) bool {
	base := filepath.Base(name)
	return strings.HasPrefix(base, "values") && strings.HasSuffix(base, ".cue")
}

// ---------------------------------------------------------------------------
// Module loading
// ---------------------------------------------------------------------------

// loadModuleBase loads the module directory while excluding all values*.cue
// files. The returned CUE value is the "abstract base": metadata and #config
// are present, but values is still constrained (not yet concrete).
//
// This approach is borrowed from the values-load-isolation experiment — it
// avoids the package-level conflict that arises when multiple values*.cue
// files are loaded together by CUE's package loader.
func loadModuleBase(t *testing.T) (*cue.Context, cue.Value) {
	t.Helper()
	dir := modulePath(t)
	ctx := cuecontext.New()

	entries, err := os.ReadDir(dir)
	require.NoError(t, err)

	var moduleFiles []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		name := e.Name()
		if !strings.HasSuffix(name, ".cue") {
			continue
		}
		if isValuesFile(name) {
			continue // values*.cue excluded — loaded separately
		}
		moduleFiles = append(moduleFiles, "./"+name)
	}

	instances := load.Instances(moduleFiles, &load.Config{Dir: dir})
	require.Len(t, instances, 1, "filtered file list should produce exactly one instance")
	require.NoError(t, instances[0].Err, "filtered load should not error")

	baseVal := ctx.BuildInstance(instances[0])
	require.NoError(t, baseVal.Err(), "abstract module base must have no error")
	return ctx, baseVal
}

// loadAuthorDefaults loads values.cue from the module directory in isolation
// and returns the concrete values subtree (the .values field).
func loadAuthorDefaults(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	path := filepath.Join(modulePath(t), "values.cue")
	content, err := os.ReadFile(path)
	require.NoError(t, err)

	v := ctx.CompileBytes(content, cue.Filename(path))
	require.NoError(t, v.Err(), "author defaults (values.cue) should compile cleanly")

	vals := v.LookupPath(cue.ParsePath("values"))
	require.True(t, vals.Exists(), "values.cue must contain a top-level 'values' field")
	return vals
}

// loadUserValues loads a user-provided values file from testdata/ and returns
// the .values subtree as a CUE value.
func loadUserValues(t *testing.T, ctx *cue.Context, filename string) cue.Value {
	t.Helper()
	path := valuesFilePath(t, filename)
	content, err := os.ReadFile(path)
	require.NoError(t, err, "user values file %s must exist", filename)

	v := ctx.CompileBytes(content, cue.Filename(path))
	require.NoError(t, v.Err(), "user values file %s should compile cleanly", filename)

	vals := v.LookupPath(cue.ParsePath("values"))
	require.True(t, vals.Exists(), "user values file %s must contain a top-level 'values' field", filename)
	return vals
}

// ---------------------------------------------------------------------------
// Go-level map merge utilities (used by Approach C and D)
// ---------------------------------------------------------------------------

// cueValueToMap decodes a concrete CUE value to a Go map via JSON marshalling.
// The test fails if the value is not concrete or cannot be marshalled.
func cueValueToMap(t *testing.T, v cue.Value) map[string]any {
	t.Helper()
	b, err := v.MarshalJSON()
	require.NoError(t, err, "cueValueToMap: MarshalJSON failed — value must be concrete")
	var m map[string]any
	require.NoError(t, json.Unmarshal(b, &m))
	return m
}

// deepMergeMap merges src into dst recursively, returning a new map.
// src wins on any key conflict (last-wins semantics).
// Nested maps are merged recursively; all other types are replaced.
func deepMergeMap(dst, src map[string]any) map[string]any {
	result := make(map[string]any, len(dst))
	for k, v := range dst {
		result[k] = v
	}
	for k, v := range src {
		if srcMap, ok := v.(map[string]any); ok {
			if dstMap, ok := result[k].(map[string]any); ok {
				result[k] = deepMergeMap(dstMap, srcMap)
				continue
			}
		}
		result[k] = v // src wins
	}
	return result
}

// mapToCUEValue encodes a Go map to a CUE value via JSON.
func mapToCUEValue(ctx *cue.Context, m map[string]any) (cue.Value, error) {
	b, err := json.Marshal(m)
	if err != nil {
		return cue.Value{}, err
	}
	v := ctx.CompileBytes(b)
	return v, v.Err()
}
