// Package config provides CLI command implementations for config operations.
package config

import (
	"bytes"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"

	opmconfig "github.com/open-platform-model/cli/internal/config"
)

// setTempHome points HOME at a fresh temp dir for the test.
func setTempHome(t *testing.T) string {
	t.Helper()
	tmpHome := t.TempDir()
	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	t.Cleanup(func() { os.Setenv("HOME", origHome) })
	return tmpHome
}

func TestNewConfigInitCmd(t *testing.T) {
	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})

	assert.Equal(t, "init", cmd.Use)
	assert.NotEmpty(t, cmd.Short)
	assert.NotEmpty(t, cmd.Long)

	// Check flags exist; --no-tidy is gone with the CUE-module retirement
	// (enhancement 0006 D39).
	assert.NotNil(t, cmd.Flags().Lookup("force"))
	assert.Nil(t, cmd.Flags().Lookup("no-tidy"))
}

func TestConfigInit_CreatesFiles(t *testing.T) {
	tmpHome := setTempHome(t)

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())

	// Check files were created: config.cue + platform.cue, and NO cue.mod
	opmDir := filepath.Join(tmpHome, ".opm")
	assert.DirExists(t, opmDir)
	assert.FileExists(t, filepath.Join(opmDir, "config.cue"))
	assert.FileExists(t, filepath.Join(opmDir, "platform.cue"))
	assert.NoDirExists(t, filepath.Join(opmDir, "cue.mod"))
}

func TestConfigInit_SecurePermissions(t *testing.T) {
	tmpHome := setTempHome(t)

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())

	// Check directory permissions (0700)
	opmDir := filepath.Join(tmpHome, ".opm")
	dirInfo, err := os.Stat(opmDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	// Check file permissions (0600)
	for _, name := range []string{"config.cue", "platform.cue"} {
		fileInfo, err := os.Stat(filepath.Join(opmDir, name))
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm(), name)
	}
}

func TestConfigInit_ExistingConfig(t *testing.T) {
	tmpHome := setTempHome(t)

	// Create existing config
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))
	configFile := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configFile, []byte("// existing config"), 0o600))

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	err := cmd.Execute()
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "already exists")
}

func TestConfigInit_ForceOverwrite(t *testing.T) {
	tmpHome := setTempHome(t)

	// Create existing config
	opmDir := filepath.Join(tmpHome, ".opm")
	require.NoError(t, os.MkdirAll(opmDir, 0o700))
	configFile := filepath.Join(opmDir, "config.cue")
	require.NoError(t, os.WriteFile(configFile, []byte("// old config"), 0o600))

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetArgs([]string{"--force"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())

	// Check file was overwritten
	content, err := os.ReadFile(configFile)
	require.NoError(t, err)
	assert.NotContains(t, string(content), "old config")
}

func TestConfigInit_ConfigContent(t *testing.T) {
	tmpHome := setTempHome(t)

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())

	// Check config.cue content: scalar data, no providers, no imports
	configFile := filepath.Join(tmpHome, ".opm", "config.cue")
	content, err := os.ReadFile(configFile)
	require.NoError(t, err)

	configStr := string(content)
	assert.Contains(t, configStr, "kubernetes")
	assert.NotContains(t, configStr, "providers")
	assert.NotContains(t, configStr, "import")

	// Check platform.cue content: seeded official catalog subscriptions
	// with explicit ranges (enhancement 0006 D39)
	platformFile := filepath.Join(tmpHome, ".opm", "platform.cue")
	platformContent, err := os.ReadFile(platformFile)
	require.NoError(t, err)

	platformStr := string(platformContent)
	assert.Contains(t, platformStr, "opmodel.dev/catalogs/opm")
	assert.Contains(t, platformStr, "opmodel.dev/catalogs/kubernetes")
	assert.Contains(t, platformStr, "range:")
}

func TestConfigInit_OutputMessage(t *testing.T) {
	tmpHome := setTempHome(t)

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	// Just verify the command executes successfully
	// Note: output.Println writes to stdout, not to cmd.SetOut()
	require.NoError(t, cmd.Execute())

	// Verify files exist (command worked correctly)
	opmDir := filepath.Join(tmpHome, ".opm")
	assert.FileExists(t, filepath.Join(opmDir, "config.cue"))
	assert.FileExists(t, filepath.Join(opmDir, "platform.cue"))
}
