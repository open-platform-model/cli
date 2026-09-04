package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue"
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
// optional debugValues, as resolveModuleValues reads them.
func moduleWithConfig(t *testing.T, k *kernel.Kernel, src string) *module.Module {
	t.Helper()
	modVal := k.CueContext().CompileString(src)
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
	mod := moduleWithConfig(t, k, `{#config: {replicas: int}, debugValues: {replicas: 1}}`)

	values, err := resolveModuleValues(k, mod, []string{valuesFile})
	require.NoError(t, err)
	require.True(t, values.Exists())
	replicas, err := values.LookupPath(cue.ParsePath("replicas")).Int64()
	require.NoError(t, err)
	assert.Equal(t, int64(3), replicas, "the -f file wins over debugValues")
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
	mod := moduleWithConfig(t, k, `{#config: {replicas: int}}`)

	_, err := resolveModuleValues(k, mod, []string{f1, f2})
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
	mod := moduleWithConfig(t, k, `{#config: close({replicas: int | *1})}`)

	_, err := resolveModuleValues(k, mod, []string{valuesFile})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "field not allowed")
	assert.True(t, positionsName(err, "values.cue"), "the violation must be attributed to the values file: %v", cueerrors.Positions(err))
}

// TestResolveModuleValues_FallbackDebugValues exercises the debugValues path.
func TestResolveModuleValues_FallbackDebugValues(t *testing.T) {
	k := kernel.New()
	mod := moduleWithConfig(t, k, `{#config: {replicas: int}, debugValues: {replicas: 5}}`)

	values, err := resolveModuleValues(k, mod, nil)
	require.NoError(t, err)
	assert.True(t, values.Exists())
}

// TestResolveModuleValues_NoDebugValues asserts the actionable error when
// the module defines neither debugValues nor a -f flag.
func TestResolveModuleValues_NoDebugValues(t *testing.T) {
	k := kernel.New()
	mod := moduleWithConfig(t, k, `{metadata: name: "x"}`)

	_, err := resolveModuleValues(k, mod, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "debugValues")
}
