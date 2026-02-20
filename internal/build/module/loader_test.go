package module

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoad_ValidModule verifies that Load returns a populated *core.Module
// with a resolved absolute path, metadata name, defaultNamespace, pkgName,
// FQN, version, UUID, CUEValue, Config, Values, and Components.
func TestLoad_ValidModule(t *testing.T) {
	ctx := cuecontext.New()
	// Use the test-module fixture from the build testdata directory.
	// We resolve relative to the module loader package location.
	modulePath := "../testdata/test-module"

	mod, err := Load(ctx, modulePath, "")
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.NotNil(t, mod.Metadata)

	// All metadata extracted from CUE evaluation
	assert.Equal(t, "test-module", mod.Metadata.Name)
	assert.Equal(t, "default", mod.Metadata.DefaultNamespace)
	assert.Equal(t, "testmodule", mod.PkgName())
	// ModulePath must be absolute after Load
	assert.NotEmpty(t, mod.ModulePath)

	assert.Equal(t, "example.com/test-module@v0#test-module", mod.Metadata.FQN, "FQN extracted from CUE eval")
	assert.Equal(t, "1.0.0", mod.Metadata.Version, "version extracted from CUE eval")
	assert.NotEmpty(t, mod.Metadata.UUID, "UUID extracted from CUE eval")

	// CUEValue must be set
	assert.True(t, mod.CUEValue().Exists(), "CUEValue must be set after Load")

	// #config and values must be extracted
	assert.True(t, mod.Config.Exists(), "#config must be extracted")
	assert.True(t, mod.Values.Exists(), "values must be extracted")

	// Components must be extracted (test-module has #components)
	assert.NotEmpty(t, mod.Components, "#components must be extracted")
}

// TestLoad_InvalidPath verifies that Load returns an error for a non-existent directory.
func TestLoad_InvalidPath(t *testing.T) {
	ctx := cuecontext.New()

	_, err := Load(ctx, "/nonexistent/path/that/does/not/exist", "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "module directory not found")
}

// TestLoad_ComputedMetadataName verifies that Load correctly resolves computed
// metadata.name expressions via CUE evaluation.
func TestLoad_ComputedMetadataName(t *testing.T) {
	ctx := cuecontext.New()
	modulePath := "../testdata/no-metadata-module"

	mod, err := Load(ctx, modulePath, "")
	require.NoError(t, err)
	require.NotNil(t, mod)

	// CUE evaluation resolves computed expressions — "computed" + "-module" = "computed-module"
	assert.Equal(t, "computed-module", mod.Metadata.Name, "computed metadata.name resolves via CUE evaluation")
	assert.Equal(t, "nometadata", mod.PkgName())
}
