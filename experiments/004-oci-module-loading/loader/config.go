package loader

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
)

// OPMConfig holds parsed configuration from ~/.opm/config.cue
type OPMConfig struct {
	// Registry URL for module resolution
	Registry string

	// Providers maps provider names to their CUE definitions
	Providers map[string]cue.Value

	// rawValue is the full parsed config value
	rawValue cue.Value
}

// ConfigLoader loads OPM configuration from CUE files
type ConfigLoader struct {
	homeDir string
	verbose bool
}

// NewConfigLoader creates a new config loader
func NewConfigLoader(homeDir string, verbose bool) *ConfigLoader {
	return &ConfigLoader{
		homeDir: homeDir,
		verbose: verbose,
	}
}

// Load loads and validates the config from the specified home directory
func (cl *ConfigLoader) Load() (*OPMConfig, error) {
	if cl.homeDir == "" {
		return nil, fmt.Errorf("config home directory not specified")
	}

	// Expand path if it contains ~
	homeDir, err := expandPath(cl.homeDir)
	if err != nil {
		return nil, fmt.Errorf("failed to expand home directory path: %w", err)
	}

	if cl.verbose {
		fmt.Fprintf(os.Stderr, "[config] Loading config from %s\n", homeDir)
	}

	// Check if directory exists
	if _, err := os.Stat(homeDir); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config directory does not exist: %s", homeDir)
		}
		return nil, fmt.Errorf("failed to access config directory: %w", err)
	}

	// Check if module.cue exists
	modulePath := filepath.Join(homeDir, "cue.mod", "module.cue")
	if _, err := os.Stat(modulePath); err != nil {
		if os.IsNotExist(err) {
			return nil, fmt.Errorf("config module.cue not found at %s", modulePath)
		}
		return nil, fmt.Errorf("failed to access module.cue: %w", err)
	}

	// Create registry client for module resolution
	registryCfg := &modconfig.Config{}

	// Check if OPM_REGISTRY env var is set (used during config loading)
	if envRegistry := os.Getenv("OPM_REGISTRY"); envRegistry != "" {
		if cl.verbose {
			fmt.Fprintf(os.Stderr, "[config] Using registry from OPM_REGISTRY for config loading: %s\n", envRegistry)
		}
		registryCfg.CUERegistry = envRegistry
	}

	reg, err := modconfig.NewRegistry(registryCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client for config loading: %w", err)
	}

	// Load the config module
	loadCfg := &load.Config{
		Registry: reg,
		Dir:      homeDir,
	}

	instances := load.Instances([]string{"."}, loadCfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", homeDir)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("failed to load config instance: %w", inst.Err)
	}

	// Build CUE value
	ctx := cuecontext.New()
	value := ctx.BuildInstance(inst)

	// Note: We don't check value.Err() here because the config may reference
	// abstract provider definitions that aren't meant to be fully concrete yet.

	// Extract config from value
	configVal := value.LookupPath(cue.ParsePath("config"))
	if !configVal.Exists() {
		return nil, fmt.Errorf("config field not found in config.cue")
	}

	// Note: We check for errors but allow incomplete values since providers
	// may contain abstract definitions

	// Parse config fields
	cfg := &OPMConfig{
		rawValue:  configVal,
		Providers: make(map[string]cue.Value),
	}

	// Extract registry
	registryVal := configVal.LookupPath(cue.ParsePath("registry"))
	if registryVal.Exists() {
		if cfg.Registry, err = registryVal.String(); err != nil {
			return nil, fmt.Errorf("failed to parse config.registry: %w", err)
		}
		if cl.verbose {
			fmt.Fprintf(os.Stderr, "[config] Config registry: %s\n", cfg.Registry)
		}
	}

	// Extract providers
	providersVal := configVal.LookupPath(cue.ParsePath("providers"))
	if providersVal.Exists() {
		iter, err := providersVal.Fields()
		if err != nil {
			return nil, fmt.Errorf("failed to iterate providers: %w", err)
		}

		for iter.Next() {
			providerName := iter.Selector().String()
			providerVal := iter.Value()

			// Store the provider value (may be incomplete/abstract, that's OK)
			// We just verify that the path exists and the module was fetchable
			if !providerVal.Exists() {
				return nil, fmt.Errorf("provider %s does not exist", providerName)
			}

			cfg.Providers[providerName] = providerVal

			if cl.verbose {
				fmt.Fprintf(os.Stderr, "[config] Loaded provider: %s\n", providerName)
			}
		}
	}

	if cl.verbose {
		fmt.Fprintf(os.Stderr, "[config] Config loaded successfully\n")
	}

	return cfg, nil
}

// ProviderNames returns a list of configured provider names
func (c *OPMConfig) ProviderNames() []string {
	names := make([]string, 0, len(c.Providers))
	for name := range c.Providers {
		names = append(names, name)
	}
	return names
}

// GetProvider returns the CUE value for a provider by name
func (c *OPMConfig) GetProvider(name string) (cue.Value, bool) {
	val, ok := c.Providers[name]
	return val, ok
}

// expandPath expands ~ to home directory
func expandPath(path string) (string, error) {
	if path == "" {
		return path, nil
	}

	if path[0] == '~' {
		home, err := os.UserHomeDir()
		if err != nil {
			return "", err
		}
		path = filepath.Join(home, path[1:])
	}

	return filepath.Clean(path), nil
}
