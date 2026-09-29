package render

import (
	"context"
	"fmt"
	"path"
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

// FromModule synthesizes an instance from a module (a local module-package
// directory, or a published module acquired from the registry when
// opts.Published is set) through kernel SynthesizeInstance and renders it
// through the same render path as
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
	if opts.Published == nil {
		if opts.ModulePath == "" {
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("module path is required")}
		}
		if pathErr := cmdutil.ValidateModuleInputPath(opts.ModulePath); pathErr != nil {
			return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: pathErr}
		}
	}

	namespace := opts.K8sConfig.Namespace.Value
	output.Debug("rendering from module", "module", opts.moduleLabel(), "namespace", namespace)

	k := config.NewKernel(opts.Config.Registry)

	src, err := acquireModule(ctx, k, opts)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	mod := src.module

	values, err := ResolveModuleValues(k, mod.Package, src.valuesOrigin, opts.ValuesFiles)
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
	var deps *platform.ModuleDeps
	if opts.PlatformFromDeps && opts.PlatformFlag == "" {
		deps, err = moduleDepsOf(mod.Source, src.moduleRoot)
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

	// A local module directory is the main module, so render provenance is
	// local (0006:D7), and the directory is the 0010:D19 module context: a
	// replaced dependency in its own cue.mod is worded from the kernel's rows
	// after the render. A published module's bytes come from the registry: no
	// local provenance, no module context.
	return renderInstance(ctx, env, inst, opts.K8sConfig, src.moduleRoot, src.local)
}

// moduleLabel names the module for logs: the directory, or
// "<path>@<version>" for a published module.
func (o ModuleOpts) moduleLabel() string {
	if p := o.Published; p != nil {
		return p.Path + "@" + p.Version
	}
	return o.ModulePath
}

// acquiredModule is a module acquired for synthesis, with what the rest of
// the pipeline needs to know about where it came from.
type acquiredModule struct {
	module *module.Module
	// valuesOrigin prefixes the debugValues source's origin: the module
	// directory, or "<path>@<version>" for a published module.
	valuesOrigin string
	// moduleRoot is the local module context, "" for a published module.
	moduleRoot string
	// local marks a module read from disk (local render provenance).
	local bool
}

// acquireModule acquires the module through the kernel's shape gate, from
// the registry when opts.Published is set, else from the local directory.
// Either way the acquire stages the module's source tree, so synthesis builds
// the instance package inside the module's own root and its cue.mod drives
// transitive resolution; for a local directory that includes any
// local-module.cue replaceWith (0006:D37).
func acquireModule(ctx context.Context, k *kernel.Kernel, opts ModuleOpts) (*acquiredModule, error) {
	if p := opts.Published; p != nil {
		mod, err := k.AcquireModuleFromRegistry(ctx, p.Import(), p.Version)
		if err != nil {
			return nil, err
		}
		return &acquiredModule{module: mod, valuesOrigin: opts.moduleLabel()}, nil
	}
	mod, err := k.AcquireModuleFromDir(ctx, opts.ModulePath)
	if err != nil {
		return nil, err
	}
	return &acquiredModule{
		module:       mod,
		valuesOrigin: opts.ModulePath,
		moduleRoot:   moduleContextRoot(opts.ModulePath),
		local:        true,
	}, nil
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
		if opts.Published != nil {
			modName = path.Base(opts.Published.Path)
		} else {
			modName = filepath.Base(opts.ModulePath)
		}
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
