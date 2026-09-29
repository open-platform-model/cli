package e2e

import (
	"errors"
	"os"
	"os/exec"
	"strings"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// skipBackupFQN is the provider-fulfilled trait the skip-unprovided test
// module attaches; its catalog ships no provider for it.
const skipBackupFQN = "opmodel.dev/catalogs/opm/traits/backup@v1alpha1"

// TestE2E_ModBuild_SkipUnprovided builds the skip-unprovided test module
// against its own deps. Without the flag the provider-fulfilled backup trait
// refuses the render with the unprovided hint; with it the build succeeds,
// the manifests reach stdout and the skip warning reaches stderr only.
func TestE2E_ModBuild_SkipUnprovided(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}
	modPath := repoPath(t, "internal/workflow/render/testdata/skip-unprovided")
	home := seedRenderHome(t)

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), home, 180*time.Second, "module", "build", modPath)
	require.Error(t, err, "stdout: %s", stdout)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode(), "stderr: %s", stderr)
	assert.Contains(t, stderr, skipBackupFQN)
	assert.Contains(t, stderr, "Hint: a provider-fulfilled contract has no provider on this platform: install one, pass --platform <dir> with a platform that carries one, or pass --skip-unprovided to render the rest")
	assert.Empty(t, stdout)

	stdout, stderr, err = runOPMWithEnv(t, t.TempDir(), home, 180*time.Second, "module", "build", modPath, "--skip-unprovided")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stdout, "kind: StatefulSet", "the component with the skipped trait still renders")
	assert.Contains(t, stdout, "kind: Deployment")
	warning := `component "db": skipped provider-fulfilled trait "` + skipBackupFQN + `" (no provider on this platform)`
	assert.Contains(t, stderr, warning)
	assert.NotContains(t, stdout, "skipped provider-fulfilled", "warnings never reach the manifest stream")
	assert.Equal(t, 1, strings.Count(stderr, warning))
}
