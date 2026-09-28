package e2e

import (
	"os"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

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
