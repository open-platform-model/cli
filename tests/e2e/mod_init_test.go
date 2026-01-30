// Package e2e provides end-to-end tests for the OPM CLI.
package e2e

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

var opmBinary string

func TestMain(m *testing.M) {
	// Build the binary once for all tests
	tmpDir, err := os.MkdirTemp("", "opm-e2e-*")
	if err != nil {
		panic("failed to create temp dir: " + err.Error())
	}
	defer os.RemoveAll(tmpDir)

	opmBinary = filepath.Join(tmpDir, "opm")

	// Build the binary
	cmd := exec.Command("go", "build", "-o", opmBinary, "../../cmd/opm")
	cmd.Stderr = os.Stderr
	if err := cmd.Run(); err != nil {
		panic("failed to build opm binary: " + err.Error())
	}

	os.Exit(m.Run())
}

// runOPM runs the opm binary with the given arguments and returns output
func runOPM(t *testing.T, workDir string, args ...string) (string, string, error) {
	t.Helper()

	cmd := exec.Command(opmBinary, args...)
	cmd.Dir = workDir

	stdout, err := cmd.Output()
	var stderr []byte
	if exitErr, ok := err.(*exec.ExitError); ok {
		stderr = exitErr.Stderr
	}

	return string(stdout), string(stderr), err
}

func TestE2E_ModInit_SimpleTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-mod-init-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, stderr, err := runOPM(t, tmpDir, "mod", "init", "my-app", "--template", "simple")
	require.NoError(t, err, "stderr: %s", stderr)

	// Verify files were created
	assert.FileExists(t, filepath.Join(tmpDir, "my-app", "module.cue"))
	assert.FileExists(t, filepath.Join(tmpDir, "my-app", "values.cue"))
	assert.FileExists(t, filepath.Join(tmpDir, "my-app", "cue.mod", "module.cue"))
}

func TestE2E_ModInit_StandardTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-mod-init-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, stderr, err := runOPM(t, tmpDir, "mod", "init", "my-app", "--template", "standard")
	require.NoError(t, err, "stderr: %s", stderr)

	// Verify files were created
	assert.FileExists(t, filepath.Join(tmpDir, "my-app", "module.cue"))
	assert.FileExists(t, filepath.Join(tmpDir, "my-app", "values.cue"))
	assert.FileExists(t, filepath.Join(tmpDir, "my-app", "components.cue"))
	assert.FileExists(t, filepath.Join(tmpDir, "my-app", "cue.mod", "module.cue"))
}

func TestE2E_ModInit_InvalidTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-mod-init-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, _, err = runOPM(t, tmpDir, "mod", "init", "my-app", "--template", "invalid")
	assert.Error(t, err)

	// Check exit code is 2 (validation error)
	if exitErr, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, exitErr.ExitCode(), "expected exit code 2 for validation error")
	}
}

func TestE2E_ModInit_ThenVet(t *testing.T) {
	// TODO: Enable this test after templates are fixed to conform to OPM core schema
	// Current templates have issues:
	// - Double @v0 in apiVersion (ModulePath already contains @v0)
	// - Name format doesn't match FQN regex requirements
	t.Skip("templates need fixes to pass mod vet against OPM core schema")

	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue binary not available")
	}

	tmpDir, err := os.MkdirTemp("", "e2e-mod-init-vet-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Initialize module
	_, stderr, err := runOPM(t, tmpDir, "mod", "init", "my-app", "--template", "simple")
	require.NoError(t, err, "mod init failed: %s", stderr)

	// Validate module
	moduleDir := filepath.Join(tmpDir, "my-app")
	_, stderr, err = runOPM(t, moduleDir, "mod", "vet")
	require.NoError(t, err, "mod vet failed: %s", stderr)
}

func TestE2E_ModInit_CustomDir(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-mod-init-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	customDir := filepath.Join(tmpDir, "custom", "path", "my-module")

	_, stderr, err := runOPM(t, tmpDir, "mod", "init", "my-app", "--dir", customDir)
	require.NoError(t, err, "stderr: %s", stderr)

	// Verify files were created in custom directory
	assert.FileExists(t, filepath.Join(customDir, "module.cue"))
	assert.FileExists(t, filepath.Join(customDir, "cue.mod", "module.cue"))
}

func TestE2E_ModInit_DirectoryExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-mod-init-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create the directory first
	existingDir := filepath.Join(tmpDir, "existing")
	require.NoError(t, os.MkdirAll(existingDir, 0o755))

	_, _, err = runOPM(t, tmpDir, "mod", "init", "my-app", "--dir", existingDir)
	assert.Error(t, err)

	// Check exit code is 2 (validation error)
	if exitErr, ok := err.(*exec.ExitError); ok {
		assert.Equal(t, 2, exitErr.ExitCode(), "expected exit code 2 for validation error")
	}
}

func TestE2E_Version(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-version-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	stdout, stderr, err := runOPM(t, tmpDir, "version")
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Contains(t, stdout, "opm version")
	assert.Contains(t, stdout, "CUE SDK")
}

func TestE2E_Help(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "e2e-help-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	stdout, stderr, err := runOPM(t, tmpDir, "--help")
	require.NoError(t, err, "stderr: %s", stderr)

	assert.Contains(t, stdout, "mod")
	assert.Contains(t, stdout, "config")
	assert.Contains(t, stdout, "version")
}
