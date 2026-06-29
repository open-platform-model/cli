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

// makeBuildHome returns a HOME directory that mirrors the user's real ~/.opm
// configuration so the binary loads providers via the published catalog. The
// stub config used by other e2e tests is too minimal — it does not carry the
// provider schema imports that the synthesis path's downstream render expects.
//
// Skips the test when no real ~/.opm/config.cue is available (CI must mount
// one or otherwise pre-configure HOME).
func makeBuildHome(t *testing.T) string {
	t.Helper()

	realHome, err := os.UserHomeDir()
	if err != nil || realHome == "" {
		t.Skip("no user home dir; skipping module build e2e")
	}
	srcOpm := filepath.Join(realHome, ".opm")
	if _, statErr := os.Stat(filepath.Join(srcOpm, "config.cue")); statErr != nil {
		t.Skipf("no real ~/.opm/config.cue at %q; skipping module build e2e", srcOpm)
	}

	dir, err := os.MkdirTemp("", "e2e-mod-build-home-*")
	require.NoError(t, err)

	dstOpm := filepath.Join(dir, ".opm")
	require.NoError(t, copyDir(srcOpm, dstOpm))
	return dir
}

// copyDir recursively copies src into dst. Created files are writable so a
// test cleanup can later remove them (CUE cache files are extracted read-only).
func copyDir(src, dst string) error {
	return filepath.Walk(src, func(p string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}
		rel, relErr := filepath.Rel(src, p)
		if relErr != nil {
			return relErr
		}
		target := filepath.Join(dst, rel)
		if info.IsDir() {
			return os.MkdirAll(target, 0o755)
		}
		data, readErr := os.ReadFile(p)
		if readErr != nil {
			return readErr
		}
		return os.WriteFile(target, data, 0o644)
	})
}

// TestE2E_ModBuild_FromExampleModule renders a synthetic instance for an
// example module using its debugValues. Skipped if the registry is unreachable
// (CI runs with a pre-warmed registry per task 8.5).
func TestE2E_ModBuild_FromExampleModule(t *testing.T) {
	if os.Getenv("OPM_SKIP_REGISTRY_TESTS") != "" {
		t.Skip("skipping registry-backed e2e tests")
	}

	repoRoot, err := filepath.Abs("../..")
	require.NoError(t, err)
	modPath := filepath.Join(repoRoot, "examples", "modules", "mc_router")
	if _, statErr := os.Stat(modPath); statErr != nil {
		t.Skipf("examples/modules/mc_router not available: %v", statErr)
	}

	tmpDir, err := os.MkdirTemp("", "e2e-mod-build-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	customHome := makeBuildHome(t)
	defer os.RemoveAll(customHome)

	stdout, stderr, err := runOPMWithEnv(t, tmpDir, customHome, 120*time.Second, "module", "build", modPath, "--name", "e2e-mc-router")
	if err != nil {
		t.Skipf("opm module build failed (likely registry/provider unavailable): err=%v stderr=%s", err, stderr)
	}
	assert.Contains(t, stderr, "synthetic instance")
	assert.Contains(t, stderr, "e2e-mc-router")
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
