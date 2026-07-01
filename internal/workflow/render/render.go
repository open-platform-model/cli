package render

import (
	"context"
	"fmt"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	internalinstancefile "github.com/open-platform-model/cli/internal/instancefile"
	"github.com/open-platform-model/cli/internal/output"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
	"github.com/open-platform-model/cli/pkg/loader"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
	"github.com/open-platform-model/cli/pkg/provider"
	pkgrender "github.com/open-platform-model/cli/pkg/render"
)

// FromInstanceFile prepares and renders an instance from a declarative #ModuleInstance CUE file.
// It parses the instance file, resolves values, and renders through the pipeline. The instance
// file must import a module to fill #module. This is typically used by platform operators to
// deploy configured instances of modules (e.g., via "opm instance apply my-app-instance.cue").
func FromInstanceFile(ctx context.Context, opts InstanceFileOpts) (*Result, error) {
	if opts.Config == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}
	if opts.K8sConfig == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("kubernetes config not resolved")}
	}
	if opts.InstanceFilePath == "" {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("instance file path is required")}
	}

	namespace := opts.K8sConfig.Namespace.Value
	providerName := opts.K8sConfig.Provider.Value

	output.Debug("rendering from instance file", "file", opts.InstanceFilePath, "namespace", namespace, "provider", providerName)

	cueCtx := opts.Config.CueContext
	if pathErr := cmdutil.ValidateInstanceInputPath(opts.InstanceFilePath); pathErr != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: pathErr}
	}
	fileInstance, err := internalinstancefile.GetInstanceFile(cueCtx, opts.InstanceFilePath)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	parseData := fileInstance.Module

	// Verify #module is filled in the instance file.
	moduleVal := parseData.Spec.LookupPath(cue.MakePath(cue.Def("module")))
	if !moduleVal.Exists() || moduleVal.Validate(cue.Concrete(true)) != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("#module is not filled in the instance file — import a module to fill it")}
	}

	// Resolve values from --values files, auto-discovered values.cue, or inline values.
	valuesVals, err := resolveInstanceValues(cueCtx, parseData.Spec, opts.InstanceFilePath, opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// Prepare the instance: validate values, fill, check concreteness, decode metadata.
	rel, err := pkgmodule.ParseModuleInstance(ctx, parseData.Spec, parseData.Module, valuesVals)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// Apply namespace override from --namespace flag or env.
	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		rel.Metadata.Namespace = namespace
	}

	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	return renderPreparedModuleInstance(ctx, rel, p)
}

// renderPreparedModuleInstance runs the render engine on a fully prepared instance
// and converts the result to unstructured resources.
func renderPreparedModuleInstance(
	ctx context.Context,
	rel *pkgmodule.Instance,
	p *provider.Provider,
) (*Result, error) {
	engineResult, err := pkgrender.ProcessModuleInstance(ctx, rel, p, pkgcore.LabelManagedByValue)
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
		Instance:   *rel.Metadata,
		Module:     *rel.Module.Metadata,
		Components: engineResult.Components,
		MatchPlan:  engineResult.MatchPlan,
		Warnings:   engineResult.Warnings,
	}, nil
}

func ShowOutput(result *Result, opts ShowOutputOpts) {
	showOutput(result, opts)
}
