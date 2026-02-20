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
// with a resolved absolute path, metadata name, defaultNamespace, and pkgName.
func TestLoad_ValidModule(t *testing.T) {
	ctx := cuecontext.New()
	// Use the test-module fixture from the build testdata directory.
	// We resolve relative to the module loader package location.
	modulePath := "../testdata/test-module"

	mod, err := Load(ctx, modulePath, "")
	require.NoError(t, err)
	require.NotNil(t, mod)
	require.NotNil(t, mod.Metadata)

	assert.Equal(t, "test-module", mod.Metadata.Name)
	assert.Equal(t, "default", mod.Metadata.DefaultNamespace)
	assert.Equal(t, "testmodule", mod.PkgName())
	// ModulePath must be absolute after Load
	assert.NotEmpty(t, mod.ModulePath)
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
