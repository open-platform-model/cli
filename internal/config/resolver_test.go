// Package config provides configuration loading and management.
package config

import (
	"os"
	"path/filepath"
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

func TestResolveKubernetes_FlagOverridesAll(t *testing.T) {
	os.Setenv("OPM_KUBECONFIG", "/env/kubeconfig")
	os.Setenv("OPM_CONTEXT", "env-context")
	os.Setenv("OPM_NAMESPACE", "env-namespace")
	defer func() {
		os.Unsetenv("OPM_KUBECONFIG")
		os.Unsetenv("OPM_CONTEXT")
		os.Unsetenv("OPM_NAMESPACE")
	}()

	result, err := ResolveKubernetes(ResolveKubernetesOptions{
		KubeconfigFlag: "/flag/kubeconfig",
		ContextFlag:    "flag-context",
		NamespaceFlag:  "flag-namespace",
		ProviderFlag:   "flag-provider",
		Config: &GlobalConfig{
			Kubernetes: KubernetesConfig{
				Kubeconfig: "/config/kubeconfig",
				Context:    "config-context",
				Namespace:  "config-namespace",
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "/flag/kubeconfig", result.Kubeconfig.Value)
	assert.Equal(t, SourceFlag, result.Kubeconfig.Source)
	assert.Equal(t, "flag-context", result.Context.Value)
	assert.Equal(t, SourceFlag, result.Context.Source)
	assert.Equal(t, "flag-namespace", result.Namespace.Value)
	assert.Equal(t, SourceFlag, result.Namespace.Source)
	assert.Equal(t, "flag-provider", result.Provider.Value)
	assert.Equal(t, SourceFlag, result.Provider.Source)
}

func TestResolveKubernetes_EnvOverridesConfig(t *testing.T) {
	os.Setenv("OPM_NAMESPACE", "env-namespace")
	defer os.Unsetenv("OPM_NAMESPACE")

	result, err := ResolveKubernetes(ResolveKubernetesOptions{
		Config: &GlobalConfig{
			Kubernetes: KubernetesConfig{
				Namespace: "config-namespace",
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "env-namespace", result.Namespace.Value)
	assert.Equal(t, SourceEnv, result.Namespace.Source)
	assert.Equal(t, "config-namespace", result.Namespace.Shadowed[SourceConfig])
}

func TestResolveKubernetes_ConfigOverridesDefault(t *testing.T) {
	result, err := ResolveKubernetes(ResolveKubernetesOptions{
		Config: &GlobalConfig{
			Kubernetes: KubernetesConfig{
				Kubeconfig: "/custom/kubeconfig",
				Namespace:  "staging",
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "/custom/kubeconfig", result.Kubeconfig.Value)
	assert.Equal(t, SourceConfig, result.Kubeconfig.Source)
	assert.Equal(t, "staging", result.Namespace.Value)
	assert.Equal(t, SourceConfig, result.Namespace.Source)
}

func TestResolveKubernetes_DefaultsUsedWhenNothingSet(t *testing.T) {
	os.Unsetenv("OPM_KUBECONFIG")
	os.Unsetenv("OPM_CONTEXT")
	os.Unsetenv("OPM_NAMESPACE")

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	expectedKubeconfig := filepath.Join(homeDir, ".kube", "config")

	result, err := ResolveKubernetes(ResolveKubernetesOptions{})
	require.NoError(t, err)

	assert.Equal(t, expectedKubeconfig, result.Kubeconfig.Value)
	assert.Equal(t, SourceDefault, result.Kubeconfig.Source)
	assert.Equal(t, "", result.Context.Value) // no default for context
	assert.Equal(t, "default", result.Namespace.Value)
	assert.Equal(t, SourceDefault, result.Namespace.Source)
}

func TestResolveKubernetes_ProviderAutoResolve_NoProviders(t *testing.T) {
	result, err := ResolveKubernetes(ResolveKubernetesOptions{
		Config: &GlobalConfig{},
	})
	require.NoError(t, err)

	assert.Equal(t, "", result.Provider.Value)
	assert.Equal(t, Source(""), result.Provider.Source)
}

func TestResolveKubernetes_ProviderFlagOverridesAutoResolve(t *testing.T) {
	result, err := ResolveKubernetes(ResolveKubernetesOptions{
		ProviderFlag: "nomad",
		Config:       &GlobalConfig{},
	})
	require.NoError(t, err)

	assert.Equal(t, "nomad", result.Provider.Value)
	assert.Equal(t, SourceFlag, result.Provider.Source)
}

func TestResolveKubernetes_AllFlags(t *testing.T) {
	os.Setenv("OPM_KUBECONFIG", "/env/kubeconfig")
	os.Setenv("OPM_CONTEXT", "env-context")
	os.Setenv("OPM_NAMESPACE", "env-namespace")
	defer func() {
		os.Unsetenv("OPM_KUBECONFIG")
		os.Unsetenv("OPM_CONTEXT")
		os.Unsetenv("OPM_NAMESPACE")
	}()

	result, err := ResolveKubernetes(ResolveKubernetesOptions{
		KubeconfigFlag: "/flag/kubeconfig",
		ContextFlag:    "flag-context",
		NamespaceFlag:  "flag-namespace",
		ProviderFlag:   "flag-provider",
		Config: &GlobalConfig{
			Kubernetes: KubernetesConfig{
				Kubeconfig: "/config/kubeconfig",
				Context:    "config-context",
				Namespace:  "config-namespace",
			},
		},
	})
	require.NoError(t, err)

	assert.Equal(t, "/flag/kubeconfig", result.Kubeconfig.Value)
	assert.Equal(t, SourceFlag, result.Kubeconfig.Source)
	assert.Equal(t, "flag-context", result.Context.Value)
	assert.Equal(t, SourceFlag, result.Context.Source)
	assert.Equal(t, "flag-namespace", result.Namespace.Value)
	assert.Equal(t, SourceFlag, result.Namespace.Source)
	assert.Equal(t, "flag-provider", result.Provider.Value)
	assert.Equal(t, SourceFlag, result.Provider.Source)
}

func TestResolveKubernetes_Defaults(t *testing.T) {
	os.Unsetenv("OPM_KUBECONFIG")
	os.Unsetenv("OPM_CONTEXT")
	os.Unsetenv("OPM_NAMESPACE")

	homeDir, err := os.UserHomeDir()
	require.NoError(t, err)
	expectedKubeconfig := filepath.Join(homeDir, ".kube", "config")

	result, err := ResolveKubernetes(ResolveKubernetesOptions{})
	require.NoError(t, err)

	assert.Equal(t, expectedKubeconfig, result.Kubeconfig.Value)
	assert.Equal(t, SourceDefault, result.Kubeconfig.Source)
	assert.Equal(t, "", result.Context.Value)
	assert.Equal(t, "default", result.Namespace.Value)
	assert.Equal(t, SourceDefault, result.Namespace.Source)
}

func TestResolveKubernetes_ProviderAutoResolve(t *testing.T) {
	// Provider auto-resolve requires exactly one provider in config.Providers.
	// We can't easily add a cue.Value in a unit test, so we verify
	// the nil/empty case: no providers = no auto-resolve.
	result, err := ResolveKubernetes(ResolveKubernetesOptions{
		Config: &GlobalConfig{Providers: nil},
	})
	require.NoError(t, err)

	assert.Equal(t, "", result.Provider.Value)
}
