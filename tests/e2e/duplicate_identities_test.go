package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestE2E_ModBuild_RefusesDuplicateIdentities drives the whole command over a
// module whose two components render one apply identity. The unit tests prove
// the helper call and the printing; this proves the refusal reaches a user
// running `opm module build`, with the validation exit code and with nothing
// on stdout — the build refuses exactly what the apply would have written
// twice.
func TestE2E_ModBuild_RefusesDuplicateIdentities(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}

	modPath, err := filepath.Abs(filepath.Join("testdata", "duplicate-identities"))
	require.NoError(t, err)

	customHome := seedRenderHome(t)

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), customHome, 180*time.Second, "module", "build", modPath, "--name", "e2e-collider")

	require.Error(t, err, "a colliding module must not build; stdout: %s", stdout)
	var exitErr *exec.ExitError
	require.ErrorAs(t, err, &exitErr)
	assert.Equal(t, 2, exitErr.ExitCode(), "a duplicate identity is a validation failure; stderr: %s", stderr)
	assert.Contains(t, stderr, "render failed")
	assert.Contains(t, stderr, "share one identity")
	assert.Contains(t, stderr, "apps/v1 Deployment")
	assert.Contains(t, stderr, "collider")
	assert.Contains(t, stderr, `component "first"`)
	assert.Contains(t, stderr, `component "second"`)
	assert.Empty(t, stdout, "a refused render prints no object")
}
