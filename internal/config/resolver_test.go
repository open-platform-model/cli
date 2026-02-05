// Package config provides configuration loading and management.
package config

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestResolveRegistry_FlagPrecedence(t *testing.T) {
	// Set up env var
	os.Setenv("OPM_REGISTRY", "env-registry.example.com")
	defer os.Unsetenv("OPM_REGISTRY")

	result := ResolveRegistry(ResolveRegistryOptions{
		FlagValue:   "flag-registry.example.com",
		ConfigValue: "config-registry.example.com",
	})

	assert.Equal(t, "flag-registry.example.com", result.Registry)
	assert.Equal(t, SourceFlag, result.Source)
	assert.Equal(t, "env-registry.example.com", result.Shadowed[SourceEnv])
	assert.Equal(t, "config-registry.example.com", result.Shadowed[SourceConfig])
}

func TestResolveRegistry_EnvPrecedence(t *testing.T) {
	os.Setenv("OPM_REGISTRY", "env-registry.example.com")
	defer os.Unsetenv("OPM_REGISTRY")

	result := ResolveRegistry(ResolveRegistryOptions{
		FlagValue:   "", // No flag
		ConfigValue: "config-registry.example.com",
	})

	assert.Equal(t, "env-registry.example.com", result.Registry)
	assert.Equal(t, SourceEnv, result.Source)
	assert.Equal(t, "config-registry.example.com", result.Shadowed[SourceConfig])
	assert.NotContains(t, result.Shadowed, SourceFlag)
}

func TestResolveRegistry_ConfigFallback(t *testing.T) {
	// Ensure env is not set
	os.Unsetenv("OPM_REGISTRY")

	result := ResolveRegistry(ResolveRegistryOptions{
		FlagValue:   "",
		ConfigValue: "config-registry.example.com",
	})

	assert.Equal(t, "config-registry.example.com", result.Registry)
	assert.Equal(t, SourceConfig, result.Source)
	assert.Empty(t, result.Shadowed)
}

func TestResolveRegistry_NoRegistry(t *testing.T) {
	os.Unsetenv("OPM_REGISTRY")

	result := ResolveRegistry(ResolveRegistryOptions{
		FlagValue:   "",
		ConfigValue: "",
	})

	assert.Empty(t, result.Registry)
	assert.Empty(t, result.Source)
}

func TestResolveConfigPath_FlagPrecedence(t *testing.T) {
	os.Setenv("OPM_CONFIG", "/env/path/config.cue")
	defer os.Unsetenv("OPM_CONFIG")

	result, err := ResolveConfigPath(ResolveConfigPathOptions{
		FlagValue: "/flag/path/config.cue",
	})
	require.NoError(t, err)

	assert.Equal(t, "/flag/path/config.cue", result.ConfigPath)
	assert.Equal(t, SourceFlag, result.Source)
	assert.Equal(t, "/env/path/config.cue", result.Shadowed[SourceEnv])
	assert.NotEmpty(t, result.Shadowed[SourceDefault])
}

func TestResolveConfigPath_EnvPrecedence(t *testing.T) {
	os.Setenv("OPM_CONFIG", "/env/path/config.cue")
	defer os.Unsetenv("OPM_CONFIG")

	result, err := ResolveConfigPath(ResolveConfigPathOptions{
		FlagValue: "", // No flag
	})
	require.NoError(t, err)

	assert.Equal(t, "/env/path/config.cue", result.ConfigPath)
	assert.Equal(t, SourceEnv, result.Source)
	assert.NotEmpty(t, result.Shadowed[SourceDefault])
}

func TestResolveConfigPath_Default(t *testing.T) {
	os.Unsetenv("OPM_CONFIG")

	result, err := ResolveConfigPath(ResolveConfigPathOptions{
		FlagValue: "",
	})
	require.NoError(t, err)

	assert.Contains(t, result.ConfigPath, ".opm")
	assert.Contains(t, result.ConfigPath, "config.cue")
	assert.Equal(t, SourceDefault, result.Source)
	assert.Empty(t, result.Shadowed)
}

func TestSource_String(t *testing.T) {
	assert.Equal(t, "flag", string(SourceFlag))
	assert.Equal(t, "env", string(SourceEnv))
	assert.Equal(t, "config", string(SourceConfig))
	assert.Equal(t, "default", string(SourceDefault))
}
