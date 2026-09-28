package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/cuemod/cuemodtest"
	opmexit "github.com/open-platform-model/cli/internal/exit"
)

// TestE2E_ModTidy_PinsCoreFromRegistry drives the whole authoring loop
// against the real registry: a module importing core with no deps fails the
// check, tidy pins core, and the check then passes. No cue binary is
// involved at any step.
func TestE2E_ModTidy_PinsCoreFromRegistry(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}

	modDir := t.TempDir()
	require.NoError(t, os.MkdirAll(filepath.Join(modDir, "cue.mod"), 0o755))
	moduleCue := `module: "testing.opmodel.dev/modules/cli/e2e-tidy@v0"
language: version: "v0.17.0"
deps: {}
`
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "cue.mod", "module.cue"), []byte(moduleCue), 0o644))
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "main.cue"), []byte(`package tidy

import "opmodel.dev/core@v2"

_core: core
`), 0o644))

	const timeout = 180 * time.Second

	_, stderr, err := runOPMWithEnv(t, modDir, homeDir, timeout, "module", "tidy", "--check")
	require.Error(t, err, "an unpinned core import must fail the check")
	assert.Equal(t, opmexit.ExitValidationError, exitCode(t, err), "stderr: %s", stderr)
	assert.Contains(t, stderr, "module is not tidy")
	assert.Contains(t, stderr, "opmodel.dev/core@v2")
	assert.Contains(t, stderr, "opm module tidy")
	data, readErr := os.ReadFile(filepath.Join(modDir, "cue.mod", "module.cue"))
	require.NoError(t, readErr)
	assert.Equal(t, moduleCue, string(data), "--check must not write")

	_, stderr, err = runOPMWithEnv(t, modDir, homeDir, timeout, "mod", "tidy")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stderr, "updated cue.mod/module.cue")
	data, readErr = os.ReadFile(filepath.Join(modDir, "cue.mod", "module.cue"))
	require.NoError(t, readErr)
	assert.Contains(t, string(data), `"opmodel.dev/core@v2"`)

	_, stderr, err = runOPMWithEnv(t, modDir, homeDir, timeout, "module", "tidy", "--check")
	require.NoError(t, err, "a freshly tidied module must pass the check; stderr: %s", stderr)
}

// TestE2E_ModTidy_RegistryFlagRoutesResolution proves the root --registry
// flag reaches tidy: both registry variables in the process environment
// point at a dead address, so only the flag's in-process registry can
// resolve the dependency. Hermetic: no network.
func TestE2E_ModTidy_RegistryFlagRoutesResolution(t *testing.T) {
	modDir, registry := cuemodtest.NewConsumer(t)
	deadEnv := []string{
		"CUE_REGISTRY=" + cuemodtest.UnreachableRegistry,
		"OPM_REGISTRY=" + cuemodtest.UnreachableRegistry,
	}

	_, stderr, err := runOPMPublish(t, modDir, deadEnv, "--registry", registry, "module", "tidy")
	require.NoError(t, err, "stderr: %s", stderr)
	got := cuemodtest.ReadModule(t, modDir)
	assert.Contains(t, got, `"`+cuemodtest.DepModule+`"`)
	assert.Contains(t, got, `"`+cuemodtest.DepNewest+`"`)

	// Control: without the flag the dead environment wins and resolution fails.
	otherDir, _ := cuemodtest.NewConsumer(t)
	_, stderr, err = runOPMPublish(t, otherDir, deadEnv, "module", "tidy")
	require.Error(t, err)
	assert.Equal(t, opmexit.ExitGeneralError, exitCode(t, err), "stderr: %s", stderr)
	assert.Contains(t, stderr, "connection refused")
}
