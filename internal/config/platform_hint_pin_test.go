package config

import (
	"context"
	"errors"
	"fmt"
	"net/http"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	liberrors "github.com/open-platform-model/library/opm/errors"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// hintPlatform writes a platform module whose cue.mod declares deps and
// whose platform.cue holds src, and returns its directory.
func hintPlatform(t *testing.T, deps, src string) string {
	t.Helper()
	dir := filepath.Join(t.TempDir(), "platform")
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o700))
	mod := "module: \"example.com/platform@v0\"\nlanguage: version: \"v0.9.0\"\n" + deps
	require.NoError(t, os.WriteFile(filepath.Join(dir, filepath.FromSlash(PlatformModuleFileName)), []byte(mod), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(dir, PlatformCUEFileName), []byte(src), 0o600))
	return dir
}

const (
	importsDep = "package platform\n\nimport d \"example.com/dep@v0\"\n\nv: d.version\n"
	depPinned  = "deps: \"example.com/dep@v0\": v: \"" + cuemodtest.DepNewest + "\"\n"
)

// TestPlatformBuildHint_Pinned pins the hint each platform build failure
// form gets. The registry-backed #registry key-mismatch hint is pinned by
// TestBuildPlatformModule_KeyImportDriftNamesTheEntry.
func TestPlatformBuildHint_Pinned(t *testing.T) {
	const pin, kind = "Pin a published build in ", "platform.cue must be a single package embedding core.#Platform"
	for _, tc := range []struct {
		name     string
		registry func(t *testing.T) string
		deps     string
		src      string
		want     string
	}{
		{"unpublished catalog pin", cuemodtest.Registry, "deps: \"example.com/dep@v0\": v: \"v0.9.0\"\n", importsDep, pin},
		{"pinned build's archive blob answers 404", func(t *testing.T) string {
			return cuemodtest.Fronted(t, cuemodtest.Registry(t), cuemodtest.BlobAnswers("example.com/dep", http.StatusNotFound))
		}, depPinned, importsDep, pin},
		{"undeclared import", cuemodtest.Registry, "", importsDep, pin},
		{"not a #Platform", cuemodtest.Registry, "", "package platform\n\nv: 1\n", kind},
		{"refused registry", func(*testing.T) string { return cuemodtest.UnreachableRegistry }, depPinned, importsDep, pin},
	} {
		t.Run(tc.name, func(t *testing.T) {
			cuemodtest.ColdCache(t)
			dir := hintPlatform(t, tc.deps, tc.src)
			_, err := BuildPlatformModule(context.Background(), dir, tc.registry(t))
			require.Error(t, err)
			var detail *oerrors.DetailError
			require.True(t, errors.As(err, &detail), "%v", err)
			assert.Contains(t, detail.Hint, tc.want, "%v", err)
		})
	}
}

// TestPlatformBuildHint_NotFoundWithoutImportPrefix holds the pin hint for a
// registry answer that does not carry cue/load's import prefix: the library
// classifies the form as a fetch failure.
func TestPlatformBuildHint_NotFoundWithoutImportPrefix(t *testing.T) {
	hint := platformBuildHint(t.TempDir(), DefaultRegistry, errors.New("cannot fetch example.com/dep@v0.9.0: module not found"))
	assert.Contains(t, hint, "Pin a published build in ")
}

// TestPlatformBuildHint_CUEErrorPath holds the #registry hint for an
// evaluation error at a path under #registry, and the default hint for one
// at any other path, without a registry. An error that only names #registry
// in its text gets the default hint too.
func TestPlatformBuildHint_CUEErrorPath(t *testing.T) {
	ctx := cuecontext.New()
	under := ctx.CompileString("#registry: \"example.com/x@v1\": enable: true & false\n").Validate()
	require.Error(t, under)
	other := ctx.CompileString("metadata: name: 1 & 2\n").Validate()
	require.Error(t, other)

	wrap := func(err error) error { return fmt.Errorf("building platform package from dir: %w", err) }
	assert.Contains(t, platformBuildHint(t.TempDir(), DefaultRegistry, wrap(under)), hintKey)
	assert.Contains(t, platformBuildHint(t.TempDir(), DefaultRegistry, wrap(other)), hintDefault)

	// The library's shape check names #registry in its text only: it
	// flattens the CUE cause and wraps the missing-field sentinel. Old hint,
	// from the word in the message: the key-and-import hint. New hint: the
	// default one. TestBuildPlatformModule_EntryWithoutCatalogIsRefusedByShape
	// drives the same form through a real build.
	shape := fmt.Errorf("validating platform package in dir: required field %q entry %q is incomplete at %q (an embedded catalog supplies it): %v: %w",
		"#registry", "example.com/x@v1", "version", "#registry.\"example.com/x@v1\".version: required field missing", liberrors.ErrMissingRequiredField)
	shapeHint := platformBuildHint(t.TempDir(), DefaultRegistry, shape)
	assert.Contains(t, shapeHint, hintDefault)
	assert.NotContains(t, shapeHint, hintKey)
}
