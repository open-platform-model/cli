package valuesnapshot

import (
	"path/filepath"
	"runtime"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"github.com/stretchr/testify/require"
)

// moduleSnapshot holds two independent cue.Value handles from the same context.
//
// Schema is the result of BuildInstance — the module as the author wrote it.
// All #config fields carry defaults, so spec fields resolve to those defaults
// (e.g. image="nginx:latest", replicas=1). No user values have been applied.
//
// Evaluated is Schema after FillPath with user-supplied values. Spec fields
// reflect the user overrides (e.g. image="nginx:1.28.2", replicas=3).
//
// Both fields are value types (cue.Value is a struct, not a pointer). Storing
// them side-by-side is free and safe — assigning or passing either is a copy.
// Operations that produce Evaluated never mutate Schema.
type moduleSnapshot struct {
	Schema    cue.Value // pre-fill: module as loaded, schema defaults visible
	Evaluated cue.Value // post-fill: user values applied, overrides visible
}

// buildSnapshot loads the shared test module and applies the standard user
// values fixture (image="nginx:1.28.2", replicas=3). Returns the snapshot
// and the context so tests can compile additional CUE values in the same runtime.
func buildSnapshot(t *testing.T) (*cue.Context, moduleSnapshot) {
	t.Helper()

	modulePath := testModulePath(t)
	ctx := cuecontext.New()

	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	schema := ctx.BuildInstance(instances[0])
	require.NoError(t, schema.Err(), "BuildInstance should succeed")

	// Standard user values: override image and replicas, leave port+debug at defaults.
	userVals := ctx.CompileString(`{
		image:    "nginx:1.28.2"
		replicas: 3
	}`)
	require.NoError(t, userVals.Err())

	evaluated := schema.FillPath(cue.MakePath(cue.Def("config")), userVals)
	require.NoError(t, evaluated.Err(), "FillPath should succeed")

	return ctx, moduleSnapshot{
		Schema:    schema,
		Evaluated: evaluated,
	}
}

// buildSnapshotWithValues loads the same test module and applies the provided
// CUE string as user values. Useful for tests that need custom overrides.
func buildSnapshotWithValues(t *testing.T, ctx *cue.Context, valuesCUE string) moduleSnapshot {
	t.Helper()

	modulePath := testModulePath(t)
	instances := load.Instances([]string{"."}, &load.Config{Dir: modulePath})
	require.Len(t, instances, 1)
	require.NoError(t, instances[0].Err)

	schema := ctx.BuildInstance(instances[0])
	require.NoError(t, schema.Err())

	userVals := ctx.CompileString(valuesCUE)
	require.NoError(t, userVals.Err(), "user values CUE should compile")

	evaluated := schema.FillPath(cue.MakePath(cue.Def("config")), userVals)
	require.NoError(t, evaluated.Err())

	return moduleSnapshot{Schema: schema, Evaluated: evaluated}
}

// testModulePath returns the absolute path to the test module fixture.
func testModulePath(t *testing.T) string {
	t.Helper()
	_, file, _, ok := runtime.Caller(0)
	if !ok {
		t.Fatal("could not determine test file path")
	}
	return filepath.Join(filepath.Dir(file), "testdata", "test_module")
}
