package loader

import (
	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
	"cuelang.org/go/mod/modconfig"
)

// LoaderConfig configures the module loader.
type LoaderConfig struct {
	// HomeDir is the path to the .opm config directory.
	// If specified, config is loaded from this directory.
	HomeDir string

	// Registry URL (defaults to CUE_REGISTRY env var or empty for default)
	// Precedence: this field > OPM_REGISTRY env > config.registry
	Registry string

	// PathOverlays are local directories to overlay on top of registry modules.
	// Files from these paths take precedence over registry-fetched modules.
	PathOverlays []string

	// ModuleDir is the directory containing the module.cue file.
	ModuleDir string

	// Verbose enables detailed logging.
	Verbose bool
}

// Loader loads CUE modules with registry and path overlay support.
type Loader struct {
	cfg      *LoaderConfig
	config   *OPMConfig // Loaded config from HomeDir
	registry modconfig.Registry
	loadCfg  *load.Config
}

// LoadResult contains the loaded module and metadata.
type LoadResult struct {
	// Value is the loaded and validated CUE value.
	Value cue.Value

	// Context is the CUE context used for loading.
	Context *cue.Context

	// ModulePath is the resolved module path.
	ModulePath string

	// OverlayedPaths lists the paths that were overlayed.
	OverlayedPaths []string

	// Config is the loaded OPM config (if HomeDir was specified)
	Config *OPMConfig
}
