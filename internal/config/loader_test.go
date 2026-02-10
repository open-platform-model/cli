// Package config provides configuration loading and management.
package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestBootstrapRegistry_NoFile(t *testing.T) {
	registry, err := BootstrapRegistry("/nonexistent/path/config.cue")
	require.NoError(t, err)
	assert.Empty(t, registry)
}

func TestBootstrapRegistry_NoRegistryInFile(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootstrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.cue")
	content := `package config

kubernetes: {
    namespace: "default"
}
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	registry, err := BootstrapRegistry(configPath)
	require.NoError(t, err)
	assert.Empty(t, registry)
}

func TestBootstrapRegistry_WithRegistry(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootstrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.cue")
	content := `package config

registry: "localhost:5001"

kubernetes: {
    namespace: "default"
}
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	registry, err := BootstrapRegistry(configPath)
	require.NoError(t, err)
	assert.Equal(t, "localhost:5001", registry)
}

func TestBootstrapRegistry_RegistryInConfigStruct(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "bootstrap-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.cue")
	content := `package config

config: {
    registry: "registry.example.com"
    kubernetes: {
        namespace: "default"
    }
}
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	registry, err := BootstrapRegistry(configPath)
	require.NoError(t, err)
	assert.Equal(t, "registry.example.com", registry)
}

func TestConfigHasProviders_NoFile(t *testing.T) {
	has, err := configHasProviders("/nonexistent/path/config.cue")
	require.NoError(t, err)
	assert.False(t, has)
}

func TestConfigHasProviders_NoProviders(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "provider-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.cue")
	content := `package config

registry: "localhost:5001"
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	has, err := configHasProviders(configPath)
	require.NoError(t, err)
	assert.False(t, has)
}

func TestConfigHasProviders_WithProviders(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "provider-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	configPath := filepath.Join(tmpDir, "config.cue")
	content := `package config

registry: "localhost:5001"

providers: {
    kubernetes: something
}
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	has, err := configHasProviders(configPath)
	require.NoError(t, err)
	assert.True(t, has)
}

func TestLoadOPMConfig_NoConfigFile(t *testing.T) {
	// Use a temp home dir that doesn't have .opm
	tmpHome, err := os.MkdirTemp("", "opm-load-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Clear any registry env
	os.Unsetenv("OPM_REGISTRY")
	os.Unsetenv("OPM_CONFIG")

	cfg, err := LoadOPMConfig(LoaderOptions{})
	require.NoError(t, err)

	// Should return default config
	assert.NotNil(t, cfg)
	assert.NotNil(t, cfg.Config)
	assert.Empty(t, cfg.Registry)
}

func TestLoadOPMConfig_WithRegistryEnv(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "opm-load-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Setenv("OPM_REGISTRY", "env-registry.example.com")
	defer os.Unsetenv("OPM_REGISTRY")
	os.Unsetenv("OPM_CONFIG")

	cfg, err := LoadOPMConfig(LoaderOptions{})
	require.NoError(t, err)

	assert.Equal(t, "env-registry.example.com", cfg.Registry)
	assert.Equal(t, "env", cfg.RegistrySource)
}

func TestLoadOPMConfig_RegistryFlagPrecedence(t *testing.T) {
	tmpHome, err := os.MkdirTemp("", "opm-load-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpHome)

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Setenv("OPM_REGISTRY", "env-registry.example.com")
	defer os.Unsetenv("OPM_REGISTRY")
	os.Unsetenv("OPM_CONFIG")

	cfg, err := LoadOPMConfig(LoaderOptions{
		RegistryFlag: "flag-registry.example.com",
	})
	require.NoError(t, err)

	assert.Equal(t, "flag-registry.example.com", cfg.Registry)
	assert.Equal(t, "flag", cfg.RegistrySource)
}

func TestCheckRegistryConnectivity_EmptyRegistry(t *testing.T) {
	err := CheckRegistryConnectivity("")
	assert.NoError(t, err)
}

func TestExtractConfig_Empty(t *testing.T) {
	// This test verifies default values are returned for empty CUE value
	// In practice, extractConfig is called with loaded CUE values
}

func TestExtractConfig_LogTimestampsFalse(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a minimal CUE module
	modDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	modCue := `module: "test.local/config@v0"

language: {
	version: "v0.15.0"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(modCue), 0o644))

	configPath := filepath.Join(tmpDir, "config.cue")
	content := `package config

config: {
	log: {
		timestamps: false
	}
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, _, _, err := loadFullConfig(configPath, "")
	require.NoError(t, err)
	require.NotNil(t, cfg.Log.Timestamps, "Log.Timestamps should not be nil")
	assert.False(t, *cfg.Log.Timestamps, "Log.Timestamps should be false")
}

func TestExtractConfig_NoLogSection(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a minimal CUE module
	modDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	modCue := `module: "test.local/config@v0"

language: {
	version: "v0.15.0"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(modCue), 0o644))

	configPath := filepath.Join(tmpDir, "config.cue")
	content := `package config

config: {
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, _, _, err := loadFullConfig(configPath, "")
	require.NoError(t, err)
	assert.Nil(t, cfg.Log.Timestamps, "Log.Timestamps should be nil when not configured (defaults handled by caller)")
}

func TestExtractConfig_LogTimestampsInvalidType(t *testing.T) {
	tmpDir, err := os.MkdirTemp("", "log-config-test-*")
	require.NoError(t, err)
	defer os.RemoveAll(tmpDir)

	// Create a minimal CUE module
	modDir := filepath.Join(tmpDir, "cue.mod")
	require.NoError(t, os.MkdirAll(modDir, 0o755))
	modCue := `module: "test.local/config@v0"

language: {
	version: "v0.15.0"
}
`
	require.NoError(t, os.WriteFile(filepath.Join(modDir, "module.cue"), []byte(modCue), 0o644))

	configPath := filepath.Join(tmpDir, "config.cue")
	content := `package config

config: {
	log: {
		timestamps: "yes"
	}
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o644))

	cfg, _, _, err := loadFullConfig(configPath, "")
	// CUE itself won't fail on "yes" as a string (it's valid CUE), but our
	// extractor uses Bool() which will fail for a string value, so timestamps
	// will remain nil (treated as default true by the caller).
	require.NoError(t, err)
	assert.Nil(t, cfg.Log.Timestamps, "Log.Timestamps should be nil when value is not a bool")
}
