package render

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	opmexit "github.com/opmodel/cli/internal/exit"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	internalreleasefile "github.com/opmodel/cli/internal/releasefile"
	"github.com/opmodel/cli/internal/releaseprocess"
	"github.com/opmodel/cli/internal/runtime/modulerelease"
	"github.com/opmodel/cli/pkg/loader"
	pkgmodule "github.com/opmodel/cli/pkg/module"
)

func Release(ctx context.Context, opts ReleaseOpts) (*Result, error) { //nolint:gocyclo
	modulePath := resolveModulePath(opts.Args)

	if opts.Config == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}
	if opts.K8sConfig == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("kubernetes config not resolved")}
	}

	namespace := opts.K8sConfig.Namespace.Value
	providerName := opts.K8sConfig.Provider.Value

	if opts.K8sConfig.Kubeconfig.Value != "" || opts.K8sConfig.Context.Value != "" {
		output.Debug("resolved kubernetes config",
			"kubeconfig", opts.K8sConfig.Kubeconfig.Value,
			"context", opts.K8sConfig.Context.Value,
			"namespace", namespace,
			"provider", providerName,
		)
	} else {
		output.Debug("resolved config", "namespace", namespace, "provider", providerName)
	}

	cueCtx := opts.Config.CueContext
	output.Debug("rendering release", "module-path", modulePath, "namespace", namespace, "provider", providerName)

	var (
		rel        *modulerelease.ModuleRelease
		valuesVals []cue.Value
	)

	if !hasReleaseFile(modulePath) {
		modVal, modErr := loader.LoadModulePackage(cueCtx, modulePath)
		if modErr != nil {
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("loading module: %w", modErr)}
		}

		switch {
		case len(opts.Values) > 0:
			var loadErr error
			valuesVals, loadErr = loadValuesFiles(cueCtx, opts.Values)
			if loadErr != nil {
				return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: loadErr}
			}
		case opts.DebugValues:
			debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
			if !debugVal.Exists() {
				return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("no release.cue found - add debugValues to module or use -f <values-file>")}
			}
			if err := debugVal.Validate(cue.Concrete(true)); err != nil {
				printValidationError("debugValues not concrete", err)
				return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: fmt.Errorf("debugValues is not concrete - module must provide complete test values"), Printed: true}
			}
			valuesVals = []cue.Value{debugVal}
		default:
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("no release.cue found - use -f <values-file> to provide values")}
		}

		releaseName := opts.ReleaseName
		if releaseName == "" {
			if nameVal := modVal.LookupPath(cue.ParsePath("metadata.name")); nameVal.Exists() {
				if n, strErr := nameVal.String(); strErr == nil && n != "" {
					releaseName = n
				}
			}
		}
		if releaseName == "" {
			releaseName = filepath.Base(modulePath)
		}

		moduleNamespace := namespace
		s := opts.K8sConfig.Namespace.Source
		if s != config.SourceFlag && s != config.SourceEnv {
			if nsVal := modVal.LookupPath(cue.ParsePath("metadata.defaultNamespace")); nsVal.Exists() {
				if ns, strErr := nsVal.String(); strErr == nil && ns != "" {
					moduleNamespace = ns
				}
			}
		}

		var synthErr error
		rel, synthErr = releaseprocess.SynthesizeModuleRelease(cueCtx, modVal, valuesVals, releaseName, moduleNamespace)
		if synthErr != nil {
			printValidationError("render failed", synthErr)
			return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: synthErr, Printed: true}
		}
	} else {
		var loadErr error
		rel, valuesVals, loadErr = loadModuleReleaseForRender(cueCtx, modulePath, opts.Values, opts.DebugValues, opts.ReleaseName)
		if loadErr != nil {
			var exitErr *opmexit.ExitError
			if ok := errors.As(loadErr, &exitErr); ok {
				return nil, exitErr
			}
			printValidationError("render failed", loadErr)
			return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: loadErr, Printed: true}
		}
	}

	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		rel.Metadata.Namespace = namespace
	}

	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	engineResult, err := releaseprocess.ProcessModuleRelease(ctx, rel, valuesVals, p)
	if err != nil {
		printValidationError("render failed", err)
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

func ReleaseFile(ctx context.Context, opts ReleaseFileOpts) (*Result, error) { //nolint:gocyclo
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
	fileRelease, err := internalreleasefile.GetReleaseFile(cueCtx, opts.ReleaseFilePath)
	if err != nil {
		printValidationError("render failed", err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	if fileRelease.Kind == internalreleasefile.KindBundleRelease {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("bundle releases are not yet supported - use a #ModuleRelease file")}
	}
	rel := fileRelease.Module

	if opts.ModulePath != "" {
		modVal, modErr := loader.LoadModulePackage(cueCtx, opts.ModulePath)
		if modErr != nil {
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("loading module from --module: %w", modErr)}
		}
		rel.RawCUE = rel.RawCUE.FillPath(cue.MakePath(cue.Def("module")), modVal)
		rel.Module.Raw = modVal
		rel.Module.Config = modVal.LookupPath(cue.ParsePath("#config"))
		rel.Config = rel.RawCUE.LookupPath(cue.ParsePath("#module.#config"))
		if rel.Module.Metadata == nil {
			rel.Module.Metadata = &pkgmodule.ModuleMetadata{}
			if err := modVal.LookupPath(cue.ParsePath("metadata")).Decode(rel.Module.Metadata); err != nil {
				return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("decoding module metadata from --module: %w", err)}
			}
		}
		if err := rel.RawCUE.Err(); err != nil {
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("filling #module from --module: %w", err)}
		}
	}

	moduleVal := rel.RawCUE.LookupPath(cue.MakePath(cue.Def("module")))
	if !moduleVal.Exists() || moduleVal.Validate(cue.Concrete(true)) != nil {
		if opts.ModulePath == "" {
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("#module is not filled in the release file - either import a module or use --module <path>")}
		}
	}

	valuesVals, err := resolveReleaseValues(cueCtx, rel.RawCUE, opts.ReleaseFilePath, opts.ValuesFiles)
	if err != nil {
		printValidationError("render failed", err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		rel.Metadata.Namespace = namespace
	}

	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	engineResult, err := releaseprocess.ProcessModuleRelease(ctx, rel, valuesVals, p)
	if err != nil {
		printValidationError("render failed", err)
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

func resolveModulePath(args []string) string {
	if len(args) == 0 {
		return "."
	}
	return args[0]
}
