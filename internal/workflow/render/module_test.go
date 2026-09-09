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

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
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

// moduleWithConfig builds a bare module value carrying a #config schema and
// optional debugValues, as resolveModuleValues reads them. It needs no kernel:
// the kernel compiles values sources in the schema value's own context.
func moduleWithConfig(t *testing.T, src string) *module.Module {
	t.Helper()
	modVal := cuecontext.New().CompileString(src)
	require.NoError(t, modVal.Err())
	return &module.Module{Package: modVal}
}

// TestResolveModuleValues_UsesValuesFile asserts that supplied -f files are
// layered and validated against #config in preference to debugValues.
func TestResolveModuleValues_UsesValuesFile(t *testing.T) {
	k := kernel.New()
	dir := t.TempDir()
	valuesFile := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	mod := moduleWithConfig(t, `{#config: {replicas: int}, debugValues: {replicas: 1}}`)

	values, err := resolveModuleValues(k, mod, dir, []string{valuesFile})
	require.NoError(t, err)
	require.Len(t, values, 1, "the -f file is the single source; debugValues are not layered under it")
	assert.Equal(t, valuesFile, values[0].Origin, "the -f file wins over debugValues")
}

// TestResolveModuleValues_ConflictNamesTheFile asserts that two -f files
// disagreeing on a value fail with the conflict attributed to a file.
func TestResolveModuleValues_ConflictNamesTheFile(t *testing.T) {
	k := kernel.New()
	dir := t.TempDir()
	f1 := filepath.Join(dir, "a.cue")
	f2 := filepath.Join(dir, "b.cue")
	require.NoError(t, os.WriteFile(f1, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))
	require.NoError(t, os.WriteFile(f2, []byte("package test\nvalues: {replicas: 4}\n"), 0o644))
	mod := moduleWithConfig(t, `{#config: {replicas: int}}`)

	_, err := resolveModuleValues(k, mod, dir, []string{f1, f2})
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

// TestResolveModuleValues_SchemaViolationRejected asserts a -f file is
// validated against #config before synthesis.
func TestResolveModuleValues_SchemaViolationRejected(t *testing.T) {
	k := kernel.New()
	valuesFile := filepath.Join(t.TempDir(), "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {bogus: 1}\n"), 0o644))
	mod := moduleWithConfig(t, `{#config: close({replicas: int | *1})}`)

	_, err := resolveModuleValues(k, mod, "mod", []string{valuesFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field not allowed")
	assert.True(t, positionsName(err, "values.cue"), "the violation must be attributed to the values file: %v", cueerrors.Positions(err))
}

// TestResolveModuleValues_FallbackDebugValues exercises the debugValues path:
// the field becomes the single values source, attributed to the module's
// debugValues and carrying the field's data.
func TestResolveModuleValues_FallbackDebugValues(t *testing.T) {
	k := kernel.New()
	mod := moduleWithConfig(t, `{#config: {replicas: int}, debugValues: {replicas: 5}}`)

	values, err := resolveModuleValues(k, mod, "mod", nil)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.Equal(t, filepath.Join("mod", "debugValues"), values[0].Origin)
	// A Source carries bytes the kernel compiles where it uses them: read the
	// field's data back through the validation primitive.
	merged, err := k.ValidateConfigDetailed(mod.ConfigSchema(), values)
	require.NoError(t, err)
	replicas, err := merged.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(5), replicas)
}

// TestResolveModuleValues_NoDebugValues asserts the actionable error when
// the module defines neither debugValues nor a -f flag.
func TestResolveModuleValues_NoDebugValues(t *testing.T) {
	k := kernel.New()
	mod := moduleWithConfig(t, `{metadata: name: "x"}`)

	_, err := resolveModuleValues(k, mod, "mod", nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "debugValues")
}
