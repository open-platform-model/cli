package cmdutil

import (
	"context"
	"fmt"
	"path/filepath"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/config"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/pkg/engine"
	"github.com/opmodel/cli/pkg/loader"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
)

// RenderResult is the output of RenderRelease, combining engine output with
// release/module metadata for use across all cmd/mod commands.
type RenderResult struct {
	// Resources is the ordered list of Kubernetes resources, already converted
	// to *unstructured.Unstructured for direct use with inventory and k8s packages.
	Resources []*unstructured.Unstructured

	// Release contains release-level metadata (name, namespace, release UUID, labels).
	Release modulerelease.ReleaseMetadata

	// Module contains module-level metadata (canonical name, FQN, version, UUID).
	Module pkgmodule.ModuleMetadata

	// Components contains summary data for each component rendered in this release.
	// Sorted by component name for deterministic output.
	Components []engine.ComponentSummary

	// MatchPlan describes which transformers matched which components.
	// Used for verbose output and debugging.
	MatchPlan *engine.MatchPlan

	// Warnings contains non-fatal warnings (e.g. unhandled traits).
	Warnings []string
}

// HasWarnings returns true if there are warnings.
func (r *RenderResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ResourceCount returns the number of rendered resources.
func (r *RenderResult) ResourceCount() int {
	return len(r.Resources)
}

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
	// via config.ResolveKubernetes before calling RenderRelease.
	K8sConfig *config.ResolvedKubernetesConfig
	// Config is the fully loaded global configuration.
	Config *config.GlobalConfig
}

// RenderRelease executes the common render pipeline shared by build, vet, apply,
// and diff commands. It loads the release package, detects its kind, loads the
// module release, loads the provider, and runs the engine renderer.
//
// On success it returns the RenderResult. On failure it returns an
// *ExitError with the appropriate exit code and Printed flag.
func RenderRelease(ctx context.Context, opts RenderReleaseOpts) (*RenderResult, error) {
	modulePath := ResolveModulePath(opts.Args)

	// Validate config is loaded
	if opts.Config == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}

	// K8sConfig must be pre-resolved by the caller
	if opts.K8sConfig == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("kubernetes config not resolved")}
	}

	namespace := opts.K8sConfig.Namespace.Value
	providerName := opts.K8sConfig.Provider.Value

	// Log resolved config at DEBUG level
	if opts.K8sConfig.Kubeconfig.Value != "" || opts.K8sConfig.Context.Value != "" {
		output.Debug("resolved kubernetes config",
			"kubeconfig", opts.K8sConfig.Kubeconfig.Value,
			"context", opts.K8sConfig.Context.Value,
			"namespace", namespace,
			"provider", providerName,
		)
	} else {
		output.Debug("resolved config",
			"namespace", namespace,
			"provider", providerName,
		)
	}

	cueCtx := opts.Config.CueContext

	// Resolve values file: use first -f flag value if provided, else empty
	// (loader will fall back to values.cue in the module directory).
	var valuesFile string
	if len(opts.Values) > 0 {
		valuesFile = opts.Values[0]
	}

	output.Debug("rendering release",
		"module-path", modulePath,
		"namespace", namespace,
		"provider", providerName,
	)

	// Load the CUE release package (release.cue + values file).
	pkg, _, err := loader.LoadReleasePackage(cueCtx, modulePath, valuesFile)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Detect release kind (ModuleRelease or BundleRelease).
	kind, err := loader.DetectReleaseKind(pkg)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}
	if kind != "ModuleRelease" {
		return nil, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("unsupported release kind %q (use bundle commands for BundleRelease)", kind),
		}
	}

	// Load the ModuleRelease from the evaluated package.
	fallbackName := filepath.Base(modulePath)
	if opts.ReleaseName != "" {
		fallbackName = opts.ReleaseName
	}
	rel, err := loader.LoadModuleReleaseFromValue(cueCtx, pkg, fallbackName)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Override namespace if provided (flag takes precedence over module default).
	if namespace != "" {
		rel.Metadata.Namespace = namespace
	}

	// Load provider from config providers map.
	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	// Run the engine renderer.
	renderer := engine.NewModuleRenderer(p, opts.Config.Matcher)
	engineResult, err := renderer.Render(ctx, rel)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Convert []*core.Resource → []*unstructured.Unstructured at the cmdutil boundary.
	resources := make([]*unstructured.Unstructured, 0, len(engineResult.Resources))
	for _, r := range engineResult.Resources {
		u, convErr := r.ToUnstructured()
		if convErr != nil {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("converting resource %s/%s to unstructured: %w", r.Kind(), r.Name(), convErr),
			}
		}
		resources = append(resources, u)
	}

	return &RenderResult{
		Resources:  resources,
		Release:    *rel.Metadata,
		Module:     *rel.Module.Metadata,
		Components: engineResult.Components,
		MatchPlan:  engineResult.MatchPlan,
		Warnings:   engineResult.Warnings,
	}, nil
}

// ShowOutputOpts controls how ShowRenderOutput displays results.
type ShowOutputOpts struct {
	Verbose bool
}

// ShowRenderOutput shows transformer match output and logs warnings.
// It returns an *ExitError if the result has errors.
func ShowRenderOutput(result *RenderResult, opts ShowOutputOpts) error {
	// Show transformer matches
	switch {
	case opts.Verbose:
		WriteVerboseMatchLog(result)
	default:
		WriteTransformerMatches(result)
	}

	// Log warnings
	releaseLog := output.ReleaseLogger(result.Release.Name)
	for _, w := range result.Warnings {
		releaseLog.Warn(w)
	}

	return nil
}
