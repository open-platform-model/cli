package render

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"path/filepath"
	"sort"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/k8s/object"
	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/inventory"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	pkgerrors "github.com/open-platform-model/cli/pkg/errors"
	"github.com/open-platform-model/cli/pkg/loader"
)

// FromInstanceFile prepares and renders an instance from a declarative
// #ModuleInstance CUE package through the library kernel (0006:D9). The
// instance package, the named directory or the directory holding the named
// .cue file, is acquired as one CUE package (instance.cue + values.cue +
// overlays) with any -f values files
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
	output.Debug("rendering from instance file", "file", opts.InstanceFilePath, "namespace", opts.K8sConfig.Namespace.Value)

	k := config.NewKernel(opts.Config.Registry)

	// Acquire the instance package (the directory containing the instance
	// file) with the -f files layered as values sources: the schema's own
	// values unification performs the merge inside the build, nothing is
	// filled from Go, and a conflict names the file it came from.
	instanceDir, moduleRoot, err := instanceContext(opts.InstanceFilePath)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}
	// A package under no module root cannot import its module: refuse it
	// here, naming the directory, before the acquire reports it as an
	// unresolved import.
	if moduleRoot == "" {
		err := errNoModuleRoot(instanceDir)
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	sources, err := loadValuesSources(k, opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	inst, err := k.AcquireInstanceFromDir(ctx, instanceDir, sources...)
	if err != nil {
		// What the package is decides, not its file names: a module package
		// is pointed at the module command that builds it.
		if modErr := cmdutil.ModulePackageError(ctx, k, instanceDir, opts.ModuleCommand, err); modErr != nil {
			return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: modErr}
		}
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// The instance file owns its namespace: a --namespace or OPM_NAMESPACE
	// that disagrees with it is refused before the platform is resolved.
	if inst.Metadata != nil {
		if err := refuseNamespaceOverride(opts.InstanceFilePath, inst.Metadata.Namespace, opts.K8sConfig.Namespace); err != nil {
			printValidationError(err)
			return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
		}
	}

	// Render provenance (0006:D7): an instance apply is local when
	// its module's cue.mod/local-module.cue replaces a dependency; otherwise it
	// resolves from registries. The same module root is the 0010:D19 module
	// context the render's replacement warnings are worded against.
	sourceLocal := loader.HasLocalModuleReplacement(moduleRoot)

	// Without --platform the render falls back to the instance package's own
	// pins: its cue.mod/module.cue and local-module.cue at the module root.
	var deps *platform.ModuleDeps
	if opts.PlatformFlag == "" {
		deps, err = instanceDepsOf(instanceDir, moduleRoot)
		if err != nil {
			printValidationError(err)
			return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
		}
	}

	// Platform resolution + acquisition only after the instance itself
	// validated: cheap failures never hit the cluster or registry.
	env, err := resolvePlatformEnv(ctx, k, opts.Config, platform.ResolveOptions{
		PlatformFlag:    opts.PlatformFlag,
		Cluster:         opts.ClusterPlatform,
		ClusterOptional: opts.ClusterOptional,
		Deps:            deps,
		DepsKind:        platform.DepsInstance,
	})
	if err != nil {
		return nil, err
	}
	env.skipUnprovided = opts.SkipUnprovided

	return renderInstance(ctx, env, inst, opts.K8sConfig, moduleRoot, sourceLocal)
}

// refuseNamespaceOverride returns a validation error when the namespace was
// set by --namespace or OPM_NAMESPACE and differs from the namespace the
// instance file declares. The file owns the namespace: it is part of the
// instance's identity (fqn and uuid), so an override would put the record
// in one namespace and the resources in another. A namespace from the
// config file or the default is not an override and is never compared, nor
// is an OPM_NAMESPACE the flag shadows.
func refuseNamespaceOverride(instancePath, declared string, ns config.ResolvedField) error {
	var source string
	switch ns.Source {
	case config.SourceFlag:
		source = "--namespace"
	case config.SourceEnv:
		source = "OPM_NAMESPACE"
	case config.SourceConfig, config.SourceDefault:
		// Not an override: the instance renders in its file's namespace.
	}
	if source == "" || ns.Value == declared {
		return nil
	}
	return &pkgerrors.ValidationError{
		Message: fmt.Sprintf("%s %q disagrees with metadata.namespace %q in %s", source, ns.Value, declared, instancePath),
		Details: fmt.Sprintf("the namespace is part of the instance's identity: to deploy to %q, set metadata.namespace: %q in the instance file (that makes a new instance; delete the one in %q if it is deployed); otherwise drop the override", ns.Value, ns.Value, declared),
	}
}

// instanceContext resolves an instance argument (a .cue file or a package
// directory) to the instance package directory and its module context. The
// context is computed from the package directory, never from the argument's
// parent, so a directory argument with its own cue.mod is its own context.
func instanceContext(arg string) (dir, moduleRoot string, err error) {
	dir, err = cmdutil.InstanceDir(arg)
	if err != nil {
		return "", "", err
	}
	return dir, moduleContextRoot(dir), nil
}

// errNoModuleRoot is the validation error for an instance package under no
// CUE module: without its own cue.mod it can import neither its module nor
// core, and it has no dependency pins to render against.
func errNoModuleRoot(instanceDir string) error {
	return fmt.Errorf("instance package %s is under no CUE module (no cue.mod/module.cue at or above it): an instance package imports its module through its own cue.mod", instanceDir)
}

// instanceDepsOf reads what an instance-deps platform is generated from: the
// instance package's committed cue.mod/module.cue at its module root and the
// replacements of its cue.mod/local-module.cue. A package under no module
// root cannot import its module, so it has no pins to read.
func instanceDepsOf(instanceDir, moduleRoot string) (*platform.ModuleDeps, error) {
	if moduleRoot == "" {
		return nil, errNoModuleRoot(instanceDir)
	}
	name := filepath.Join(moduleRoot, "cue.mod", "module.cue")
	data, err := os.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("reading the instance package's dependency pins: %w", err)
	}
	replacements, err := loader.LocalReplacements(moduleRoot)
	if err != nil {
		return nil, err
	}
	return &platform.ModuleDeps{
		ModFile:      data,
		ModFileName:  name,
		Replacements: replacements,
		ModuleRoot:   moduleRoot,
	}, nil
}

// moduleContextRoot is the effective module context of a render entry: the
// module root (nearest cue.mod/module.cue) above dir — the directory holding
// an instance file, or the module directory itself. "" when dir is under no
// module root. It is where a developer's cue.mod/local-module.cue lives, so
// it drives both the render provenance signal (0006:D7) and the wording of
// the render's replacement warnings (0010:D19).
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
// SkipUnprovided is the caller's --skip-unprovided, passed through: which
// demands are skippable is the kernel's rule.
func newRenderInput(env *renderEnv, inst *module.Instance) kernel.RenderInput {
	return kernel.RenderInput{
		Instance:          inst,
		Platform:          env.platform,
		RuntimeName:       RuntimeName,
		Skew:              env.skew,
		LocalReplacements: true,
		SkipUnprovided:    env.skipUnprovided,
	}
}

// renderInstance runs the kernel's single render verb on a source-carrying
// instance and adapts the result to the workflow Result. Every failure the
// kernel reports — a skew refusal before evaluation, a local-replacement
// refusal, or the fail-closed gate after it (unresolved demands, unmatched
// components, an over-subscribed provider contract, a failed pair) — exits
// as a validation failure with the kernel's message and the diagnostics
// printed beside it. A successful render is then refused when two of its
// objects share one apply identity, before anything downstream can receive
// the set. After that the 0010:D19 replacement warnings are emitted from the
// kernel's rows against moduleRoot, the module context, and every demand
// skipped under --skip-unprovided is warned about on the log stream.
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
		if hint := refusalHint(err, env.resolution); hint != "" {
			output.Details("Hint: " + hint)
		}
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	if err := refuseDuplicateIdentities(out); err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	for _, w := range replacementWarnings(out.Diagnostics.Replacements, moduleRoot, env.resolution.Carried) {
		output.Warn(w)
	}
	for _, w := range formatSkipped(out.Diagnostics.Skipped) {
		output.Warn(w)
	}

	// One CUE export per object: the render digest hashes the exported JSON
	// and the apply objects are the objects decoded from those same bytes.
	exported, err := object.Export(object.Resources(out.Compiled))
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("converting rendered resources: %w", err)}
	}

	renderDigest, err := inventory.ComputeRenderDigest(exported)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: err}
	}

	result := newResult(env, out, renderDigest, decodeUnifiedValues(inst.Values()), sourceLocal)

	for i := range exported {
		result.Resources = append(result.Resources, exported[i].Object)
	}

	// Instance metadata is the kernel's decode, taken whole; the namespace
	// flag/env override applies to the apply target after it.
	if inst.Metadata != nil {
		result.Instance = *inst.Metadata
	}
	if k8sCfg != nil {
		if s := k8sCfg.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
			result.Instance.Namespace = k8sCfg.Namespace.Value
		}
	}

	// The embedded module's metadata carries the full registry modulePath for
	// the canonical spec.module reference.
	result.Module = moduleMetadataOf(inst)

	return result, nil
}

// refuseDuplicateIdentities returns the library's duplicate-identity error
// when two of a render's compiled objects share one apply identity, nil
// otherwise. A build refuses exactly what an apply would have written twice,
// and the CLI and the operator word the refusal identically because both
// raise the library's error (0015:D15). Pure over the render output, so it is
// tested without a render.
func refuseDuplicateIdentities(out *kernel.RenderResult) error {
	if dups := object.Duplicates(out.Compiled); len(dups) > 0 {
		return &object.DuplicateIdentitiesError{Duplicates: dups}
	}
	return nil
}

// newResult assembles the workflow Result from the render output and the
// render environment. Warnings are the render's advisory facts worded by the CLI from the
// diagnostics rows (unhandled optional traits, skew under the warn policy);
// the 0010:D19 local-replacement warnings are emitted directly by renderInstance
// from the replacement rows, after the render.
func newResult(env *renderEnv, out *kernel.RenderResult, renderDigest string, values map[string]any, sourceLocal bool) *Result {
	return &Result{
		Pairs:        out.Diagnostics.Pairs,
		Warnings:     formatAdvisories(out.Diagnostics),
		Platform:     env.resolution,
		RenderDigest: renderDigest,
		Values:       values,
		SourceLocal:  sourceLocal,
		Skipped:      out.Diagnostics.Skipped,
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

// moduleMetadataOf is the metadata of the module the instance embeds, or zero
// metadata when the instance carries none. The library returns nil both for
// an absent #module and for embedded metadata that does not decode as a
// whole (an open identity field); the render result carries either as a
// module with no metadata.
func moduleMetadataOf(inst *module.Instance) module.ModuleMetadata {
	if m := inst.ModuleMetadata(); m != nil {
		return *m
	}
	return module.ModuleMetadata{}
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
