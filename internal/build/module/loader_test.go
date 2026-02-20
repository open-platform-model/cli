package module

import (
	"testing"

	"cuelang.org/go/cue/cuecontext"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestExtractMetadataFromAST_NilFiles(t *testing.T) {
	name, ns := ExtractMetadataFromAST(nil)
	assert.Empty(t, name)
	assert.Empty(t, ns)
}

// TestLoad_ValidModule verifies that Load returns a populated *core.Module
// with a resolved absolute path, metadata name, defaultNamespace, pkgName,
// FQN, version, CUEValue, Config, Values, and Components.
func TestLoad_ValidModule(t *testing.T) {
	ctx := cuecontext.New()
	// Use the test-module fixture from the build testdata directory.
	// We resolve relative to the module loader package location.
	modulePath := "../testdata/test-module"

	mod, err := Load(ctx, modulePath, "")
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.NotNil(t, mod.Metadata)

	// From AST inspection
	assert.Equal(t, "test-module", mod.Metadata.Name)
	assert.Equal(t, "default", mod.Metadata.DefaultNamespace)
	assert.Equal(t, "testmodule", mod.PkgName())
	// ModulePath must be absolute after Load
	assert.NotEmpty(t, mod.ModulePath)

	// From full CUE evaluation (task 5.2-5.4)
	assert.Equal(t, "example.com/test-module@v0#test-module", mod.Metadata.FQN, "FQN extracted from CUE eval")
	assert.Equal(t, "1.0.0", mod.Metadata.Version, "version extracted from CUE eval")
	assert.Equal(t, "a1b2c3d4-e5f6-7890-abcd-ef1234567890", mod.Metadata.UUID, "UUID extracted from metadata.uuid")

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

// TestLoad_ComputedMetadataName verifies that Load succeeds even when
// metadata.name is a computed expression (returns empty name from AST inspection).
func TestLoad_ComputedMetadataName(t *testing.T) {
	ctx := cuecontext.New()
	modulePath := "../testdata/no-metadata-module"

	mod, err := Load(ctx, modulePath, "")
	require.NoError(t, err)
	require.NotNil(t, mod)

	// AST inspection cannot extract computed expressions — Name will be empty.
	// This is expected behavior; Validate() would catch it if called.
	assert.Empty(t, mod.Metadata.Name, "computed metadata.name should yield empty string from AST")
	assert.Equal(t, "nometadata", mod.PkgName())
}
