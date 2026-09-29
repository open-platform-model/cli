package render

import (
	"bytes"
	"context"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/mod/modfile"
	"cuelang.org/go/mod/module"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"k8s.io/apimachinery/pkg/runtime"
	"k8s.io/apimachinery/pkg/runtime/schema"
	dynamicfake "k8s.io/client-go/dynamic/fake"

	"github.com/open-platform-model/library/opm/kernel"

	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/instinit"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/tests/fixtures"
)

// instanceDepsRegistry routes both OPM domains to GHCR: the podinfo test
// fixture lives on the testing domain, core and the catalogs on the
// production one.
const instanceDepsRegistry = "opmodel.dev=ghcr.io/open-platform-model,testing.opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"

// writeInitInstance writes an instance package for the published podinfo
// fixture exactly as `opm instance init` does: the module and its own core
// pin rendered by instinit, the values picked from the module, and the
// closure completed by the tidy inside instinit.Write. Returns the package
// directory.
func writeInitInstance(t *testing.T, k *kernel.Kernel) string {
	t.Helper()
	ctx := context.Background()
	coord := fixtures.Must(t, "podinfo")

	fixtureModFile := filepath.Join("..", "..", "..", "tests", "fixtures", "modules", "podinfo", "cue.mod", "module.cue")
	data, err := os.ReadFile(fixtureModFile)
	require.NoError(t, err)
	mf, err := modfile.Parse(data, fixtureModFile)
	require.NoError(t, err)
	coreDep, ok := mf.Deps["opmodel.dev/core@v2"]
	require.True(t, ok, "the podinfo fixture pins core")

	modVersion, err := module.NewVersion(coord.ModulePath, coord.Tag())
	require.NoError(t, err)
	core, err := module.NewVersion("opmodel.dev/core@v2", coreDep.Version)
	require.NoError(t, err)

	mod, err := k.AcquireModuleFromRegistry(ctx, coord.ModulePath, coord.Tag())
	if err != nil {
		t.Skipf("podinfo fixture unavailable (registry/cache): %v", err)
	}
	values, source, err := instinit.PickValues(mod.Package)
	require.NoError(t, err)
	files, err := instinit.Render(instinit.Input{
		Name:              "hello",
		Namespace:         "default",
		PackageModulePath: instinit.DefaultPackageModulePath("hello"),
		Module:            modVersion,
		Core:              core,
		Values:            values,
		Source:            source,
	})
	require.NoError(t, err)

	dir := filepath.Join(t.TempDir(), "hello")
	require.NoError(t, instinit.Write(ctx, dir, files, instanceDepsRegistry))
	return dir
}

// TestInstanceDeps_InitPackageListsItsCatalogsAndRenders pins the input the
// instance-deps platform is generated from: a package `opm instance init`
// writes lists, after its tidy, the catalog its module imports, and a
// platform generated from that list renders the instance.
func TestInstanceDeps_InitPackageListsItsCatalogsAndRenders(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed tests")
	}
	ctx := context.Background()
	k := config.NewKernel(instanceDepsRegistry)
	if _, err := k.SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	dir := writeInitInstance(t, k)

	modFileName := filepath.Join(dir, "cue.mod", "module.cue")
	data, err := os.ReadFile(modFileName)
	require.NoError(t, err)
	mf, err := modfile.Parse(data, modFileName)
	require.NoError(t, err)
	assert.Contains(t, mf.Deps, "opmodel.dev/catalogs/opm@v4", "the tidied package lists its module's catalog transitively")

	deps, err := instanceDepsOf(dir, dir)
	require.NoError(t, err)
	assert.Equal(t, data, deps.ModFile, "the deps are the package's own committed module file")

	cfg := &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue"), Registry: instanceDepsRegistry}
	env, err := resolvePlatformEnv(ctx, k, cfg, platform.ResolveOptions{Deps: deps, DepsKind: platform.DepsInstance})
	require.NoError(t, err)
	assert.Equal(t, platform.SourceModuleDeps, env.resolution.Source)
	assert.Equal(t, platform.DepsInstance, env.resolution.DepsKind)
	require.NotEmpty(t, env.resolution.Catalogs)
	assert.Contains(t, env.resolution.Catalogs[0], "opmodel.dev/catalogs/opm@v4")

	inst, err := k.AcquireInstanceFromDir(ctx, dir)
	require.NoError(t, err)
	result, err := renderInstance(ctx, env, inst, &config.ResolvedKubernetesConfig{}, dir, false)
	require.NoError(t, err)
	assert.NotEmpty(t, result.Resources, "the instance renders against the platform generated from its own pins")
}

// TestInstanceDepsOf_NoModuleRootIsAnError asserts a package under no CUE
// module has no pins to fall back to, and the error names the directory.
func TestInstanceDepsOf_NoModuleRootIsAnError(t *testing.T) {
	dir := t.TempDir()
	_, err := instanceDepsOf(dir, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), dir)
	assert.Contains(t, err.Error(), "under no CUE module")
}

// TestInstanceDepsOf_CarriesThePackageReplacements asserts the instance
// package's own cue.mod/local-module.cue is read beside its module file.
func TestInstanceDepsOf_CarriesThePackageReplacements(t *testing.T) {
	root := writeModuleContext(t, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "../catalog_opm/opm"`)

	deps, err := instanceDepsOf(root, root)
	require.NoError(t, err)
	assert.Equal(t, filepath.Join(root, "cue.mod", "module.cue"), deps.ModFileName)
	assert.Equal(t, root, deps.ModuleRoot)
	require.Len(t, deps.Replacements, 1)
	assert.Equal(t, "opmodel.dev/catalogs/opm@v4", deps.Replacements[0].Path)
}

// absentClusterPlatform is the production cluster getter over a fake
// dynamic client holding no Platform: every read is a real NotFound.
func absentClusterPlatform() platform.ClusterPlatformGetter {
	fake := dynamicfake.NewSimpleDynamicClientWithCustomListKinds(runtime.NewScheme(),
		map[schema.GroupVersionResource]string{inventory.PlatformGVR: "PlatformList"})
	return platform.ClusterPlatformGetterFor(fake)
}

// captureRenderLog redirects the CLI's log sink for the test and returns it.
func captureRenderLog(t *testing.T) *bytes.Buffer {
	t.Helper()
	var buf bytes.Buffer
	output.SetLogWriter(&buf)
	t.Cleanup(func() { output.SetLogWriter(os.Stderr) })
	return &buf
}

// TestFromInstanceFile_AbsentClusterPlatformFallsBackToInstanceDeps covers
// "Absent Platform falls back to the deps" for an instance render on the
// apply path (a cluster getter, the cluster not optional): the cluster holds
// no Platform, so the render warns naming the instance's own deps and
// renders against a platform generated from the package's pins.
func TestFromInstanceFile_AbsentClusterPlatformFallsBackToInstanceDeps(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed tests")
	}
	k := config.NewKernel(instanceDepsRegistry)
	if _, err := k.SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	dir := writeInitInstance(t, k)
	logBuf := captureRenderLog(t)

	result, err := FromInstanceFile(context.Background(), InstanceFileOpts{
		InstanceFilePath: dir,
		ClusterPlatform:  absentClusterPlatform(),
		Config:           &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue"), Registry: instanceDepsRegistry},
		K8sConfig:        &config.ResolvedKubernetesConfig{},
	})
	require.NoError(t, err, "log: %s", logBuf.String())

	assert.Equal(t, platform.SourceModuleDeps, result.Platform.Source)
	assert.Equal(t, platform.DepsInstance, result.Platform.DepsKind)
	assert.Equal(t, filepath.Join(dir, "cue.mod", "module.cue"), result.Platform.Location)
	assert.NotEmpty(t, result.Resources)
	assert.Contains(t, logBuf.String(), "cluster Platform not used (no Platform CR in the cluster) — rendering against the instance's own deps")
	assert.Contains(t, logBuf.String(), "platform: instance deps (")
}
