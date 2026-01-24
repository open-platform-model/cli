package config

import (
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNewLoader(t *testing.T) {
	loader := NewLoader()
	assert.NotNil(t, loader)
	assert.NotNil(t, loader.v)
}

func TestLoaderLoad(t *testing.T) {
	t.Run("loads config from file", func(t *testing.T) {
		// Create temp config file
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")

		content := `
kubeconfig: /path/to/kubeconfig
context: production
namespace: my-namespace
registry: ghcr.io/myorg
cacheDir: /custom/cache
`
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0o644))

		loader := NewLoader()
		cfg, err := loader.Load(configFile)

		require.NoError(t, err)
		assert.Equal(t, "/path/to/kubeconfig", cfg.Kubeconfig)
		assert.Equal(t, "production", cfg.Context)
		assert.Equal(t, "my-namespace", cfg.Namespace)
		assert.Equal(t, "ghcr.io/myorg", cfg.Registry)
		assert.Equal(t, "/custom/cache", cfg.CacheDir)
	})

	t.Run("returns empty config for missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "nonexistent.yaml")

		loader := NewLoader()
		cfg, err := loader.Load(configFile)

		require.NoError(t, err)
		assert.Empty(t, cfg.Kubeconfig)
		assert.Empty(t, cfg.Namespace)
	})

	t.Run("loads from environment variables", func(t *testing.T) {
		// Set env vars
		t.Setenv("OPM_KUBECONFIG", "/env/kubeconfig")
		t.Setenv("OPM_NAMESPACE", "env-namespace")
		t.Setenv("OPM_REGISTRY", "env-registry")

		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "empty.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte(""), 0o644))

		loader := NewLoader()
		cfg, err := loader.Load(configFile)

		require.NoError(t, err)
		assert.Equal(t, "/env/kubeconfig", cfg.Kubeconfig)
		assert.Equal(t, "env-namespace", cfg.Namespace)
		assert.Equal(t, "env-registry", cfg.Registry)
	})

	t.Run("env vars override file values", func(t *testing.T) {
		// Set env var
		t.Setenv("OPM_NAMESPACE", "env-namespace")

		// Create temp config file with different value
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		content := `namespace: file-namespace`
		require.NoError(t, os.WriteFile(configFile, []byte(content), 0o644))

		loader := NewLoader()
		cfg, err := loader.Load(configFile)

		require.NoError(t, err)
		assert.Equal(t, "env-namespace", cfg.Namespace)
	})
}

func TestLoaderLoadWithDefaults(t *testing.T) {
	tmpDir := t.TempDir()
	configFile := filepath.Join(tmpDir, "empty.yaml")
	require.NoError(t, os.WriteFile(configFile, []byte(""), 0o644))

	loader := NewLoader()
	cfg, err := loader.LoadWithDefaults(configFile)

	require.NoError(t, err)
	assert.Equal(t, "~/.kube/config", cfg.Kubeconfig)
	assert.Equal(t, "default", cfg.Namespace)
	assert.Equal(t, "~/.opm/cache", cfg.CacheDir)
}

func TestConfigFileExists(t *testing.T) {
	t.Run("returns true for existing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "config.yaml")
		require.NoError(t, os.WriteFile(configFile, []byte(""), 0o644))

		exists, err := ConfigFileExists(configFile)
		require.NoError(t, err)
		assert.True(t, exists)
	})

	t.Run("returns false for missing file", func(t *testing.T) {
		tmpDir := t.TempDir()
		configFile := filepath.Join(tmpDir, "nonexistent.yaml")

		exists, err := ConfigFileExists(configFile)
		require.NoError(t, err)
		assert.False(t, exists)
	})
}

func TestExpandPath(t *testing.T) {
	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)

	tests := []struct {
		name     string
		input    string
		expected string
	}{
		{
			name:     "empty path",
			input:    "",
			expected: "",
		},
		{
			name:     "absolute path",
			input:    "/absolute/path",
			expected: "/absolute/path",
		},
		{
			name:     "relative path",
			input:    "relative/path",
			expected: "relative/path",
		},
		{
			name:     "home directory only",
			input:    "~",
			expected: homeDir,
		},
		{
			name:     "path with tilde",
			input:    "~/some/path",
			expected: filepath.Join(homeDir, "some/path"),
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			result, err := ExpandPath(tt.input)
			require.NoError(t, err)
			assert.Equal(t, tt.expected, result)
		})
	}
}

func TestConfigMerge(t *testing.T) {
	t.Run("merge overwrites non-empty values", func(t *testing.T) {
		base := &Config{
			Kubeconfig: "base-kubeconfig",
			Namespace:  "base-namespace",
		}
		other := &Config{
			Kubeconfig: "other-kubeconfig",
			Context:    "other-context",
		}

		base.Merge(other)

		assert.Equal(t, "other-kubeconfig", base.Kubeconfig)
		assert.Equal(t, "other-context", base.Context)
		assert.Equal(t, "base-namespace", base.Namespace)
	})

	t.Run("merge with nil does nothing", func(t *testing.T) {
		base := &Config{
			Kubeconfig: "base-kubeconfig",
		}

		base.Merge(nil)

		assert.Equal(t, "base-kubeconfig", base.Kubeconfig)
	})
}

func TestConfigIsEmpty(t *testing.T) {
	t.Run("empty config", func(t *testing.T) {
		cfg := &Config{}
		assert.True(t, cfg.IsEmpty())
	})

	t.Run("non-empty config", func(t *testing.T) {
		cfg := &Config{Namespace: "test"}
		assert.False(t, cfg.IsEmpty())
	})
}

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	assert.Equal(t, "~/.kube/config", cfg.Kubeconfig)
	assert.Equal(t, "default", cfg.Namespace)
	assert.Equal(t, "~/.opm/cache", cfg.CacheDir)
	assert.Empty(t, cfg.Context)
	assert.Empty(t, cfg.Registry)
}
