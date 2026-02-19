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

// GlobalFlags holds raw CLI flag values set by the user.
// These are populated by the root command before calling config.Load.
type GlobalFlags struct {
	// Config is the --config flag value (path to config file).
	Config string
	// Registry is the --registry flag value.
	Registry string
	// Verbose is the --verbose flag value.
	Verbose bool
	// Timestamps is the --timestamps flag value.
	Timestamps bool
}

// GlobalConfig is the single consolidated runtime configuration type.
// It is populated by config.Load and holds all configuration the CLI needs.
type GlobalConfig struct {
	// Kubernetes contains resolved Kubernetes-specific settings from config file.
	Kubernetes KubernetesConfig

	// Log contains logging-related settings from config file.
	Log LogConfig

	// Registry is the resolved registry URL after applying precedence.
	// Set by config.Load using flag > env > config precedence.
	Registry string

	// ConfigPath is the resolved config file path.
	// Set by config.Load.
	ConfigPath string

	// Providers maps provider names to their loaded CUE definitions.
	// Key: provider alias (e.g., "kubernetes")
	// Value: loaded CUE value referencing the provider's #Provider definition
	Providers map[string]cue.Value

	// CueContext is the CUE context used to load providers.
	// Shared with module loader to ensure all values are from the same runtime.
	CueContext *cue.Context

	// Flags holds the raw CLI flag values as set by the user.
	Flags GlobalFlags
}
