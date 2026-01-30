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

func TestNewModTidyCmd(t *testing.T) {
	cmd := NewModTidyCmd()

	assert.Equal(t, "tidy", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check flag exists
	assert.NotNil(t, cmd.Flags().Lookup("dir"))
}

func TestModTidy_DirectoryNotFound(t *testing.T) {
	cmd := NewModTidyCmd()
	cmd.SetArgs([]string{"--dir", "/nonexistent/path"})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "does not exist")
}

func TestModTidy_NotACUEModule(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mod-tidy-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Empty directory, no cue.mod/module.cue
	cmd := NewModTidyCmd()
	cmd.SetArgs([]string{"--dir", tmpDir})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "not a CUE module")
}

func TestModTidy_ValidModule(t *testing.T) {
	if _, err := exec.LookPath("cue"); err != nil {
		t.Skip("cue binary not available")
	}

	tmpDir, err := os.MkdirTemp("", "mod-tidy-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a valid CUE module
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

	cmd := NewModTidyCmd()
	cmd.SetArgs([]string{"--dir", tmpDir})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.NoError(t, err)
}

func TestModTidy_DefaultDir(t *testing.T) {
	cmd := NewModTidyCmd()

	// Check default value of --dir flag
	dirFlag := cmd.Flags().Lookup("dir")
	require.NotNil(t, dirFlag)
	assert.Equal(t, ".", dirFlag.DefValue)
}
