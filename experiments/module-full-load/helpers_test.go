package modulefullload

import (
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/require"
)

// buildBaseValue is a test helper that loads the test module and returns
// the evaluated base cue.Value. Equivalent to what module.Load() will do
// after the design change.
func buildBaseValue(t *testing.T) (*cue.Context, cue.Value) {
	t.Helper()
	modulePath := testModulePath(t)
	ctx := cuecontext.New()
	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)
	val := ctx.BuildInstance(instances[0])
	require.NoError(t, val.Err())
	return ctx, val
}

// applyValues simulates the build phase: takes the base value and a user
// values CUE string, compiles the values, and fills #config.
// Returns the filled cue.Value (concrete module).
func applyValues(t *testing.T, ctx *cue.Context, base cue.Value, valuesCUE string) cue.Value {
	t.Helper()
	userVals := ctx.CompileString(valuesCUE)
	require.NoError(t, userVals.Err(), "user values should compile without error")

	// Unify user values with the #config schema to validate types.
	configSchema := base.LookupPath(cue.MakePath(cue.Def("config")))
	require.True(t, configSchema.Exists())
	validated := configSchema.Unify(userVals)
	require.NoError(t, validated.Err(), "user values should unify with #config schema")

	// FillPath injects the concrete user values into the base module.
	filled := base.FillPath(cue.MakePath(cue.Def("config")), userVals)
	require.NoError(t, filled.Err(), "FillPath should not error")
	return filled
}

// testModulePath returns the absolute path to the experiment's test module fixture.
func testModulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "test_module")
}

// testModuleValuesPath returns the absolute path to the external user values fixture.
// This file lives outside the module directory, simulating a --values file supplied
// by the end-user at release time.
func testModuleValuesPath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "test_module_values.cue")
}
