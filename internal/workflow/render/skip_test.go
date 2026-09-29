package render

import (
	"context"
	"errors"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/config"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// skipFixtureRegistry routes core and the catalogs to GHCR; the
// skip-unprovided fixture pins nothing on the testing domain.
const skipFixtureRegistry = "opmodel.dev=ghcr.io/open-platform-model,registry.cue.works"

// backupFQN is the provider-fulfilled trait the skip-unprovided fixture's
// db component attaches while its backup value is true.
const backupFQN = "opmodel.dev/catalogs/opm/traits/backup@v1alpha1"

// skipFixture returns the absolute path of the skip-unprovided test module,
// skipping the test when core cannot be fetched.
func skipFixture(t *testing.T) string {
	t.Helper()
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed tests")
	}
	if _, err := config.NewKernel(skipFixtureRegistry).SchemaCache().Get(); err != nil {
		t.Skipf("core v2 schema unavailable (registry/cache): %v", err)
	}
	dir, err := filepath.Abs(filepath.Join("testdata", "skip-unprovided"))
	require.NoError(t, err)
	return dir
}

func skipFixtureConfig(t *testing.T) *config.GlobalConfig {
	t.Helper()
	return &config.GlobalConfig{ConfigPath: filepath.Join(t.TempDir(), "config.cue"), Registry: skipFixtureRegistry}
}

// Without the flag the kernel input is exactly what it was before the
// switch existed; with it, the switch is carried through and nothing else
// moves.
func TestNewRenderInput_CarriesSkipUnprovided(t *testing.T) {
	inst := &module.Instance{}
	off := newRenderInput(&renderEnv{skew: kernel.SkewWarn}, inst)
	assert.Equal(t, kernel.RenderInput{
		Instance:          inst,
		RuntimeName:       RuntimeName,
		Skew:              kernel.SkewWarn,
		LocalReplacements: true,
	}, off, "the flag off leaves the kernel input as before")

	on := newRenderInput(&renderEnv{skew: kernel.SkewWarn, skipUnprovided: true}, inst)
	assert.True(t, on.SkipUnprovided)
	on.SkipUnprovided = false
	assert.Equal(t, off, on, "the flag moves only the switch")
}

// requireRefusedUnprovided asserts err is the kernel's refusal of the
// fixture's backup demand, marked unprovided, exiting as a validation
// failure.
func requireRefusedUnprovided(t *testing.T, err error) {
	t.Helper()
	require.Error(t, err)
	var exitErr *opmexit.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, opmexit.ExitValidationError, exitErr.Code)
	var renderErr *kernel.RenderError
	require.True(t, errors.As(err, &renderErr))
	require.Len(t, renderErr.Diagnostics.Unresolved, 1)
	assert.Equal(t, backupFQN, renderErr.Diagnostics.Unresolved[0].FQN)
	assert.True(t, renderErr.Diagnostics.Unresolved[0].Unprovided)
}

// assertSkippedBackup asserts a render with the flag skipped exactly the
// fixture's backup trait and still rendered both components.
func assertSkippedBackup(t *testing.T, result *Result) {
	t.Helper()
	require.Len(t, result.Skipped, 1)
	s := result.Skipped[0]
	assert.Equal(t, "db", s.Component)
	assert.Equal(t, backupFQN, s.FQN)
	assert.Equal(t, "trait", s.Kind)
	assert.False(t, s.ComponentOmitted, "a skipped trait leaves its component rendering")
	kinds := map[string]bool{}
	for _, r := range result.Resources {
		kinds[r.GetKind()] = true
	}
	assert.True(t, kinds["StatefulSet"], "db still renders")
	assert.True(t, kinds["Deployment"], "web renders")
}

// Both entry points carry SkipUnprovided to the kernel: the module path
// (FromModule) and the instance path (FromInstanceFile) refuse the fixture
// without it and render it with it.
func TestSkipUnprovided_BothEntryPoints(t *testing.T) {
	dir := skipFixture(t)
	ctx := context.Background()
	captureRenderLog(t)

	moduleOpts := func(skip bool) ModuleOpts {
		return ModuleOpts{ModulePath: dir, SkipUnprovided: skip, Config: skipFixtureConfig(t), K8sConfig: &config.ResolvedKubernetesConfig{}}
	}
	instanceOpts := func(skip bool) InstanceFileOpts {
		return InstanceFileOpts{InstanceFilePath: filepath.Join(dir, "instance"), SkipUnprovided: skip, Config: skipFixtureConfig(t), K8sConfig: &config.ResolvedKubernetesConfig{}}
	}

	t.Run("module", func(t *testing.T) {
		_, err := FromModule(ctx, moduleOpts(false))
		requireRefusedUnprovided(t, err)
		result, err := FromModule(ctx, moduleOpts(true))
		require.NoError(t, err)
		assertSkippedBackup(t, result)
	})
	t.Run("instance", func(t *testing.T) {
		_, err := FromInstanceFile(ctx, instanceOpts(false))
		requireRefusedUnprovided(t, err)
		result, err := FromInstanceFile(ctx, instanceOpts(true))
		require.NoError(t, err)
		assertSkippedBackup(t, result)
	})
}

// A render with nothing to skip is the same with and without the flag: the
// same objects, the same digest, no skipped rows.
func TestSkipUnprovided_NothingToSkipIsUnchanged(t *testing.T) {
	dir := skipFixture(t)
	ctx := context.Background()
	captureRenderLog(t)
	values := filepath.Join(t.TempDir(), "values.cue")
	require.NoError(t, os.WriteFile(values, []byte("backup: false\n"), 0o600))

	render := func(skip bool) *Result {
		result, err := FromModule(ctx, ModuleOpts{
			ModulePath:     dir,
			ValuesFiles:    []string{values},
			SkipUnprovided: skip,
			Config:         skipFixtureConfig(t),
			K8sConfig:      &config.ResolvedKubernetesConfig{},
		})
		require.NoError(t, err)
		return result
	}
	off, on := render(false), render(true)

	assert.Empty(t, off.Skipped)
	assert.Empty(t, on.Skipped)
	assert.Equal(t, off.RenderDigest, on.RenderDigest)
	require.Len(t, on.Resources, len(off.Resources))
	for i := range off.Resources {
		assert.Equal(t, off.Resources[i].Object, on.Resources[i].Object)
	}
}
