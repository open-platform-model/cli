package e2e

import (
	"errors"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestE2E_ReleaseVet_Output(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	testdataDir := filepath.Join(cwd, "testdata", "vet-errors")

	tmpDir, err := os.MkdirTemp("", "e2e-rel-vet-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, stderr, err := runOPM(t, tmpDir, "rel", "vet",
		filepath.Join(testdataDir, "release", "release.cue"),
		"-f", filepath.Join(testdataDir, "release", "values.cue"))

	// Assert exit code 2
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode())

	// Assert grouped shape exists
	assert.Contains(t, stderr, "render failed: 2 issues")
	assert.Contains(t, stderr, "field not allowed")
	assert.Contains(t, stderr, "values.test")
	assert.Contains(t, stderr, "conflicting values")
	assert.Contains(t, stderr, "values.media.test")

	// Anti-regression: Assert flattened shape does NOT exist
	// (Checking for the literal dash prefix that cue/errors prints when flattened)
	assert.NotContains(t, stderr, "ERRO render failed: - ")
}

func TestE2E_ModuleVet_Output(t *testing.T) {
	cwd, err := os.Getwd()
	require.NoError(t, err)
	testdataDir := filepath.Join(cwd, "testdata", "vet-errors")

	tmpDir, err := os.MkdirTemp("", "e2e-mod-vet-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	_, stderr, err := runOPM(t, tmpDir, "mod", "vet",
		filepath.Join(testdataDir, "module"),
		"-f", filepath.Join(testdataDir, "release", "values.cue"))

	// Assert exit code 2
	require.Error(t, err)
	var exitErr *exec.ExitError
	require.True(t, errors.As(err, &exitErr))
	assert.Equal(t, 2, exitErr.ExitCode())

	// Assert grouped shape exists
	assert.Contains(t, stderr, "values do not satisfy #config: 2 issues")
	assert.Contains(t, stderr, "field not allowed")
	assert.Contains(t, stderr, "values.test")
	assert.Contains(t, stderr, "conflicting values")

	// Anti-regression: Assert flattened shape does NOT exist
	assert.NotContains(t, stderr, "ERRO values do not satisfy #config: - ")
}
