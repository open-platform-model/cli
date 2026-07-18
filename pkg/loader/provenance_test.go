package loader

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func writeFile(t *testing.T, path, content string) {
	t.Helper()
	require.NoError(t, os.MkdirAll(filepath.Dir(path), 0o755))
	require.NoError(t, os.WriteFile(path, []byte(content), 0o644))
}

func TestModuleRootFrom(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cue.mod", "module.cue"), `module: "example.com/main@v0"`)
	sub := filepath.Join(root, "components", "web")
	require.NoError(t, os.MkdirAll(sub, 0o755))

	assert.Equal(t, root, ModuleRootFrom(sub), "should walk up to the module root")
	assert.Equal(t, root, ModuleRootFrom(root))
	assert.Equal(t, "", ModuleRootFrom(t.TempDir()), "a dir with no cue.mod yields no root")
}

func TestHasLocalModuleReplacement_DetectsReplaceWith(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cue.mod", "module.cue"), `module: "example.com/main@v0"
language: version: "v0.17.0"`)
	writeFile(t, filepath.Join(root, "cue.mod", "local-module.cue"),
		`deps: "opmodel.dev/modules/podinfo@v0": replaceWith: "../podinfo"`)

	assert.True(t, HasLocalModuleReplacement(root))
}

func TestHasLocalModuleReplacement_ForkReplacementCounts(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cue.mod", "local-module.cue"),
		`deps: "opmodel.dev/modules/podinfo@v0": replaceWith: "example.com/fork@v0"`)

	assert.True(t, HasLocalModuleReplacement(root), "an alternative-module fork replacement also marks the render")
}

func TestHasLocalModuleReplacement_AbsentIsRegistry(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cue.mod", "module.cue"), `module: "example.com/main@v0"`)

	assert.False(t, HasLocalModuleReplacement(root), "no local-module.cue → registry provenance")
}

func TestHasLocalModuleReplacement_DepsWithoutReplaceWith(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cue.mod", "local-module.cue"),
		`deps: "opmodel.dev/modules/podinfo@v0": v: "v0.1.0"`)

	assert.False(t, HasLocalModuleReplacement(root), "a deps entry without replaceWith is not a replacement")
}

func TestHasLocalModuleReplacement_EmptyRootIsFalse(t *testing.T) {
	assert.False(t, HasLocalModuleReplacement(""))
}
