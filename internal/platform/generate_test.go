package platform

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/platformmodule"
	"github.com/open-platform-model/library/opm/schema"
)

// fakeModFiles is a fixture module graph: module version string -> the paths
// and versions that module requires. A version absent from the map is an
// unpublished build. No network, no cache.
type fakeModFiles struct {
	graph map[string][]platformmodule.Dep
	calls []string
}

func (f *fakeModFiles) ModFile(_ context.Context, mv module.Version) (*modfile.File, error) {
	f.calls = append(f.calls, mv.String())
	deps, ok := f.graph[mv.String()]
	if !ok {
		return nil, fmt.Errorf("module %s: module not found", mv)
	}
	mf := &modfile.File{Module: mv.Path(), Deps: map[string]*modfile.Dep{}}
	for _, d := range deps {
		mf.Deps[d.Path] = &modfile.Dep{Version: d.Version}
	}
	if err := mf.Init(); err != nil {
		return nil, err
	}
	return mf, nil
}

// fixtureGraph publishes core at the kernel's verified release (what Roots
// pins) and one catalog build requiring an older core plus a transitive
// dependency.
func fixtureGraph() *fakeModFiles {
	return &fakeModFiles{graph: map[string][]platformmodule.Dep{
		"opmodel.dev/core@" + schema.DefaultSchemaVersion(): nil,
		"opmodel.dev/core@v2.0.0-alpha.6":                   nil,
		"opmodel.dev/catalogs/opm@v4.0.1": {
			{Path: "cue.dev/x/k8s.io@v0", Version: "v0.10.0"},
			{Path: platformmodule.CorePath, Version: "v2.0.0-alpha.6"},
		},
		"cue.dev/x/k8s.io@v0.10.0": nil,
	}}
}

func clusterSpec() Spec {
	return Spec{
		Name:    "cluster",
		Type:    "kubernetes",
		Entries: []Entry{{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true}},
	}
}

func TestGenerateClusterModule_WritesModuleUnderContentHash(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache", "platforms")

	dir, err := GenerateClusterModule(context.Background(), clusterSpec(), GenerateOptions{CacheDir: cache, ModFiles: fixtureGraph()})
	require.NoError(t, err)

	assert.Equal(t, cache, filepath.Dir(dir), "the module lives directly under the cache")
	assert.Len(t, filepath.Base(dir), 64, "the directory is named by the hex SHA-256 of its files")

	modFile, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(modFile), ClusterPlatformModulePath)
	assert.Contains(t, string(modFile), `"opmodel.dev/catalogs/opm@v4"`)
	assert.Contains(t, string(modFile), `"v4.0.1"`)
	assert.Contains(t, string(modFile), schema.DefaultSchemaVersion(), "core is pinned at the library's verified release")
	assert.Contains(t, string(modFile), `"cue.dev/x/k8s.io@v0"`, "the transitive closure is pinned")

	platformFile, err := os.ReadFile(filepath.Join(dir, "platform.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(platformFile), `"opmodel.dev/catalogs/opm@v4"`)
	assert.Contains(t, string(platformFile), `"4.0.1"`, "the subscription version is stamped as the entry's expected version")
}

func TestGenerateClusterModule_UnchangedSpecReusesDirectory(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache", "platforms")
	src := fixtureGraph()

	first, err := GenerateClusterModule(context.Background(), clusterSpec(), GenerateOptions{CacheDir: cache, ModFiles: src})
	require.NoError(t, err)
	// A marker survives only if the directory is reused, never rewritten.
	marker := filepath.Join(first, ".reused")
	require.NoError(t, os.WriteFile(marker, []byte("x"), 0o600))

	second, err := GenerateClusterModule(context.Background(), clusterSpec(), GenerateOptions{CacheDir: cache, ModFiles: src})
	require.NoError(t, err)

	assert.Equal(t, first, second)
	_, err = os.Stat(marker)
	assert.NoError(t, err, "identical content must not be rewritten")
	entries, err := os.ReadDir(cache)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "no staging leftovers")
}

func TestGenerateClusterModule_StaleDirectoryIsReplaced(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache", "platforms")

	dir, err := GenerateClusterModule(context.Background(), clusterSpec(), GenerateOptions{CacheDir: cache, ModFiles: fixtureGraph()})
	require.NoError(t, err)
	// Corrupt one file under the hash name: the content no longer matches
	// the name, so the next generation must rewrite it.
	platformFile := filepath.Join(dir, "platform.cue")
	require.NoError(t, os.WriteFile(platformFile, []byte("// corrupted\n"), 0o600))

	again, err := GenerateClusterModule(context.Background(), clusterSpec(), GenerateOptions{CacheDir: cache, ModFiles: fixtureGraph()})
	require.NoError(t, err)
	assert.Equal(t, dir, again)
	restored, err := os.ReadFile(platformFile)
	require.NoError(t, err)
	assert.NotEqual(t, "// corrupted\n", string(restored))
}

func TestGenerateClusterModule_DifferentSpecDifferentDirectory(t *testing.T) {
	cache := filepath.Join(t.TempDir(), "cache", "platforms")

	a, err := GenerateClusterModule(context.Background(), clusterSpec(), GenerateOptions{CacheDir: cache, ModFiles: fixtureGraph()})
	require.NoError(t, err)
	disabled := clusterSpec()
	disabled.Entries[0].Enable = false
	b, err := GenerateClusterModule(context.Background(), disabled, GenerateOptions{CacheDir: cache, ModFiles: fixtureGraph()})
	require.NoError(t, err)

	assert.NotEqual(t, a, b)
}

func TestGenerateClusterModule_MissingVersionFailsBeforeRegistry(t *testing.T) {
	src := fixtureGraph()
	legacy := Spec{Name: "cluster", Type: "kubernetes", Entries: []Entry{{Path: "opmodel.dev/catalogs/opm", Enable: true}}}

	_, err := GenerateClusterModule(context.Background(), legacy, GenerateOptions{CacheDir: t.TempDir(), ModFiles: src})
	require.Error(t, err)
	assert.ErrorIs(t, err, ErrEntryMissingVersion)
	assert.Contains(t, err.Error(), `"opmodel.dev/catalogs/opm"`)
	assert.Contains(t, err.Error(), "predate the scalar-version subscription shape")
	assert.Empty(t, src.calls, "the legacy-CR refusal must precede any registry access")
}

func TestGenerateClusterModule_UnpublishedPinNamesIt(t *testing.T) {
	spec := clusterSpec()
	spec.Entries[0].Version = "4.9.9"

	_, err := GenerateClusterModule(context.Background(), spec, GenerateOptions{CacheDir: t.TempDir(), ModFiles: fixtureGraph()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "opmodel.dev/catalogs/opm@v4.9.9")
}

func TestGenerateClusterModule_RequiresCacheDir(t *testing.T) {
	_, err := GenerateClusterModule(context.Background(), clusterSpec(), GenerateOptions{ModFiles: fixtureGraph()})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cache directory")
}

func TestHashFiles_OrderIndependent(t *testing.T) {
	a := platformmodule.Files{"x": []byte("1"), "y": []byte("2")}
	b := platformmodule.Files{"y": []byte("2"), "x": []byte("1")}
	assert.Equal(t, hashFiles(a), hashFiles(b))
	assert.NotEqual(t, hashFiles(a), hashFiles(platformmodule.Files{"x": []byte("1"), "y": []byte("3")}))
}
