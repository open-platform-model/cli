package modulefullload

// ---------------------------------------------------------------------------
// User-supplied values: external file flow
//
// In production the end-user provides a --values file that lives outside the
// module directory. The build phase loads it, extracts the inner values struct,
// validates it against #config, and applies it via FillPath.
//
// testdata/test_module_values.cue simulates that file:
//
//   package values
//
//   values: {
//       image:    "nginx:1.28.2"
//       replicas: 3
//   }
//
// The file is intentionally partial — port and debug are omitted. Since all
// #config fields carry CUE defaults, the schema fills those gaps automatically.
//
// These tests prove:
//   - The external file loads and the inner values struct is readable
//   - Valid user values unify cleanly with #config (structural validation)
//   - Invalid user values (wrong type) are caught during #config unification
//   - FillPath with user values produces concrete component specs
//   - User values override schema defaults (image, replicas)
//   - Omitted fields retain their schema defaults after FillPath
// ---------------------------------------------------------------------------

import (
	"os"
	"testing"

	"cuelang.org/go/cue"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// loadUserValues reads the external test_module_values.cue fixture, compiles
// it, and returns the inner values struct (unwrapping the "values:" field).
// This mirrors the production path: read the --values file from disk, compile,
// extract the top-level values field.
func loadUserValues(t *testing.T, ctx *cue.Context) cue.Value {
	t.Helper()
	path := testModuleValuesPath(t)
	content, err := os.ReadFile(path)
	require.NoError(t, err, "user values fixture should be readable")

	fileVal := ctx.CompileBytes(content, cue.Filename(path))
	require.NoError(t, fileVal.Err(), "user values fixture should compile without error")

	// The file wraps values under a top-level "values:" field — extract it.
	inner := fileVal.LookupPath(cue.ParsePath("values"))
	require.True(t, inner.Exists(), `"values" field should exist in test_module_values.cue`)
	require.NoError(t, inner.Err())
	return inner
}

// TestUserValues_LoadExternalFile proves the fixture file loads correctly and
// the inner values struct contains the expected literal values.
func TestUserValues_LoadExternalFile(t *testing.T) {
	ctx, _ := buildBaseValue(t)

	userVals := loadUserValues(t, ctx)

	image, err := userVals.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err, "values.image should be a readable string")
	assert.Equal(t, "nginx:1.28.2", image, "values.image should match fixture")

	replicas, err := userVals.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err, "values.replicas should be a readable int")
	assert.Equal(t, int64(3), replicas, "values.replicas should match fixture")
}

// TestUserValues_ValidateAgainstConfig proves that the external user values
// unify cleanly with the #config schema. This is the validation gate that runs
// before FillPath — catching type mismatches before they reach the components.
func TestUserValues_ValidateAgainstConfig(t *testing.T) {
	ctx, base := buildBaseValue(t)

	userVals := loadUserValues(t, ctx)
	configSchema := base.LookupPath(cue.MakePath(cue.Def("config")))
	require.True(t, configSchema.Exists())

	unified := configSchema.Unify(userVals)
	assert.NoError(t, unified.Err(), "user values should unify with #config without error")
	assert.NoError(t, unified.Validate(), "unified value should pass structural validation")
}

// TestUserValues_InvalidField_Rejected proves that a user values file with an
// invalid type is caught during #config unification — before FillPath is called.
// The error is surfaced on the unified value, not as a Go error from Unify().
func TestUserValues_InvalidField_Rejected(t *testing.T) {
	ctx, base := buildBaseValue(t)

	// replicas must be int & >=1; provide a string instead.
	badVals := ctx.CompileString(`{
		image:    "nginx:1.28.2"
		replicas: "three"
	}`)
	require.NoError(t, badVals.Err())

	configSchema := base.LookupPath(cue.MakePath(cue.Def("config")))
	require.True(t, configSchema.Exists())

	unified := configSchema.Unify(badVals)
	assert.Error(t, unified.Validate(), "string value for int field should fail #config validation")
}

// TestUserValues_ApplyToModule proves the full user-values flow end to end:
// load the external file, extract the inner struct, apply via FillPath, and
// verify that component specs carry the user-supplied values.
func TestUserValues_ApplyToModule(t *testing.T) {
	ctx, base := buildBaseValue(t)

	userVals := loadUserValues(t, ctx)
	filled := base.FillPath(cue.MakePath(cue.Def("config")), userVals)
	require.NoError(t, filled.Err(), "FillPath with user values should not error")

	// web.spec.image should be the user value.
	image, err := filled.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.28.2", image, "web.spec.image should reflect user value")

	// web.spec.replicas should be the user value.
	replicas, err := filled.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas, "web.spec.replicas should reflect user value")

	// worker also picks up the user image (it references #config.image).
	workerImage, err := filled.LookupPath(cue.ParsePath("#components.worker.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.28.2", workerImage, "worker.spec.image should reflect user value")
}

// TestUserValues_OverridesSchemaDefaults proves that user-supplied values take
// precedence over the schema defaults defined in #config. Both image and replicas
// are set to values distinct from their defaults ("nginx:latest" and 1 respectively).
func TestUserValues_OverridesSchemaDefaults(t *testing.T) {
	ctx, base := buildBaseValue(t)

	// Confirm schema defaults before applying user values.
	defaultImage, err := base.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:latest", defaultImage, "schema default for image should be nginx:latest")

	defaultReplicas, err := base.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), defaultReplicas, "schema default for replicas should be 1")

	// Apply user values — both are different from their schema defaults.
	userVals := loadUserValues(t, ctx)
	filled := base.FillPath(cue.MakePath(cue.Def("config")), userVals)
	require.NoError(t, filled.Err())

	// User image (nginx:1.28.2) replaces schema default (nginx:latest).
	filledImage, err := filled.LookupPath(cue.ParsePath("#components.web.spec.image")).String()
	require.NoError(t, err)
	assert.Equal(t, "nginx:1.28.2", filledImage, "user image should override schema default")
	assert.NotEqual(t, defaultImage, filledImage, "image must differ from schema default")

	// User replicas (3) replaces schema default (1).
	filledReplicas, err := filled.LookupPath(cue.ParsePath("#components.web.spec.replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), filledReplicas, "user replicas should override schema default")
	assert.NotEqual(t, defaultReplicas, filledReplicas, "replicas must differ from schema default")
}

// TestUserValues_OmittedFieldsRetainSchemaDefaults proves that fields not present
// in the user values file (port, debug) retain their schema defaults after FillPath.
// The #config schema carries `port: int | *8080` and `debug: bool | *false` — CUE
// resolves these to concrete defaults regardless of whether the user provides them.
func TestUserValues_OmittedFieldsRetainSchemaDefaults(t *testing.T) {
	ctx, base := buildBaseValue(t)

	// User values only provide image and replicas — port and debug are omitted.
	userVals := loadUserValues(t, ctx)
	filled := base.FillPath(cue.MakePath(cue.Def("config")), userVals)
	require.NoError(t, filled.Err())

	// port was not in the user values; schema default (*8080) should apply.
	port, err := filled.LookupPath(cue.ParsePath("#components.web.spec.port")).Int64()
	require.NoError(t, err, "spec.port should be readable (schema default) after partial FillPath")
	assert.Equal(t, int64(8080), port, "spec.port should retain schema default 8080")

	// The filled components are still fully concrete (all fields resolved).
	componentsVal := filled.LookupPath(cue.MakePath(cue.Def("components")))
	comps := extractComponents(t, componentsVal)
	require.NotEmpty(t, comps)
	for name, comp := range comps {
		assert.True(t, comp.isConcrete(),
			"component %q should be fully concrete after partial FillPath (schema defaults fill gaps)", name)
	}
}
