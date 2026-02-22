package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/loader"
)

// ---------------------------------------------------------------------------
// Path helpers
// ---------------------------------------------------------------------------

// fixture returns the absolute path to a named module in testdata/.
func fixture(t *testing.T, name string) string {
	t.Helper()
	abs, _ := filepath.Abs(filepath.Join("testdata", name))
	return abs
}

// invalidFixture returns the absolute path to a named module in testdata/invalid/.
func invalidFixture(t *testing.T, name string) string {
	t.Helper()
	abs, _ := filepath.Abs(filepath.Join("testdata", "invalid", name))
	return abs
}

// ---------------------------------------------------------------------------
// Path resolution (pre-load validation)
// ---------------------------------------------------------------------------

// TestLoadModule_PathResolution covers path resolution behaviour:
// relative paths are resolved, non-existent paths are rejected,
// and directories missing cue.mod/ are rejected.
func TestLoadModule_PathResolution(t *testing.T) {
	ctx := cuecontext.New()

	tests := []struct {
		name        string
		modulePath  func(t *testing.T) string
		wantErr     bool
		errContains string
	}{
		{
			name: "relative path resolves to valid module",
			modulePath: func(t *testing.T) string {
				return filepath.Join("testdata", "test-module")
			},
			wantErr: false,
		},
		{
			name: "non-existent path returns error",
			modulePath: func(t *testing.T) string {
				return "/nonexistent/path/that/does/not/exist"
			},
			wantErr:     true,
			errContains: "module directory not found",
		},
		{
			name: "directory without cue.mod returns error",
			modulePath: func(t *testing.T) string {
				return t.TempDir()
			},
			wantErr:     true,
			errContains: "not a CUE module",
		},
	}

	for _, tc := range tests {
		t.Run(tc.name, func(t *testing.T) {
			mod, err := loader.LoadModule(ctx, tc.modulePath(t), "")
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				assert.Nil(t, mod)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, mod)
				assert.True(t, filepath.IsAbs(mod.ModulePath), "ModulePath must be absolute after LoadModule")
			}
		})
	}
}

// TestLoadModule_RegistryEnvVarIsCleanedUp verifies that CUE_REGISTRY is set during
// LoadModule when a registry is provided and is unset after LoadModule returns.
func TestLoadModule_RegistryEnvVarIsCleanedUp(t *testing.T) {
	os.Unsetenv("CUE_REGISTRY") //nolint:errcheck
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module"), "fake-registry.test")
	require.NoError(t, err)
	assert.NotNil(t, mod)
	assert.Empty(t, os.Getenv("CUE_REGISTRY"), "CUE_REGISTRY must be unset after LoadModule returns")
}

// ---------------------------------------------------------------------------
// Error paths
// ---------------------------------------------------------------------------

// TestLoadModule_InstanceLoadError verifies that a CUE syntax error causes LoadModule to
// return an error wrapping inst.Err.
func TestLoadModule_InstanceLoadError(t *testing.T) {
	ctx := cuecontext.New()
	_, err := loader.LoadModule(ctx, invalidFixture(t, "syntax-error"), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading module:")
}

// TestLoadModule_EvaluationError verifies that a CUE type conflict that is valid
// syntax but fails at evaluation causes LoadModule to return a wrapped error.
func TestLoadModule_EvaluationError(t *testing.T) {
	ctx := cuecontext.New()
	_, err := loader.LoadModule(ctx, invalidFixture(t, "eval-error"), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evaluating module CUE:")
}

// ---------------------------------------------------------------------------
// Extra values*.cue files: filtered, not rejected
// ---------------------------------------------------------------------------

// TestLoadModule_ExtraValuesFilesFilteredSilently proves that a module directory
// containing values_prod.cue alongside values.cue loads without error, and
// that mod.Values reflects only values.cue — the extra file has no effect.
func TestLoadModule_ExtraValuesFilesFilteredSilently(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "extra-values-module"), "")
	require.NoError(t, err, "module with values_prod.cue must load without error")
	require.NotNil(t, mod)

	// mod.Values must come from values.cue, not values_prod.cue.
	require.True(t, mod.Values.Exists(), "mod.Values must be populated from values.cue")

	image, imgErr := mod.Values.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, imgErr, "values.image must be a concrete string")
	assert.Equal(t, "nginx:default", image,
		"image must match values.cue default, not values_prod.cue override")

	replicas, repErr := mod.Values.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, repErr, "values.replicas must be a concrete int")
	assert.Equal(t, int64(1), replicas,
		"replicas must match values.cue default, not values_prod.cue override")
}

// ---------------------------------------------------------------------------
// Approach A: explicit filtered file list, Pattern A modules
// ---------------------------------------------------------------------------

// TestLoadModule_ApproachA_ModuleRawHasNoConcreteValues proves that after loading
// a Pattern A module (test-module: no inline values in module.cue), mod.Raw
// does not contain a concrete values field. values.cue was excluded from the
// package load; only the abstract #config schema is present in the package.
func TestLoadModule_ApproachA_ModuleRawHasNoConcreteValues(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module"), "")
	require.NoError(t, err)

	valuesInRaw := mod.Raw.LookupPath(cue.ParsePath("values"))
	assert.False(t, valuesInRaw.Exists(),
		"mod.Raw must not have a concrete values field after Approach A LoadModule: values.cue was excluded from load.Instances")
}

// TestLoadModule_ApproachA_DefaultValuesLoadedSeparately proves that values.cue,
// loaded separately by the Approach A strategy, provides the expected concrete
// defaults in mod.Values — distinct from anything in mod.Raw.
func TestLoadModule_ApproachA_DefaultValuesLoadedSeparately(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module"), "")
	require.NoError(t, err)

	require.True(t, mod.Values.Exists(),
		"mod.Values must be populated from values.cue (Approach A)")

	image, imgErr := mod.Values.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, imgErr, "values.image must be a concrete string from values.cue")
	assert.Equal(t, "nginx:1.25", image)

	replicas, repErr := mod.Values.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, repErr, "values.replicas must be a concrete int from values.cue")
	assert.Equal(t, int64(2), replicas)
}

// TestLoadModule_ApproachA_PackageFilesAllRetained proves that filtering values*.cue
// does not accidentally drop other .cue files. All non-values package content
// (metadata, #config, #components) must be present in mod.Raw after the
// Approach A load.
func TestLoadModule_ApproachA_PackageFilesAllRetained(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module"), "")
	require.NoError(t, err)

	assert.True(t, mod.Raw.LookupPath(cue.ParsePath("metadata.name")).Exists(),
		"metadata.name must be present in mod.Raw")
	assert.True(t, mod.Raw.LookupPath(cue.ParsePath("#config")).Exists(),
		"#config must be present in mod.Raw")
	assert.True(t, mod.Raw.LookupPath(cue.ParsePath("#components.web")).Exists(),
		"#components.web must be present in mod.Raw")
}

// ---------------------------------------------------------------------------
// Pattern B: inline values, no values.cue
// ---------------------------------------------------------------------------

// TestLoadModule_InlineValues_PopulatesModValues proves that when a module defines
// values inline in module.cue with no values.cue present, mod.Values is
// populated from mod.Raw's inline values field.
func TestLoadModule_InlineValues_PopulatesModValues(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "inline-values-module"), "")
	require.NoError(t, err)

	require.True(t, mod.Values.Exists(),
		"mod.Values must be set from inline values when no values.cue exists")

	image, err := mod.Values.LookupPath(cue.ParsePath("image")).String()
	require.NoError(t, err, "values.image must be concrete in inline-values-module")
	assert.Equal(t, "nginx:stable", image)

	replicas, err := mod.Values.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err, "values.replicas must be concrete in inline-values-module")
	assert.Equal(t, int64(2), replicas)
}

// TestLoadModule_InlineValues_NoSeparateValuesFile proves that inline-values-module
// has no values.cue — mod.Values comes exclusively from the inline values
// field in module.cue, not from a separately loaded file.
func TestLoadModule_InlineValues_NoSeparateValuesFile(t *testing.T) {
	dir := fixture(t, "inline-values-module")
	_, err := os.Stat(filepath.Join(dir, "values.cue"))
	assert.True(t, os.IsNotExist(err),
		"inline-values-module must not contain values.cue")
}

// ---------------------------------------------------------------------------
// Metadata extraction
// ---------------------------------------------------------------------------

// TestLoadModule_Success verifies that LoadModule returns a fully populated *core.Module
// with all fields set and that the module passes Validate().
// Uses the test-module fixture (pure Pattern A).
func TestLoadModule_Success(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module"), "")
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.NotNil(t, mod.Metadata)

	assert.True(t, filepath.IsAbs(mod.ModulePath))
	assert.Equal(t, "test-module", mod.Metadata.Name)
	assert.Equal(t, "example.com/test-module@v0#test-module", mod.Metadata.FQN)
	assert.Equal(t, "1.0.0", mod.Metadata.Version)
	assert.NotEmpty(t, mod.Metadata.UUID)
	assert.Equal(t, "default", mod.Metadata.DefaultNamespace)
	assert.NotEmpty(t, mod.Metadata.Labels)

	assert.True(t, mod.Raw.Exists(), "Raw must be set after LoadModule")
	assert.True(t, mod.Config.Exists(), "#config must be extracted")
	assert.True(t, mod.Values.Exists(), "Values must be populated from values.cue")
	assert.NotEmpty(t, mod.Components, "#components must be extracted")

	require.NoError(t, mod.Validate())
}

// TestLoadModule_PartialMetadata verifies that a module with only some metadata fields
// defined loads without error, with absent fields as zero values.
func TestLoadModule_PartialMetadata(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "no-metadata-module"), "")
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.NotNil(t, mod.Metadata)

	assert.Equal(t, "computed-module", mod.Metadata.Name)
	assert.Equal(t, "1.0.0", mod.Metadata.Version)

	// Absent fields remain zero values.
	assert.Empty(t, mod.Metadata.FQN)
	assert.Empty(t, mod.Metadata.UUID)
	assert.Empty(t, mod.Metadata.DefaultNamespace)
	assert.Empty(t, mod.Metadata.Labels)

	// Values come from values.cue (pure Pattern A after fixture cleanup).
	assert.True(t, mod.Values.Exists(), "Values must be populated from values.cue")
	assert.True(t, mod.Raw.Exists())
}

// TestLoadModule_NoValues verifies that a module without a values field and without
// values.cue loads without error and that Module.Values is a zero cue.Value.
func TestLoadModule_NoValues(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module-no-values"), "")
	require.NoError(t, err)
	require.NotNil(t, mod)

	assert.False(t, mod.Values.Exists(), "Values must be zero when neither values.cue nor inline values exist")
	assert.True(t, mod.Config.Exists(), "#config is present")
	assert.NotEmpty(t, mod.Components)
}

// TestLoadModule_NoComponents verifies that a module without #components loads
// without error and that Module.Components is nil.
func TestLoadModule_NoComponents(t *testing.T) {
	ctx := cuecontext.New()
	fixturesPath, _ := filepath.Abs(filepath.Join("..", "..", "tests", "fixtures", "valid", "simple-module"))

	mod, err := loader.LoadModule(ctx, fixturesPath, "")
	require.NoError(t, err)
	require.NotNil(t, mod)

	assert.Nil(t, mod.Components)
	assert.False(t, mod.Config.Exists(), "#config must be zero when absent from module")
	assert.Equal(t, "simple-module", mod.Metadata.Name)
	assert.True(t, mod.Raw.Exists())
	// simple-module has values.cue (Pattern A after fixture cleanup).
	assert.True(t, mod.Values.Exists(), "Values must be populated from values.cue")
}
