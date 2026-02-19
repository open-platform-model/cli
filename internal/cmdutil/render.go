package cmdutil

import (
	"context"
	"fmt"

	"github.com/opmodel/cli/internal/build"
	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/internal/errors"
	"github.com/opmodel/cli/internal/output"
)

// RenderReleaseOpts holds the inputs for RenderRelease.
type RenderReleaseOpts struct {
	// Args from the cobra command (first arg is module path).
	Args []string
	// Values files (-f flags).
	Values []string
	// ReleaseName overrides the default release name.
	ReleaseName string
	// K8sConfig is the pre-resolved Kubernetes configuration.
	// All fields (namespace, provider, kubeconfig, context) must already be resolved
	// via cmdutil.ResolveKubernetes before calling RenderRelease.
	K8sConfig *config.ResolvedKubernetesConfig
	// OPMConfig loaded by the root command.
	OPMConfig *config.OPMConfig
	// Registry URL resolved by the root command.
	Registry string
}

// RenderRelease executes the common render pipeline preamble shared by
// build, vet, apply, and diff commands. It resolves the module path,
// validates config, builds RenderOptions from the pre-resolved K8s config,
// creates the pipeline, and executes Render.
//
// On success it returns the RenderResult. On failure it returns an
// *ExitError with the appropriate exit code and Printed flag.
func RenderRelease(ctx context.Context, opts RenderReleaseOpts) (*build.RenderResult, error) {
	modulePath := ResolveModulePath(opts.Args)

	// Validate OPM config is loaded
	if opts.OPMConfig == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}

	// K8sConfig must be pre-resolved by the caller
	if opts.K8sConfig == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("kubernetes config not resolved")}
	}

	namespace := opts.K8sConfig.Namespace.Value
	provider := opts.K8sConfig.Provider.Value

	// Log resolved config at DEBUG level
	if opts.K8sConfig.Kubeconfig.Value != "" || opts.K8sConfig.Context.Value != "" {
		output.Debug("resolved kubernetes config",
			"kubeconfig", opts.K8sConfig.Kubeconfig.Value,
			"context", opts.K8sConfig.Context.Value,
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
	renderOpts := build.RenderOptions{
		ModulePath: modulePath,
		Values:     opts.Values,
		Name:       opts.ReleaseName,
		Namespace:  namespace,
		Provider:   provider,
		Registry:   opts.Registry,
	}

	if err := renderOpts.Validate(); err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}

	// Create and execute pipeline
	pipeline := build.NewPipeline(opts.OPMConfig)

	output.Debug("rendering release",
		"module-path", modulePath,
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
	Verbose bool
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
	case opts.Verbose:
		WriteVerboseMatchLog(result)
	default:
		WriteTransformerMatches(result)
	}

	// Log warnings
	releaseLog := output.ReleaseLogger(result.Release.Name)
	if result.HasWarnings() {
		for _, w := range result.Warnings {
			releaseLog.Warn(w)
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
