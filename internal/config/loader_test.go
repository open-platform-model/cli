// Package config provides configuration loading and management.
package config

import (
	"os"
	"path/filepath"
	"testing"

	"cuelang.org/go/cue/cuecontext"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// writeConfig writes content as config.cue in a fresh temp dir and returns
// its path.
func writeConfig(t *testing.T, content string) string {
	t.Helper()
	dir := t.TempDir()
	configPath := filepath.Join(dir, "config.cue")
	require.NoError(t, os.WriteFile(configPath, []byte(content), 0o600))
	return configPath
}

func TestLoad_NoConfigFile(t *testing.T) {
	// Use a temp home dir that doesn't have .opm
	tmpHome := t.TempDir()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	// Clear any registry env
	os.Unsetenv("OPM_REGISTRY")
	os.Unsetenv("OPM_CONFIG")

	var cfg GlobalConfig
	err := Load(&cfg, LoaderOptions{})
	require.NoError(t, err)

	// Should populate with defaults (empty registry when no config or env)
	assert.Empty(t, cfg.Registry)
	assert.Equal(t, "~/.kube/config", cfg.Kubernetes.Kubeconfig)
	assert.Equal(t, "default", cfg.Kubernetes.Namespace)
	assert.Equal(t, APIWarningsWarn, cfg.Log.Kubernetes.APIWarnings)
}

func TestLoad_WithRegistryEnv(t *testing.T) {
	tmpHome := t.TempDir()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Setenv("OPM_REGISTRY", "env-registry.example.com")
	defer os.Unsetenv("OPM_REGISTRY")
	os.Unsetenv("OPM_CONFIG")

	var cfg GlobalConfig
	err := Load(&cfg, LoaderOptions{})
	require.NoError(t, err)

	assert.Equal(t, "env-registry.example.com", cfg.Registry)
}

func TestLoad_RegistryFlagPrecedence(t *testing.T) {
	tmpHome := t.TempDir()

	origHome := os.Getenv("HOME")
	os.Setenv("HOME", tmpHome)
	defer os.Setenv("HOME", origHome)

	os.Setenv("OPM_REGISTRY", "env-registry.example.com")
	defer os.Unsetenv("OPM_REGISTRY")
	os.Unsetenv("OPM_CONFIG")

	var cfg GlobalConfig
	err := Load(&cfg, LoaderOptions{
		RegistryFlag: "flag-registry.example.com",
	})
	require.NoError(t, err)

	// Flag takes precedence over env
	assert.Equal(t, "flag-registry.example.com", cfg.Registry)
}

func TestLoad_RegistryFromConfigFile(t *testing.T) {
	// Single-pass: the registry comes out of the parsed file, no bootstrap
	// pre-pass. The package clause must be accepted (existing user files
	// carry one).
	configPath := writeConfig(t, `package config

config: {
	registry: "registry.example.com"
	kubernetes: {
		namespace: "default"
	}
}
`)

	os.Unsetenv("OPM_REGISTRY")

	var cfg GlobalConfig
	err := Load(&cfg, LoaderOptions{ConfigFlag: configPath})
	require.NoError(t, err)

	assert.Equal(t, "registry.example.com", cfg.Registry)
}

func TestLoadConfigFile_LogTimestampsFalse(t *testing.T) {
	// No cue.mod needed: the config file is import-free data parsed in a
	// single pass.
	configPath := writeConfig(t, `package config

config: {
	log: {
		timestamps: false
	}
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`)

	var cfg GlobalConfig
	_, err := loadConfigFile(&cfg, configPath)
	require.NoError(t, err)
	require.NotNil(t, cfg.Log.Timestamps, "Log.Timestamps should not be nil")
	assert.False(t, *cfg.Log.Timestamps, "Log.Timestamps should be false")
}

func TestLoadConfigFile_NoLogSection(t *testing.T) {
	configPath := writeConfig(t, `package config

config: {
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`)

	var cfg GlobalConfig
	_, err := loadConfigFile(&cfg, configPath)
	require.NoError(t, err)
	assert.Nil(t, cfg.Log.Timestamps, "Log.Timestamps should be nil when not configured")
}

func TestLoadConfigFile_LogTimestampsInvalidType(t *testing.T) {
	configPath := writeConfig(t, `package config

config: {
	log: {
		timestamps: "yes"
	}
	kubernetes: {
		kubeconfig: "~/.kube/config"
		namespace: "default"
	}
}
`)

	var cfg GlobalConfig
	_, err := loadConfigFile(&cfg, configPath)
	// Schema validation should catch the type error (string instead of bool)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation")
}

func TestLoadConfigFile_ReturnsRegistry(t *testing.T) {
	configPath := writeConfig(t, `package config

config: {
	registry: "localhost:5001"
}
`)

	var cfg GlobalConfig
	registry, err := loadConfigFile(&cfg, configPath)
	require.NoError(t, err)
	assert.Equal(t, "localhost:5001", registry)
}

func TestLoadConfigFile_DefaultTemplateIsValid(t *testing.T) {
	// The template written by `opm config init` must load cleanly.
	configPath := writeConfig(t, DefaultConfigTemplate)

	var cfg GlobalConfig
	registry, err := loadConfigFile(&cfg, configPath)
	require.NoError(t, err)
	assert.Equal(t, DefaultRegistry, registry)
	assert.Equal(t, "default", cfg.Kubernetes.Namespace)
}

func TestExtractConfig_LogKubernetesAPIWarnings(t *testing.T) {
	tests := []struct {
		name     string
		cueInput string
		want     string
	}{
		{
			name: "default warn when not specified",
			cueInput: `
package config
config: {
	registry: "test.example.com"
}
`,
			want: "warn",
		},
		{
			name: "explicit warn value",
			cueInput: `
package config
config: {
	registry: "test.example.com"
	log: {
		kubernetes: {
			apiWarnings: "warn"
		}
	}
}
`,
			want: "warn",
		},
		{
			name: "debug value",
			cueInput: `
package config
config: {
	registry: "test.example.com"
	log: {
		kubernetes: {
			apiWarnings: "debug"
		}
	}
}
`,
			want: "debug",
		},
		{
			name: "suppress value",
			cueInput: `
package config
config: {
	registry: "test.example.com"
	log: {
		kubernetes: {
			apiWarnings: "suppress"
		}
	}
}
`,
			want: "suppress",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			ctx := cuecontext.New()
			value := ctx.CompileString(tt.cueInput)
			assert.NoError(t, value.Err())

			var cfg GlobalConfig
			extractConfigInto(&cfg, value)
			// Apply the same default as loadConfigFile does
			if cfg.Log.Kubernetes.APIWarnings == "" {
				cfg.Log.Kubernetes.APIWarnings = "warn"
			}
			assert.Equal(t, tt.want, cfg.Log.Kubernetes.APIWarnings)
		})
	}
}

func TestValidateConfigSchema_ValidMinimal(t *testing.T) {
	ctx := cuecontext.New()
	configCUE := `package config

config: {
	kubernetes: {
		namespace: "default"
	}
}
`
	value := ctx.CompileString(configCUE)
	require.NoError(t, value.Err())

	err := validateConfigSchema(ctx, value, "test-config.cue")
	assert.NoError(t, err)
}

func TestValidateConfigSchema_ValidFull(t *testing.T) {
	ctx := cuecontext.New()
	configCUE := `package config

config: {
	registry: "opmodel.dev=localhost:5000+insecure,registry.cue.works"

	kubernetes: {
		kubeconfig: "~/.kube/config"
		context: "prod"
		namespace: "my-app"
	}

	log: {
		timestamps: true
		kubernetes: {
			apiWarnings: "debug"
		}
	}
}
`
	value := ctx.CompileString(configCUE)
	require.NoError(t, value.Err())

	err := validateConfigSchema(ctx, value, "test-config.cue")
	assert.NoError(t, err)
}

func TestValidateConfigSchema_ProvidersRejected(t *testing.T) {
	// The providers field was removed by enhancement 0006 D39. A pre-D39
	// config must fail with a migration hint naming the removed field.
	ctx := cuecontext.New()
	configCUE := `package config

config: {
	registry: "localhost:5000"
	providers: {
		kubernetes: {}
	}
	kubernetes: {
		namespace: "default"
	}
}
`
	value := ctx.CompileString(configCUE)
	require.NoError(t, value.Err())

	err := validateConfigSchema(ctx, value, "test-config.cue")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "providers")
	assert.Contains(t, err.Error(), "opm config init")
}

func TestValidateConfigSchema_CacheDirRejected(t *testing.T) {
	ctx := cuecontext.New()
	configCUE := `package config

config: {
	cacheDir: "/tmp/cache"
	kubernetes: {
		namespace: "default"
	}
}
`
	value := ctx.CompileString(configCUE)
	require.NoError(t, value.Err())

	err := validateConfigSchema(ctx, value, "test-config.cue")
	require.Error(t, err)
	assert.Contains(t, err.Error(), "cacheDir")
}

func TestValidateConfigSchema_UnknownField(t *testing.T) {
	ctx := cuecontext.New()
	configCUE := `package config

config: {
	registry: "localhost:5000"
	unknownField: "this should fail"
	kubernetes: {
		namespace: "default"
	}
}
`
	value := ctx.CompileString(configCUE)
	require.NoError(t, value.Err())

	err := validateConfigSchema(ctx, value, "test-config.cue")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation failed")
}

func TestValidateConfigSchema_InvalidNamespace(t *testing.T) {
	ctx := cuecontext.New()
	configCUE := `package config

config: {
	kubernetes: {
		namespace: "UPPERCASE-not-allowed"
	}
}
`
	value := ctx.CompileString(configCUE)
	require.NoError(t, value.Err())

	err := validateConfigSchema(ctx, value, "test-config.cue")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation failed")
}

func TestValidateConfigSchema_InvalidAPIWarnings(t *testing.T) {
	ctx := cuecontext.New()
	configCUE := `package config

config: {
	log: {
		kubernetes: {
			apiWarnings: "invalid-value"
		}
	}
}
`
	value := ctx.CompileString(configCUE)
	require.NoError(t, value.Err())

	err := validateConfigSchema(ctx, value, "test-config.cue")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation failed")
}

func TestValidateConfigSchema_InvalidTimestampsType(t *testing.T) {
	ctx := cuecontext.New()
	configCUE := `package config

config: {
	log: {
		timestamps: "should-be-bool"
	}
}
`
	value := ctx.CompileString(configCUE)
	require.NoError(t, value.Err())

	err := validateConfigSchema(ctx, value, "test-config.cue")
	assert.Error(t, err)
	assert.Contains(t, err.Error(), "schema validation failed")
}

func TestLoadConfigFile_ImportsRejected(t *testing.T) {
	// Data-only contract (0006 D39): even CUE stdlib imports are rejected,
	// mirroring the platform-file guard.
	configPath := writeConfig(t, `package config

import "strings"

config: {
	registry: strings.ToLower("LOCALHOST:5000")
}
`)

	var cfg GlobalConfig
	_, err := loadConfigFile(&cfg, configPath)
	require.Error(t, err)
	assert.Contains(t, err.Error(), "data-only")
}
