// Package config provides configuration loading and management.
package config

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"cuelang.org/go/cue"
	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"

	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// hackPlatformDir is the repo's maintained platform module (hack/platform/),
// pinned by the root deps task and mirrored by hack/kind-platform.yaml.
const hackPlatformDir = "../../hack/platform"

// copyHackPlatform copies hack/platform/ into a fresh temp dir and returns
// it, so a test can edit its pins or entries.
func copyHackPlatform(t *testing.T) string {
	t.Helper()
	dst := filepath.Join(t.TempDir(), "platform")
	require.NoError(t, filepath.WalkDir(hackPlatformDir, func(p string, d os.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, err := filepath.Rel(hackPlatformDir, p)
		if err != nil {
			return err
		}
		out := filepath.Join(dst, rel)
		if d.IsDir() {
			return os.MkdirAll(out, 0o700)
		}
		data, err := os.ReadFile(p)
		if err != nil {
			return err
		}
		return os.WriteFile(out, data, 0o600)
	}))
	return dst
}

// hackCatalogPins reads the catalog pins of a platform module's cue.mod.
func hackCatalogPins(t *testing.T, dir string) map[string]string {
	t.Helper()
	name := filepath.Join(dir, filepath.FromSlash(PlatformModuleFileName))
	data, err := os.ReadFile(name)
	require.NoError(t, err)
	f, err := modfile.Parse(data, name)
	require.NoError(t, err)
	pins := map[string]string{}
	require.Contains(t, f.Deps, DefaultCatalogPath)
	pins[DefaultCatalogPath] = f.Deps[DefaultCatalogPath].Version
	return pins
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

func TestLegacyPlatformFilePath_SiblingOfConfig(t *testing.T) {
	dir := filepath.Join("custom", "dir")
	got := LegacyPlatformFilePath(filepath.Join(dir, "config.cue"))
	assert.Equal(t, filepath.Join(dir, "platform.cue"), got)
}

func TestLegacyPlatformDirPath_SiblingOfConfig(t *testing.T) {
	dir := filepath.Join("custom", "dir")
	got := LegacyPlatformDirPath(filepath.Join(dir, "config.cue"))
	assert.Equal(t, filepath.Join(dir, "platform"), got)
}

func TestBuildPlatformModule_MissingDirectory(t *testing.T) {
	dir := filepath.Join(t.TempDir(), "platform")
	_, err := BuildPlatformModule(context.Background(), dir, DefaultRegistry)
	require.Error(t, err)
	assert.ErrorIs(t, err, oerrors.ErrValidation)
	assert.Contains(t, err.Error(), dir)
	assert.Contains(t, err.Error(), "opm platform pull")
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
	assert.Contains(t, err.Error(), "opm platform pull")
}

func TestBuildPlatformModule_HackPlatformBuilds(t *testing.T) {
	// Registry-backed: the maintained hack/platform/ module must build
	// against the published core and catalogs, and each entry's version
	// must be the build its cue.mod pins (derived readout, 0019:D5).
	dir := copyHackPlatform(t)
	pins := hackCatalogPins(t, dir)

	p, err := BuildPlatformModule(buildCtx(t), dir, DefaultRegistry)
	skipIfRegistryUnavailable(t, err)
	require.NoError(t, err)
	require.NotNil(t, p)
	require.NotNil(t, p.Metadata)
	assert.Equal(t, "cluster", p.Metadata.Name)
	assert.Equal(t, "kubernetes", p.Metadata.Type)

	registry := p.Package.LookupPath(cue.MakePath(cue.Def("registry")))
	require.True(t, registry.Exists())
	entry := registry.LookupPath(cue.MakePath(cue.Str(DefaultCatalogPath)))
	require.True(t, entry.Exists(), DefaultCatalogPath)
	version, err := entry.LookupPath(cue.ParsePath("version")).String()
	require.NoError(t, err)
	assert.Equal(t, strings.TrimPrefix(pins[DefaultCatalogPath], "v"), version, "version derived from the pinned catalog")
	enable, err := entry.LookupPath(cue.ParsePath("enable")).Bool()
	require.NoError(t, err)
	assert.True(t, enable, "enabled by default")
}

func TestBuildPlatformModule_UnpublishedPinNamesTheDependency(t *testing.T) {
	// Registry-backed: a pin naming a build that does not exist fails the
	// build naming the dependency, with the hint pointing at cue.mod.
	dir := copyHackPlatform(t)
	pins := hackCatalogPins(t, dir)
	modPath := filepath.Join(dir, "cue.mod", "module.cue")
	content, err := os.ReadFile(modPath)
	require.NoError(t, err)
	bumped := strings.Replace(string(content), pins[DefaultCatalogPath], "v4.9.9", 1)
	require.NotEqual(t, string(content), bumped)
	require.NoError(t, os.WriteFile(modPath, []byte(bumped), 0o600))

	_, err = BuildPlatformModule(buildCtx(t), dir, DefaultRegistry)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)
	assert.ErrorIs(t, err, oerrors.ErrValidation)
	assert.Contains(t, err.Error(), DefaultCatalogPath+".9.9")
	assert.Contains(t, err.Error(), modPath)
}

func TestBuildPlatformModule_KeyImportDriftNamesTheEntry(t *testing.T) {
	// Registry-backed: an entry keyed at a path other than the embedded
	// catalog's module path fails the 0019:D5 binding at a path naming the
	// entry.
	dir := copyHackPlatform(t)
	cuePath := filepath.Join(dir, "platform.cue")
	content, err := os.ReadFile(cuePath)
	require.NoError(t, err)
	const rekeyed = "opmodel.dev/core@v2"
	drifted := strings.Replace(string(content), `"`+DefaultCatalogPath+`": #catalog: opm`, `"`+rekeyed+`": #catalog: opm`, 1)
	require.NotEqual(t, string(content), drifted)
	require.NoError(t, os.WriteFile(cuePath, []byte(drifted), 0o600))

	_, err = BuildPlatformModule(buildCtx(t), dir, DefaultRegistry)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)
	assert.ErrorIs(t, err, oerrors.ErrValidation)
	msg := err.Error()
	assert.Contains(t, msg, `#registry."`+rekeyed+`"`)
	assert.Contains(t, msg, "conflicting values")
	assert.Contains(t, msg, "must equal the module path")
	var detail *oerrors.DetailError
	require.True(t, errors.As(err, &detail), "%v", err)
	assert.Contains(t, detail.Hint, hintKey)
	assert.Contains(t, detail.Hint, cuePath)
}

func TestBuildPlatformModule_EntryWithoutCatalogIsRefusedByShape(t *testing.T) {
	// Registry-backed: a second entry that embeds no catalog is refused by
	// the library's shape check, which names #registry in its text only and
	// keeps no CUE path. Matched on the word "#registry", it got the hint
	// "Each #registry entry's key must equal the module path of the catalog
	// it imports (#catalog); fix the entry named above in <dir>/platform.cue".
	dir := copyHackPlatform(t)
	cuePath := filepath.Join(dir, "platform.cue")
	content, err := os.ReadFile(cuePath)
	require.NoError(t, err)
	const entry = `"` + DefaultCatalogPath + `": #catalog: opm`
	bare := strings.Replace(string(content), entry, entry+"\n\t\"example.com/x@v1\": {}", 1)
	require.NotEqual(t, string(content), bare)
	require.NoError(t, os.WriteFile(cuePath, []byte(bare), 0o600))

	_, err = BuildPlatformModule(buildCtx(t), dir, DefaultRegistry)
	skipIfRegistryUnavailable(t, err)
	require.Error(t, err)
	assert.ErrorIs(t, err, oerrors.ErrValidation)
	assert.ErrorIs(t, err, liberrors.ErrMissingRequiredField)
	var detail *oerrors.DetailError
	require.True(t, errors.As(err, &detail), "%v", err)
	assert.Contains(t, detail.Message, `required field "#registry" entry "example.com/x@v1"`)
	assert.Contains(t, detail.Hint, hintKey)
}
