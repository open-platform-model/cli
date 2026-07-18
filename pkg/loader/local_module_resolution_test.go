package loader

import (
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestLoaderHonorsLocalModuleReplaceWith proves the CLI's instance-file loader
// resolves a cue.mod/local-module.cue `replaceWith` onto a local, never-published
// module directory — no registry required (enhancement 0006 D9/D37). The loader
// goes through CUE's standard load.Instances, so the SDK's local-module handling
// applies.
func TestLoaderHonorsLocalModuleReplaceWith(t *testing.T) {
	// Hermetic: isolate the CUE cache so resolution cannot reach a shared cache
	// or the network — the local replaceWith must satisfy the import on its own.
	t.Setenv("CUE_CACHE_DIR", t.TempDir())

	base := t.TempDir()
	mainDir := filepath.Join(base, "main")
	libDir := filepath.Join(base, "lib")

	// Never-published library module the main module imports.
	writeFile(t, filepath.Join(libDir, "cue.mod", "module.cue"),
		"module: \"test.example/lib@v0\"\nlanguage: version: \"v0.17.0\"\n")
	writeFile(t, filepath.Join(libDir, "lib.cue"),
		"package lib\n\n#Thing: {x: int}\n")

	// Main module: the dep exists solely to be replaced, so its version is
	// omitted (a never-published path resolves purely via the local replaceWith).
	writeFile(t, filepath.Join(mainDir, "cue.mod", "module.cue"),
		"module: \"test.example/main@v0\"\nlanguage: version: \"v0.17.0\"\ndeps: \"test.example/lib@v0\": {}\n")
	writeFile(t, filepath.Join(mainDir, "cue.mod", "local-module.cue"),
		"deps: \"test.example/lib@v0\": replaceWith: \"../lib\"\n")
	writeFile(t, filepath.Join(mainDir, "instance.cue"),
		"package main\n\nimport lib \"test.example/lib@v0\"\n\nout: lib.#Thing & {x: 1}\n")

	// Sanity: our provenance detector sees the replacement.
	require.True(t, HasLocalModuleReplacement(mainDir))

	ctx := cuecontext.New()
	val, _, err := LoadInstanceFile(ctx, filepath.Join(mainDir, "instance.cue"), LoadOptions{})
	require.NoError(t, err, "loader must resolve the local-module replaceWith without a registry")

	x, err := val.LookupPath(cue.ParsePath("out.x")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(1), x, "the imported, locally-replaced definition resolved and unified")
}
