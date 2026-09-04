// Package config provides configuration loading and management.
package config

// APIWarningsWarn is the default value for LogKubernetesConfig.APIWarnings.
// It causes Kubernetes API deprecation warnings to be logged at WARN level.
const APIWarningsWarn = "warn"

// Skew policy values for GlobalConfig.SkewPolicy (enhancement 0019 D18):
// the render's response when a module requires a newer build of an
// OPM-namespace path than the platform pins.
const (
	// SkewPolicyWarn renders against the platform's build and reports the
	// skew as a warning. The default when config.cue carries no skewPolicy.
	SkewPolicyWarn = "warn"
	// SkewPolicyRefuse fails the render before evaluation.
	SkewPolicyRefuse = "refuse"
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

	// SkewPolicy is the config file's skewPolicy (SkewPolicyWarn when
	// absent). It governs renders against the local default platform and
	// --platform directories; when the cluster Platform CR is the source,
	// the CR's spec.skewPolicy takes precedence. There is no flag.
	SkewPolicy string

	// Registry is the resolved registry URL after applying precedence.
	// Set by config.Load using flag > env > config precedence.
	Registry string

	// RegistryResolution records how Registry was resolved — the winning
	// source and any shadowed values. Set by config.Load beside Registry;
	// first consumer is `opm registry login`'s resolution report.
	RegistryResolution ResolveRegistryResult

	// ConfigPath is the resolved config file path.
	// Set by config.Load.
	ConfigPath string

	// Flags holds the raw CLI flag values as set by the user.
	Flags GlobalFlags
}
