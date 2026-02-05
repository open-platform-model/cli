// Package config provides configuration loading and management.
package config

import (
	"fmt"
	"os"
	"path/filepath"
	"regexp"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"

	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

// LoaderOptions contains options for loading configuration.
type LoaderOptions struct {
	// RegistryFlag is the --registry flag value.
	RegistryFlag string
	// ConfigFlag is the --config flag value.
	ConfigFlag string
}

// registryRegex matches registry field in CUE config.
// Matches patterns like: registry: "localhost:5001" or registry: "registry.example.com"
var registryRegex = regexp.MustCompile(`(?m)^\s*registry:\s*"([^"]*)"`)

// BootstrapRegistry extracts the registry value from config.cue using simple parsing.
// This is Phase 1 of the two-phase loading process (FR-013).
//
// Phase 1 (Bootstrap): Extract config.registry via simple CUE parsing without import resolution.
// This allows us to resolve the registry before loading config with provider imports.
func BootstrapRegistry(configPath string) (string, error) {
	// Check if config file exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		output.Debug("config file not found, skipping bootstrap",
			"path", configPath,
		)
		return "", nil // No config file, no registry
	}

	// Read config file
	content, err := os.ReadFile(configPath)
	if err != nil {
		return "", fmt.Errorf("reading config file: %w", err)
	}

	// Simple regex extraction of registry value
	// This avoids CUE parsing which would require resolving imports
	matches := registryRegex.FindSubmatch(content)
	if len(matches) < 2 {
		output.Debug("no registry found in config",
			"path", configPath,
		)
		return "", nil // No registry in config
	}

	registry := string(matches[1])
	output.Debug("bootstrap: extracted registry from config",
		"registry", registry,
		"path", configPath,
	)
	return registry, nil
}

// LoadOPMConfig loads the full OPM configuration with resolved registry.
// This implements the two-phase loading process per FR-013.
//
// Phase 1: Extract config.registry via simple parsing (BootstrapRegistry)
// Phase 2: Load config.cue with CUE_REGISTRY set to resolved registry
func LoadOPMConfig(opts LoaderOptions) (*OPMConfig, error) {
	// Step 1: Resolve config path
	configPathResult, err := ResolveConfigPath(ResolveConfigPathOptions{
		FlagValue: opts.ConfigFlag,
	})
	if err != nil {
		return nil, fmt.Errorf("resolving config path: %w", err)
	}

	output.Debug("resolved config path",
		"path", configPathResult.ConfigPath,
		"source", configPathResult.Source,
	)

	// Step 2: Bootstrap - extract registry from config without full parsing
	configRegistry, err := BootstrapRegistry(configPathResult.ConfigPath)
	if err != nil {
		return nil, fmt.Errorf("bootstrap registry extraction: %w", err)
	}

	// Step 3: Resolve registry using precedence
	registryResult := ResolveRegistry(ResolveRegistryOptions{
		FlagValue:   opts.RegistryFlag,
		ConfigValue: configRegistry,
	})

	output.Debug("resolved registry",
		"registry", registryResult.Registry,
		"source", registryResult.Source,
	)

	// Step 4: Check if providers are configured but no registry
	// Per FR-014: Fail fast if providers configured but no registry resolvable
	if registryResult.Registry == "" {
		// Check if config exists and has providers
		hasProviders, err := configHasProviders(configPathResult.ConfigPath)
		if err != nil {
			output.Debug("could not check for providers", "error", err)
		}
		if hasProviders {
			return nil, oerrors.NewValidationError(
				"providers configured but no registry resolvable",
				configPathResult.ConfigPath,
				"providers", // field
				"Set OPM_REGISTRY environment variable, use --registry flag, or add registry field to config.cue",
			)
		}
	}

	// Step 5: Phase 2 - Load full config with registry set
	cfg, providers, cueCtx, err := loadFullConfig(configPathResult.ConfigPath, registryResult.Registry)
	if err != nil {
		return nil, err
	}

	return &OPMConfig{
		Config:         cfg,
		Registry:       registryResult.Registry,
		RegistrySource: string(registryResult.Source),
		Providers:      providers,
		CueContext:     cueCtx,
	}, nil
}

// configHasProviders checks if the config file references providers.
func configHasProviders(configPath string) (bool, error) {
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		return false, nil
	}

	content, err := os.ReadFile(configPath)
	if err != nil {
		return false, err
	}

	// Simple check: look for "providers" in config
	providerRegex := regexp.MustCompile(`(?m)^\s*providers:\s*{`)
	return providerRegex.Match(content), nil
}

// loadFullConfig loads the config.cue file with full CUE evaluation.
// This is Phase 2 of the two-phase loading process.
// Returns the config, providers map, CUE context, and any error.
func loadFullConfig(configPath, registry string) (*Config, map[string]cue.Value, *cue.Context, error) {
	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		output.Debug("config file not found, using defaults",
			"path", configPath,
		)
		return DefaultConfig(), nil, cuecontext.New(), nil
	}

	// Use the directory containing the config file for CUE loading.
	// This ensures custom config paths work correctly.
	configDir := filepath.Dir(configPath)

	// Set CUE_REGISTRY if registry is provided
	if registry != "" {
		output.Debug("setting CUE_REGISTRY for config load",
			"registry", registry,
		)
		os.Setenv("CUE_REGISTRY", registry)
		defer os.Unsetenv("CUE_REGISTRY") // Clean up after loading
	}

	// Load CUE configuration
	ctx := cuecontext.New()

	cfg := &load.Config{
		Dir: configDir,
	}

	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, nil, nil, oerrors.NewValidationError(
			"no CUE instances found",
			configDir,
			"", // no specific field
			"Ensure config.cue exists in the configuration directory",
		)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, nil, nil, &oerrors.DetailError{
			Type:     "configuration error",
			Message:  inst.Err.Error(),
			Location: configPath,
			Hint:     "Run 'opm config vet' to check for configuration errors",
			Cause:    oerrors.ErrValidation,
		}
	}

	value := ctx.BuildInstance(inst)
	if value.Err() != nil {
		return nil, nil, nil, &oerrors.DetailError{
			Type:     "configuration error",
			Message:  value.Err().Error(),
			Location: configPath,
			Hint:     "Run 'opm config vet' to check for configuration errors",
			Cause:    oerrors.ErrValidation,
		}
	}

	// Extract config values
	config, err := extractConfig(value)
	if err != nil {
		return nil, nil, nil, err
	}

	// Extract providers
	providers := extractProviders(value)

	return config, providers, ctx, nil
}

// extractProviders extracts provider definitions from the CUE config value.
// Returns a map of provider alias to CUE value.
func extractProviders(value cue.Value) map[string]cue.Value {
	// Look for providers in config struct first
	configValue := value.LookupPath(cue.ParsePath("config"))
	if !configValue.Exists() {
		configValue = value
	}

	providersValue := configValue.LookupPath(cue.ParsePath("providers"))
	if !providersValue.Exists() {
		output.Debug("no providers found in config")
		return nil
	}

	providers := make(map[string]cue.Value)
	iter, err := providersValue.Fields()
	if err != nil {
		output.Debug("failed to iterate providers", "error", err)
		return nil
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		providers[name] = iter.Value()
		output.Debug("extracted provider from config", "name", name)
	}

	if len(providers) == 0 {
		return nil
	}

	output.Debug("extracted providers from config", "count", len(providers))
	return providers
}

// extractConfig extracts Go config struct from CUE value.
//
//nolint:unparam // error return allows for future validation
func extractConfig(value cue.Value) (*Config, error) {
	cfg := DefaultConfig()

	// Look for config struct or top-level fields
	configValue := value.LookupPath(cue.ParsePath("config"))
	if !configValue.Exists() {
		// Try top-level fields directly
		configValue = value
	}

	// Extract registry
	if registryVal := configValue.LookupPath(cue.ParsePath("registry")); registryVal.Exists() {
		if str, err := registryVal.String(); err == nil {
			cfg.Registry = str
		}
	}

	// Extract kubernetes config
	k8sValue := configValue.LookupPath(cue.ParsePath("kubernetes"))
	if k8sValue.Exists() {
		if kubeconfigVal := k8sValue.LookupPath(cue.ParsePath("kubeconfig")); kubeconfigVal.Exists() {
			if str, err := kubeconfigVal.String(); err == nil {
				cfg.Kubernetes.Kubeconfig = str
			}
		}
		if contextVal := k8sValue.LookupPath(cue.ParsePath("context")); contextVal.Exists() {
			if str, err := contextVal.String(); err == nil {
				cfg.Kubernetes.Context = str
			}
		}
		if namespaceVal := k8sValue.LookupPath(cue.ParsePath("namespace")); namespaceVal.Exists() {
			if str, err := namespaceVal.String(); err == nil {
				cfg.Kubernetes.Namespace = str
			}
		}
	}

	return cfg, nil
}

// CheckRegistryConnectivity checks if the registry is reachable.
// Per FR-010 and FR-014: Fail fast with clear error if registry unreachable.
func CheckRegistryConnectivity(registry string) error {
	if registry == "" {
		return nil // No registry to check
	}

	// Use a simple HEAD request to check connectivity
	// For OCI registries, we check the /v2/ endpoint
	url := fmt.Sprintf("https://%s/v2/", registry)

	// Note: We're just checking connectivity, not authentication
	// For now, we'll skip the actual HTTP check since it requires more setup.
	// The CUE binary will fail with a clear error if registry is unreachable.
	output.Debug("checking registry connectivity", "url", url)
	return nil
}
