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

// TestLoadModule_PathResolution covers path resolution behavior:
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
	_ = os.Unsetenv("CUE_REGISTRY")
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
// containing values_prod.cue loads without error. The loader filters all values*.cue
// files from the package load — no values are loaded by the loader in v1alpha1
// (values discovery is the builder's responsibility at build time).
func TestLoadModule_ExtraValuesFilesFilteredSilently(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "extra-values-module"), "")
	require.NoError(t, err, "module with values_prod.cue must load without error")
	require.NotNil(t, mod)

	// The module itself must be valid — metadata, config, components present.
	assert.True(t, mod.Raw.Exists(), "mod.Raw must be set")
	assert.True(t, mod.Config.Exists(), "#config must be extracted")
	assert.NotEmpty(t, mod.Metadata.Name, "metadata.name must be populated")
}

// ---------------------------------------------------------------------------
// Approach A: explicit filtered file list, Pattern A modules
// ---------------------------------------------------------------------------

// TestLoadModule_ApproachA_ModuleRawHasNoConcreteValues proves that after loading
// a module, mod.Raw does not contain a top-level values field. In v1alpha1, modules
// do not define a values field — defaults live in #config, and user values are
// supplied at build time via --values / -f flags.
func TestLoadModule_ApproachA_ModuleRawHasNoConcreteValues(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module"), "")
	require.NoError(t, err)

	valuesInRaw := mod.Raw.LookupPath(cue.ParsePath("values"))
	assert.False(t, valuesInRaw.Exists(),
		"mod.Raw must not have a concrete values field after LoadModule: values.cue was excluded from load.Instances")
}

// TestLoadModule_ApproachA_PackageFilesAllRetained proves that filtering values*.cue
// does not accidentally drop other .cue files. All non-values package content
// (metadata, #config, #components) must be present in mod.Raw after load.
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
// Metadata extraction
// ---------------------------------------------------------------------------

// TestLoadModule_Success verifies that LoadModule returns a fully populated *module.Module
// with all fields set and that the module passes Validate().
// Uses the test-module fixture.
func TestLoadModule_Success(t *testing.T) {
	ctx := cuecontext.New()
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module"), "")
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.NotNil(t, mod.Metadata)

	assert.True(t, filepath.IsAbs(mod.ModulePath))
	assert.Equal(t, "test-module", mod.Metadata.Name)
	assert.Equal(t, "example.com/modules", mod.Metadata.ModulePath)
	assert.Equal(t, "example.com/modules/test-module:1.0.0", mod.Metadata.FQN)
	assert.Equal(t, "1.0.0", mod.Metadata.Version)
	assert.NotEmpty(t, mod.Metadata.UUID)
	assert.Equal(t, "default", mod.Metadata.DefaultNamespace)

	assert.True(t, mod.Raw.Exists(), "Raw must be set after LoadModule")
	assert.True(t, mod.Config.Exists(), "#config must be extracted")
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
	assert.Equal(t, "example.com/modules", mod.Metadata.ModulePath)
	assert.Equal(t, "1.0.0", mod.Metadata.Version)
	assert.Equal(t, "example.com/modules/computed-module:1.0.0", mod.Metadata.FQN)

	// Absent fields remain zero values.
	assert.Empty(t, mod.Metadata.UUID)
	assert.Empty(t, mod.Metadata.DefaultNamespace)

	assert.True(t, mod.Raw.Exists())
}

// TestLoadModule_NoValues verifies that any module loads without error since all
// modules in v1alpha1 carry defaults in #config rather than in a values.cue file.
// The loader does not load values — values discovery is the builder's responsibility.
func TestLoadModule_NoValues(t *testing.T) {
	ctx := cuecontext.New()
	// test-module has no values.cue; it carries defaults inside #config.
	mod, err := loader.LoadModule(ctx, fixture(t, "test-module"), "")
	require.NoError(t, err)
	require.NotNil(t, mod)

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
	assert.True(t, mod.Config.Exists(), "#config is present (simple-module defines it with defaults)")
	assert.Equal(t, "simple-module", mod.Metadata.Name)
	assert.True(t, mod.Raw.Exists())
}
