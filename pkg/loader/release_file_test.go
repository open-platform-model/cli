package loader

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// makeReleaseFileFixture creates a temp directory with a cue.mod and a release .cue file.
// Returns the path to the created .cue file.
func makeReleaseFileFixture(t *testing.T, filename, content string) string {
	t.Helper()
	dir := t.TempDir()

	// Create minimal cue.mod so load.Instances can find the module root.
	modDir := filepath.Join(dir, "cue.mod")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(`module: "test.example.com/releases@v0"
language: version: "v0.15.0"
`), 0o644))

	filePath := filepath.Join(dir, filename)
	require.NoError(t, os.WriteFile(filePath, []byte(content), 0o644))
	return filePath
}

// makeModuleFixture creates a temp module directory with a minimal CUE package.
func makeModuleFixture(t *testing.T) string {
	t.Helper()
	dir := t.TempDir()

	modDir := filepath.Join(dir, "cue.mod")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(`module: "example.com/mymodule@v0"
language: version: "v0.15.0"
`), 0o644))

	require.NoError(t, os.WriteFile(filepath.Join(dir, "module.cue"), []byte(`package mymodule

metadata: {
	name:    "my-module"
	version: "0.1.0"
}
#config: {
	replicas: *1 | int
}
`), 0o644))
	return dir
}

// TestLoadReleaseFile tests loading standalone .cue release files.
func TestLoadReleaseFile(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("valid ModuleRelease file", func(t *testing.T) {
		filePath := makeReleaseFileFixture(t, "release.cue", `package releases

kind: "ModuleRelease"
metadata: name: "my-release"
`)
		val, dir, err := LoadReleaseFile(ctx, filePath, "")
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.NoError(t, val.Err())

		kind, kindErr := val.LookupPath(cue.ParsePath("kind")).String()
		require.NoError(t, kindErr)
		assert.Equal(t, "ModuleRelease", kind)
	})

	t.Run("valid BundleRelease file", func(t *testing.T) {
		filePath := makeReleaseFileFixture(t, "bundle_release.cue", `package releases

kind: "BundleRelease"
metadata: name: "my-bundle"
`)
		val, dir, err := LoadReleaseFile(ctx, filePath, "")
		require.NoError(t, err)
		assert.NotEmpty(t, dir)
		assert.NoError(t, val.Err())

		kind, kindErr := val.LookupPath(cue.ParsePath("kind")).String()
		require.NoError(t, kindErr)
		assert.Equal(t, "BundleRelease", kind)
	})

	t.Run("invalid CUE syntax", func(t *testing.T) {
		filePath := makeReleaseFileFixture(t, "bad.cue", `package releases

this is not valid CUE !!!`)
		_, _, err := LoadReleaseFile(ctx, filePath, "")
		require.Error(t, err)
	})

	t.Run("file not found", func(t *testing.T) {
		_, _, err := LoadReleaseFile(ctx, "/nonexistent/path/release.cue", "")
		require.Error(t, err)
	})

	t.Run("returns parent directory", func(t *testing.T) {
		filePath := makeReleaseFileFixture(t, "my_release.cue", `package releases

kind: "ModuleRelease"
`)
		_, dir, err := LoadReleaseFile(ctx, filePath, "")
		require.NoError(t, err)
		assert.Equal(t, filepath.Dir(filePath), dir)
	})
}

// TestLoadModulePackage tests loading a module CUE package from a directory.
func TestLoadModulePackage(t *testing.T) {
	ctx := cuecontext.New()

	t.Run("valid module directory", func(t *testing.T) {
		dir := makeModuleFixture(t)
		val, err := LoadModulePackage(ctx, dir)
		require.NoError(t, err)
		assert.NoError(t, val.Err())

		// The module should have a metadata field.
		name, nameErr := val.LookupPath(cue.ParsePath("metadata.name")).String()
		require.NoError(t, nameErr)
		assert.Equal(t, "my-module", name)
	})

	t.Run("missing directory", func(t *testing.T) {
		_, err := LoadModulePackage(ctx, "/nonexistent/module/dir")
		require.Error(t, err)
		assert.Contains(t, err.Error(), "accessing module directory")
	})

	t.Run("path is a file not directory", func(t *testing.T) {
		dir := t.TempDir()
		filePath := filepath.Join(dir, "somefile.cue")
		require.NoError(t, os.WriteFile(filePath, []byte(`kind: "x"`), 0o644))

		_, err := LoadModulePackage(ctx, filePath)
		require.Error(t, err)
		assert.Contains(t, err.Error(), "not a directory")
	})
}
