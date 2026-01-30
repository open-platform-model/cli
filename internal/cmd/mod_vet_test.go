// Package cmd provides CLI command implementations.
package cmd

import (
	"bytes"
	"os"
	"os/exec"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModVetCmd(t *testing.T) {
	cmd := NewModVetCmd()

	assert.Equal(t, "vet", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check flags exist
	assert.NotNil(t, cmd.Flags().Lookup("concrete"))
	assert.NotNil(t, cmd.Flags().Lookup("dir"))
}

func TestModVet_DirectoryNotFound(t *testing.T) {
	cmd := NewModVetCmd()
	cmd.SetArgs([]string{"--dir", "/nonexistent/path"})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestModVet_NotAModule(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mod-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Empty directory, no module.cue
	cmd := NewModVetCmd()
	cmd.SetArgs([]string{"--dir", tmpDir})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not an OPM module")
}

// skipIfNoCUEBinary skips tests that require the cue binary
func skipIfNoCUEBinary(t *testing.T) {
	t.Helper()
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue binary not available")
	}
}

func TestModVet_ValidModule(t *testing.T) {
	skipIfNoCUEBinary(t)

	tmpDir, err := os.MkdirTemp("", "mod-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a valid CUE module
	cueModDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o755))

	moduleCue := `module: "test.local/testmod@v0"
language: version: "v0.15.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleCue), 0o644))

	// Create module.cue (required by OPM)
	opmModule := `package testmod

name: "test-module"
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "module.cue"), []byte(opmModule), 0o644))

	cmd := NewModVetCmd()
	cmd.SetArgs([]string{"--dir", tmpDir})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.NoError(t, err)
}

func TestModVet_InvalidModule(t *testing.T) {
	skipIfNoCUEBinary(t)

	tmpDir, err := os.MkdirTemp("", "mod-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create cue.mod
	cueModDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o755))

	moduleCue := `module: "test.local/testmod@v0"
language: version: "v0.15.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleCue), 0o644))

	// Create invalid module.cue (conflicting values)
	invalidModule := `package testmod

x: 1
x: 2
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "module.cue"), []byte(invalidModule), 0o644))

	cmd := NewModVetCmd()
	cmd.SetArgs([]string{"--dir", tmpDir})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.Error(t, err)
}

func TestModVet_ConcreteFlag(t *testing.T) {
	skipIfNoCUEBinary(t)

	tmpDir, err := os.MkdirTemp("", "mod-vet-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create cue.mod
	cueModDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(cueModDir, 0o755))

	moduleCue := `module: "test.local/testmod@v0"
language: version: "v0.15.0"
`
	require.NoError(t, os.WriteFile(filepath.Join(cueModDir, "module.cue"), []byte(moduleCue), 0o644))

	// Create module with incomplete value (should fail with --concrete)
	incompleteModule := `package testmod

x: int
`
	require.NoError(t, os.WriteFile(filepath.Join(tmpDir, "module.cue"), []byte(incompleteModule), 0o644))

	cmd := NewModVetCmd()
	cmd.SetArgs([]string{"--dir", tmpDir, "--concrete"})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.Error(t, err)
}

func TestModVet_DefaultDir(t *testing.T) {
	cmd := NewModVetCmd()

	// Check default value of --dir flag
	dirFlag := cmd.Flags().Lookup("dir")
	require.NotNil(t, dirFlag)
	assert.Equal(t, ".", dirFlag.DefValue)
}
