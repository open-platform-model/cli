package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	cueerrors "cuelang.org/go/cue/errors"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/pkg/loader"
)

func TestFromModule_NilConfig(t *testing.T) {
	_, err := FromModule(context.Background(), ModuleOpts{ModulePath: "./mod", Config: nil, K8sConfig: nil})
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "configuration not loaded")
}

func TestFromModule_NilK8sConfig(t *testing.T) {
	_, err := FromModule(context.Background(), ModuleOpts{
		ModulePath: "./mod",
		Config:     &config.GlobalConfig{},
		K8sConfig:  nil,
	})
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitGeneralError, exitErr.Code)
	assert.Contains(t, exitErr.Error(), "kubernetes config not resolved")
}

func TestFromModule_RejectsInstancePackage(t *testing.T) {
	dir := t.TempDir()
	require.NoError(t, os.WriteFile(filepath.Join(dir, "instance.cue"), []byte("package test\n"), 0o644))

	_, err := FromModule(context.Background(), ModuleOpts{
		ModulePath: dir,
		Config:     &config.GlobalConfig{},
		K8sConfig:  &config.ResolvedKubernetesConfig{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instance package")
}

// packageWithConfig compiles a bare module package value carrying a #config
// schema and optional debugValues, as ResolveModuleValues reads them. It
// needs no kernel: the kernel compiles values sources in the schema value's
// own context.
func packageWithConfig(t *testing.T, src string) cue.Value {
	t.Helper()
	pkg := cuecontext.New().CompileString(src)
	require.NoError(t, pkg.Err())
	return pkg
}

// TestResolveModuleValues_UsesValuesFile asserts that supplied -f files are
// the sources, in preference to debugValues.
func TestResolveModuleValues_UsesValuesFile(t *testing.T) {
	k := kernel.New()
	dir := t.TempDir()
	valuesFile := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	pkg := packageWithConfig(t, `{#config: {replicas: int}, debugValues: {replicas: 1}}`)

	values, err := ResolveModuleValues(k, pkg, dir, []string{valuesFile})
	require.NoError(t, err)
	require.Len(t, values, 1, "the -f file is the single source; debugValues are not layered under it")
	assert.Equal(t, valuesFile, values[0].Origin, "the -f file wins over debugValues")
}

// TestResolveModuleValues_ReturnsSourcesUnvalidated asserts the resolver
// carries a -f file that violates #config: validating is the caller's step,
// so vet and build each check the sources exactly once.
func TestResolveModuleValues_ReturnsSourcesUnvalidated(t *testing.T) {
	k := kernel.New()
	valuesFile := filepath.Join(t.TempDir(), "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {bogus: 1}\n"), 0o644))
	pkg := packageWithConfig(t, `{#config: close({replicas: int | *1})}`)

	values, err := ResolveModuleValues(k, pkg, "mod", []string{valuesFile})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, valuesFile, values[0].Origin)
}

// TestValidateValuesFiles_ConflictNamesTheFile asserts that two -f files
// disagreeing on a value fail with the conflict attributed to a file.
func TestValidateValuesFiles_ConflictNamesTheFile(t *testing.T) {
	k := kernel.New()
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.cue")
	f2 := filepath.Join(dir, "b.cue")
	require.NoError(t, os.WriteFile(f1, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	require.NoError(t, os.WriteFile(f2, []byte("package test\nvalues: {replicas: 4}\n"), 0o644))
	pkg := packageWithConfig(t, `{#config: {replicas: int}}`)

	sources, err := ResolveModuleValues(k, pkg, dir, []string{f1, f2})
	require.NoError(t, err)
	err = validateValuesFiles(k, pkg.LookupPath(schema.Config), sources)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "conflicting values")
	assert.True(t, positionsName(err, "b.cue"), "the conflict must be attributed to a values file: %v", cueerrors.Positions(err))
}

// positionsName reports whether any CUE error position in err names a file
// with the given basename.
func positionsName(err error, base string) bool {
	for _, pos := range cueerrors.Positions(err) {
		if filepath.Base(pos.Filename()) == base {
			return true
		}
	}
	return false
}

// TestValidateValuesFiles_SchemaViolationRejected asserts a -f file is
// validated against #config, with the violation attributed to the file.
func TestValidateValuesFiles_SchemaViolationRejected(t *testing.T) {
	k := kernel.New()
	valuesFile := filepath.Join(t.TempDir(), "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {bogus: 1}\n"), 0o644))
	pkg := packageWithConfig(t, `{#config: close({replicas: int | *1})}`)

	sources, err := ResolveModuleValues(k, pkg, "mod", []string{valuesFile})
	require.NoError(t, err)
	err = validateValuesFiles(k, pkg.LookupPath(schema.Config), sources)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field not allowed")
	assert.True(t, positionsName(err, "values.cue"), "the violation must be attributed to the values file: %v", cueerrors.Positions(err))
}

// TestValidateValuesFiles_NoConfigSchema asserts the actionable error when
// -f files are supplied to a module that declares no #config.
func TestValidateValuesFiles_NoConfigSchema(t *testing.T) {
	k := kernel.New()
	pkg := packageWithConfig(t, `{metadata: name: "x"}`)

	err := validateValuesFiles(k, pkg.LookupPath(schema.Config), nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "#config")
}

// TestResolveModuleValues_FallbackDebugValues exercises the debugValues path:
// the field becomes the single values source, attributed to the module's
// debugValues and carrying the field's data.
func TestResolveModuleValues_FallbackDebugValues(t *testing.T) {
	k := kernel.New()
	pkg := packageWithConfig(t, `{#config: {replicas: int}, debugValues: {replicas: 5}}`)

	values, err := ResolveModuleValues(k, pkg, "mod", nil)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, filepath.Join("mod", "debugValues"), values[0].Origin)
	// A Source carries bytes the kernel compiles where it uses them: read the
	// field's data back through the validation primitive.
	merged, err := k.ValidateConfigDetailed(pkg.LookupPath(schema.Config), values)
	require.NoError(t, err)
	replicas, err := merged.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas)
}

// TestResolveModuleValues_NoDebugValues asserts the actionable error when
// the module defines neither debugValues nor a -f flag.
func TestResolveModuleValues_NoDebugValues(t *testing.T) {
	k := kernel.New()
	pkg := packageWithConfig(t, `{metadata: name: "x"}`)

	_, err := ResolveModuleValues(k, pkg, "mod", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "debugValues")
}

func TestModuleDepsOf_OverlayOnlySourceCarriesNoReplacements(t *testing.T) {
	// The registry-acquired shape: a synthetic root, the tree in memory, no
	// local module context.
	root := filepath.Join(string(filepath.Separator), "opm-registry", "example.com", "app@v0.1.0")
	modFile := []byte("module: \"example.com/app@v0\"\nlanguage: version: \"v0.17.0\"\n")
	src := &module.Source{Root: root, Overlay: map[string][]byte{filepath.Join(root, "cue.mod", "module.cue"): modFile}}

	deps, err := moduleDepsOf(src, "")
	require.NoError(t, err)
	assert.Equal(t, modFile, deps.ModFile)
	assert.Equal(t, filepath.Join(root, "cue.mod", "module.cue"), deps.ModFileName)
	assert.Empty(t, deps.Replacements)
	assert.Empty(t, deps.ModuleRoot)
}

func TestModuleDepsOf_DirectoryModuleCarriesItsReplacements(t *testing.T) {
	root := writeModuleContext(t, `deps: "opmodel.dev/catalogs/opm@v4": replaceWith: "../catalog_opm/opm"`)
	modFile, err := os.ReadFile(filepath.Join(root, "cue.mod", "module.cue"))
	require.NoError(t, err)
	src := &module.Source{Root: root, Overlay: map[string][]byte{filepath.Join(root, "cue.mod", "module.cue"): modFile}}

	deps, err := moduleDepsOf(src, root)
	require.NoError(t, err)
	assert.Equal(t, root, deps.ModuleRoot)
	assert.Equal(t, []loader.LocalReplacement{{Path: "opmodel.dev/catalogs/opm@v4", ReplaceWith: "../catalog_opm/opm"}}, deps.Replacements)
}

func TestModuleDepsOf_MissingModFileIsAnError(t *testing.T) {
	root := t.TempDir()
	_, err := moduleDepsOf(&module.Source{Root: root, Overlay: map[string][]byte{filepath.Join(root, "x.cue"): nil}}, "")
	require.Error(t, err)
	assert.Contains(t, err.Error(), filepath.Join(root, "cue.mod", "module.cue"))

	_, err = moduleDepsOf(nil, "")
	require.Error(t, err)
}

// TestFromModule_PlatformFromDepsNeedsNoLocalDefault renders the
// module-with-debug-values fixture (no catalog, no components) with PlatformFromDeps and
// an OPM home holding no platform/: the platform is generated from the
// module's pins under the home's cache, and the render yields zero objects.
func TestFromModule_PlatformFromDepsNeedsNoLocalDefault(t *testing.T) {
	const registry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "module-with-debug-values"))
	require.NoError(t, err)
	if _, err := config.NewKernel(registry).SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	configPath := filepath.Join(t.TempDir(), "config.cue")

	result, err := FromModule(context.Background(), ModuleOpts{
		ModulePath:       dir,
		Name:             "debug-values", // the default "<name>-debug" keeps the fixture's underscore
		PlatformFromDeps: true,
		Config:           &config.GlobalConfig{ConfigPath: configPath, Registry: registry},
		K8sConfig:        &config.ResolvedKubernetesConfig{},
	})
	require.NoError(t, err)

	assert.Equal(t, platform.SourceModuleDeps, result.Platform.Source)
	assert.Empty(t, result.Platform.Catalogs, "the fixture pins no catalog")
	assert.Equal(t, config.PlatformCacheDir(configPath), filepath.Dir(result.Platform.Dir))
	assert.Empty(t, result.Resources)
	assert.Empty(t, result.Warnings, "no skew row against the module's own pins")
	_, err = os.Stat(config.PlatformDir(configPath))
	assert.ErrorIs(t, err, os.ErrNotExist)
}

// TestModuleSource_CarriesCommittedModFile pins the input the module-deps
// platform is generated from: a module acquired from a directory carries its
// committed cue.mod/module.cue in Source.Overlay, keyed under Source.Root,
// so the dependency pins are read from the acquired source with no second
// load, the same way a registry-acquired module's are.
func TestModuleSource_CarriesCommittedModFile(t *testing.T) {
	dir, err := filepath.Abs(filepath.Join("..", "..", "..", "tests", "fixtures", "valid", "simple-module"))
	require.NoError(t, err)
	k := config.NewKernel("opmodel.dev=ghcr.io/open-platform-model,registry.cue.works")
	if _, err := k.SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}

	mod, err := k.AcquireModuleFromDir(context.Background(), dir)
	require.NoError(t, err)
	require.True(t, mod.HasSource())

	want, err := os.ReadFile(filepath.Join(dir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	got, ok := mod.Source.Overlay[filepath.Join(mod.Source.Root, "cue.mod", "module.cue")]
	require.True(t, ok, "overlay carries cue.mod/module.cue at Source.Root")
	assert.Equal(t, string(want), string(got))
}
