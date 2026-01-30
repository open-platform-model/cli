// Package config provides configuration loading and management.
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	require.NotNil(t, cfg)

	// Check Kubernetes defaults
	assert.Equal(t, "~/.kube/config", cfg.Kubernetes.Kubeconfig)
	assert.Equal(t, "default", cfg.Kubernetes.Namespace)
	assert.Empty(t, cfg.Kubernetes.Context) // No default context

	// Check cache dir default
	assert.Equal(t, "~/.opm/cache", cfg.CacheDir)

	// Registry should be empty by default
	assert.Empty(t, cfg.Registry)
}

func TestConfig_Fields(t *testing.T) {
	cfg := &Config{
		Registry: "registry.example.com",
		Kubernetes: KubernetesConfig{
			Kubeconfig: "/custom/kubeconfig",
			Context:    "my-cluster",
			Namespace:  "my-namespace",
		},
		CacheDir: "/custom/cache",
	}

	assert.Equal(t, "registry.example.com", cfg.Registry)
	assert.Equal(t, "/custom/kubeconfig", cfg.Kubernetes.Kubeconfig)
	assert.Equal(t, "my-cluster", cfg.Kubernetes.Context)
	assert.Equal(t, "my-namespace", cfg.Kubernetes.Namespace)
	assert.Equal(t, "/custom/cache", cfg.CacheDir)
}

func TestResolvedValue(t *testing.T) {
	rv := ResolvedValue{
		Key:    "registry",
		Value:  "registry.example.com",
		Source: "env",
		Shadowed: map[string]any{
			"config":  "config-registry.example.com",
			"default": "",
		},
	}

	assert.Equal(t, "registry", rv.Key)
	assert.Equal(t, "registry.example.com", rv.Value)
	assert.Equal(t, "env", rv.Source)
	assert.Len(t, rv.Shadowed, 2)
	assert.Equal(t, "config-registry.example.com", rv.Shadowed["config"])
}

func TestOPMConfig(t *testing.T) {
	cfg := DefaultConfig()
	opmCfg := &OPMConfig{
		Config:         cfg,
		Registry:       "resolved-registry.example.com",
		RegistrySource: "env",
		Providers:      nil, // Providers are CUE values, tested separately
	}

	assert.NotNil(t, opmCfg.Config)
	assert.Equal(t, "resolved-registry.example.com", opmCfg.Registry)
	assert.Equal(t, "env", opmCfg.RegistrySource)
}

func TestKubernetesConfig_ZeroValue(t *testing.T) {
	var k8sCfg KubernetesConfig

	// Zero values should be empty strings
	assert.Empty(t, k8sCfg.Kubeconfig)
	assert.Empty(t, k8sCfg.Context)
	assert.Empty(t, k8sCfg.Namespace)
}
