// Package config provides configuration management for the OPM CLI.
package config

// Config represents the OPM CLI configuration.
// Loaded from ~/.opm/config.yaml, validated against embedded CUE schema.
type Config struct {
	// Kubeconfig is the path to the kubeconfig file.
	// Env: OPM_KUBECONFIG, Default: ~/.kube/config
	Kubeconfig string `yaml:"kubeconfig,omitempty" json:"kubeconfig,omitempty" mapstructure:"kubeconfig"`

	// Context is the Kubernetes context to use.
	// Env: OPM_CONTEXT, Default: current-context from kubeconfig
	Context string `yaml:"context,omitempty" json:"context,omitempty" mapstructure:"context"`

	// Namespace is the default namespace for operations.
	// Env: OPM_NAMESPACE, Default: "default"
	Namespace string `yaml:"namespace,omitempty" json:"namespace,omitempty" mapstructure:"namespace"`

	// Registry is the default OCI registry for publish/get.
	// Env: OPM_REGISTRY
	Registry string `yaml:"registry,omitempty" json:"registry,omitempty" mapstructure:"registry"`

	// CacheDir is the local cache directory.
	// Env: OPM_CACHE_DIR, Default: ~/.opm/cache
	CacheDir string `yaml:"cacheDir,omitempty" json:"cacheDir,omitempty" mapstructure:"cacheDir"`
}

// DefaultConfig returns a Config with all default values populated.
// Used by `opm config init` to generate initial config file.
func DefaultConfig() *Config {
	return &Config{
		Kubeconfig: "~/.kube/config",
		Namespace:  "default",
		CacheDir:   "~/.opm/cache",
	}
}

// IsEmpty returns true if the config has no values set.
func (c *Config) IsEmpty() bool {
	return c.Kubeconfig == "" &&
		c.Context == "" &&
		c.Namespace == "" &&
		c.Registry == "" &&
		c.CacheDir == ""
}

// Merge merges another config into this one.
// Non-empty values from other override values in this config.
func (c *Config) Merge(other *Config) {
	if other == nil {
		return
	}
	if other.Kubeconfig != "" {
		c.Kubeconfig = other.Kubeconfig
	}
	if other.Context != "" {
		c.Context = other.Context
	}
	if other.Namespace != "" {
		c.Namespace = other.Namespace
	}
	if other.Registry != "" {
		c.Registry = other.Registry
	}
	if other.CacheDir != "" {
		c.CacheDir = other.CacheDir
	}
}

// WithDefaults returns a copy of the config with defaults applied.
func (c *Config) WithDefaults() *Config {
	result := DefaultConfig()
	result.Merge(c)
	return result
}
