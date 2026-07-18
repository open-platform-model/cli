package render

import (
	"context"
	"fmt"
	"path/filepath"

	opmexit "github.com/open-platform-model/cli/internal/exit"

	"cuelang.org/go/cue"

	"github.com/open-platform-model/cli/internal/cmdutil"
	"github.com/open-platform-model/cli/internal/config"
	"github.com/open-platform-model/cli/internal/output"
	"github.com/open-platform-model/cli/pkg/loader"
	pkgmodule "github.com/open-platform-model/cli/pkg/module"
)

// FromModule synthesizes a #ModuleInstance from a module-package directory and
// renders it through the same pipeline as FromInstanceFile. Values come from
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

	cueCtx := opts.Config.CueContext
	namespace := opts.K8sConfig.Namespace.Value
	providerName := opts.K8sConfig.Provider.Value

	output.Debug("rendering from module", "path", opts.ModulePath, "namespace", namespace, "provider", providerName)

	synthOpts := loader.SynthesizeOptions{Name: opts.Name}
	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		synthOpts.Namespace = namespace
	}

	synth, err := loader.SynthesizeModuleInstanceFromPackage(cueCtx, opts.ModulePath, synthOpts)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	values, err := resolveModuleValues(cueCtx, synth.ModuleValue, opts.ValuesFiles)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	mod := buildModuleFromValue(synth.ModuleValue, opts.ModulePath)

	// Read back the synthesized metadata.name for the banner.
	synthName := lookupStringOrDefault(synth.Spec, "metadata.name", "<unnamed>")
	modName := synth.ModuleName
	if modName == "" {
		modName = filepath.Base(opts.ModulePath)
	}
	output.Info(fmt.Sprintf("Building synthetic instance %q for module %q", synthName, modName))

	rel, err := pkgmodule.ParseModuleInstance(ctx, synth.Spec, mod, values)
	if err != nil {
		printValidationError(err)
		return nil, &opmexit.ExitError{Code: opmexit.ExitValidationError, Err: err, Printed: true}
	}

	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		rel.Metadata.Namespace = namespace
	}

	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &opmexit.ExitError{Code: opmexit.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	// A module apply always renders a local module directory (the main module is
	// local), so render provenance is local (enhancement 0006 D7).
	return renderPreparedModuleInstance(ctx, rel, p, true)
}

// resolveModuleValues mirrors `opm module vet`: -f files override debugValues.
func resolveModuleValues(cueCtx *cue.Context, modVal cue.Value, valuesFiles []string) ([]cue.Value, error) {
	if len(valuesFiles) > 0 {
		return loadValuesFiles(cueCtx, valuesFiles)
	}
	debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
	if !debugVal.Exists() {
		return nil, fmt.Errorf("module does not define debugValues - add debugValues or provide values with -f")
	}
	return []cue.Value{debugVal}, nil
}

// buildModuleFromValue constructs a pkgmodule.Module from a loaded module
// CUE value. Mirrors the bare-module side of internal/instancefile.bareModuleInstance
// but for a directly-loaded module value (no #module wrapper yet).
func buildModuleFromValue(modVal cue.Value, modulePath string) pkgmodule.Module {
	meta := &pkgmodule.ModuleMetadata{}
	if mv := modVal.LookupPath(cue.ParsePath("metadata")); mv.Exists() {
		// Best-effort decode: leaves zero-value fields if metadata is partial.
		if err := mv.Decode(meta); err != nil {
			_ = err
		}
	}
	return pkgmodule.Module{
		Metadata:   meta,
		Config:     modVal.LookupPath(cue.ParsePath("#config")),
		Raw:        modVal,
		ModulePath: modulePath,
	}
}

func lookupStringOrDefault(v cue.Value, path, fallback string) string {
	field := v.LookupPath(cue.ParsePath(path))
	if !field.Exists() {
		return fallback
	}
	s, err := field.String()
	if err != nil || s == "" {
		return fallback
	}
	return s
}
