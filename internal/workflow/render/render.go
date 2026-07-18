package render

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
	"github.com/open-platform-model/cli/pkg/loader"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
)

// FromInstanceFile prepares and renders an instance from a declarative
// #ModuleInstance CUE package through the library kernel (0006 D9). The
// package directory containing the instance file is loaded as one CUE
// package (instance.cue + values.cue + overlays), the embedded #module is
// decoded, and the kernel validates, matches, and compiles against the
// resolved platform.
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
	if pathErr := cmdutil.ValidateInstanceInputPath(opts.InstanceFilePath); pathErr != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: pathErr}
	}

	output.Debug("rendering from instance file", "file", opts.InstanceFilePath, "namespace", opts.K8sConfig.Namespace.Value)

	k := NewKernel(opts.Config)

	// Load the instance package (the directory containing the instance file).
	instanceDir, err := resolveInstanceDir(opts.InstanceFilePath)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	instVal, err := k.LoadInstancePackage(ctx, instanceDir, loaderfile.LoadOptions{Registry: opts.Config.Registry})
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// The embedded #module must be filled by the package's own import.
	moduleVal := instVal.LookupPath(schema.Module)
	if !moduleVal.Exists() {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("#module is not filled in the instance package — import a module to fill it")}
	}
	mod, err := k.NewModuleFromValue(moduleVal)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// Values: -f files (unified) win; otherwise the package's own values
	// (values.cue / inline) already live in the loaded package and
	// ProcessModuleInstance enforces concreteness.
	values, err := unifyValuesFiles(k.CueContext(), opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	inst, err := k.ProcessModuleInstance(ctx, instVal, *mod, values)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// Render provenance (enhancement 0006 D7): an instance apply is local when
	// its module's cue.mod/local-module.cue replaces a dependency; otherwise it
	// resolves from registries.
	sourceLocal := false
	if abs, absErr := filepath.Abs(opts.InstanceFilePath); absErr == nil {
		sourceLocal = loader.HasLocalModuleReplacement(loader.ModuleRootFrom(filepath.Dir(abs)))
	}

	// Platform resolution + materialization only after the instance itself
	// validated: cheap failures never hit the cluster or registry.
	env, err := resolvePlatformEnv(ctx, k, opts.Config, opts.PlatformFlag, opts.ClusterPlatform)
	if err != nil {
		return nil, err
	}

	return compileInstance(ctx, env, inst, opts.K8sConfig, sourceLocal)
}

// compileInstance runs the kernel compile on a processed instance and adapts
// the result to the workflow Result.
func compileInstance(
	ctx context.Context,
	env *renderEnv,
	inst *module.Instance,
	k8sCfg *config.ResolvedKubernetesConfig,
	sourceLocal bool,
) (*Result, error) {
	out, err := env.kernel.Compile(ctx, kernel.CompileInput{
		ModuleInstance: inst,
		Platform:       env.platform,
		RuntimeName:    RuntimeName,
	})
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	converted := make([]*pkgcore.Resource, 0, len(out.Compiled))
	for _, c := range out.Compiled {
		converted = append(converted, &pkgcore.Resource{
			Value:       c.Value,
			Instance:    c.Instance,
			Component:   c.Component,
			Transformer: c.Transformer,
		})
	}

	result := &Result{
		Components:  out.Components,
		MatchPlan:   out.MatchPlan,
		Warnings:    out.Warnings,
		Platform:    env.resolution,
		Values:      decodeUnifiedValues(inst.Package.LookupPath(schema.Values)),
		SourceLocal: sourceLocal,
	}

	for _, r := range converted {
		u, convErr := r.ToUnstructured()
		if convErr != nil {
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("converting resource %s/%s to unstructured: %w", r.Kind(), r.Name(), convErr)}
		}
		result.Resources = append(result.Resources, u)
	}

	// Instance metadata from the kernel's decode; namespace flag/env override
	// applies to the apply target, mirroring the legacy pipeline.
	if inst.Metadata != nil {
		result.Instance = pkgmodule.InstanceMetadata{
			Name:      inst.Metadata.Name,
			Namespace: inst.Metadata.Namespace,
			UUID:      inst.Metadata.UUID,
			Labels:    inst.Metadata.Labels,
		}
	}
	if k8sCfg != nil {
		if s := k8sCfg.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
			result.Instance.Namespace = k8sCfg.Namespace.Value
		}
	}

	// Module metadata decoded from the embedded #module value (carries
	// nameSnakeCase for the canonical spec.module reference — D6/D37).
	result.Module = decodeModuleMetadata(inst.Package.LookupPath(schema.Module))

	return result, nil
}

// decodeModuleMetadata decodes the CLI's module metadata (including
// nameSnakeCase) from a module CUE value.
func decodeModuleMetadata(moduleVal cue.Value) pkgmodule.ModuleMetadata {
	meta := pkgmodule.ModuleMetadata{}
	if !moduleVal.Exists() {
		return meta
	}
	if mv := moduleVal.LookupPath(cue.ParsePath("metadata")); mv.Exists() {
		// Best-effort decode: leaves zero-value fields if metadata is partial.
		if err := mv.Decode(&meta); err != nil {
			output.Debug("could not decode module metadata", "err", err)
		}
	}
	return meta
}

// decodeUnifiedValues converts the instance's concrete, merged values into a
// JSON-shaped map for the ModuleInstance CR's spec.values. A non-existent or
// undecodable value yields nil (spec.values omitted).
func decodeUnifiedValues(v cue.Value) map[string]any {
	if !v.Exists() {
		return nil
	}
	data, err := v.MarshalJSON()
	if err != nil {
		output.Debug("could not encode instance values for spec.values", "err", err)
		return nil
	}
	var m map[string]any
	if err := json.Unmarshal(data, &m); err != nil {
		output.Debug("could not decode instance values for spec.values", "err", err)
		return nil
	}
	return m
}

func ShowOutput(result *Result, opts ShowOutputOpts) {
	showOutput(result, opts)
}
