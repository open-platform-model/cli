// Package schema defines the embedded CUE schema for OPM CLI configuration.
// This schema validates config.cue files loaded by the CLI.
package schema

// #CLIConfig is the root schema for the CLI configuration file.
// The config.cue file must contain a 'config' struct that matches #Config.
#CLIConfig: {
	config: #Config
	...
}

// #Config defines the structure of the config struct.
//
// The config file is scalar data only — no CUE imports, no providers
// (retired by enhancement 0006 D39; catalog selection lives in the
// sibling ~/.opm/platform.cue, see schema/platform.cue).
#Config: {
	// registry is the default registry for CUE module resolution.
	// Can be overridden by --registry flag or OPM_REGISTRY env var.
	// Format supports multiple registries separated by commas, with options like +insecure.
	// Example: "opmodel.dev=localhost:5000+insecure,registry.cue.works"
	registry?: string

	// kubernetes contains Kubernetes-specific settings.
	kubernetes?: #KubernetesConfig

	// log contains logging configuration.
	log?: #LogConfig
}

// #KubernetesConfig contains Kubernetes-specific settings.
#KubernetesConfig: {
	// kubeconfig is the path to the kubeconfig file.
	// Env: OPM_KUBECONFIG, Default: ~/.kube/config
	kubeconfig?: string

	// context is the Kubernetes context to use.
	// Env: OPM_CONTEXT, Default: current-context from kubeconfig
	context?: string

	// namespace is the default namespace for operations.
	// Env: OPM_NAMESPACE, Default: "default"
	// Must be RFC-1123 compliant (lowercase alphanumeric and hyphens).
	namespace?: string & =~"^[a-z0-9]([-a-z0-9]*[a-z0-9])?$"
}

// #LogConfig contains logging-related settings.
#LogConfig: {
	// timestamps controls whether timestamps are shown in log output.
	// Override with --timestamps flag.
	timestamps?: bool

	// kubernetes contains Kubernetes-related logging settings.
	kubernetes?: #LogKubernetesConfig
}

// #LogKubernetesConfig contains Kubernetes-related logging settings.
#LogKubernetesConfig: {
	// apiWarnings controls how Kubernetes API deprecation warnings are displayed.
	// Valid values: "warn" (default), "debug", "suppress"
	// - "warn": Show as WARN level in log output
	// - "debug": Only show with --verbose flag
	// - "suppress": Drop entirely
	apiWarnings?: "warn" | "debug" | "suppress"
}
