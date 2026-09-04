// Package config provides CLI command implementations for config operations.
package config

import (
	"bytes"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"cuelang.org/go/mod/modfile"
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

	// Check files were created: config.cue + the platform module, and NO
	// data-only platform.cue and NO cue.mod beside config.cue (0019 D5).
	opmDir := filepath.Join(tmpHome, ".opm")
	assert.DirExists(t, opmDir)
	assert.FileExists(t, filepath.Join(opmDir, "config.cue"))
	assert.FileExists(t, filepath.Join(opmDir, "platform", "cue.mod", "module.cue"))
	assert.FileExists(t, filepath.Join(opmDir, "platform", "platform.cue"))
	assert.NoFileExists(t, filepath.Join(opmDir, "platform.cue"))
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
	for _, dir := range []string{opmDir, filepath.Join(opmDir, "platform"), filepath.Join(opmDir, "platform", "cue.mod")} {
		dirInfo, err := os.Stat(dir)
		require.NoError(t, err)
		assert.Equal(t, os.FileMode(0o700), dirInfo.Mode().Perm(), dir)
	}

	// Check file permissions (0600)
	for _, name := range []string{"config.cue", "platform/cue.mod/module.cue", "platform/platform.cue"} {
		fileInfo, err := os.Stat(filepath.Join(opmDir, filepath.FromSlash(name)))
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

	// Check the platform module (0019 D5): cue.mod pins exactly one build
	// for core and each seeded catalog; platform.cue carries one #registry
	// entry per catalog embedding it by import, no version scalar, no
	// filter vocabulary, no retired kubernetes catalog.
	platformDir := filepath.Join(tmpHome, ".opm", "platform")
	modContent, err := os.ReadFile(filepath.Join(platformDir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	modFile, err := modfile.Parse(modContent, "cue.mod/module.cue")
	require.NoError(t, err)
	assert.Equal(t, "opmodel.dev/platforms/local@v0", modFile.Module)
	require.Len(t, modFile.Deps, 3)
	assert.Contains(t, modFile.Deps, "opmodel.dev/core@v2")
	assert.Contains(t, modFile.Deps, "opmodel.dev/catalogs/opm@v4")
	assert.Contains(t, modFile.Deps, "opmodel.dev/catalogs/k8s@v1")
	for path, dep := range modFile.Deps {
		assert.NotEmpty(t, dep.Version, path)
	}

	platformContent, err := os.ReadFile(filepath.Join(platformDir, "platform.cue"))
	require.NoError(t, err)
	platformStr := string(platformContent)
	assert.Contains(t, platformStr, "core.#Platform")
	assert.Contains(t, platformStr, `"opmodel.dev/catalogs/opm@v4": #catalog:`)
	assert.Contains(t, platformStr, `"opmodel.dev/catalogs/k8s@v1": #catalog:`)
	assert.Equal(t, 2, strings.Count(platformStr, "#catalog:"), "exactly two registry entries")
	assert.NotContains(t, platformStr, "version:")
	assert.NotContains(t, platformStr, "opmodel.dev/catalogs/kubernetes")
	assert.NotContains(t, platformStr, "filter")
}

func TestConfigInit_SeededPlatformOffersRawEscapeHatch(t *testing.T) {
	tmpHome := setTempHome(t)

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})

	require.NoError(t, cmd.Execute())

	// A module demanding a contract from the raw passthrough catalog must
	// already have a matching entry in the seeded default, with no user
	// edit to the platform module: the entry imports the k8s catalog and
	// its build is pinned in cue.mod.
	platformDir := filepath.Join(tmpHome, ".opm", "platform")
	platformContent, err := os.ReadFile(filepath.Join(platformDir, "platform.cue"))
	require.NoError(t, err)
	assert.Contains(t, string(platformContent), `"opmodel.dev/catalogs/k8s@v1": #catalog: k8s`,
		"seeded platform must already subscribe to the raw escape-hatch catalog")

	modContent, err := os.ReadFile(filepath.Join(platformDir, "cue.mod", "module.cue"))
	require.NoError(t, err)
	modFile, err := modfile.Parse(modContent, "cue.mod/module.cue")
	require.NoError(t, err)
	require.Contains(t, modFile.Deps, "opmodel.dev/catalogs/k8s@v1")
	assert.NotEmpty(t, modFile.Deps["opmodel.dev/catalogs/k8s@v1"].Version)
}

func TestConfigInit_RemovesLegacyPlatformFile(t *testing.T) {
	// A pre-0019 data-only ~/.opm/platform.cue is removed when the module
	// is written, whether init is fresh or forced (spec: legacy file is
	// migrated).
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
			assert.FileExists(t, filepath.Join(opmDir, "platform", "platform.cue"))
			assert.FileExists(t, filepath.Join(opmDir, "platform", "cue.mod", "module.cue"))
		})
	}
}

func TestConfigInit_ForceRewritesPlatformModule(t *testing.T) {
	// --force overwrites a hand-edited module with the seeded one.
	tmpHome := setTempHome(t)
	opmDir := filepath.Join(tmpHome, ".opm")
	platformDir := filepath.Join(opmDir, "platform")
	require.NoError(t, os.MkdirAll(filepath.Join(platformDir, "cue.mod"), 0o700))
	require.NoError(t, os.WriteFile(filepath.Join(opmDir, "config.cue"), []byte("// old config"), 0o600))
	require.NoError(t, os.WriteFile(filepath.Join(platformDir, "platform.cue"), []byte("bogus: true\n"), 0o600))

	cmd := NewConfigInitCmd(&opmconfig.GlobalConfig{})
	cmd.SetArgs([]string{"--force"})
	cmd.SetOut(&bytes.Buffer{})
	cmd.SetErr(&bytes.Buffer{})
	require.NoError(t, cmd.Execute())

	content, err := os.ReadFile(filepath.Join(platformDir, "platform.cue"))
	require.NoError(t, err)
	assert.Equal(t, opmconfig.DefaultPlatformCUE, string(content))
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
	assert.FileExists(t, filepath.Join(opmDir, "platform", "platform.cue"))
}
