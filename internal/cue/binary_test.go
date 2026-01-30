// Package cue provides CUE SDK integration and binary delegation.
package cue

import (
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	oerrors "github.com/opmodel/cli/internal/errors"
)

func TestExtractMajorMinor(t *testing.T) {
	tests := []struct {
		input string
		want  string
	}{
		{"v0.15.0", "0.15"},
		{"0.15.0", "0.15"},
		{"v1.2.3", "1.2"},
		{"v0.15.0-alpha.1", "0.15"},
		{"0.15", "0.15"},
		{"0", "0"},
		{"", ""},
	}

	for _, tt := range tests {
		t.Run(tt.input, func(t *testing.T) {
			got := extractMajorMinor(tt.input)
			assert.Equal(t, tt.want, got)
		})
	}
}

func TestCheckCUEVersion_NoBinary(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set PATH to empty to ensure cue binary is not found
	os.Setenv("PATH", "")

	err := CheckCUEVersion()
	assert.Error(t, err)

	// Should be a NotFound error
	var notFoundErr *oerrors.ErrorDetail
	if assert.ErrorAs(t, err, &notFoundErr) {
		assert.Equal(t, oerrors.ErrNotFound, notFoundErr.Cause)
	}
}

func TestRunCUECommand_NoBinary(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set PATH to empty
	os.Setenv("PATH", "")

	err := RunCUECommand(".", []string{"version"}, "")
	assert.Error(t, err)

	// Should be a NotFound error
	var notFoundErr *oerrors.ErrorDetail
	if assert.ErrorAs(t, err, &notFoundErr) {
		assert.Equal(t, oerrors.ErrNotFound, notFoundErr.Cause)
	}
}

// skipIfNoCUE skips the test if cue binary is not available
func skipIfNoCUE(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue binary not available")
	}
}

func TestRunCUECommand_WithRegistry(t *testing.T) {
	skipIfNoCUE(t)

	// Create a minimal CUE module for testing
	tmpDir, err := os.MkdirTemp("", "cue-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create cue.mod/module.cue
	cueModDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o755))

	moduleCue := `module: "test.local/testmod@v0"
language: version: "v0.15.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleCue), 0o644))

	// Create a simple CUE file
	testCue := `package testmod
x: 1
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.cue"), []byte(testCue), 0o644))

	// Run cue vet (should succeed)
	err = RunCUECommand(tmpDir, []string{"vet", "./..."}, "")
	assert.NoError(t, err)
}

func TestRunCUECommand_ValidationFailure(t *testing.T) {
	skipIfNoCUE(t)

	// Create a CUE module with invalid content
	tmpDir, err := os.MkdirTemp("", "cue-test-invalid-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create cue.mod/module.cue
	cueModDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o755))

	moduleCue := `module: "test.local/testmod@v0"
language: version: "v0.15.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleCue), 0o644))

	// Create an invalid CUE file (conflicting values)
	invalidCue := `package testmod
x: 1
x: 2
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "invalid.cue"), []byte(invalidCue), 0o644))

	// Run cue vet (should fail with validation error)
	err = RunCUECommand(tmpDir, []string{"vet", "./..."}, "")
	assert.Error(t, err)
}

func TestVet_NoBinary(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set PATH to empty
	os.Setenv("PATH", "")

	err := Vet(".", false, "")
	assert.Error(t, err)
}

func TestVet_Success(t *testing.T) {
	skipIfNoCUE(t)

	// Create a valid CUE module
	tmpDir, err := os.MkdirTemp("", "cue-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create cue.mod/module.cue
	cueModDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o755))

	moduleCue := `module: "test.local/testmod@v0"
language: version: "v0.15.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleCue), 0o644))

	// Create a valid CUE file
	testCue := `package testmod
x: int & >0
x: 42
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.cue"), []byte(testCue), 0o644))

	err = Vet(tmpDir, false, "")
	assert.NoError(t, err)
}

func TestTidy_NoBinary(t *testing.T) {
	// Save original PATH
	origPath := os.Getenv("PATH")
	defer os.Setenv("PATH", origPath)

	// Set PATH to empty
	os.Setenv("PATH", "")

	err := Tidy(".", "")
	assert.Error(t, err)
}

func TestTidy_Success(t *testing.T) {
	skipIfNoCUE(t)

	// Create a valid CUE module
	tmpDir, err := os.MkdirTemp("", "cue-tidy-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create cue.mod/module.cue
	cueModDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o755))

	moduleCue := `module: "test.local/testmod@v0"
language: version: "v0.15.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleCue), 0o644))

	// Create a simple CUE file
	testCue := `package testmod
x: 1
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "test.cue"), []byte(testCue), 0o644))

	err = Tidy(tmpDir, "")
	assert.NoError(t, err)
}
