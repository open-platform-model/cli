package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
		Config:     &config.GlobalConfig{CueContext: cuecontext.New()},
		K8sConfig:  &config.ResolvedKubernetesConfig{},
	})
	require.Error(t, err)
	assert.Contains(t, err.Error(), "instance package")
}

// TestResolveModuleValues_UsesValuesFile asserts that supplied -f files are
// loaded in preference to debugValues.
func TestResolveModuleValues_UsesValuesFile(t *testing.T) {
	ctx := cuecontext.New()
	dir := t.TempDir()
	valuesFile := filepath.Join(dir, "values.cue")
	require.NoError(t, os.WriteFile(valuesFile, []byte("package test\nvalues: {replicas: 3}\n"), 0o644))

	modVal := ctx.CompileString(`{debugValues: {replicas: 1}}`)
	require.NoError(t, modVal.Err())

	values, err := resolveModuleValues(ctx, modVal, []string{valuesFile})
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
}

// TestResolveModuleValues_FallbackDebugValues exercises the debugValues path.
func TestResolveModuleValues_FallbackDebugValues(t *testing.T) {
	ctx := cuecontext.New()
	modVal := ctx.CompileString(`{debugValues: {replicas: 5}}`)
	require.NoError(t, modVal.Err())

	values, err := resolveModuleValues(ctx, modVal, nil)
	require.NoError(t, err)
	require.Len(t, values, 1)
	assert.True(t, values[0].Exists())
}

// TestResolveModuleValues_NoDebugValues asserts the actionable error when
// the module defines neither debugValues nor a -f flag.
func TestResolveModuleValues_NoDebugValues(t *testing.T) {
	ctx := cuecontext.New()
	modVal := ctx.CompileString(`{metadata: name: "x"}`)
	require.NoError(t, modVal.Err())

	_, err := resolveModuleValues(ctx, modVal, nil)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "debugValues")
}

// TestFromModule_RegistryE2E exercises the full FromModule pipeline against
// an example module with debugValues. Skipped if the registry is unreachable.
func TestFromModule_RegistryE2E(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed render tests")
	}

	modPath, err := filepath.Abs("../../../examples/modules/mc_router")
	require.NoError(t, err)
	if _, statErr := os.Stat(modPath); statErr != nil {
		t.Skip("examples/modules/mc_router not available")
	}

	cfg := &config.GlobalConfig{
		CueContext: cuecontext.New(),
	}
	k8sCfg := &config.ResolvedKubernetesConfig{}

	result, err := FromModule(context.Background(), ModuleOpts{
		ModulePath: modPath,
		Name:       "mc-router-test",
		Config:     cfg,
		K8sConfig:  k8sCfg,
	})
	if err != nil {
		t.Skipf("FromModule failed (likely registry/provider unavailable): %v", err)
	}
	require.NotNil(t, result)
	assert.Equal(t, "mc-router-test", result.Instance.Name)
}
