// Package config provides configuration loading and management.
package config

import (
	"context"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/schema"

	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// writePlatform writes content as platform.cue in a fresh temp dir and
// returns its path (legacy data-file shape).
func writePlatform(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	path := filepath.Join(dir, "platform.cue")
	require.NoError(t, os.WriteFile(path, []byte(content), 0o600))
	return path
}

// seedPlatformModule writes the default platform module into a fresh temp
// dir and returns the directory.
func seedPlatformModule(t *testing.T) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), PlatformDirName)
	require.NoError(t, WritePlatformModule(dir))
	return dir
}

// buildCtx returns a bounded context for registry-backed builds.
func buildCtx(t *testing.T) context.Context {
	t.Helper()
	ctx, cancel := context.WithTimeout(context.Background(), 120*time.Second)
	t.Cleanup(cancel)
	return ctx
}

// skipIfRegistryUnavailable skips the test when err looks like the registry
// could not be reached, matching the repo's other registry-backed tests:
// they prove behavior against GHCR when it is reachable and never
// false-fail offline.
func skipIfRegistryUnavailable(t *testing.T, err error) {
	t.Helper()
	if err == nil {
		return
	}
	msg := err.Error()
	for _, needle := range []string{"dial tcp", "no such host", "connection refused", "i/o timeout", "context deadline exceeded", "TLS handshake", "network is unreachable"} {
		if strings.Contains(msg, needle) {
			t.Skipf("registry unavailable: %v", err)
		}
	}
}

func TestPlatformDir_SiblingOfConfig(t *testing.T) {
	dir := filepath.Join("custom", "dir")
	got := PlatformDir(filepath.Join(dir, "config.cue"))
	assert.Equal(t, filepath.Join(dir, "platform"), got)
}

func TestLegacyPlatformFilePath_SiblingOfConfig(t *testing.T) {
	dir := filepath.Join("custom", "dir")
	got := LegacyPlatformFilePath(filepath.Join(dir, "config.cue"))
	assert.Equal(t, filepath.Join(dir, "platform.cue"), got)
}

func TestDefaultPaths_PlatformDirUnderOpmHome(t *testing.T) {
	home := t.TempDir()
	t.Setenv("HOME", home)

	paths, err := DefaultPaths()
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(home, ".opm", "platform"), paths.PlatformDir)
	assert.Equal(t, PlatformDir(paths.ConfigFile), paths.PlatformDir, "DefaultPaths and PlatformDir agree on the sibling rule")
}

func TestDefaultPlatformModuleFile_PinsCoreAndBothCatalogs(t *testing.T) {
	// The seeded cue.mod is the platform's catalog selection (0019 D5): it
	// must parse as a module file, carry the reserved module path, and pin
	// exactly one build for core and for each first-party catalog.
	f, err := modfile.Parse([]byte(DefaultPlatformModuleFile), PlatformModuleFileName)
	require.NoError(t, err)

	assert.Equal(t, DefaultPlatformModulePath, f.Module)
	require.Len(t, f.Deps, 3)
	require.Contains(t, f.Deps, DefaultCorePath)
	assert.Equal(t, DefaultCorePin, f.Deps[DefaultCorePath].Version)
	for i, path := range DefaultCatalogPaths {
		require.Contains(t, f.Deps, path)
		assert.Equal(t, DefaultCatalogPins[i], f.Deps[path].Version, path)
	}
}

func TestDefaultPlatformCUE_EntriesImportTheirCatalogs(t *testing.T) {
	// Exactly two #registry entries keyed by the catalog paths, each
	// carrying its catalog by import; no version scalar, no filter
	// vocabulary, no retired kubernetes catalog.
	src := DefaultPlatformCUE
	assert.Contains(t, src, "core.#Platform")
	assert.Contains(t, src, `metadata: name: "cluster"`)
	assert.Contains(t, src, `type: "kubernetes"`)
	for _, path := range DefaultCatalogPaths {
		assert.Contains(t, src, "\t"+catalogImportName(path)+" \""+path+"\"", "catalog imported under its package name")
		assert.Contains(t, src, "\t\""+path+"\": #catalog: "+catalogImportName(path), "entry keyed by path carries the import")
	}
	assert.Equal(t, 2, strings.Count(src, "#catalog:"), "exactly two registry entries")
	assert.NotContains(t, src, "version:")
	assert.NotContains(t, src, "filter")
	assert.NotContains(t, src, "opmodel.dev/catalogs/kubernetes")
}

func TestCatalogImportName(t *testing.T) {
	assert.Equal(t, "opm", catalogImportName("opmodel.dev/catalogs/opm@v4"))
	assert.Equal(t, "k8s", catalogImportName("opmodel.dev/catalogs/k8s@v1"))
	assert.Equal(t, "core", catalogImportName("opmodel.dev/core@v2"))
}

func TestWritePlatformModule_WritesBothFilesSecurely(t *testing.T) {
	dir := seedPlatformModule(t)

	for _, name := range []string{PlatformModuleFileName, PlatformCUEFileName} {
		info, err := os.Stat(filepath.Join(dir, filepath.FromSlash(name)))
		require.NoError(t, err, name)
		assert.Equal(t, os.FileMode(0o600), info.Mode().Perm(), name)
	}
	for _, sub := range []string{"", "cue.mod"} {
		info, err := os.Stat(filepath.Join(dir, sub))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), info.Mode().Perm(), sub)
	}

	modContent, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	assert.Equal(t, DefaultPlatformModuleFile, string(modContent))
	cueContent, err := os.ReadFile(filepath.Join(dir, "platform.cue"))
	require.NoError(t, err)
	assert.Equal(t, DefaultPlatformCUE, string(cueContent))
}

func TestWritePlatformModule_OverwritesInPlace(t *testing.T) {
	dir := seedPlatformModule(t)
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte("bogus: true\n"), 0o600))

	require.NoError(t, WritePlatformModule(dir))
	cueContent, err := os.ReadFile(filepath.Join(dir, "platform.cue"))
	require.NoError(t, err)
	assert.Equal(t, DefaultPlatformCUE, string(cueContent))
}

func TestBuildPlatformModule_MissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "platform")
	_, err := BuildPlatformModule(context.Background(), dir, DefaultRegistry)
	require.Error(t, err)
	assert.ErrorIs(t, err, oerrors.ErrValidation)
	assert.Contains(t, err.Error(), dir)
	assert.Contains(t, err.Error(), "opm config init")
}

func TestBuildPlatformModule_NotAModule(t *testing.T) {
	// A directory holding a bare platform.cue (the legacy data shape moved
	// into a directory) is not a module: no cue.mod/module.cue.
	dir := filepath.Join(t.TempDir(), "platform")
	require.NoError(t, os.MkdirAll(dir, 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "platform.cue"), []byte("name: \"cluster\"\n"), 0o600))

	_, err := BuildPlatformModule(context.Background(), dir, DefaultRegistry)
	require.Error(t, err)
	assert.ErrorIs(t, err, oerrors.ErrValidation)
	assert.Contains(t, err.Error(), "cue.mod/module.cue")
	assert.Contains(t, err.Error(), "opm config init")
}

func TestBuildPlatformModule_DefaultTemplateBuilds(t *testing.T) {
	// Registry-backed: the seeded module must build against the published
	// core and catalogs, and each entry's version must be the build its
	// cue.mod pins (derived readout, 0019 D5).
	dir := seedPlatformModule(t)

	p, err := BuildPlatformModule(buildCtx(t), dir, DefaultRegistry)
	skipIfRegistryUnavailable(t, err)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, "cluster", p.Metadata.Name)
	assert.Equal(t, "kubernetes", p.Metadata.Type)

	registry := p.Package.LookupPath(schema.Registry)
	require.True(t, registry.Exists())
	for i, path := range DefaultCatalogPaths {
		entry := registry.LookupPath(cue.MakePath(cue.Str(path)))
		require.True(t, entry.Exists(), path)
		version, err := entry.LookupPath(cue.ParsePath("version")).String()
		require.NoError(t, err, path)
		assert.Equal(t, strings.TrimPrefix(DefaultCatalogPins[i], "v"), version, "%s version derived from the pinned catalog", path)
		enable, err := entry.LookupPath(cue.ParsePath("enable")).Bool()
		require.NoError(t, err, path)
		assert.True(t, enable, "%s enabled by default", path)
	}
}

func TestBuildPlatformModule_UnpublishedPinNamesTheDependency(t *testing.T) {
	// Registry-backed: a pin naming a build that does not exist fails the
	// build naming the dependency, with the hint pointing at cue.mod.
	dir := seedPlatformModule(t)
	modPath := filepath.Join(dir, "cue.mod", "module.cue")
	content, err := os.ReadFile(modPath)
	require.NoError(t, err)
	bumped := strings.Replace(string(content), DefaultCatalogPins[0], "v4.9.9", 1)
	require.NotEqual(t, string(content), bumped)
	require.NoError(t, os.WriteFile(modPath, []byte(bumped), 0o600))

	_, err = BuildPlatformModule(buildCtx(t), dir, DefaultRegistry)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)
	assert.ErrorIs(t, err, oerrors.ErrValidation)
	assert.Contains(t, err.Error(), DefaultCatalogPaths[0]+".9.9")
	assert.Contains(t, err.Error(), modPath)
}

func TestBuildPlatformModule_KeyImportDriftNamesTheEntry(t *testing.T) {
	// Registry-backed: an entry keyed at one catalog but embedding the other
	// fails the D5 binding at a path naming the entry.
	dir := seedPlatformModule(t)
	cuePath := filepath.Join(dir, "platform.cue")
	content, err := os.ReadFile(cuePath)
	require.NoError(t, err)
	opmName, k8sName := catalogImportName(DefaultCatalogPaths[0]), catalogImportName(DefaultCatalogPaths[1])
	swapped := strings.NewReplacer(
		"#catalog: "+opmName+"\n", "#catalog: "+k8sName+"\n",
		"#catalog: "+k8sName+"\n", "#catalog: "+opmName+"\n",
	).Replace(string(content))
	require.NotEqual(t, string(content), swapped)
	require.NoError(t, os.WriteFile(cuePath, []byte(swapped), 0o600))

	_, err = BuildPlatformModule(buildCtx(t), dir, DefaultRegistry)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)
	assert.ErrorIs(t, err, oerrors.ErrValidation)
	// CUE reports the conflict at whichever swapped entry it evaluates
	// first; either names a #registry entry by its key.
	msg := err.Error()
	named := strings.Contains(msg, `#registry."`+DefaultCatalogPaths[0]+`"`) ||
		strings.Contains(msg, `#registry."`+DefaultCatalogPaths[1]+`"`)
	assert.True(t, named, "conflict must be reported at a #registry entry path: %s", msg)
	assert.Contains(t, msg, "conflicting values")
	assert.Contains(t, msg, "must equal the module path")
}

// Legacy data-only platform file loader (deleted by cli-render-switch).

func TestLoadLegacyPlatformFile_MinimalValid(t *testing.T) {
	path := writePlatform(t, `name: "cluster"
type: "kubernetes"
`)
	_, err := LoadLegacyPlatformFile(path)
	require.NoError(t, err)
}

func TestLoadLegacyPlatformFile_MissingFile(t *testing.T) {
	_, err := LoadLegacyPlatformFile("/nonexistent/platform.cue")
	require.Error(t, err)
	assert.True(t, os.IsNotExist(err), "missing file should surface as os.IsNotExist")
}

func TestLoadLegacyPlatformFile_ImportsRejected(t *testing.T) {
	path := writePlatform(t, `import "strings"

name: strings.ToLower("Cluster")
type: "kubernetes"
`)
	_, err := LoadLegacyPlatformFile(path)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data-only")
	assert.Contains(t, err.Error(), "opm config init --force")
}

func TestLoadLegacyPlatformFile_SchemaAndSyntaxErrors(t *testing.T) {
	tests := []struct {
		name    string
		content string
		want    string
	}{
		{"missing required fields", "registry: {\n\t\"opmodel.dev/catalogs/opm@v2\": {version: \"2.0.0-alpha.3\"}\n}\n", "platform schema validation failed"},
		{"filter vocabulary rejected", "name: \"cluster\"\ntype: \"kubernetes\"\nregistry: {\n\t\"opmodel.dev/catalogs/opm@v2\": {\n\t\tversion: \"2.0.0-alpha.3\"\n\t\tfilter: range: \">=1.0.0-0 <2.0.0-0\"\n\t}\n}\n", "platform schema validation failed"},
		{"unknown field rejected", "name: \"cluster\"\ntype: \"kubernetes\"\nbogus: true\n", "platform schema validation failed"},
		{"syntax error", "name: \"cluster\n", "platform file error"},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			path := writePlatform(t, tt.content)
			_, err := LoadLegacyPlatformFile(path)
			require.Error(t, err)
			assert.Contains(t, err.Error(), tt.want)
		})
	}
}

func TestHackPlatformMirror_PinsMatchSeed(t *testing.T) {
	// hack/platform/cue.mod/module.cue is the kind dev-cluster mirror of the
	// seeded module; its pins must equal the seed's so an offline build and
	// an in-cluster render evaluate the same catalog builds.
	mirrorPath := filepath.Join("..", "..", "hack", "platform", "cue.mod", "module.cue")
	content, err := os.ReadFile(mirrorPath)
	if err != nil {
		t.Skipf("hack mirror not present: %v", err)
	}
	mirror, err := modfile.Parse(content, mirrorPath)
	require.NoError(t, err)
	seed, err := modfile.Parse([]byte(DefaultPlatformModuleFile), PlatformModuleFileName)
	require.NoError(t, err)

	// The mirror is kept in `cue mod tidy`'s canonical form, so it may carry
	// transitive pins the offline seed does not; every root the seed pins
	// must be pinned identically.
	assert.Equal(t, seed.Module, mirror.Module)
	for path, dep := range seed.Deps {
		require.Contains(t, mirror.Deps, path)
		assert.Equal(t, dep.Version, mirror.Deps[path].Version, "%s pin drifted between the seed and hack/platform", path)
	}
}
