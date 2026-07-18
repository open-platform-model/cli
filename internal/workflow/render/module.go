package render

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	loaderfile "github.com/open-platform-model/library/opm/helper/loader/file"
	"github.com/open-platform-model/library/opm/helper/synth"
	"github.com/open-platform-model/library/opm/module"
	"github.com/open-platform-model/library/opm/schema"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
)

// FromModule synthesizes an instance from a module-package directory through
// kernel SynthesizeInstance and renders it through the same compile path as
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

	modVal, err := k.LoadModulePackage(ctx, opts.ModulePath, loaderfile.LoadOptions{Registry: opts.Config.Registry})
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}
	mod, err := k.NewModuleFromValue(modVal)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// Stage the local directory as the module's source tree: synthesis
	// builds the instance package inside the module's own root, so the
	// module import resolves locally (no registry round-trip for the module
	// itself) and its cue.mod — including any local-module.cue replaceWith
	// (D37) — drives transitive resolution.
	src, err := stageLocalModuleSource(opts.ModulePath)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("staging module source: %w", err)}
	}
	mod.Source = src

	values, err := resolveModuleValues(k.CueContext(), modVal, opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	modName, synthName, synthNamespace := syntheticIdentity(mod, opts, namespace)

	output.Info(fmt.Sprintf("Building synthetic instance %q for module %q", synthName, modName))

	inst, err := k.SynthesizeInstance(ctx, synth.InstanceInput{
		Module:    mod,
		Name:      synthName,
		Namespace: synthNamespace,
		Values:    values,
	})
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	// Platform resolution + materialization only after synthesis validated
	// the values: cheap failures never hit the cluster or registry.
	env, err := resolvePlatformEnv(ctx, k, opts.Config, opts.PlatformFlag, opts.ClusterPlatform)
	if err != nil {
		return nil, err
	}

	// A module apply always renders a local module directory (the main module is
	// local), so render provenance is local (enhancement 0006 D7).
	return compileInstance(ctx, env, inst, opts.K8sConfig, true)
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

// stageLocalModuleSource builds a module.Source overlay from a local module
// directory so kernel synthesis can use it as the build's main module. Every
// regular file under the directory is staged (VCS and build-artifact
// directories skipped).
func stageLocalModuleSource(dir string) (*module.Source, error) {
	absDir, err := filepath.Abs(dir)
	if err != nil {
		return nil, err
	}
	// Root-scoped filesystem: every read stays inside the module directory
	// even under concurrent symlink swaps (gosec G122).
	root, err := os.OpenRoot(absDir)
	if err != nil {
		return nil, err
	}
	defer root.Close()

	overlay := make(map[string]load.Source)
	err = fs.WalkDir(root.FS(), ".", func(rel string, d fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if d.IsDir() {
			switch d.Name() {
			case ".git", ".build", "node_modules":
				return fs.SkipDir
			}
			return nil
		}
		if !d.Type().IsRegular() {
			return nil
		}
		content, readErr := fs.ReadFile(root.FS(), rel)
		if readErr != nil {
			return readErr
		}
		overlay[filepath.Join(absDir, filepath.FromSlash(rel))] = load.FromBytes(content)
		return nil
	})
	if err != nil {
		return nil, err
	}
	if len(overlay) == 0 {
		return nil, fmt.Errorf("module directory %s contains no files", dir)
	}
	return &module.Source{Root: absDir, Overlay: overlay}, nil
}

// resolveModuleValues mirrors `opm module vet`: -f files override debugValues.
// The returned value is a single unified cue.Value (the kernel's synthesis
// takes one values input).
func resolveModuleValues(cueCtx *cue.Context, modVal cue.Value, valuesFiles []string) (cue.Value, error) {
	if len(valuesFiles) > 0 {
		return unifyValuesFiles(cueCtx, valuesFiles)
	}
	debugVal := modVal.LookupPath(schema.DebugValues)
	if !debugVal.Exists() {
		return cue.Value{}, fmt.Errorf("module does not define debugValues - add debugValues or provide values with -f")
	}
	return debugVal, nil
}
