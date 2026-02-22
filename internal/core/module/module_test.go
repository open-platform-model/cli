package module

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeModuleDir creates a temp directory with a cue.mod/ subdirectory for testing.
func makeModuleDir(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()
	require.NoError(t, os.Mkdir(filepath.Join(dir, "cue.mod"), 0o755))
	return dir
}

// --- ResolvePath tests ---

func TestModule_ResolvePath_ValidPath(t *testing.T) {
	dir := makeModuleDir(t)
	mod := &Module{ModulePath: dir}

	err := mod.ResolvePath()

	require.NoError(t, err)
	assert.Equal(t, dir, mod.ModulePath)
}

func TestModule_ResolvePath_RelativeToAbsolute(t *testing.T) {
	dir := makeModuleDir(t)
	// Change to a known directory and use a relative path
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(dir))

	mod := &Module{ModulePath: "."}

	err = mod.ResolvePath()

	require.NoError(t, err)
	// ModulePath should now be the absolute path of dir
	absDir, _ := filepath.Abs(dir)
	assert.Equal(t, absDir, mod.ModulePath)
}

func TestModule_ResolvePath_NonExistentDirectory(t *testing.T) {
	mod := &Module{ModulePath: "/nonexistent/path/that/does/not/exist"}

	err := mod.ResolvePath()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module directory not found")
}

func TestModule_ResolvePath_MissingCueMod(t *testing.T) {
	dir := t.TempDir() // no cue.mod/ created
	mod := &Module{ModulePath: dir}

	err := mod.ResolvePath()

	assert.Error(t, err)
	assert.Contains(t, err.Error(), "cue.mod")
}

func TestModule_ResolvePath_MutatesModulePath(t *testing.T) {
	dir := makeModuleDir(t)
	orig, err := os.Getwd()
	require.NoError(t, err)
	t.Cleanup(func() { _ = os.Chdir(orig) })
	require.NoError(t, os.Chdir(dir))

	mod := &Module{ModulePath: "."}
	require.NoError(t, mod.ResolvePath())

	// Path must be absolute after resolution
	assert.True(t, filepath.IsAbs(mod.ModulePath), "ModulePath should be absolute after ResolvePath")
}

// --- Validate tests ---

func validModule(t *testing.T, modulePath string) *Module {
	t.Helper()
	ctx := cuecontext.New()
	val := ctx.CompileString(`{}`)
	mod := &Module{
		ModulePath: modulePath,
		Metadata: &ModuleMetadata{
			Name: "my-module",
			FQN:  "example.com/my-module@v0#my-module",
		},
	}
	mod.Raw = val
	return mod
}

func TestModule_Validate_FullyPopulatedPasses(t *testing.T) {
	mod := validModule(t, "/some/path")
	assert.NoError(t, mod.Validate())
}

func TestModule_Validate_NilMetadataFails(t *testing.T) {
	mod := &Module{ModulePath: "/some/path", Metadata: nil}
	err := mod.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata is nil")
}

func TestModule_Validate_EmptyModulePathFails(t *testing.T) {
	mod := validModule(t, "")
	err := mod.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "module path is empty")
}

func TestModule_Validate_EmptyNameFails(t *testing.T) {
	mod := validModule(t, "/some/path")
	mod.Metadata.Name = ""
	err := mod.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "metadata.name is empty")
}

func TestModule_Validate_EmptyFQNFails(t *testing.T) {
	// FQN is now extracted during Load() (full CUE evaluation). Validate() checks it.
	mod := validModule(t, "/some/path")
	mod.Metadata.FQN = "" // explicitly empty — should fail
	err := mod.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "fqn is empty")
}

func TestModule_Validate_ZeroRawValueFails(t *testing.T) {
	// Raw must be set by module.Load(). A zero value means Load() was not called.
	mod := validModule(t, "/some/path")
	mod.Raw = cue.Value{} // reset to zero value
	err := mod.Validate()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "CUE value is not set")
}
