// Package cmd provides CLI command implementations.
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/spf13/cobra"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewModInitCmd(t *testing.T) {
	cmd := NewModInitCmd()

	assert.Equal(t, "init <module-name>", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check flags exist
	assert.NotNil(t, cmd.Flags().Lookup("template"))
	assert.NotNil(t, cmd.Flags().Lookup("dir"))
}

func TestModInit_RequiresArgs(t *testing.T) {
	cmd := NewModInitCmd()
	cmd.SetArgs([]string{})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	// Cobra's ExactArgs(1) returns "accepts 1 arg(s), received 0"
	assert.Contains(t, err.Error(), "accepts 1 arg")
}

func TestModInit_InvalidTemplate(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mod-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	cmd := NewModInitCmd()
	cmd.SetArgs([]string{"test-app", "--template", "invalid", "--dir", filepath.Join(tmpDir, "out")})

	// Capture output
	var outBuf, errBuf bytes.Buffer
	cmd.SetOut(&outBuf)
	cmd.SetErr(&errBuf)

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "unknown template")
}

func TestModInit_DirectoryExists(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mod-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create the target directory
	targetDir := filepath.Join(tmpDir, "existing-dir")
	require.NoError(t, os.MkdirAll(targetDir, 0o755))

	cmd := NewModInitCmd()
	cmd.SetArgs([]string{"test-app", "--dir", targetDir})

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestModInit_Simple(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mod-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	targetDir := filepath.Join(tmpDir, "my-app")

	cmd := NewModInitCmd()
	cmd.SetArgs([]string{"my-app", "--template", "simple", "--dir", targetDir})

	// Silence output
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Check files were created
	assert.FileExists(t, filepath.Join(targetDir, "module.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "values.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "cue.mod", "module.cue"))
}

func TestModInit_Standard(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mod-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	targetDir := filepath.Join(tmpDir, "my-app")

	cmd := NewModInitCmd()
	cmd.SetArgs([]string{"my-app", "--template", "standard", "--dir", targetDir})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Check files were created
	assert.FileExists(t, filepath.Join(targetDir, "module.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "values.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "components.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "cue.mod", "module.cue"))
}

func TestModInit_Advanced(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mod-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	targetDir := filepath.Join(tmpDir, "my-app")

	cmd := NewModInitCmd()
	cmd.SetArgs([]string{"my-app", "--template", "advanced", "--dir", targetDir})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Check files and directories were created
	assert.FileExists(t, filepath.Join(targetDir, "module.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "values.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "components.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "scopes.cue"))
	assert.FileExists(t, filepath.Join(targetDir, "policies.cue"))
	assert.DirExists(t, filepath.Join(targetDir, "components"))
	assert.DirExists(t, filepath.Join(targetDir, "scopes"))
}

func TestModInit_DefaultDir(t *testing.T) {
	// Save and change working directory
	origWd, err := os.Getwd()
	require.NoError(t, err)

	tmpDir, err := os.MkdirTemp("", "mod-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	require.NoError(t, os.Chdir(tmpDir))
	defer os.Chdir(origWd)

	cmd := NewModInitCmd()
	cmd.SetArgs([]string{"test-module", "--template", "simple"})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Should create directory with module name
	assert.DirExists(t, filepath.Join(tmpDir, "test-module"))
	assert.FileExists(t, filepath.Join(tmpDir, "test-module", "module.cue"))
}

func TestModInit_ContentSubstitution(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "mod-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	targetDir := filepath.Join(tmpDir, "my-special-app")

	cmd := NewModInitCmd()
	cmd.SetArgs([]string{"my-special-app", "--template", "simple", "--dir", targetDir})

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Check module path contains the module name
	cueModContent, err := os.ReadFile(filepath.Join(targetDir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(cueModContent), "my-special-app")
}

func TestGetFileDescription(t *testing.T) {
	tests := []struct {
		filename string
		want     string
	}{
		{"module.cue", "Module definition"},
		{"values.cue", "Default values"},
		{"components.cue", "Component definitions"},
		{"cue.mod/module.cue", "CUE module metadata"},
		{"scopes.cue", "Scope definitions"},
		{"policies.cue", "Policy definitions"},
		{"debug_values.cue", "Debug-specific values"},
		{"components/api.cue", "Component template"},
		{"scopes/backend.cue", "Scope template"},
		{"unknown.cue", ""},
	}

	for _, tt := range tests {
		t.Run(tt.filename, func(t *testing.T) {
			got := getFileDescription(tt.filename)
			assert.Equal(t, tt.want, got)
		})
	}
}

// newTestCommand creates a command with silenced output for testing
func newTestCommand(cmd *cobra.Command) *cobra.Command {
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	return cmd
}
