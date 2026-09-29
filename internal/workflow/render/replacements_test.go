package render

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
)

// writeModuleContext writes a module root carrying the given local-module.cue
// (none when local is empty) and returns it.
func writeModuleContext(t *testing.T, local string) string {
	t.Helper()
	root := t.TempDir()
	writeD19File(t, filepath.Join(root, "cue.mod", "module.cue"),
		"module: \"example.com/app@v0\"\nlanguage: version: \"v0.17.0\"\n")
	if local != "" {
		writeD19File(t, filepath.Join(root, "cue.mod", "local-module.cue"), local)
	}
	return root
}

func localFileOf(root string) string {
	return filepath.Join(root, "cue.mod", "local-module.cue")
}

func TestReplacementWarnings_CleanContextIsSilent(t *testing.T) {
	got := replacementWarnings(nil, writeModuleContext(t, ""), nil)

	assert.Empty(t, got, "no rows and no file: nothing to warn about")
	assert.NotNil(t, got, "never nil, so callers range without a nil check")
}

func TestReplacementWarnings_NoModuleContextReadsNoFile(t *testing.T) {
	assert.Empty(t, replacementWarnings(nil, "", nil))
}

// A published module renders with no module context: only the platform's
// rows are worded, and a local-module.cue in the working directory is not
// read.
func TestReplacementWarnings_PublishedModuleWordsOnlyPlatformRows(t *testing.T) {
	t.Chdir(writeModuleContext(t, `deps: "test.example/lib@v0": replaceWith: "../lib"`))
	rows := []kernel.Replacement{{Path: "opmodel.dev/catalogs/opm@v4", Target: "/home/dev/catalog_opm", By: "platform"}}

	got := replacementWarnings(rows, "", nil)

	assert.Equal(t, []string{
		"local replacement in effect: opmodel.dev/catalogs/opm@v4 served from /home/dev/catalog_opm (platform); rendered bytes may not correspond to any published build",
	}, got)
}

func TestReplacementWarnings_HonoredPlatformRow(t *testing.T) {
	rows := []kernel.Replacement{{Path: "opmodel.dev/catalogs/opm@v4", Target: "/home/dev/catalog_opm", By: "platform"}}

	got := replacementWarnings(rows, writeModuleContext(t, ""), nil)

	assert.Equal(t, []string{
		"local replacement in effect: opmodel.dev/catalogs/opm@v4 served from /home/dev/catalog_opm (platform); rendered bytes may not correspond to any published build",
	}, got)
}

func TestReplacementWarnings_HonoredInstanceRow(t *testing.T) {
	root := writeModuleContext(t, `deps: "test.example/lib@v0": replaceWith: "../lib"`)
	rows := []kernel.Replacement{{Path: "test.example/lib@v0", Target: "/home/dev/lib", By: "instance"}}

	got := replacementWarnings(rows, root, nil)

	assert.Equal(t, []string{
		"local replacement in effect: test.example/lib@v0 served from /home/dev/lib (instance); rendered bytes may not correspond to any published build",
	}, got, "the row names the kernel's absolute target, and the file's own entry is honored, not inert")
}

func TestReplacementWarnings_InertModuleEntry(t *testing.T) {
	root := writeModuleContext(t, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "../catalog_opm"`)

	got := replacementWarnings(nil, root, nil)

	assert.Equal(t, []string{
		"local replacement of opmodel.dev/catalogs/opm@v4 in " + localFileOf(root) + " is ignored: the platform names that path; redirect it in the platform module's cue.mod/local-module.cue",
	}, got)
}

func TestReplacementWarnings_PlatformRowDoesNotHonourTheModuleEntry(t *testing.T) {
	// The platform replaced the same path: its row is in effect, the module's
	// own redirect of that path is still inert (the platform names it).
	root := writeModuleContext(t, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "../mine"`)
	rows := []kernel.Replacement{{Path: "opmodel.dev/catalogs/opm@v4", Target: "/home/dev/theirs", By: "platform"}}

	got := replacementWarnings(rows, root, nil)

	require.Len(t, got, 2)
	assert.Contains(t, got[0], "in effect: opmodel.dev/catalogs/opm@v4 served from /home/dev/theirs (platform)")
	assert.Contains(t, got[1], "local replacement of opmodel.dev/catalogs/opm@v4 in "+localFileOf(root)+" is ignored")
}

func TestReplacementWarnings_CarriedCatalogIsTheModules(t *testing.T) {
	// A module-deps platform carried the module's own redirect: the kernel
	// reports the row as the platform's, but it is the module's, and the
	// module file's entry is honored rather than ignored.
	root := writeModuleContext(t, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "../catalog_opm/opm"`)
	rows := []kernel.Replacement{{Path: "opmodel.dev/catalogs/opm@v4", Target: "/home/dev/catalog_opm/opm", By: "platform"}}
	carried := map[string]string{"opmodel.dev/catalogs/opm@v4": "/home/dev/catalog_opm/opm"}

	got := replacementWarnings(rows, root, carried)

	assert.Equal(t, []string{
		"local replacement in effect: opmodel.dev/catalogs/opm@v4 served from /home/dev/catalog_opm/opm (instance); rendered bytes may not correspond to any published build",
	}, got, "one in-effect line under the module side's label, no ignored line")
}

func TestReplacementWarnings_CarriedSetOnlyClaimsItsOwnTarget(t *testing.T) {
	// A platform row for the same path with a different target is not the
	// carried redirect: it keeps the platform label and the module entry is
	// still reported ignored.
	root := writeModuleContext(t, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "../mine"`)
	rows := []kernel.Replacement{{Path: "opmodel.dev/catalogs/opm@v4", Target: "/home/dev/theirs", By: "platform"}}

	got := replacementWarnings(rows, root, map[string]string{"opmodel.dev/catalogs/opm@v4": "/home/dev/mine"})

	require.Len(t, got, 2)
	assert.Contains(t, got[0], "served from /home/dev/theirs (platform)")
	assert.Contains(t, got[1], "is ignored")
}

func TestReplacementWarnings_MixedKeepsRowOrderThenInertByPath(t *testing.T) {
	root := writeModuleContext(t, `deps: {
	"opmodel.dev/catalogs/opm@v4": replaceWith: "../catalog_opm"
	"test.example/lib@v0": replaceWith: "../lib"
	"opmodel.dev/catalogs/k8s@v1": replaceWith: "../catalog_k8s"
}`)
	// Rows arrive in the kernel's path order; the builder does not reorder them.
	rows := []kernel.Replacement{
		{Path: "example.com/shared@v1", Target: "/home/dev/shared", By: "platform"},
		{Path: "test.example/lib@v0", Target: "/home/dev/lib", By: "instance"},
	}

	got := replacementWarnings(rows, root, nil)

	require.Len(t, got, 4)
	assert.Contains(t, got[0], "in effect: example.com/shared@v1 served from /home/dev/shared (platform)")
	assert.Contains(t, got[1], "in effect: test.example/lib@v0 served from /home/dev/lib (instance)")
	assert.Contains(t, got[2], "local replacement of opmodel.dev/catalogs/k8s@v1 in ")
	assert.Contains(t, got[3], "local replacement of opmodel.dev/catalogs/opm@v4 in ")
	for _, w := range got[:2] {
		assert.Contains(t, w, "rendered bytes may not correspond to any published build")
	}
	for _, w := range got[2:] {
		assert.Contains(t, w, "the platform names that path; redirect it in the platform module's cue.mod/local-module.cue")
	}
}

func TestReplacementWarnings_StableAcrossCalls(t *testing.T) {
	root := writeModuleContext(t, `deps: {
	"b.example/two@v0": replaceWith: "./two"
	"a.example/one@v0": replaceWith: "./one"
}`)
	rows := []kernel.Replacement{{Path: "c.example/three@v0", Target: "/three", By: "platform"}}

	first := replacementWarnings(rows, root, nil)
	for range 5 {
		assert.Equal(t, first, replacementWarnings(rows, root, nil))
	}
}

func TestReplacementWarnings_UnparseableFileYieldsRowsOnly(t *testing.T) {
	root := writeModuleContext(t, `deps: {`)
	rows := []kernel.Replacement{{Path: "test.example/lib@v0", Target: "/home/dev/lib", By: "instance"}}

	got := replacementWarnings(rows, root, nil)

	assert.Len(t, got, 1, "the kernel refuses a malformed file before any row exists; the builder stays defensive")
	assert.Contains(t, got[0], "in effect: test.example/lib@v0")
	_, statErr := os.Stat(localFileOf(root))
	require.NoError(t, statErr, "the file is present, only unparseable")
}
