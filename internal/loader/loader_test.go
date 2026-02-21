package loader_test

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/opmodel/cli/internal/loader"
)

// legacyFixture returns the absolute path to a named module in the loader
// testdata directory.
func legacyFixture(t *testing.T, name string) string {
	t.Helper()
	// filepath.Abs never errors on Linux/macOS; path validity is checked by Load.
	abs, _ := filepath.Abs(filepath.Join("testdata", name))
	return abs
}

// invalidFixture returns the absolute path to a named module in the local
// testdata/invalid directory, which contains deliberately broken CUE modules
// for error-path testing.
func invalidFixture(t *testing.T, name string) string {
	t.Helper()
	abs, _ := filepath.Abs(filepath.Join("testdata", "invalid", name))
	return abs
}

// TestLoad_PathResolution covers path resolution behavior:
// relative paths are resolved, non-existent paths are rejected,
// and directories missing cue.mod/ are rejected.
func TestLoad_PathResolution(t *testing.T) {
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
			mod, err := loader.Load(ctx, tc.modulePath(t), "")
			if tc.wantErr {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tc.errContains)
				assert.Nil(t, mod)
			} else {
				require.NoError(t, err)
				assert.NotNil(t, mod)
				assert.True(t, filepath.IsAbs(mod.ModulePath), "ModulePath must be absolute after Load")
			}
		})
	}
}

// TestLoad_RegistryEnvVarIsCleanedUp verifies that CUE_REGISTRY is set during
// Load when a registry is provided and is unset after Load returns (defer fires).
func TestLoad_RegistryEnvVarIsCleanedUp(t *testing.T) {
	// Ensure no pre-existing value interferes.
	os.Unsetenv("CUE_REGISTRY") //nolint:errcheck

	ctx := cuecontext.New()
	// The test-module fixture has no external CUE dependencies, so load.Instances
	// succeeds even with a non-existent registry string.
	mod, err := loader.Load(ctx, legacyFixture(t, "test-module"), "fake-registry.test")
	require.NoError(t, err)
	assert.NotNil(t, mod)

	// The defer os.Unsetenv inside Load must have fired by the time Load returns.
	assert.Empty(t, os.Getenv("CUE_REGISTRY"), "CUE_REGISTRY must be unset after Load returns")
}

// TestLoad_InstanceLoadError verifies that a CUE syntax error in the module
// file causes Load to return an error wrapping inst.Err.
func TestLoad_InstanceLoadError(t *testing.T) {
	ctx := cuecontext.New()
	_, err := loader.Load(ctx, invalidFixture(t, "syntax-error"), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "loading module:")
}

// TestLoad_EvaluationError verifies that a CUE type conflict that is valid
// syntax but fails at evaluation causes Load to return a wrapped error.
func TestLoad_EvaluationError(t *testing.T) {
	ctx := cuecontext.New()
	_, err := loader.Load(ctx, invalidFixture(t, "eval-error"), "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "evaluating module CUE:")
}

// TestLoad_Success verifies that Load returns a fully populated *core.Module
// with all fields set and that the module passes Validate().
// Uses the test-module fixture which defines all metadata, #config, values, and #components.
func TestLoad_Success(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.Load(ctx, legacyFixture(t, "test-module"), "")
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.NotNil(t, mod.Metadata)

	// Path is resolved to absolute
	assert.True(t, filepath.IsAbs(mod.ModulePath))

	// All metadata fields populated
	assert.Equal(t, "test-module", mod.Metadata.Name)
	assert.Equal(t, "example.com/test-module@v0#test-module", mod.Metadata.FQN)
	assert.Equal(t, "1.0.0", mod.Metadata.Version)
	assert.NotEmpty(t, mod.Metadata.UUID)
	assert.Equal(t, "default", mod.Metadata.DefaultNamespace)
	assert.NotEmpty(t, mod.Metadata.Labels)

	// Raw is set
	assert.True(t, mod.Raw.Exists(), "Raw must be set after Load")

	// #config and values extracted
	assert.True(t, mod.Config.Exists(), "#config must be extracted")
	assert.True(t, mod.Values.Exists(), "values must be extracted")

	// #components extracted
	assert.NotEmpty(t, mod.Components, "#components must be extracted")

	// Validate passes
	require.NoError(t, mod.Validate())
}

// TestLoad_PartialMetadata verifies that a module with only some metadata fields
// defined loads without error, with absent fields as zero values.
// Uses no-metadata-module which has name and version but no fqn, uuid, defaultNamespace, or labels.
func TestLoad_PartialMetadata(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.Load(ctx, legacyFixture(t, "no-metadata-module"), "")
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.NotNil(t, mod.Metadata)

	// Present fields are populated
	assert.Equal(t, "computed-module", mod.Metadata.Name)
	assert.Equal(t, "1.0.0", mod.Metadata.Version)

	// Absent fields remain zero values — no error
	assert.Empty(t, mod.Metadata.FQN)
	assert.Empty(t, mod.Metadata.UUID)
	assert.Empty(t, mod.Metadata.DefaultNamespace)
	assert.Empty(t, mod.Metadata.Labels)

	// Raw is still set
	assert.True(t, mod.Raw.Exists())
}

// TestLoad_NoValues verifies that a module without a values field loads without
// error and that Module.Values is a zero cue.Value.
// Uses test-module-no-values which defines #config and #components but no values.
func TestLoad_NoValues(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.Load(ctx, legacyFixture(t, "test-module-no-values"), "")
	require.NoError(t, err)
	require.NotNil(t, mod)

	assert.False(t, mod.Values.Exists(), "Values must be zero when absent from module")
	// Other fields are still populated
	assert.True(t, mod.Config.Exists(), "#config is present")
	assert.NotEmpty(t, mod.Components)
}

// TestLoad_NoComponents verifies that a module without #components loads
// without error and that Module.Components is nil.
// Uses the simple-module fixture from tests/fixtures/valid/ which has no #components
// and no #config field.
func TestLoad_NoComponents(t *testing.T) {
	ctx := cuecontext.New()

	fixturesPath, _ := filepath.Abs(filepath.Join("..", "..", "tests", "fixtures", "valid", "simple-module"))

	mod, err := loader.Load(ctx, fixturesPath, "")
	require.NoError(t, err)
	require.NotNil(t, mod)

	// Components is nil when #components is absent
	assert.Nil(t, mod.Components)

	// Config is zero when #config is absent
	assert.False(t, mod.Config.Exists(), "#config must be zero when absent from module")

	// Other fields are populated
	assert.Equal(t, "simple-module", mod.Metadata.Name)
	assert.True(t, mod.Raw.Exists())
	assert.True(t, mod.Values.Exists())
}
