// Package config provides configuration loading and management.
package config

import (
	"os"
)

// Source indicates where a configuration value came from.
type Source string

const (
	// SourceFlag indicates value came from command-line flag.
	SourceFlag Source = "flag"
	// SourceEnv indicates value came from environment variable.
	SourceEnv Source = "env"
	// SourceConfig indicates value came from config file.
	SourceConfig Source = "config"
	// SourceDefault indicates value is the built-in default.
	SourceDefault Source = "default"
)

// ResolveRegistryOptions contains options for registry resolution.
type ResolveRegistryOptions struct {
	// FlagValue is the --registry flag value (empty if not set).
	FlagValue string
	// ConfigValue is the registry value from config file (empty if not set).
	ConfigValue string
}

// ResolveRegistryResult contains the resolved registry and its source.
type ResolveRegistryResult struct {
	// Registry is the resolved registry URL.
	Registry string
	// Source indicates where the registry came from.
	Source Source
	// Shadowed contains values that were overridden by higher precedence.
	Shadowed map[Source]string
}

// ResolveRegistry resolves the registry URL using precedence:
// (1) --registry flag, (2) OPM_REGISTRY env, (3) config.registry
//
// Per FR-009: The CLI MUST resolve the registry URL using this precedence.
func ResolveRegistry(opts ResolveRegistryOptions) ResolveRegistryResult {
	result := ResolveRegistryResult{
		Shadowed: make(map[Source]string),
	}

	// Collect all potential values
	envValue := os.Getenv("OPM_REGISTRY")

	// Resolve using precedence: flag > env > config
	switch {
	case opts.FlagValue != "":
		result.Registry = opts.FlagValue
		result.Source = SourceFlag
		// Record shadowed values
		if envValue != "" {
			result.Shadowed[SourceEnv] = envValue
		}
		if opts.ConfigValue != "" {
			result.Shadowed[SourceConfig] = opts.ConfigValue
		}
	case envValue != "":
		result.Registry = envValue
		result.Source = SourceEnv
		// Record shadowed values
		if opts.ConfigValue != "" {
			result.Shadowed[SourceConfig] = opts.ConfigValue
		}
	case opts.ConfigValue != "":
		result.Registry = opts.ConfigValue
		result.Source = SourceConfig
	}
	// If none set, Registry stays empty and Source is zero value

	return result
}

// ResolveConfigPathOptions contains options for config path resolution.
type ResolveConfigPathOptions struct {
	// FlagValue is the --config flag value (empty if not set).
	FlagValue string
}

// ResolveConfigPathResult contains the resolved config path and its source.
type ResolveConfigPathResult struct {
	// ConfigPath is the resolved config file path.
	ConfigPath string
	// Source indicates where the config path came from.
	Source Source
	// Shadowed contains values that were overridden by higher precedence.
	Shadowed map[Source]string
}

// ResolveConfigPath resolves the config file path using precedence:
// (1) --config flag, (2) OPM_CONFIG env, (3) ~/.opm/config.cue default
//
// Per FR-018: The CLI MUST resolve configuration values using precedence.
func ResolveConfigPath(opts ResolveConfigPathOptions) (ResolveConfigPathResult, error) {
	result := ResolveConfigPathResult{
		Shadowed: make(map[Source]string),
	}

	envValue := os.Getenv("OPM_CONFIG")

	// Get default path
	paths, err := DefaultPaths()
	if err != nil {
		return result, err
	}
	defaultPath := paths.ConfigFile

	// Resolve using precedence: flag > env > default
	switch {
	case opts.FlagValue != "":
		result.ConfigPath = opts.FlagValue
		result.Source = SourceFlag
		// Record shadowed values
		if envValue != "" {
			result.Shadowed[SourceEnv] = envValue
		}
		result.Shadowed[SourceDefault] = defaultPath
	case envValue != "":
		result.ConfigPath = envValue
		result.Source = SourceEnv
		// Record shadowed values
		result.Shadowed[SourceDefault] = defaultPath
	default:
		result.ConfigPath = defaultPath
		result.Source = SourceDefault
	}

	return result, nil
}

// SourceConfigAuto indicates provider was auto-resolved from single configured provider.
const SourceConfigAuto Source = "config-auto"

// ResolvedField contains a resolved configuration value with source tracking.
type ResolvedField struct {
	// Value is the resolved value.
	Value string
	// Source indicates where the value came from.
	Source Source
	// Shadowed contains values that were overridden by higher precedence.
	Shadowed map[Source]string
}

// ResolvedConfig contains all resolved configuration values.
type ResolvedConfig struct {
	ConfigPath ResolvedField
	Registry   ResolvedField
	Kubeconfig ResolvedField
	Context    ResolvedField
	Namespace  ResolvedField
	Provider   ResolvedField
}

// ResolveAllOptions contains options for resolving all configuration values.
type ResolveAllOptions struct {
	// Flag values
	ConfigFlag     string
	RegistryFlag   string
	KubeconfigFlag string
	ContextFlag    string
	NamespaceFlag  string
	ProviderFlag   string

	// Config values (from loaded config file)
	Config *Config

	// Provider names (keys from loaded providers map)
	ProviderNames []string
}

// ResolveAll resolves all configuration values using precedence: Flag > Env > Config > Default.
func ResolveAll(opts ResolveAllOptions) (*ResolvedConfig, error) {
	result := &ResolvedConfig{}

	// Resolve config path
	configPathResult, err := ResolveConfigPath(ResolveConfigPathOptions{
		FlagValue: opts.ConfigFlag,
	})
	if err != nil {
		return nil, err
	}
	result.ConfigPath = ResolvedField{
		Value:    configPathResult.ConfigPath,
		Source:   configPathResult.Source,
		Shadowed: configPathResult.Shadowed,
	}

	// Resolve registry
	var configRegistry string
	if opts.Config != nil {
		configRegistry = opts.Config.Registry
	}
	registryResult := ResolveRegistry(ResolveRegistryOptions{
		FlagValue:   opts.RegistryFlag,
		ConfigValue: configRegistry,
	})
	result.Registry = ResolvedField{
		Value:    registryResult.Registry,
		Source:   registryResult.Source,
		Shadowed: registryResult.Shadowed,
	}

	// Resolve kubeconfig
	result.Kubeconfig = resolveStringField(
		opts.KubeconfigFlag,
		"OPM_KUBECONFIG",
		func() string {
			if opts.Config != nil {
				return opts.Config.Kubernetes.Kubeconfig
			}
			return ""
		},
		"~/.kube/config",
	)
	// Expand tilde in kubeconfig path
	result.Kubeconfig.Value = ExpandTilde(result.Kubeconfig.Value)

	// Resolve context
	result.Context = resolveStringField(
		opts.ContextFlag,
		"OPM_CONTEXT",
		func() string {
			if opts.Config != nil {
				return opts.Config.Kubernetes.Context
			}
			return ""
		},
		"", // no default for context
	)

	// Resolve namespace
	result.Namespace = resolveStringField(
		opts.NamespaceFlag,
		"OPM_NAMESPACE",
		func() string {
			if opts.Config != nil {
				return opts.Config.Kubernetes.Namespace
			}
			return ""
		},
		"default",
	)

	// Resolve provider with auto-resolution
	result.Provider = resolveProvider(opts.ProviderFlag, opts.ProviderNames)

	return result, nil
}

// resolveStringField resolves a single configuration field using Flag > Env > Config > Default precedence.
func resolveStringField(flagValue, envVar string, configGetter func() string, defaultValue string) ResolvedField {
	result := ResolvedField{
		Shadowed: make(map[Source]string),
	}

	envValue := os.Getenv(envVar)
	configValue := configGetter()

	// Resolve using precedence: flag > env > config > default
	switch {
	case flagValue != "":
		result.Value = flagValue
		result.Source = SourceFlag
		// Record shadowed values
		if envValue != "" {
			result.Shadowed[SourceEnv] = envValue
		}
		if configValue != "" {
			result.Shadowed[SourceConfig] = configValue
		}
		if defaultValue != "" {
			result.Shadowed[SourceDefault] = defaultValue
		}
	case envValue != "":
		result.Value = envValue
		result.Source = SourceEnv
		// Record shadowed values
		if configValue != "" {
			result.Shadowed[SourceConfig] = configValue
		}
		if defaultValue != "" {
			result.Shadowed[SourceDefault] = defaultValue
		}
	case configValue != "":
		result.Value = configValue
		result.Source = SourceConfig
		// Record shadowed values
		if defaultValue != "" {
			result.Shadowed[SourceDefault] = defaultValue
		}
	default:
		result.Value = defaultValue
		result.Source = SourceDefault
	}

	return result
}

// resolveProvider resolves the provider field with auto-resolution logic.
func resolveProvider(flagValue string, providerNames []string) ResolvedField {
	result := ResolvedField{
		Shadowed: make(map[Source]string),
	}

	// If flag is set, use it
	if flagValue != "" {
		result.Value = flagValue
		result.Source = SourceFlag
		return result
	}

	// Auto-resolve if exactly one provider configured
	if len(providerNames) == 1 {
		result.Value = providerNames[0]
		result.Source = SourceConfigAuto
		return result
	}

	// Otherwise, provider remains empty
	result.Value = ""
	result.Source = "" // no source when empty
	return result
}
