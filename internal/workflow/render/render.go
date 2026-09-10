package render

import (
	"context"
	"encoding/json"
	"fmt"
	"path/filepath"
	"sort"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	pkgcore "github.com/open-platform-model/cli/pkg/core"
	"github.com/open-platform-model/cli/pkg/loader"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
)

// FromInstanceFile prepares and renders an instance from a declarative
// #ModuleInstance CUE package through the library kernel (0006 D9). The
// package directory containing the instance file is acquired as one CUE
// package (instance.cue + values.cue + overlays) with any -f values files
// passed as the acquire's trailing values sources, so the instance the render
// imports already carries them; the kernel then renders it against the
// resolved platform in one build.
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

	// Acquire the instance package (the directory containing the instance
	// file) with the -f files layered as values sources: the schema's own
	// values unification performs the merge inside the build, nothing is
	// filled from Go, and a conflict names the file it came from.
	instanceDir, err := resolveInstanceDir(opts.InstanceFilePath)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	sources, err := loadValuesSources(k, opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	inst, err := k.AcquireInstanceFromDir(ctx, instanceDir, sources...)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// Render provenance (enhancement 0006 D7): an instance apply is local when
	// its module's cue.mod/local-module.cue replaces a dependency; otherwise it
	// resolves from registries. The same module root is the D19 module
	// context the render's replacement warnings are worded against.
	moduleRoot := moduleContextRoot(filepath.Dir(opts.InstanceFilePath))
	sourceLocal := loader.HasLocalModuleReplacement(moduleRoot)

	// Platform resolution + acquisition only after the instance itself
	// validated: cheap failures never hit the cluster or registry.
	env, err := resolvePlatformEnv(ctx, k, opts.Config, opts.PlatformFlag, opts.ClusterPlatform)
	if err != nil {
		return nil, err
	}

	return renderInstance(ctx, env, inst, opts.K8sConfig, moduleRoot, sourceLocal)
}

// moduleContextRoot is the effective module context of a render entry: the
// module root (nearest cue.mod/module.cue) above dir — the directory holding
// an instance file, or the module directory itself. "" when dir is under no
// module root. It is where a developer's cue.mod/local-module.cue lives, so
// it drives both the render provenance signal (0006 D7) and the wording of
// the render's replacement warnings (0010 D19).
func moduleContextRoot(dir string) string {
	abs, err := filepath.Abs(dir)
	if err != nil {
		return ""
	}
	return loader.ModuleRootFrom(abs)
}

// newRenderInput assembles the kernel's render input for the resolved
// environment. LocalReplacements is always on: the CLI renders a developer's
// checkout, so a cue.mod/local-module.cue in the module context or the
// platform module is the developer's request and the render honors it (the
// kernel refuses such an input when the switch is off); the kernel reports
// what it honored on the diagnostics and renderInstance words the rows.
func newRenderInput(env *renderEnv, inst *module.Instance) kernel.RenderInput {
	return kernel.RenderInput{
		Instance:          inst,
		Platform:          env.platform,
		RuntimeName:       RuntimeName,
		Skew:              env.skew,
		LocalReplacements: true,
	}
}

// renderInstance runs the kernel's single render verb on a source-carrying
// instance and adapts the result to the workflow Result. Every failure the
// kernel reports — a skew refusal before evaluation, a local-replacement
// refusal, or the fail-closed gate after it (unresolved demands, unmatched
// components, an over-subscribed provider contract, a failed pair) — exits
// as a validation failure with the kernel's message and the diagnostics
// printed beside it. After a successful render the D19 replacement warnings
// are emitted from the kernel's rows against moduleRoot, the module context.
func renderInstance(
	ctx context.Context,
	env *renderEnv,
	inst *module.Instance,
	k8sCfg *config.ResolvedKubernetesConfig,
	moduleRoot string,
	sourceLocal bool,
) (*Result, error) {
	out, err := env.kernel.Render(ctx, newRenderInput(env, inst))
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	for _, w := range replacementWarnings(out.Diagnostics.Replacements, moduleRoot) {
		output.Warn(w)
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

	renderDigest, err := inventory.ComputeRenderDigest(converted)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	result := newResult(env, out, renderDigest, decodeUnifiedValues(inst.Package.LookupPath(schema.Values)), sourceLocal)

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

	// Module metadata decoded from the embedded #module value (carries the
	// full registry modulePath for the canonical spec.module reference).
	result.Module = decodeModuleMetadata(inst.Package.LookupPath(schema.Module))

	return result, nil
}

// newResult assembles the workflow Result from the render output and the
// render environment. PlatformSpec is the seed document decoded from the
// exact built platform the render consumed — the D12 write-if-absent seeding
// writes it verbatim, with no re-read of the platform module at apply time.
// Warnings are the render's advisory facts worded by the CLI from the
// diagnostics rows (unhandled optional traits, skew under the warn policy);
// the D19 local-replacement warnings are emitted directly by renderInstance
// from the replacement rows, after the render.
func newResult(env *renderEnv, out *kernel.RenderResult, renderDigest string, values map[string]any, sourceLocal bool) *Result {
	return &Result{
		Pairs:        out.Diagnostics.Pairs,
		Warnings:     formatAdvisories(out.Diagnostics),
		Platform:     env.resolution,
		PlatformSpec: env.spec,
		RenderDigest: renderDigest,
		Values:       values,
		SourceLocal:  sourceLocal,
	}
}

// formatAdvisories words the render's advisory diagnostics as the CLI's
// warnings (the kernel-render spec): one line per resolved-versions row
// marked Newer — the warn skew policy let the render proceed against the
// platform's build — followed by one line per unhandled optional trait, in
// component order. The kernel reports both as rows and attaches no message
// strings; the sentences are the CLI's, kept identical to the ones the kernel
// used to word so existing output does not move. Never nil: no advisories is
// an empty list.
func formatAdvisories(d kernel.RenderDiagnostics) []string {
	warnings := []string{}
	for _, r := range d.ResolvedVersions {
		if !r.Newer {
			continue
		}
		warnings = append(warnings, fmt.Sprintf(
			"version skew on %q: module requires %s, platform carries %s; rendering against the platform's build",
			r.Path, r.ModuleVersion, r.PlatformVersion))
	}
	comps := make([]string, 0, len(d.UnhandledTraits))
	for c := range d.UnhandledTraits {
		comps = append(comps, c)
	}
	sort.Strings(comps)
	for _, c := range comps {
		for _, fqn := range d.UnhandledTraits[c] {
			warnings = append(warnings, fmt.Sprintf(
				"component %q: trait %q is not handled by any matched transformer (values will be ignored)", c, fqn))
		}
	}
	return warnings
}

// decodeModuleMetadata decodes the CLI's module metadata from a module CUE
// value.
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
