package e2e

import (
	"context"
	"errors"
	"net"
	"os"
	"path/filepath"
	"testing"
	"time"

	"cuelang.org/go/mod/modfile"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/cuemod"
	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	"github.com/open-platform-model/cli/tests/fixtures"
)

// podinfoInstancePackage hand-writes the three files `opm instance init`
// generates for the podinfo fixture into a fresh directory and returns it:
// a module file pinning the fixture and core at the fixture's own core pin,
// an instance.cue binding the fixture to a #ModuleInstance, and a values.cue
// carrying the fixture's debugValues. It also returns the parsed module file
// it wrote.
func podinfoInstancePackage(t *testing.T) (dir string, written *modfile.File) {
	t.Helper()
	coord := fixtures.Must(t, "podinfo")
	fixturesDir, err := fixtures.Dir()
	require.NoError(t, err)
	fixtureMod, err := os.ReadFile(filepath.Join(fixturesDir, "podinfo", "cue.mod", "module.cue"))
	require.NoError(t, err)
	parsed, err := modfile.Parse(fixtureMod, "module.cue")
	require.NoError(t, err)
	coreDep := parsed.Deps["opmodel.dev/core@v2"]
	require.NotNil(t, coreDep, "the podinfo fixture must pin core")

	written = &modfile.File{
		Module:   "instance.local/podinfo@v0",
		Language: &modfile.Language{Version: "v0.17.0"},
		Deps: map[string]*modfile.Dep{
			coord.ModulePath:      {Version: coord.Tag()},
			"opmodel.dev/core@v2": {Version: coreDep.Version},
		},
	}
	modBytes, err := modfile.Format(written)
	require.NoError(t, err)

	dir = t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(dir, "cue.mod"), 0o755))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "cue.mod", "module.cue"), modBytes, 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte(`package instance

import (
	core "opmodel.dev/core@v2"
	opmModule "`+coord.ModulePath+`"
)

core.#ModuleInstance

metadata: {
	name:      "podinfo"
	namespace: "default"
}

#module: opmModule
`), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(dir, "values.cue"), []byte(`// Values from the module's debugValues.
package instance

values: {
	image: {
		repository: "ghcr.io/stefanprodan/podinfo"
		tag:        "6.7.1"
		digest:     ""
	}
	replicas: 1
}
`), 0o644))
	return dir, written
}

// readModFile parses dir's cue.mod/module.cue.
func readModFile(t *testing.T, dir string) *modfile.File {
	t.Helper()
	data, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	f, err := modfile.Parse(data, "module.cue")
	require.NoError(t, err)
	return f
}

// TestE2E_InstanceInit_GeneratedPackageTidiesAndLoads proves what `opm
// instance init` relies on (design.md, Risks): tidying a package that pins
// the module and core explicitly keeps both pins and the language version
// and only adds the rest of the closure; the tidied package loads through the
// kernel, passes a tidy check, and still loads with the registry unreachable
// once the module cache is warm.
func TestE2E_InstanceInit_GeneratedPackageTidiesAndLoads(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}
	if userCache, err := os.UserCacheDir(); err == nil && os.Getenv("CUE_CACHE_DIR") == "" {
		t.Setenv("CUE_CACHE_DIR", filepath.Join(userCache, "cue"))
	}
	ctx, cancel := context.WithTimeout(context.Background(), 180*time.Second)
	defer cancel()

	dir, written := podinfoInstancePackage(t)

	res, err := cuemod.Tidy(ctx, dir, cuemod.TidyOptions{Registry: config.DefaultRegistry})
	require.NoError(t, err)
	assert.True(t, res.ModuleUpdated, "tidy must add the rest of the closure")

	tidied := readModFile(t, dir)
	assert.Equal(t, written.Module, tidied.Module)
	require.NotNil(t, tidied.Language)
	assert.Equal(t, written.Language.Version, tidied.Language.Version, "tidy must keep the language version")
	for path, dep := range written.Deps {
		require.Contains(t, tidied.Deps, path)
		assert.Equal(t, dep.Version, tidied.Deps[path].Version, "tidy must keep the %s pin", path)
	}
	assert.Greater(t, len(tidied.Deps), len(written.Deps), "tidy must add the closure beyond the two written pins")

	_, err = config.NewKernel(config.DefaultRegistry).AcquireInstanceFromDir(ctx, dir)
	require.NoError(t, err, "the tidied package must load through the kernel")

	_, err = cuemod.Tidy(ctx, dir, cuemod.TidyOptions{Registry: config.DefaultRegistry, Check: true})
	require.NoError(t, err, "the tidied package must pass a tidy check")

	_, err = config.NewKernel(cuemodtest.UnreachableRegistry).AcquireInstanceFromDir(ctx, dir)
	require.NoError(t, err, "the tidied package must load from a warm cache with the registry unreachable")
}

// TestE2E_InstanceInit_TidyRegistryFailureShape records how a registry
// failure surfaces from cuemod.Tidy on an empty module cache, the shape the
// init command's connectivity classifier matches on.
func TestE2E_InstanceInit_TidyRegistryFailureShape(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}
	t.Setenv("CUE_CACHE_DIR", t.TempDir())
	ctx, cancel := context.WithTimeout(context.Background(), 60*time.Second)
	defer cancel()

	dir, _ := podinfoInstancePackage(t)
	_, err := cuemod.Tidy(ctx, dir, cuemod.TidyOptions{Registry: cuemodtest.UnreachableRegistry})
	require.Error(t, err)
	var netErr net.Error
	t.Logf("tidy error (errors.As net.Error: %v, type %T): %v", errors.As(err, &netErr), err, err)
	assert.Contains(t, err.Error(), "connection refused")
}
