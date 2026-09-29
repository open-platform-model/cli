package e2e

import (
	"bytes"
	"context"
	"os"
	"os/exec"
	"path/filepath"
	"testing"
	"time"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	"github.com/open-platform-model/cli/internal/config"
)

// runOPMWithEnv runs the opm binary with a custom HOME and configurable
// timeout. Used for commands like `module build` that need a config.cue
// matching the test environment's registry layout. Captures stdout and stderr
// independently regardless of exit status.
func runOPMWithEnv(t *testing.T, workDir, customHome string, timeout time.Duration, args ...string) (stdout, stderr string, err error) {
	t.Helper()

	ctx, cancel := context.WithTimeout(context.Background(), timeout)
	defer cancel()

	cmd := exec.CommandContext(ctx, opmBinary, args...)
	cmd.Dir = workDir
	cmd.Env = append(os.Environ(), "HOME="+customHome)

	var stdoutBuf, stderrBuf bytes.Buffer
	cmd.Stdout = &stdoutBuf
	cmd.Stderr = &stderrBuf

	err = cmd.Run()
	return stdoutBuf.String(), stderrBuf.String(), err
}

// TestE2E_ModBuild_FromExampleModule renders a synthetic instance for a module
// using its debugValues. It builds the repo-local podinfo fixture — a current
// core@v2-line module (tests/fixtures/modules/podinfo) — so the test carries no
// dependency on a retired-line example or a sibling checkout. Skipped if the
// registry is unreachable (CI runs with a pre-warmed registry per task 8.5).
func TestE2E_ModBuild_FromExampleModule(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	modPath := filepath.Join(repoRoot, "tests", "fixtures", "modules", "podinfo")
	if _, statErr := os.Stat(modPath); statErr != nil {
		t.Skipf("tests/fixtures/modules/podinfo not available: %v", statErr)
	}

	tmpDir, err := os.MkdirTemp("", "e2e-mod-build-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// A hermetic HOME holding config.cue but no platform/: an author's build
	// renders against a platform generated from the module's own deps, so it
	// needs no local default and reads nothing from the developer's ~/.opm.
	customHome := seedConfigOnlyHome(t)

	stdout, stderr, err := runOPMWithEnv(t, tmpDir, customHome, 180*time.Second, "module", "build", modPath, "--name", "e2e-podinfo")
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stderr, "synthetic instance")
	assert.Contains(t, stderr, "e2e-podinfo")
	assert.Contains(t, stderr, "platform: module deps (opmodel.dev/catalogs/opm@v4 v4.0.1; generated module "+filepath.Join(customHome, ".opm", "cache", "platforms"))
	assert.NotContains(t, stderr, "version skew", "the module's own pins cannot skew")
	assert.NotEmpty(t, stdout, "expected manifest output on stdout")
}

// TestE2E_ModBuild_PlatformFlagOverridesDeps renders the same module against
// an explicit platform module: the flag wins over the module's deps and the
// provenance line names it.
func TestE2E_ModBuild_PlatformFlagOverridesDeps(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	modPath := filepath.Join(repoRoot, "tests", "fixtures", "modules", "podinfo")
	if _, statErr := os.Stat(modPath); statErr != nil {
		t.Skipf("tests/fixtures/modules/podinfo not available: %v", statErr)
	}

	customHome := seedRenderHome(t)
	platformDir := config.PlatformDir(renderHomeConfigPath(customHome))

	stdout, stderr, err := runOPMWithEnv(t, t.TempDir(), customHome, 180*time.Second, "module", "build", modPath, "--name", "e2e-podinfo", "--platform", platformDir)
	require.NoError(t, err, "stderr: %s", stderr)
	assert.Contains(t, stderr, "platform: "+platformDir+" (--platform)")
	assert.NotContains(t, stderr, "module deps")
	assert.NotEmpty(t, stdout, "expected manifest output on stdout")
}

// TestE2E_ModBuild_RejectsFileArgument confirms that a file path produces an
// actionable error directing the user to `opm instance build`.
func TestE2E_ModBuild_RejectsFileArgument(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-mod-build-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	filePath := filepath.Join(tmpDir, "module.cue")
	require.NoError(t, os.WriteFile(filePath, []byte("package x\n"), 0o644))

	_, stderr, err := runOPM(t, tmpDir, "module", "build", filePath)
	require.Error(t, err)
	assert.Contains(t, stderr, "expects a directory")
	assert.Contains(t, stderr, "opm instance build")
}
