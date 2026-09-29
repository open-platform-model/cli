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

// setTempHome points HOME at a fresh temp dir for the test. The CUE module
// cache stays where it is (CUE_CACHE_DIR pinned to the real user cache):
// a registry-backed build would otherwise extract read-only module files
// under the temp HOME and break t.TempDir's cleanup.
func setTempHome(t *testing.T) string {
	t.Helper()
	if os.Getenv("CUE_CACHE_DIR") == "" {
		if cacheDir, err := os.UserCacheDir(); err == nil {
			os.Setenv("CUE_CACHE_DIR", filepath.Join(cacheDir, "cue"))
			t.Cleanup(func() { os.Unsetenv("CUE_CACHE_DIR") })
		}
	}
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
	// (0006:D39).
	assert.NotNil(t, cmd.Flags().Lookup("force"))
	assert.Nil(t, cmd.Flags().Lookup("no-tidy"))
}

func TestConfigInit_CreatesFiles(t *testing.T) {
	tmpHome := setTempHome(t)

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())

	// config.cue and nothing else: no platform module, no data-only
	// platform.cue and no cue.mod beside config.cue.
	opmDir := filepath.Join(tmpHome, ".opm")
	assert.DirExists(t, opmDir)
	assert.FileExists(t, filepath.Join(opmDir, "config.cue"))
	assert.NoDirExists(t, filepath.Join(opmDir, "platform"))
	assert.NoFileExists(t, filepath.Join(opmDir, "platform.cue"))
	assert.NoDirExists(t, filepath.Join(opmDir, "cue.mod"))
	entries, err := os.ReadDir(opmDir)
	require.NoError(t, err)
	assert.Len(t, entries, 1, "init writes config.cue only")
}

func TestConfigInit_SecurePermissions(t *testing.T) {
	tmpHome := setTempHome(t)

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())

	opmDir := filepath.Join(tmpHome, ".opm")
	dirInfo, err := os.Stat(opmDir)
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm())

	fileInfo, err := os.Stat(filepath.Join(opmDir, "config.cue"))
	require.NoError(t, err)
	assert.Equal(t, os.FileMode(0o600), fileInfo.Mode().Perm())
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
}

func TestConfigInit_RemovesLegacyPlatformFile(t *testing.T) {
	// A pre-0019 data-only ~/.opm/platform.cue is removed, whether init is
	// fresh or forced.
	tests := []struct {
		name       string
		withConfig bool
		args       []string
	}{
		{name: "force over existing config", withConfig: true, args: []string{"--force"}},
		{name: "fresh init with stale platform file", withConfig: false, args: nil},
	}
	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			tmpHome := setTempHome(t)
			opmDir := filepath.Join(tmpHome, ".opm")
			require.NoError(t, os.MkdirAll(opmDir, 0o700))
			if tt.withConfig {
				require.NoError(t, os.WriteFile(filepath.Join(opmDir, "config.cue"), []byte("// old config"), 0o600))
			}
			legacy := filepath.Join(opmDir, "platform.cue")
			require.NoError(t, os.WriteFile(legacy, []byte("name: \"cluster\"\ntype: \"kubernetes\"\n"), 0o600))

			cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
			cmd.SetArgs(tt.args)
			cmd.SetOut(&bytes.Buffer{})
			cmd.SetErr(&bytes.Buffer{})
			require.NoError(t, cmd.Execute())

			assert.NoFileExists(t, legacy, "legacy platform.cue must be removed")
			assert.NoDirExists(t, filepath.Join(opmDir, "platform"), "init writes no platform module")
		})
	}
}

// TestConfigInit_ForceLeavesPlatformDirectoryAlone covers "An existing
// platform directory is left alone": --force rewrites config.cue and never
// touches a ~/.opm/platform/ from an earlier release.
func TestConfigInit_ForceLeavesPlatformDirectoryAlone(t *testing.T) {
	tmpHome := setTempHome(t)
	opmDir := filepath.Join(tmpHome, ".opm")
	platformDir := filepath.Join(opmDir, "platform")
	require.NoError(t, os.MkdirAll(filepath.Join(platformDir, "cue.mod"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(opmDir, "config.cue"), []byte("// old config"), 0o600))
	modFile := filepath.Join(platformDir, "cue.mod", "module.cue")
	require.NoError(t, os.WriteFile(modFile, []byte("module: \"example.com/p@v0\"\n"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(platformDir, "platform.cue"), []byte("hand: \"edited\"\n"), 0o600))

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetArgs([]string{"--force"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	require.NoError(t, cmd.Execute())

	content, err := os.ReadFile(filepath.Join(platformDir, "platform.cue"))
	require.NoError(t, err)
	assert.Equal(t, "hand: \"edited\"\n", string(content))
	content, err = os.ReadFile(modFile)
	require.NoError(t, err)
	assert.Equal(t, "module: \"example.com/p@v0\"\n", string(content))
	entries, err := os.ReadDir(platformDir)
	require.NoError(t, err)
	assert.Len(t, entries, 2, "nothing is added to the leftover directory")
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
}
