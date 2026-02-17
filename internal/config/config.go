// Package config provides configuration loading and management.
package config

import (
	"cuelang.org/go/cue"
)

// KubernetesConfig contains Kubernetes-specific settings.
type KubernetesConfig struct {
	// Kubeconfig is the path to the kubeconfig file.
	// Env: OPM_KUBECONFIG, Default: ~/.kube/config
	Kubeconfig string `json:"kubeconfig,omitempty"`

	// Context is the Kubernetes context to use.
	// Env: OPM_CONTEXT, Default: current-context from kubeconfig
	Context string `json:"context,omitempty"`

	// Namespace is the default namespace for operations.
	// Env: OPM_NAMESPACE, Default: "default"
	Namespace string `json:"namespace,omitempty"`
}

// LogKubernetesConfig contains Kubernetes-related logging settings.
type LogKubernetesConfig struct {
	// APIWarnings controls how Kubernetes API deprecation warnings are displayed.
	// Valid values: "warn" (default), "debug", "suppress"
	// - "warn": Show as WARN level in log output
	// - "debug": Only show with --verbose flag
	// - "suppress": Drop entirely
	APIWarnings string `json:"apiWarnings,omitempty"`
}

// LogConfig contains logging-related settings.
type LogConfig struct {
	// Timestamps controls whether timestamps are shown in log output.
	// Default: true. Override with --timestamps flag.
	Timestamps *bool `json:"timestamps,omitempty"`

	// Kubernetes contains Kubernetes-related logging settings.
	// Non-optional because APIWarnings has a default value.
	Kubernetes LogKubernetesConfig `json:"kubernetes"`
}

// Config represents the OPM CLI configuration.
// Loaded from ~/.opm/config.cue, validated against embedded CUE schema.
type Config struct {
	// Registry is the default registry for all CUE module resolution.
	// When set, all CUE imports resolve from this registry (passed to CUE via CUE_REGISTRY).
	// Env: OPM_REGISTRY
	Registry string `json:"registry,omitempty"`

	// Kubernetes contains Kubernetes-specific settings.
	Kubernetes KubernetesConfig `json:"kubernetes,omitempty"`

	// Log contains logging-related settings.
	Log LogConfig `json:"log,omitempty"`
}

// DefaultConfig returns a Config with all default values populated.
// Used by `opm config init` to generate initial config file.
func DefaultConfig() *Config {
	return &Config{
		Kubernetes: KubernetesConfig{
			Kubeconfig: "~/.kube/config",
			Namespace:  "default",
		},
	}
}

// OPMConfig represents the fully-loaded CUE configuration.
// This includes provider definitions loaded from imports.
type OPMConfig struct {
	// Config contains the basic configuration fields.
	Config *Config

	// Registry is the resolved registry URL after applying precedence.
	Registry string

	// RegistrySource indicates where the registry URL came from.
	RegistrySource string // "flag", "env", "config"

	// Providers maps provider names to their loaded CUE definitions.
	// Key: provider alias (e.g., "kubernetes")
	// Value: loaded CUE value referencing the provider's #Provider definition
	Providers map[string]cue.Value

	// CueContext is the CUE context used to load providers.
	// Shared with module loader to ensure all values are from the same runtime.
	CueContext *cue.Context
}
