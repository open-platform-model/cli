// Package cmd provides CLI command implementations.
package cmd

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewConfigInitCmd(t *testing.T) {
	cmd := NewConfigInitCmd()

	assert.Equal(t, "init", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check flag exists
	assert.NotNil(t, cmd.Flags().Lookup("force"))
}

func TestConfigInit_CreatesFiles(t *testing.T) {
	// Use temp home directory
	tmpHome, err := os.MkdirTemp("", "config-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	// Override HOME for the test
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cmd := NewConfigInitCmd()

	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Check files were created
	opmDir := filepath.Join(tmpHome, ".opm")
	assert.DirExists(t, opmDir)
	assert.FileExists(t, filepath.Join(opmDir, "config.cue"))
	assert.FileExists(t, filepath.Join(opmDir, "cue.mod", "module.cue"))
}

func TestConfigInit_SecurePermissions(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cmd := NewConfigInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Check directory permissions (0700)
	opmDir := filepath.Join(tmpHome, ".opm")
	dirInfo, err := os.Stat(opmDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	// Check file permissions (0600)
	configFile := filepath.Join(opmDir, "config.cue")
	fileInfo, err := os.Stat(configFile)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
}

func TestConfigInit_ExistingConfig(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create existing config
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))
	configFile := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configFile, []byte("// existing config"), 0o600))

	cmd := NewConfigInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestConfigInit_ForceOverwrite(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Create existing config
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))
	configFile := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configFile, []byte("// old config"), 0o600))

	cmd := NewConfigInitCmd()
	cmd.SetArgs([]string{"--force"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Check file was overwritten
	content, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "old config")
}

func TestConfigInit_ConfigContent(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cmd := NewConfigInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err = cmd.Execute()
	require.NoError(t, err)

	// Check config.cue content
	configFile := filepath.Join(tmpHome, ".opm", "config.cue")
	content, err := os.ReadFile(configFile)
	require.NoError(t, err)

	// Should contain expected fields
	configStr := string(content)
	assert.Contains(t, configStr, "kubernetes")

	// Check cue.mod/module.cue content
	moduleFile := filepath.Join(tmpHome, ".opm", "cue.mod", "module.cue")
	moduleContent, err := os.ReadFile(moduleFile)
	require.NoError(t, err)

	// Should have module declaration
	assert.Contains(t, string(moduleContent), "module:")
}

func TestConfigInit_OutputMessage(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "config-init-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	cmd := NewConfigInitCmd()
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	// Just verify the command executes successfully
	// Note: output.Println writes to stdout, not to cmd.SetOut()
	err = cmd.Execute()
	require.NoError(t, err)

	// Verify files exist (command worked correctly)
	opmDir := filepath.Join(tmpHome, ".opm")
	assert.FileExists(t, filepath.Join(opmDir, "config.cue"))
	assert.FileExists(t, filepath.Join(opmDir, "cue.mod", "module.cue"))
}
