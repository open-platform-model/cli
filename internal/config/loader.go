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
	return registry, nil
}

// Load loads the full OPM configuration into cfg, applying precedence rules.
// This implements the two-phase loading process per FR-013.
//
// Phase 1: Extract config.registry via simple parsing (BootstrapRegistry)
// Phase 2: Load config.cue with CUE_REGISTRY set to resolved registry
//
// Load sets: cfg.ConfigPath, cfg.Registry, cfg.Kubernetes, cfg.Log, cfg.Providers, cfg.CueContext.
// The caller sets cfg.Flags before or after calling Load.
func Load(cfg *GlobalConfig, opts LoaderOptions) error {
	// Step 1: Resolve config path
	configPathResult, err := ResolveConfigPath(ResolveConfigPathOptions{
		FlagValue: opts.ConfigFlag,
	})
	if err != nil {
		return fmt.Errorf("resolving config path: %w", err)
	}

	cfg.ConfigPath = configPathResult.ConfigPath

	output.Debug("resolved config path",
		"path", configPathResult.ConfigPath,
		"source", configPathResult.Source,
	)

	// Step 2: Bootstrap - extract registry from config without full parsing
	configRegistry, err := BootstrapRegistry(configPathResult.ConfigPath)
	if err != nil {
		return fmt.Errorf("bootstrap registry extraction: %w", err)
	}

	// Step 3: Resolve registry using precedence
	registryResult := ResolveRegistry(ResolveRegistryOptions{
		FlagValue:   opts.RegistryFlag,
		ConfigValue: configRegistry,
	})

	cfg.Registry = registryResult.Registry

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
			return oerrors.NewValidationError(
				"providers configured but no registry resolvable",
				configPathResult.ConfigPath,
				"providers", // field
				"Set OPM_REGISTRY environment variable, use --registry flag, or add registry field to config.cue",
			)
		}
	}

	// Step 5: Phase 2 - Load full config with registry set
	err = loadFullConfig(cfg, configPathResult.ConfigPath, registryResult.Registry)
	if err != nil {
		return err
	}

	return nil
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

// loadFullConfig loads the config.cue file with full CUE evaluation,
// populating cfg fields directly.
// This is Phase 2 of the two-phase loading process.
func loadFullConfig(cfg *GlobalConfig, configPath, registry string) error {
	// Check if config exists
	if _, err := os.Stat(configPath); os.IsNotExist(err) {
		output.Debug("config file not found, using defaults",
			"path", configPath,
		)
		// Apply defaults inline
		cfg.Kubernetes = KubernetesConfig{
			Kubeconfig: "~/.kube/config",
			Namespace:  "default",
		}
		cfg.Log.Kubernetes.APIWarnings = "warn"
		cfg.CueContext = cuecontext.New()
		return nil
	}

	// Use the directory containing the config file for CUE loading.
	// This ensures custom config paths work correctly.
	configDir := filepath.Dir(configPath)

	// Set CUE_REGISTRY if registry is provided
	if registry != "" {
		os.Setenv("CUE_REGISTRY", registry)
		defer os.Unsetenv("CUE_REGISTRY") // Clean up after loading
	}

	// Load CUE configuration
	ctx := cuecontext.New()

	cueLoadCfg := &load.Config{
		Dir: configDir,
	}

	instances := load.Instances([]string{"."}, cueLoadCfg)
	if len(instances) == 0 {
		return oerrors.NewValidationError(
			"no CUE instances found",
			configDir,
			"", // no specific field
			"Ensure config.cue exists in the configuration directory",
		)
	}

	inst := instances[0]
	if inst.Err != nil {
		return &oerrors.DetailError{
			Type:     "configuration error",
			Message:  inst.Err.Error(),
			Location: configPath,
			Hint:     "Run 'opm config vet' to check for configuration errors",
			Cause:    oerrors.ErrValidation,
		}
	}

	value := ctx.BuildInstance(inst)
	if value.Err() != nil {
		return &oerrors.DetailError{
			Type:     "configuration error",
			Message:  value.Err().Error(),
			Location: configPath,
			Hint:     "Run 'opm config vet' to check for configuration errors",
			Cause:    oerrors.ErrValidation,
		}
	}

	// Validate against embedded schema
	if err := validateConfigSchema(ctx, value, configPath); err != nil {
		return err
	}

	// Extract config values into cfg
	extractConfigInto(cfg, value)

	// Set default for APIWarnings if not specified
	if cfg.Log.Kubernetes.APIWarnings == "" {
		cfg.Log.Kubernetes.APIWarnings = "warn"
	}

	// Extract providers
	cfg.Providers = extractProviders(value)
	cfg.CueContext = ctx

	return nil
}

// validateConfigSchema validates the loaded CUE value against the embedded schema.
func validateConfigSchema(ctx *cue.Context, value cue.Value, configPath string) error {
	// Compile the embedded schema
	schema := ctx.CompileBytes(configSchemaCUE, cue.Filename("schema/config.cue"))
	if schema.Err() != nil {
		return fmt.Errorf("compiling embedded config schema: %w", schema.Err())
	}

	// Look up #CLIConfig definition
	def := schema.LookupPath(cue.ParsePath("#CLIConfig"))
	if !def.Exists() {
		return fmt.Errorf("embedded schema missing #CLIConfig definition")
	}

	// Unify user config with schema
	unified := def.Unify(value)
	if err := unified.Validate(cue.Concrete(true)); err != nil {
		// CUE validation error - extract meaningful parts
		return &oerrors.DetailError{
			Type:     "schema validation failed",
			Message:  err.Error(),
			Location: configPath,
			Hint:     "Check your config.cue against the expected schema. Run 'opm config vet' for validation.",
			Cause:    oerrors.ErrValidation,
		}
	}

	return nil
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
	}

	if len(providers) == 0 {
		return nil
	}

	return providers
}

// extractConfigInto populates cfg fields from the CUE value.
//
//nolint:unparam // error return allows for future validation
func extractConfigInto(cfg *GlobalConfig, value cue.Value) {
	// Apply defaults first
	cfg.Kubernetes = KubernetesConfig{
		Kubeconfig: "~/.kube/config",
		Namespace:  "default",
	}

	// Look for config struct or top-level fields
	configValue := value.LookupPath(cue.ParsePath("config"))
	if !configValue.Exists() {
		// Try top-level fields directly
		configValue = value
	}

	// Extract registry (already resolved separately, but keep for completeness)
	// Note: cfg.Registry is already set by the caller via ResolveRegistry

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

	// Extract log config
	logValue := configValue.LookupPath(cue.ParsePath("log"))
	if logValue.Exists() {
		if tsVal := logValue.LookupPath(cue.ParsePath("timestamps")); tsVal.Exists() {
			if b, err := tsVal.Bool(); err == nil {
				cfg.Log.Timestamps = &b
			}
		}

		// Extract log.kubernetes.apiWarnings
		logK8sValue := logValue.LookupPath(cue.ParsePath("kubernetes"))
		if logK8sValue.Exists() {
			if apiWarningsVal := logK8sValue.LookupPath(cue.ParsePath("apiWarnings")); apiWarningsVal.Exists() {
				if str, err := apiWarningsVal.String(); err == nil {
					cfg.Log.Kubernetes.APIWarnings = str
				}
			}
		}
	}
}
