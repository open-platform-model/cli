package loader

import (
	"fmt"
	"os"

	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
)

// NewLoader creates a new loader with the given configuration.
func NewLoader(cfg *LoaderConfig) (*Loader, error) {
	if cfg == nil {
		cfg = &LoaderConfig{}
	}

	// Default to current directory
	if cfg.ModuleDir == "" {
		cfg.ModuleDir = "."
	}

	var opmConfig *OPMConfig
	var registryURL string
	var registrySource string

	// Load config if HomeDir is specified
	if cfg.HomeDir != "" {
		configLoader := NewConfigLoader(cfg.HomeDir, cfg.Verbose)
		loadedConfig, err := configLoader.Load()
		if err != nil {
			return nil, fmt.Errorf("failed to load config: %w", err)
		}
		opmConfig = loadedConfig

		// Use config registry as base
		if opmConfig.Registry != "" {
			registryURL = opmConfig.Registry
			registrySource = "config"
		}
	}

	// Apply precedence chain: flag > env > config
	// Check OPM_REGISTRY env var
	if envRegistry := os.Getenv("OPM_REGISTRY"); envRegistry != "" {
		registryURL = envRegistry
		registrySource = "OPM_REGISTRY env"
	}

	// Check explicit flag override (highest priority)
	if cfg.Registry != "" {
		registryURL = cfg.Registry
		registrySource = "flag"
	}

	// Fall back to CUE_REGISTRY if nothing else is set
	if registryURL == "" {
		if envRegistry := os.Getenv("CUE_REGISTRY"); envRegistry != "" {
			registryURL = envRegistry
			registrySource = "CUE_REGISTRY env"
		}
	}

	if cfg.Verbose && registryURL != "" {
		fmt.Fprintf(os.Stderr, "[loader] Using registry: %s (from %s)\n", registryURL, registrySource)
	}

	// Create registry client
	registryCfg := &modconfig.Config{}
	if registryURL != "" {
		registryCfg.CUERegistry = registryURL
	}

	reg, err := modconfig.NewRegistry(registryCfg)
	if err != nil {
		return nil, fmt.Errorf("failed to create registry client: %w", err)
	}

	// Create load.Config with registry
	loadCfg := &load.Config{
		Registry: reg,
		Dir:      cfg.ModuleDir,
	}

	l := &Loader{
		cfg:      cfg,
		config:   opmConfig,
		registry: reg,
		loadCfg:  loadCfg,
	}

	return l, nil
}

// Load loads the module from the configured directory with registry and overlay support.
func (l *Loader) Load() (*LoadResult, error) {
	if l.cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[loader] Loading module from: %s\n", l.cfg.ModuleDir)
	}

	// Apply path overlays if configured
	if len(l.cfg.PathOverlays) > 0 {
		if err := l.applyPathOverlays(); err != nil {
			return nil, fmt.Errorf("failed to apply path overlays: %w", err)
		}
	}

	// Load CUE instances
	if l.cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[loader] Loading CUE instances...\n")
	}

	instances := load.Instances([]string{"."}, l.loadCfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", l.cfg.ModuleDir)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("failed to load CUE instance: %w", inst.Err)
	}

	// Build CUE value
	ctx := cuecontext.New()
	value := ctx.BuildInstance(inst)

	// Note: We don't check value.Err() here because the module may contain
	// abstract definitions (e.g., #Module, #Provider) that are not concrete
	// at the root level. This is expected.

	if l.cfg.Verbose {
		fmt.Fprintf(os.Stderr, "[loader] Module loaded successfully\n")
		if len(l.cfg.PathOverlays) > 0 {
			fmt.Fprintf(os.Stderr, "[loader] Applied %d path overlay(s)\n", len(l.cfg.PathOverlays))
		}
	}

	return &LoadResult{
		Value:          value,
		Context:        ctx,
		ModulePath:     l.cfg.ModuleDir,
		OverlayedPaths: l.cfg.PathOverlays,
		Config:         l.config,
	}, nil
}
