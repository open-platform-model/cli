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
