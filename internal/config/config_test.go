// Package config provides configuration loading and management.
package config

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGlobalConfig_Fields(t *testing.T) {
	cfg := &GlobalConfig{
		Registry:   "registry.example.com",
		ConfigPath: "/home/user/.opm/config.cue",
		Kubernetes: KubernetesConfig{
			Kubeconfig: "/custom/kubeconfig",
			Context:    "my-cluster",
			Namespace:  "my-namespace",
		},
		Log: LogConfig{
			Kubernetes: LogKubernetesConfig{
				APIWarnings: "debug",
			},
		},
		Flags: GlobalFlags{
			Config:   "/custom/config.cue",
			Registry: "flag-registry.example.com",
			Verbose:  true,
		},
	}

	assert.Equal(t, "registry.example.com", cfg.Registry)
	assert.Equal(t, "/home/user/.opm/config.cue", cfg.ConfigPath)
	assert.Equal(t, "/custom/kubeconfig", cfg.Kubernetes.Kubeconfig)
	assert.Equal(t, "my-cluster", cfg.Kubernetes.Context)
	assert.Equal(t, "my-namespace", cfg.Kubernetes.Namespace)
	assert.Equal(t, "debug", cfg.Log.Kubernetes.APIWarnings)
	assert.Equal(t, "/custom/config.cue", cfg.Flags.Config)
	assert.Equal(t, "flag-registry.example.com", cfg.Flags.Registry)
	assert.True(t, cfg.Flags.Verbose)
}

func TestGlobalFlags_ZeroValue(t *testing.T) {
	var flags GlobalFlags

	assert.Empty(t, flags.Config)
	assert.Empty(t, flags.Registry)
	assert.False(t, flags.Verbose)
	assert.False(t, flags.Timestamps)
}

func TestKubernetesConfig_ZeroValue(t *testing.T) {
	var k8sCfg KubernetesConfig

	// Zero values should be empty strings
	assert.Empty(t, k8sCfg.Kubeconfig)
	assert.Empty(t, k8sCfg.Context)
	assert.Empty(t, k8sCfg.Namespace)
}
