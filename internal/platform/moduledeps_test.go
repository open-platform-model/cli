package platform

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/helper/platformmodule"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/pkg/loader"
)

// moduleFileWith is a module's committed cue.mod/module.cue pinning deps
// (path -> version).
func moduleFileWith(t *testing.T, deps map[string]string) []byte {
	t.Helper()
	f := &modfile.File{
		Module:   "example.com/modules/app@v0",
		Language: &modfile.Language{Version: "v0.17.0"},
		Deps:     map[string]*modfile.Dep{},
	}
	for path, v := range deps {
		f.Deps[path] = &modfile.Dep{Version: v}
	}
	data, err := modfile.Format(f)
	require.NoError(t, err)
	return data
}

// newerCore is a core release above the kernel's verified one.
const newerCore = "v2.99.0"

// depsGraph extends fixtureGraph with a core release newer than the verified
// one and a module outside the catalogs namespace.
func depsGraph() *fakeModFiles {
	g := fixtureGraph()
	g.graph["opmodel.dev/core@"+newerCore] = nil
	g.graph["example.com/lib@v0.1.0"] = nil
	return g
}

func TestModuleDepsRoots(t *testing.T) {
	verified := schema.DefaultSchemaVersion()
	tests := []struct {
		name        string
		deps        map[string]string
		wantEntries []platformmodule.Entry
		wantCore    string
	}{
		{
			name:        "catalog plus core",
			deps:        map[string]string{"opmodel.dev/catalogs/opm@v4": "v4.0.1", platformmodule.CorePath: verified},
			wantEntries: []platformmodule.Entry{{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true}},
			wantCore:    verified,
		},
		{
			name:        "no catalog",
			deps:        map[string]string{platformmodule.CorePath: verified},
			wantEntries: []platformmodule.Entry{},
			wantCore:    verified,
		},
		{
			name: "non-OPM deps ignored",
			deps: map[string]string{
				"opmodel.dev/catalogs/opm@v4": "v4.0.1",
				"example.com/lib@v0":          "v0.1.0",
				platformmodule.CorePath:       verified,
			},
			wantEntries: []platformmodule.Entry{{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true}},
			wantCore:    verified,
		},
		{
			name:        "older core raised to the verified floor",
			deps:        map[string]string{"opmodel.dev/catalogs/opm@v4": "v4.0.1", platformmodule.CorePath: "v2.0.0-alpha.6"},
			wantEntries: []platformmodule.Entry{{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true}},
			wantCore:    verified,
		},
		{
			name:        "newer core kept",
			deps:        map[string]string{"opmodel.dev/catalogs/opm@v4": "v4.0.1", platformmodule.CorePath: newerCore},
			wantEntries: []platformmodule.Entry{{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true}},
			wantCore:    newerCore,
		},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			mf, err := modfile.Parse(moduleFileWith(t, tt.deps), "module.cue")
			require.NoError(t, err)

			entries, roots := moduleDepsRoots(mf)
			assert.Equal(t, tt.wantEntries, entries)
			for _, r := range roots {
				assert.NotEqual(t, "example.com/lib@v0", r.Path, "a non-catalog dependency is never a root")
			}

			pinned, err := platformmodule.Closure(context.Background(), depsGraph(), roots)
			require.NoError(t, err)
			assert.Contains(t, pinned, platformmodule.Dep{Path: platformmodule.CorePath, Version: tt.wantCore})
		})
	}
}

// writeCheckout writes a catalog checkout whose cue.mod/module.cue requires
// deps, and returns its directory.
func writeCheckout(t *testing.T, dir, path string, deps map[string]string) {
	t.Helper()
	f := &modfile.File{Module: path, Language: &modfile.Language{Version: "v0.17.0"}, Deps: map[string]*modfile.Dep{}}
	for p, v := range deps {
		f.Deps[p] = &modfile.Dep{Version: v}
	}
	data, err := modfile.Format(f)
	require.NoError(t, err)
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), data, 0o600))
}

func TestReplacedModFiles(t *testing.T) {
	ctx := context.Background()
	moduleRoot := t.TempDir()
	checkout := filepath.Join(moduleRoot, "..", filepath.Base(moduleRoot)+"-checkout")
	writeCheckout(t, checkout, "opmodel.dev/catalogs/opm@v4", map[string]string{"example.com/lib@v0": "v0.1.0"})
	t.Cleanup(func() { _ = os.RemoveAll(checkout) })

	base := depsGraph()
	src := newReplacedModFiles(base, ModuleDeps{
		ModuleRoot: moduleRoot,
		Replacements: []loader.LocalReplacement{
			{Path: "opmodel.dev/catalogs/opm@v4", ReplaceWith: "../" + filepath.Base(checkout)},
			{Path: "opmodel.dev/catalogs/k8s@v1", ReplaceWith: "example.com/fork@v1", TargetVersion: "v1.0.0-alpha.2"},
		},
	})
	base.graph["example.com/fork@v1.0.0-alpha.2"] = []platformmodule.Dep{{Path: "example.com/lib@v0", Version: "v0.1.0"}}

	t.Run("directory target serves the checkout's module file", func(t *testing.T) {
		mf, err := src.ModFile(ctx, module.MustNewVersion("opmodel.dev/catalogs/opm@v4", "v4.9.9"))
		require.NoError(t, err, "an unpublished pin is served from the checkout")
		assert.Equal(t, "opmodel.dev/catalogs/opm@v4", mf.QualifiedModule())
		assert.Contains(t, mf.Deps, "example.com/lib@v0")
		assert.Empty(t, base.calls, "the registry is not consulted for a directory replacement")
	})

	t.Run("module target delegates at the target version", func(t *testing.T) {
		mf, err := src.ModFile(ctx, module.MustNewVersion("opmodel.dev/catalogs/k8s@v1", "v1.0.0-alpha.2"))
		require.NoError(t, err)
		assert.Contains(t, mf.Deps, "example.com/lib@v0")
		assert.Equal(t, []string{"example.com/fork@v1.0.0-alpha.2"}, base.calls)
	})

	t.Run("other paths delegate unchanged", func(t *testing.T) {
		base.calls = nil
		_, err := src.ModFile(ctx, module.MustNewVersion(platformmodule.CorePath, "v2.0.0-alpha.6"))
		require.NoError(t, err)
		assert.Equal(t, []string{"opmodel.dev/core@v2.0.0-alpha.6"}, base.calls)
	})
}

func TestGenerateModuleDepsModule(t *testing.T) {
	ctx := context.Background()
	catalogAndCore := map[string]string{
		"opmodel.dev/catalogs/opm@v4": "v4.0.1",
		platformmodule.CorePath:       "v2.0.0-alpha.6",
	}

	t.Run("registry entries and transitive pin", func(t *testing.T) {
		cache := t.TempDir()
		dir, entries, carried, err := GenerateModuleDepsModule(ctx,
			ModuleDeps{ModFile: moduleFileWith(t, catalogAndCore), ModFileName: "module.cue"},
			GenerateOptions{CacheDir: cache, ModFiles: depsGraph()})
		require.NoError(t, err)

		assert.Equal(t, cache, filepath.Dir(dir))
		assert.Equal(t, []platformmodule.Entry{{Path: "opmodel.dev/catalogs/opm@v4", Version: "4.0.1", Enable: true}}, entries)
		assert.Empty(t, carried)

		modFile, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
		require.NoError(t, err)
		assert.Contains(t, string(modFile), ModuleDepsPlatformModulePath)
		assert.Contains(t, string(modFile), `"cue.dev/x/k8s.io@v0"`, "the catalog's own dependency is closed over")
		assert.Contains(t, string(modFile), schema.DefaultSchemaVersion(), "core is raised to the verified release")

		platformFile, err := os.ReadFile(filepath.Join(dir, "platform.cue"))
		require.NoError(t, err)
		assert.Contains(t, string(platformFile), `metadata: name: "module-deps"`)
		assert.Contains(t, string(platformFile), `"4.0.1"`)

		_, err = os.Stat(filepath.Join(dir, "cue.mod", "local-module.cue"))
		assert.ErrorIs(t, err, os.ErrNotExist, "no local file when nothing is carried")
	})

	t.Run("carried replacement is written with an absolute target", func(t *testing.T) {
		moduleRoot := t.TempDir()
		checkout := filepath.Join(moduleRoot, "checkout")
		writeCheckout(t, checkout, "opmodel.dev/catalogs/opm@v4", map[string]string{"cue.dev/x/k8s.io@v0": "v0.10.0"})

		dir, _, carried, err := GenerateModuleDepsModule(ctx,
			ModuleDeps{
				ModFile:     moduleFileWith(t, catalogAndCore),
				ModFileName: "module.cue",
				ModuleRoot:  moduleRoot,
				Replacements: []loader.LocalReplacement{
					{Path: "opmodel.dev/catalogs/opm@v4", ReplaceWith: "./checkout"},
					{Path: "example.com/unpinned@v0", ReplaceWith: "./elsewhere"},
				},
			},
			GenerateOptions{CacheDir: t.TempDir(), ModFiles: depsGraph()})
		require.NoError(t, err)

		assert.Equal(t, map[string]string{"opmodel.dev/catalogs/opm@v4": checkout}, carried,
			"only a path the generated module pins is carried")
		local, err := os.ReadFile(filepath.Join(dir, "cue.mod", "local-module.cue"))
		require.NoError(t, err)
		assert.Contains(t, string(local), checkout)
		assert.NotContains(t, string(local), "unpinned")

		entries, err := loader.LocalReplacements(dir)
		require.NoError(t, err, "the carried file parses against the generated module file")
		assert.Equal(t, []loader.LocalReplacement{{Path: "opmodel.dev/catalogs/opm@v4", ReplaceWith: checkout}}, entries)
	})

	t.Run("carried module target is written with its version", func(t *testing.T) {
		graph := depsGraph()
		graph.graph["example.com/fork@v4.0.1"] = []platformmodule.Dep{{Path: "example.com/lib@v0", Version: "v0.1.0"}}

		dir, _, carried, err := GenerateModuleDepsModule(ctx,
			ModuleDeps{
				ModFile:     moduleFileWith(t, catalogAndCore),
				ModFileName: "module.cue",
				Replacements: []loader.LocalReplacement{
					{Path: "opmodel.dev/catalogs/opm@v4", ReplaceWith: "example.com/fork@v4.0.1", TargetVersion: "v4.0.1"},
				},
			},
			GenerateOptions{CacheDir: t.TempDir(), ModFiles: graph})
		require.NoError(t, err)

		assert.Equal(t, map[string]string{"opmodel.dev/catalogs/opm@v4": "example.com/fork@v4"}, carried)
		modFile, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
		require.NoError(t, err)
		assert.Contains(t, string(modFile), `"example.com/lib@v0"`, "the closure walked the fork's requirements")
		entries, err := loader.LocalReplacements(dir)
		require.NoError(t, err)
		assert.Equal(t, []loader.LocalReplacement{{Path: "opmodel.dev/catalogs/opm@v4", ReplaceWith: "example.com/fork@v4", TargetVersion: "v4.0.1"}}, entries)
	})

	t.Run("an unpublished pin served from a checkout generates", func(t *testing.T) {
		moduleRoot := t.TempDir()
		writeCheckout(t, filepath.Join(moduleRoot, "checkout"), "opmodel.dev/catalogs/opm@v4", nil)
		unpublished := map[string]string{"opmodel.dev/catalogs/opm@v4": "v4.9.9", platformmodule.CorePath: "v2.0.0-alpha.6"}

		_, entries, _, err := GenerateModuleDepsModule(ctx,
			ModuleDeps{
				ModFile: moduleFileWith(t, unpublished), ModFileName: "module.cue", ModuleRoot: moduleRoot,
				Replacements: []loader.LocalReplacement{{Path: "opmodel.dev/catalogs/opm@v4", ReplaceWith: "./checkout"}},
			},
			GenerateOptions{CacheDir: t.TempDir(), ModFiles: depsGraph()})
		require.NoError(t, err)
		assert.Equal(t, "4.9.9", entries[0].Version)
	})

	t.Run("second call returns the same directory untouched", func(t *testing.T) {
		cache := t.TempDir()
		deps := ModuleDeps{ModFile: moduleFileWith(t, catalogAndCore), ModFileName: "module.cue"}
		first, _, _, err := GenerateModuleDepsModule(ctx, deps, GenerateOptions{CacheDir: cache, ModFiles: depsGraph()})
		require.NoError(t, err)
		marker := filepath.Join(first, ".reused")
		require.NoError(t, os.WriteFile(marker, []byte("x"), 0o600))

		second, _, _, err := GenerateModuleDepsModule(ctx, deps, GenerateOptions{CacheDir: cache, ModFiles: depsGraph()})
		require.NoError(t, err)
		assert.Equal(t, first, second)
		_, err = os.Stat(marker)
		assert.NoError(t, err, "identical content must not be rewritten")
	})

	t.Run("an unpublished pin fails naming path and version", func(t *testing.T) {
		deps := map[string]string{"opmodel.dev/catalogs/opm@v4": "v4.9.9", platformmodule.CorePath: "v2.0.0-alpha.6"}
		_, _, _, err := GenerateModuleDepsModule(ctx,
			ModuleDeps{ModFile: moduleFileWith(t, deps), ModFileName: "module.cue"},
			GenerateOptions{CacheDir: t.TempDir(), ModFiles: depsGraph()})
		require.Error(t, err)
		assert.Contains(t, err.Error(), "module deps platform: resolving dependency opmodel.dev/catalogs/opm@v4.9.9")
	})

	t.Run("an unparseable module file is the module's error", func(t *testing.T) {
		_, _, _, err := GenerateModuleDepsModule(ctx,
			ModuleDeps{ModFile: []byte("module: "), ModFileName: "/m/cue.mod/module.cue"},
			GenerateOptions{CacheDir: t.TempDir(), ModFiles: depsGraph()})
		require.Error(t, err)
		assert.ErrorIs(t, err, ErrModuleDepsFile)
		assert.Contains(t, err.Error(), "/m/cue.mod/module.cue")
	})
}
