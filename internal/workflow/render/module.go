package render

import (
	"context"
	"fmt"
	"path/filepath"
	"strings"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/internal/platform"
	"github.com/open-platform-model/cli/pkg/loader"
)

// FromModule synthesizes an instance from a module-package directory through
// kernel SynthesizeInstance and renders it through the same render path as
// FromInstanceFile (0006:D9; retires the CLI's synthetic-wrapper module and
// the last #ModuleRelease application — 0002 carryover). Values come from
// `-f` files when supplied, else from the module's `debugValues`.
func FromModule(ctx context.Context, opts ModuleOpts) (*Result, error) {
	if opts.Config == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}
	if opts.K8sConfig == nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("kubernetes config not resolved")}
	}
	if opts.ModulePath == "" {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("module path is required")}
	}
	if pathErr := cmdutil.ValidateModuleInputPath(opts.ModulePath); pathErr != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: pathErr}
	}

	namespace := opts.K8sConfig.Namespace.Value
	output.Debug("rendering from module", "path", opts.ModulePath, "namespace", namespace)

	k := config.NewKernel(opts.Config.Registry)

	// Acquire the module package through the kernel's shape gate. The acquire
	// stages the local directory as the module's source tree, so synthesis
	// builds the instance package inside the module's own root: the module
	// import resolves locally (no registry round-trip for the module itself)
	// and its cue.mod — including any local-module.cue replaceWith (0006:D37) —
	// drives transitive resolution.
	mod, err := k.AcquireModuleFromDir(ctx, opts.ModulePath)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	values, err := ResolveModuleValues(k, mod.Package, opts.ModulePath, opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// -f files are checked against #config before synthesis, so a conflict
	// or a violation is attributed to the file it came from and a cheap
	// failure never reaches the synthesized build. debugValues are left to
	// the build itself.
	if len(opts.ValuesFiles) > 0 {
		if err := validateValuesFiles(k, mod.ConfigSchema(), values); err != nil {
			printValidationError(err)
			return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
		}
	}

	modName, synthName, synthNamespace := syntheticIdentity(mod, opts, namespace)

	output.Info(fmt.Sprintf("Building synthetic instance %q for module %q", synthName, modName))

	inst, err := k.SynthesizeInstance(ctx, kernel.InstanceInput{
		Module:    mod,
		Name:      synthName,
		Namespace: synthNamespace,
		Values:    values,
	})
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// The author's platform is generated from the module's own deps unless
	// --platform names one; the deps are read from the acquired source.
	moduleRoot := moduleContextRoot(opts.ModulePath)
	var deps *platform.ModuleDeps
	if opts.PlatformFromDeps && opts.PlatformFlag == "" {
		deps, err = moduleDepsOf(mod.Source, moduleRoot)
		if err != nil {
			printValidationError(err)
			return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
		}
	}

	// Platform resolution + acquisition only after synthesis validated the
	// values: cheap failures never hit the cluster or registry.
	env, err := resolvePlatformEnv(ctx, k, opts.Config, opts.PlatformFlag, opts.ClusterPlatform, deps)
	if err != nil {
		return nil, err
	}

	// A module apply always renders a local module directory (the main module is
	// local), so render provenance is local (0006:D7). The module
	// directory is the 0010:D19 module context: a replaced dependency in its own
	// cue.mod is worded from the kernel's rows after the render.
	return renderInstance(ctx, env, inst, opts.K8sConfig, moduleRoot, true)
}

// moduleDepsOf reads what a module-deps platform is generated from out of
// the acquired module's source: the committed cue.mod/module.cue from the
// overlay, and the replacements of the local module context at moduleRoot.
// A module with no local context (moduleRoot "", a published module) carries
// no replacements.
func moduleDepsOf(src *module.Source, moduleRoot string) (*platform.ModuleDeps, error) {
	if src == nil || src.Root == "" {
		return nil, fmt.Errorf("module carries no source tree to read its dependency pins from")
	}
	name := filepath.Join(src.Root, "cue.mod", "module.cue")
	data, ok := src.Overlay[name]
	if !ok {
		return nil, fmt.Errorf("module source carries no %s", name)
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

// defaultNamespace is the synthetic-instance namespace when no
// --namespace/env override is given.
const defaultNamespace = "default"

// syntheticIdentity derives the synthetic instance identity: caller-supplied
// name or "<module.metadata.name>-debug" with every "_" hyphenated (a module
// name is a CUE package name, "my_app"; an instance name is a DNS label);
// namespace from --namespace/env override, else "default".
func syntheticIdentity(mod *module.Module, opts ModuleOpts, namespace string) (modName, synthName, synthNamespace string) {
	if mod.Metadata != nil {
		modName = mod.Metadata.Name
	}
	if modName == "" {
		modName = filepath.Base(opts.ModulePath)
	}
	synthName = opts.Name
	if synthName == "" {
		synthName = strings.ReplaceAll(modName, "_", "-") + "-debug"
	}
	synthNamespace = defaultNamespace
	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		synthNamespace = namespace
	}
	return modName, synthName, synthNamespace
}

// validateValuesFiles checks -f sources against a module's #config through
// the kernel's layered validation, which reports schema violations and merge
// conflicts at their source positions. A module without #config cannot
// validate values files.
func validateValuesFiles(k *kernel.Kernel, configSchema cue.Value, sources []kernel.Source) error {
	if !configSchema.Exists() {
		return fmt.Errorf("module does not define #config; values files cannot be validated")
	}
	_, err := k.ValidateConfigDetailed(configSchema, sources)
	return err
}
