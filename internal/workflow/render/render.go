package render

import (
	"context"
	"fmt"

	opmexit "github.com/opmodel/cli/internal/exit"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/cmdutil"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	internalreleasefile "github.com/opmodel/cli/internal/releasefile"
	"github.com/opmodel/cli/pkg/loader"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/provider"
	pkgrender "github.com/opmodel/cli/pkg/render"
)

// FromReleaseFile prepares and renders a release from a declarative #ModuleRelease CUE file.
// It parses the release file, resolves values, and renders through the pipeline. The release
// file must import a module to fill #module. This is typically used by platform operators to
// deploy configured instances of modules (e.g., via "opm release apply my-app-release.cue").
func FromReleaseFile(ctx context.Context, opts ReleaseFileOpts) (*Result, error) {
	if opts.Config == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}
	if opts.K8sConfig == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("kubernetes config not resolved")}
	}
	if opts.ReleaseFilePath == "" {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("release file path is required")}
	}

	namespace := opts.K8sConfig.Namespace.Value
	providerName := opts.K8sConfig.Provider.Value

	output.Debug("rendering from release file", "file", opts.ReleaseFilePath, "namespace", namespace, "provider", providerName)

	cueCtx := opts.Config.CueContext
	if pathErr := cmdutil.ValidateReleaseInputPath(opts.ReleaseFilePath); pathErr != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: pathErr}
	}
	fileRelease, err := internalreleasefile.GetReleaseFile(cueCtx, opts.ReleaseFilePath)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	if fileRelease.Kind == internalreleasefile.KindBundleRelease {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("bundle releases are not yet supported - use a #ModuleRelease file")}
	}
	rel := fileRelease.Module

	moduleVal := rel.RawCUE.LookupPath(cue.MakePath(cue.Def("module")))
	if !moduleVal.Exists() || moduleVal.Validate(cue.Concrete(true)) != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("#module is not filled in the release file — import a module to fill it")}
	}

	valuesVals, err := resolveReleaseValues(cueCtx, rel.RawCUE, opts.ReleaseFilePath, opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	var namespaceOverride string
	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		namespaceOverride = namespace
	}

	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	return renderPreparedModuleRelease(ctx, rel, valuesVals, p, namespaceOverride)
}

// renderPreparedModuleRelease is the execution tail for FromReleaseFile.
// It applies the namespace override, runs the render engine, and converts the result to unstructured resources.
func renderPreparedModuleRelease(
	ctx context.Context,
	rel *pkgmodule.Release,
	valuesVals []cue.Value,
	p *provider.Provider,
	namespaceOverride string,
) (*Result, error) {
	if namespaceOverride != "" {
		rel.Metadata.Namespace = namespaceOverride
	}

	engineResult, err := pkgrender.ProcessModuleRelease(ctx, rel, valuesVals, p)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	resources := make([]*unstructured.Unstructured, 0, len(engineResult.Resources))
	for _, r := range engineResult.Resources {
		u, convErr := r.ToUnstructured()
		if convErr != nil {
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("converting resource %s/%s to unstructured: %w", r.Kind(), r.Name(), convErr)}
		}
		resources = append(resources, u)
	}

	return &Result{
		Resources:  resources,
		Release:    *rel.Metadata,
		Module:     *rel.Module.Metadata,
		Components: engineResult.Components,
		MatchPlan:  engineResult.MatchPlan,
		Warnings:   engineResult.Warnings,
	}, nil
}

func ShowOutput(result *Result, opts ShowOutputOpts) {
	showOutput(result, opts)
}
