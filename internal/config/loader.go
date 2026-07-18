// Package config provides configuration loading and management.
package config

import (
	"fmt"
	"os"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"

	"github.com/open-platform-model/cli/internal/output"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// defaultKubeconfig is the built-in kubeconfig path default.
const defaultKubeconfig = "~/.kube/config"

// LoaderOptions contains options for loading configuration.
type LoaderOptions struct {
	// RegistryFlag is the --registry flag value.
	RegistryFlag string
	// ConfigFlag is the --config flag value.
	ConfigFlag string
}

// Load loads the OPM configuration into cfg, applying precedence rules.
//
// Loading is single-pass: ~/.opm/config.cue is import-free scalar data
// (enhancement 0006 D39), so the file is parsed and validated exactly once
// and the registry resolves by ordinary flag > env > config precedence
// afterwards. There is no registry-bootstrap pre-pass.
//
// Load sets: cfg.ConfigPath, cfg.Registry, cfg.Kubernetes, cfg.Log,
// cfg.CueContext. The caller sets cfg.Flags before or after calling Load.
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

	// Step 2: Parse and validate the config file (single pass).
	configRegistry, err := loadConfigFile(cfg, configPathResult.ConfigPath)
	if err != nil {
		return err
	}

	// Step 3: Resolve registry using precedence flag > env > config.
	registryResult := ResolveRegistry(ResolveRegistryOptions{
		FlagValue:   opts.RegistryFlag,
		ConfigValue: configRegistry,
	})

	cfg.Registry = registryResult.Registry

	output.Debug("resolved registry",
		"registry", registryResult.Registry,
		"source", registryResult.Source,
	)

	return nil
}

// loadConfigFile parses configPath, validates it against the embedded schema,
// and populates cfg's config-file fields. It returns the registry value
// declared in the file (empty when absent) for precedence resolution by the
// caller.
//
// A missing config file is not an error: defaults apply.
func loadConfigFile(cfg *GlobalConfig, configPath string) (string, error) {
	ctx := cuecontext.New()
	// Shim until kernel adoption (0006 C2 Phase C): downstream render code
	// still expects a shared CUE context on GlobalConfig.
	cfg.CueContext = ctx

	content, err := os.ReadFile(configPath)
	if err != nil {
		if os.IsNotExist(err) {
			output.Debug("config file not found, using defaults",
				"path", configPath,
			)
			applyDefaults(cfg)
			return "", nil
		}
		return "", fmt.Errorf("reading config file: %w", err)
	}

	value := ctx.CompileBytes(content, cue.Filename(configPath))
	if value.Err() != nil {
		return "", &oerrors.DetailError{
			Type:     "configuration error",
			Message:  value.Err().Error(),
			Location: configPath,
			Hint:     "Run 'opm config vet' to check for configuration errors",
			Cause:    oerrors.ErrValidation,
		}
	}

	// Validate against embedded schema
	if err := validateConfigSchema(ctx, value, configPath); err != nil {
		return "", err
	}

	// Extract config values into cfg
	extractConfigInto(cfg, value)

	// Set default for APIWarnings if not specified
	if cfg.Log.Kubernetes.APIWarnings == "" {
		cfg.Log.Kubernetes.APIWarnings = APIWarningsWarn
	}

	// Extract the file's registry value for precedence resolution.
	configValue := value.LookupPath(cue.ParsePath("config"))
	if !configValue.Exists() {
		configValue = value
	}
	registry := ""
	if regVal := configValue.LookupPath(cue.ParsePath("registry")); regVal.Exists() {
		if str, err := regVal.String(); err == nil {
			registry = str
		}
	}

	return registry, nil
}

// extractConfigInto populates cfg fields from the CUE value.
func extractConfigInto(cfg *GlobalConfig, value cue.Value) {
	// Apply defaults first
	cfg.Kubernetes = KubernetesConfig{
		Kubeconfig: defaultKubeconfig,
		Namespace:  "default",
	}

	// Look for config struct or top-level fields
	configValue := value.LookupPath(cue.ParsePath("config"))
	if !configValue.Exists() {
		// Try top-level fields directly
		configValue = value
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

// applyDefaults fills cfg with built-in defaults for the no-config-file case.
func applyDefaults(cfg *GlobalConfig) {
	cfg.Kubernetes = KubernetesConfig{
		Kubeconfig: defaultKubeconfig,
		Namespace:  "default",
	}
	cfg.Log.Kubernetes.APIWarnings = APIWarningsWarn
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
		return &oerrors.DetailError{
			Type:     "schema validation failed",
			Message:  err.Error(),
			Location: configPath,
			Hint:     removedFieldHint(err.Error()),
			Cause:    oerrors.ErrValidation,
		}
	}

	return nil
}

// removedFieldHint returns a migration hint when the validation error points
// at a field removed by enhancement 0006 D39 (providers, cacheDir), and the
// generic vet hint otherwise.
func removedFieldHint(errMsg string) string {
	switch {
	case strings.Contains(errMsg, "providers"):
		return "The 'providers' field was removed — catalog selection now lives in ~/.opm/platform.cue. Re-run 'opm config init' (or delete the providers block and any ~/.opm/cue.mod/)"
	case strings.Contains(errMsg, "cacheDir"):
		return "The 'cacheDir' field was removed. Re-run 'opm config init' (or delete the field)"
	default:
		return "Check your config.cue against the expected schema. Run 'opm config vet' for validation."
	}
}
