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

// moduleFile is a strict module.cue: the parser the reader shares with
// cue/load and the kernel requires a language version.
const moduleFile = "module: \"example.com/main@v0\"\nlanguage: version: \"v0.17.0\"\n"

// writeModuleRoot writes a module root with the given local-module.cue (none
// when local is empty) and returns it.
func writeModuleRoot(t *testing.T, local string) string {
	t.Helper()
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cue.mod", "module.cue"), moduleFile)
	if local != "" {
		writeFile(t, filepath.Join(root, "cue.mod", "local-module.cue"), local)
	}
	return root
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

func TestLocalReplacements_AbsentFileIsEmpty(t *testing.T) {
	entries, err := LocalReplacements(writeModuleRoot(t, ""))
	require.NoError(t, err)
	assert.Empty(t, entries, "no local-module.cue is the normal case, not an error")
}

func TestLocalReplacements_EmptyRootIsEmpty(t *testing.T) {
	entries, err := LocalReplacements("")
	require.NoError(t, err)
	assert.Empty(t, entries)
}

func TestLocalReplacements_DirectoryEntry(t *testing.T) {
	root := writeModuleRoot(t, `deps: "opmodel.dev/modules/podinfo@v0": replaceWith: "../podinfo"`)

	entries, err := LocalReplacements(root)
	require.NoError(t, err)
	assert.Equal(t, []LocalReplacement{{Path: "opmodel.dev/modules/podinfo@v0", ReplaceWith: "../podinfo"}}, entries,
		"a directory target is returned as written, relative to the module root")
}

func TestLocalReplacements_ModulePathEntry(t *testing.T) {
	root := writeModuleRoot(t, `deps: "opmodel.dev/modules/podinfo@v0": replaceWith: "example.com/fork@v0"`)

	entries, err := LocalReplacements(root)
	require.NoError(t, err)
	assert.Equal(t, []LocalReplacement{{Path: "opmodel.dev/modules/podinfo@v0", ReplaceWith: "example.com/fork@v0"}}, entries)
}

func TestLocalReplacements_PinsWithoutReplaceWithAreNotEntries(t *testing.T) {
	root := writeModuleRoot(t, `deps: {
	"opmodel.dev/modules/podinfo@v0": v: "v0.1.0"
	"example.com/lib@v0": replaceWith: "/abs/lib"
}`)

	entries, err := LocalReplacements(root)
	require.NoError(t, err)
	assert.Equal(t, []LocalReplacement{{Path: "example.com/lib@v0", ReplaceWith: "/abs/lib"}}, entries,
		"a deps entry that only pins a version is not a replacement")
}

func TestLocalReplacements_SortedByPath(t *testing.T) {
	root := writeModuleRoot(t, `deps: {
	"z.example/last@v0": replaceWith: "./z"
	"a.example/first@v0": replaceWith: "./a"
	"m.example/middle@v0": replaceWith: "./m"
}`)

	entries, err := LocalReplacements(root)
	require.NoError(t, err)
	paths := make([]string, 0, len(entries))
	for _, e := range entries {
		paths = append(paths, e.Path)
	}
	assert.Equal(t, []string{"a.example/first@v0", "m.example/middle@v0", "z.example/last@v0"}, paths)
}

func TestLocalReplacements_MalformedFileNamesIt(t *testing.T) {
	root := writeModuleRoot(t, `deps: {`)

	entries, err := LocalReplacements(root)
	require.Error(t, err)
	assert.Nil(t, entries)
	assert.Contains(t, err.Error(), filepath.Join("cue.mod", "local-module.cue"))
}

func TestLocalReplacements_MissingModuleFileIsAnError(t *testing.T) {
	root := t.TempDir()
	writeFile(t, filepath.Join(root, "cue.mod", "local-module.cue"),
		`deps: "opmodel.dev/modules/podinfo@v0": replaceWith: "../podinfo"`)

	_, err := LocalReplacements(root)
	require.Error(t, err, "the local file is read against module.cue; without it nothing can be resolved")
	assert.Contains(t, err.Error(), filepath.Join("cue.mod", "module.cue"))
}

func TestHasLocalModuleReplacement_DetectsReplaceWith(t *testing.T) {
	root := writeModuleRoot(t, `deps: "opmodel.dev/modules/podinfo@v0": replaceWith: "../podinfo"`)

	assert.True(t, HasLocalModuleReplacement(root))
}

func TestHasLocalModuleReplacement_ForkReplacementCounts(t *testing.T) {
	root := writeModuleRoot(t, `deps: "opmodel.dev/modules/podinfo@v0": replaceWith: "example.com/fork@v0"`)

	assert.True(t, HasLocalModuleReplacement(root), "an alternative-module fork replacement also marks the render")
}

func TestHasLocalModuleReplacement_AbsentIsRegistry(t *testing.T) {
	assert.False(t, HasLocalModuleReplacement(writeModuleRoot(t, "")), "no local-module.cue → registry provenance")
}

func TestHasLocalModuleReplacement_DepsWithoutReplaceWith(t *testing.T) {
	root := writeModuleRoot(t, `deps: "opmodel.dev/modules/podinfo@v0": v: "v0.1.0"`)

	assert.False(t, HasLocalModuleReplacement(root), "a deps entry without replaceWith is not a replacement")
}

func TestHasLocalModuleReplacement_MalformedCountsAsLocal(t *testing.T) {
	root := writeModuleRoot(t, `deps: {`)

	assert.True(t, HasLocalModuleReplacement(root), "a present file that does not parse only exists to carry a replacement")
}

func TestHasLocalModuleReplacement_EmptyRootIsFalse(t *testing.T) {
	assert.False(t, HasLocalModuleReplacement(""))
}
