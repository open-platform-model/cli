package cmdutil

import (
	"context"
	"fmt"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

// RenderModuleOpts holds the inputs for RenderModule.
type RenderModuleOpts struct {
	// Args from the cobra command (first arg is module path).
	Args []string
	// Render flags (values, namespace, release-name, provider).
	Render *RenderFlags
	// K8s connection flags (optional — only needed for kubeconfig/context resolution).
	K8s *K8sFlags
	// OPMConfig loaded by the root command.
	OPMConfig *config.OPMConfig
	// Registry URL resolved by the root command.
	Registry string
}

// RenderModule executes the common render pipeline preamble shared by
// build, vet, apply, and diff commands. It resolves the module path,
// validates config, resolves K8s settings, builds RenderOptions,
// creates the pipeline, and executes Render.
//
// On success it returns the RenderResult. On failure it returns an
// *ExitError with the appropriate exit code and Printed flag.
func RenderModule(ctx context.Context, opts RenderModuleOpts) (*build.RenderResult, error) {
	modulePath := ResolveModulePath(opts.Args)

	// Validate OPM config is loaded
	if opts.OPMConfig == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}

	// Resolve Kubernetes configuration
	kubeconfigFlag := ""
	contextFlag := ""
	if opts.K8s != nil {
		kubeconfigFlag = opts.K8s.Kubeconfig
		contextFlag = opts.K8s.Context
	}

	namespaceFlag := ""
	providerFlag := ""
	if opts.Render != nil {
		namespaceFlag = opts.Render.Namespace
		providerFlag = opts.Render.Provider
	}

	k8sConfig, err := ResolveKubernetes(opts.OPMConfig, kubeconfigFlag, contextFlag, namespaceFlag, providerFlag)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("resolving kubernetes config: %w", err)}
	}

	namespace := k8sConfig.Namespace.Value
	provider := k8sConfig.Provider.Value

	// Log resolved config at DEBUG level
	if opts.K8s != nil {
		output.Debug("resolved kubernetes config",
			"kubeconfig", k8sConfig.Kubeconfig.Value,
			"context", k8sConfig.Context.Value,
			"namespace", namespace,
			"provider", provider,
		)
	} else {
		output.Debug("resolved config",
			"namespace", namespace,
			"provider", provider,
		)
	}

	// Build render options
	var values []string
	var releaseName string
	if opts.Render != nil {
		values = opts.Render.Values
		releaseName = opts.Render.ReleaseName
	}

	renderOpts := build.RenderOptions{
		ModulePath: modulePath,
		Values:     values,
		Name:       releaseName,
		Namespace:  namespace,
		Provider:   provider,
		Registry:   opts.Registry,
	}

	if err := renderOpts.Validate(); err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	// Create and execute pipeline
	pipeline := build.NewPipeline(opts.OPMConfig)

	output.Debug("rendering module",
		"module", modulePath,
		"namespace", namespace,
		"provider", provider,
	)

	result, err := pipeline.Render(ctx, renderOpts)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	return result, nil
}

// ShowOutputOpts controls how ShowRenderOutput displays results.
type ShowOutputOpts struct {
	Verbose     bool
	VerboseJSON bool
}

// ShowRenderOutput checks for render errors, shows transformer match output,
// and logs warnings. It returns an *ExitError if the result has errors.
func ShowRenderOutput(result *build.RenderResult, opts ShowOutputOpts) error {
	// Check for render errors
	if result.HasErrors() {
		PrintRenderErrors(result.Errors)
		return &oerrors.ExitError{
			Code:    oerrors.ExitValidationError,
			Err:     fmt.Errorf("%d render error(s)", len(result.Errors)),
			Printed: true,
		}
	}

	// Show transformer matches
	switch {
	case opts.VerboseJSON:
		WriteBuildVerboseJSON(result)
	case opts.Verbose:
		WriteVerboseMatchLog(result)
	default:
		WriteTransformerMatches(result)
	}

	// Log warnings
	modLog := output.ModuleLogger(result.Module.Name)
	if result.HasWarnings() {
		for _, w := range result.Warnings {
			modLog.Warn(w)
		}
	}

	return nil
}

// ResolveKubernetes resolves Kubernetes configuration values from flags,
// environment, and config. It accepts an OPMConfig and individual flag values
// (which may be empty strings) and returns resolved K8s config.
func ResolveKubernetes(opmConfig *config.OPMConfig, kubeconfigFlag, contextFlag, namespaceFlag, providerFlag string) (*config.ResolvedKubernetesConfig, error) {
	var cfg *config.Config
	var providerNames []string

	if opmConfig != nil {
		cfg = opmConfig.Config
		if opmConfig.Providers != nil {
			for name := range opmConfig.Providers {
				providerNames = append(providerNames, name)
			}
		}
	}

	return config.ResolveKubernetes(config.ResolveKubernetesOptions{
		KubeconfigFlag: kubeconfigFlag,
		ContextFlag:    contextFlag,
		NamespaceFlag:  namespaceFlag,
		ProviderFlag:   providerFlag,
		Config:         cfg,
		ProviderNames:  providerNames,
	})
}
