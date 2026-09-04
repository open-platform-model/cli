// Package config provides configuration loading and management.
package config

import (
	"fmt"
	"os"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/parser"

	"github.com/open-platform-model/cli/internal/output"
	oerrors "github.com/open-platform-model/cli/pkg/errors"
)

// defaultKubeconfig is the built-in kubeconfig path default.
const defaultKubeconfig = "~/.kube/config"

// configErrType is the DetailError type used for config file parse/load
// failures.
const configErrType = "configuration error"

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
	cfg.RegistryResolution = registryResult

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

	// Data-only contract (D39): reject CUE imports before evaluation.
	// CompileBytes would happily resolve stdlib imports otherwise. (The
	// platform beside this file is a CUE module since 0019 D5 and is built,
	// not parsed as data.)
	astFile, err := parser.ParseFile(configPath, content)
	if err != nil {
		return "", &oerrors.DetailError{
			Type:     configErrType,
			Message:  err.Error(),
			Location: configPath,
			Hint:     "Run 'opm config vet' to check for configuration errors",
			Cause:    oerrors.ErrValidation,
		}
	}
	if fileHasImports(astFile) {
		return "", &oerrors.DetailError{
			Type:     configErrType,
			Message:  "the config file must be data-only — CUE imports are not allowed",
			Location: configPath,
			Hint:     "Remove the import declarations (config.cue is scalar data since 0006 D39); re-run 'opm config init' for a fresh template",
			Cause:    oerrors.ErrValidation,
		}
	}

	value := ctx.CompileBytes(content, cue.Filename(configPath))
	if value.Err() != nil {
		return "", &oerrors.DetailError{
			Type:     configErrType,
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

	// Absent skewPolicy means warn (0019 D18).
	if cfg.SkewPolicy == "" {
		cfg.SkewPolicy = SkewPolicyWarn
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
	if k8sValue := configValue.LookupPath(cue.ParsePath("kubernetes")); k8sValue.Exists() {
		setString(&cfg.Kubernetes.Kubeconfig, k8sValue, "kubeconfig")
		setString(&cfg.Kubernetes.Context, k8sValue, "context")
		setString(&cfg.Kubernetes.Namespace, k8sValue, "namespace")
	}

	// Extract skewPolicy (validated against the schema's disjunction already).
	setString(&cfg.SkewPolicy, configValue, "skewPolicy")

	// Extract log config
	if logValue := configValue.LookupPath(cue.ParsePath("log")); logValue.Exists() {
		if tsVal := logValue.LookupPath(cue.ParsePath("timestamps")); tsVal.Exists() {
			if b, err := tsVal.Bool(); err == nil {
				cfg.Log.Timestamps = &b
			}
		}
		setString(&cfg.Log.Kubernetes.APIWarnings, logValue, "kubernetes.apiWarnings")
	}
}

// setString stores the string at path under parent into dst when the field
// exists and is a string; otherwise dst keeps its current (default) value.
func setString(dst *string, parent cue.Value, path string) {
	v := parent.LookupPath(cue.ParsePath(path))
	if !v.Exists() {
		return
	}
	if str, err := v.String(); err == nil {
		*dst = str
	}
}

// applyDefaults fills cfg with built-in defaults for the no-config-file case.
func applyDefaults(cfg *GlobalConfig) {
	cfg.Kubernetes = KubernetesConfig{
		Kubeconfig: defaultKubeconfig,
		Namespace:  "default",
	}
	cfg.Log.Kubernetes.APIWarnings = APIWarningsWarn
	cfg.SkewPolicy = SkewPolicyWarn
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

// removedFieldHint returns a targeted hint when the validation error points
// at a field removed by enhancement 0006 D39 (providers, cacheDir) or at a
// closed-enum key whose CUE error elides the allowed values (skewPolicy),
// and the generic vet hint otherwise.
func removedFieldHint(errMsg string) string {
	switch {
	case strings.Contains(errMsg, "providers"):
		return "The 'providers' field was removed — catalog selection now lives in the platform module ~/.opm/platform/. Re-run 'opm config init' (or delete the providers block and any ~/.opm/cue.mod/)"
	case strings.Contains(errMsg, "cacheDir"):
		return "The 'cacheDir' field was removed. Re-run 'opm config init' (or delete the field)"
	case strings.Contains(errMsg, "skewPolicy"):
		return fmt.Sprintf("skewPolicy must be %q or %q (default %q when omitted)", SkewPolicyWarn, SkewPolicyRefuse, SkewPolicyWarn)
	default:
		return "Check your config.cue against the expected schema. Run 'opm config vet' for validation."
	}
}
