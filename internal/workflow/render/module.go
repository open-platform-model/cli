package render

import (
	"context"
	"fmt"
	"path/filepath"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"github.com/open-platform-model/library/opm/kernel"
	"github.com/open-platform-model/library/opm/module"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
)

// FromModule synthesizes an instance from a module-package directory through
// kernel SynthesizeInstance and renders it through the same render path as
// FromInstanceFile (0006 D9; retires the CLI's synthetic-wrapper module and
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

	k := NewKernel(opts.Config)

	// Acquire the module package through the kernel's shape gate. The acquire
	// stages the local directory as the module's source tree, so synthesis
	// builds the instance package inside the module's own root: the module
	// import resolves locally (no registry round-trip for the module itself)
	// and its cue.mod — including any local-module.cue replaceWith (D37) —
	// drives transitive resolution.
	mod, err := k.AcquireModuleFromDir(ctx, opts.ModulePath)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	values, err := resolveModuleValues(k, mod, opts.ModulePath, opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
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

	// Platform resolution + acquisition only after synthesis validated the
	// values: cheap failures never hit the cluster or registry.
	env, err := resolvePlatformEnv(ctx, k, opts.Config, opts.PlatformFlag, opts.ClusterPlatform)
	if err != nil {
		return nil, err
	}

	// A module apply always renders a local module directory (the main module is
	// local), so render provenance is local (enhancement 0006 D7). The module
	// directory is the D19 module context: a replaced dependency in its own
	// cue.mod is worded from the kernel's rows after the render.
	return renderInstance(ctx, env, inst, opts.K8sConfig, moduleContextRoot(opts.ModulePath), true)
}

// defaultNamespace is the synthetic-instance namespace when no
// --namespace/env override is given.
const defaultNamespace = "default"

// syntheticIdentity derives the synthetic instance identity: caller-supplied
// name or "<module.metadata.name>-debug"; namespace from --namespace/env
// override, else "default".
func syntheticIdentity(mod *module.Module, opts ModuleOpts, namespace string) (modName, synthName, synthNamespace string) {
	if mod.Metadata != nil {
		modName = mod.Metadata.Name
	}
	if modName == "" {
		modName = filepath.Base(opts.ModulePath)
	}
	synthName = opts.Name
	if synthName == "" {
		synthName = modName + "-debug"
	}
	synthNamespace = defaultNamespace
	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		synthNamespace = namespace
	}
	return modName, synthName, synthNamespace
}

// resolveModuleValues mirrors `opm module vet`: -f files override debugValues.
// The files are layered as kernel values sources and checked through the
// kernel's layered validation against the module's #config before synthesis,
// so a conflict or a violation is attributed to the file it came from; the
// returned sources are what synthesis renders into the instance's values
// file. Without -f files the module's own debugValues are the single source
// (DebugValuesSource), attributed to the module directory.
func resolveModuleValues(k *kernel.Kernel, mod *module.Module, moduleDir string, valuesFiles []string) ([]kernel.Source, error) {
	if len(valuesFiles) > 0 {
		sources, err := loadValuesSources(k, valuesFiles)
		if err != nil {
			return nil, err
		}
		schemaVal := mod.ConfigSchema()
		if !schemaVal.Exists() {
			return nil, fmt.Errorf("module does not define #config; values files cannot be validated")
		}
		if _, err := k.ValidateConfigDetailed(schemaVal, sources); err != nil {
			return nil, err
		}
		return sources, nil
	}
	src, err := DebugValuesSource(k, mod, filepath.Join(moduleDir, "debugValues"))
	if err != nil {
		return nil, err
	}
	return []kernel.Source{src}, nil
}
