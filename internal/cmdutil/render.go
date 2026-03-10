package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	internalreleasefile "github.com/opmodel/cli/internal/releasefile"
	"github.com/opmodel/cli/pkg/engine"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/loader"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
	"github.com/opmodel/cli/pkg/releaseprocess"
)

// RenderResult is the output of RenderRelease, combining engine output with
// release/module metadata for use across all cmd/mod commands.
type RenderResult struct {
	// Resources is the ordered list of Kubernetes resources, already converted
	// to *unstructured.Unstructured for direct use with inventory and k8s packages.
	Resources []*unstructured.Unstructured

	// Release contains release-level metadata (name, namespace, release UUID, labels).
	Release modulerelease.ReleaseMetadata

	// Module contains module-level metadata (canonical name, FQN, version, UUID).
	Module pkgmodule.ModuleMetadata

	// Components contains summary data for each component rendered in this release.
	// Sorted by component name for deterministic output.
	Components []engine.ComponentSummary

	// MatchPlan describes which transformers matched which components.
	// Used for verbose output and debugging.
	MatchPlan *engine.MatchPlan

	// Warnings contains non-fatal warnings (e.g. unhandled traits).
	Warnings []string
}

// HasWarnings returns true if there are warnings.
func (r *RenderResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ResourceCount returns the number of rendered resources.
func (r *RenderResult) ResourceCount() int {
	return len(r.Resources)
}

// RenderReleaseOpts holds the inputs for RenderRelease.
type RenderReleaseOpts struct {
	// Args from the cobra command (first arg is module path).
	Args []string
	// Values files (-f flags).
	Values []string
	// ReleaseName overrides the default release name.
	ReleaseName string
	// K8sConfig is the pre-resolved Kubernetes configuration.
	// All fields (namespace, provider, kubeconfig, context) must already be resolved
	// via config.ResolveKubernetes before calling RenderRelease.
	K8sConfig *config.ResolvedKubernetesConfig
	// Config is the fully loaded global configuration.
	Config *config.GlobalConfig
	// DebugValues instructs RenderRelease to extract the module's debugValues field
	// and use it as the values source instead of a values file. Intended for
	// opm mod vet when no -f flag is provided.
	DebugValues bool
}

// RenderFromReleaseFileOpts holds inputs for RenderFromReleaseFile.
type RenderFromReleaseFileOpts struct {
	// ReleaseFilePath is the path to the .cue release file (required).
	// May be a directory, in which case release.cue inside it is used.
	ReleaseFilePath string
	// ValuesFiles are values CUE files (optional, from --values/-f).
	// When empty, values.cue next to the release file is used if it exists.
	// If neither is found and the release package has no concrete values field,
	// an error is returned.
	ValuesFiles []string
	// ModulePath is the path to a local module directory (optional, from --module).
	ModulePath string
	// K8sConfig is the pre-resolved Kubernetes configuration.
	K8sConfig *config.ResolvedKubernetesConfig
	// Config is the fully loaded global configuration.
	Config *config.GlobalConfig
}

// hasReleaseFile reports whether a release.cue file exists inside modulePath.
// modulePath may be a directory (checked directly) or a file path (its parent
// directory is checked). Returns false on any stat error.
func hasReleaseFile(modulePath string) bool {
	// Determine the candidate directory.
	dir := modulePath
	info, err := os.Stat(modulePath)
	if err == nil && !info.IsDir() {
		dir = filepath.Dir(modulePath)
	}
	_, statErr := os.Stat(filepath.Join(dir, "release.cue"))
	return statErr == nil
}

// RenderRelease executes the common render pipeline shared by build, vet, and apply
// commands. It loads the release (or synthesizes one from the module), loads the
// provider, and runs the engine renderer.
//
// When release.cue is present, the existing LoadReleasePackage path is used.
// When release.cue is absent, a synthesis path is taken:
//   - If DebugValues is true and no -f flag is given, debugValues is extracted from the module.
//   - If -f is given, values are loaded from the first -f file.
//   - SynthesizeModuleRelease builds a *ModuleRelease without a release file.
//
// On success it returns the RenderResult. On failure it returns an
// *ExitError with the appropriate exit code and Printed flag.
func RenderRelease(ctx context.Context, opts RenderReleaseOpts) (*RenderResult, error) { //nolint:gocyclo // orchestration function; complexity is inherent
	modulePath := ResolveModulePath(opts.Args)

	// Validate config is loaded
	if opts.Config == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}

	// K8sConfig must be pre-resolved by the caller
	if opts.K8sConfig == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("kubernetes config not resolved")}
	}

	namespace := opts.K8sConfig.Namespace.Value
	providerName := opts.K8sConfig.Provider.Value

	// Log resolved config at DEBUG level
	if opts.K8sConfig.Kubeconfig.Value != "" || opts.K8sConfig.Context.Value != "" {
		output.Debug("resolved kubernetes config",
			"kubeconfig", opts.K8sConfig.Kubeconfig.Value,
			"context", opts.K8sConfig.Context.Value,
			"namespace", namespace,
			"provider", providerName,
		)
	} else {
		output.Debug("resolved config",
			"namespace", namespace,
			"provider", providerName,
		)
	}

	cueCtx := opts.Config.CueContext

	output.Debug("rendering release",
		"module-path", modulePath,
		"namespace", namespace,
		"provider", providerName,
	)

	// rel and valuesVals are populated by either the synthesis path or the normal release path.
	var (
		rel        *modulerelease.ModuleRelease
		valuesVals []cue.Value
	)

	if !hasReleaseFile(modulePath) {
		// ── Synthesis path: no release.cue present ────────────────────────────
		// Load the module package directly.
		modVal, modErr := loader.LoadModulePackage(cueCtx, modulePath)
		if modErr != nil {
			return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading module: %w", modErr)}
		}

		// Resolve values: -f flag takes precedence over debugValues.
		// Three cases, expressed as a switch on what we have:
		//   - One or more -f files: load and unify all of them.
		//   - No -f and DebugValues: true: extract debugValues from the module.
		//   - No -f and DebugValues: false: no values source — error.
		switch {
		case len(opts.Values) > 0:
			var loadErr error
			valuesVals, loadErr = loadValuesFiles(cueCtx, opts.Values)
			if loadErr != nil {
				return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: loadErr}
			}
		case opts.DebugValues:
			// DebugValues path — extract debugValues from the module.
			// Only attempted when DebugValues: true (callers set this when no -f
			// flag is given). Keeping the guard here preserves the spec contract:
			// a future caller that sets DebugValues: false without -f will get a
			// clear error rather than silently falling through to debugValues.
			debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
			if !debugVal.Exists() {
				return nil, &oerrors.ExitError{
					Code: oerrors.ExitGeneralError,
					Err:  fmt.Errorf("no release.cue found — add debugValues to module or use -f <values-file>"),
				}
			}
			if err := debugVal.Validate(cue.Concrete(true)); err != nil {
				PrintValidationError("debugValues not concrete", err)
				return nil, &oerrors.ExitError{
					Code:    oerrors.ExitValidationError,
					Err:     fmt.Errorf("debugValues is not concrete — module must provide complete test values"),
					Printed: true,
				}
			}
			valuesVals = []cue.Value{debugVal}
		default:
			// No -f flag and DebugValues: false — no values source available.
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("no release.cue found — use -f <values-file> to provide values"),
			}
		}

		// Task 3.5: Resolve releaseName.
		// Priority: opts.ReleaseName → module.metadata.name → filepath.Base(modulePath)
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

		// Task 3.6: Resolve moduleNamespace.
		// Use module.metadata.defaultNamespace when namespace is not from flag/env.
		// Flag/env override is applied post-synthesis (task 3.9).
		moduleNamespace := namespace // default: whatever was resolved
		s := opts.K8sConfig.Namespace.Source
		if s != config.SourceFlag && s != config.SourceEnv {
			if nsVal := modVal.LookupPath(cue.ParsePath("metadata.defaultNamespace")); nsVal.Exists() {
				if ns, strErr := nsVal.String(); strErr == nil && ns != "" {
					moduleNamespace = ns
				}
			}
		}

		// Task 3.7: Call SynthesizeModuleRelease.
		var synthErr error
		rel, synthErr = releaseprocess.SynthesizeModuleRelease(cueCtx, modVal, valuesVals, releaseName, moduleNamespace)
		if synthErr != nil {
			PrintValidationError("render failed", synthErr)
			return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: synthErr, Printed: true}
		}
	} else {
		// ── Normal path: release.cue is present ───────────────────────────────
		var loadErr error
		rel, valuesVals, loadErr = loadModuleReleaseForRender(cueCtx, modulePath, opts.Values, opts.DebugValues, opts.ReleaseName)
		if loadErr != nil {
			var exitErr *oerrors.ExitError
			if ok := errors.As(loadErr, &exitErr); ok {
				return nil, exitErr
			}
			PrintValidationError("render failed", loadErr)
			return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: loadErr, Printed: true}
		}
	}

	// ── Common tail: shared by both synthesis and normal paths ─────────────────

	// Task 3.9 / existing behavior: override namespace only when the user
	// explicitly provided one via flag or env var. Config file and built-in
	// default are transparent to the release; the release definition owns its
	// target namespace.
	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		rel.Metadata.Namespace = namespace
	}

	// Load provider from config providers map.
	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	// Run the release-processing pipeline.
	engineResult, err := releaseprocess.ProcessModuleRelease(ctx, rel, valuesVals, p)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Convert []*core.Resource → []*unstructured.Unstructured at the cmdutil boundary.
	resources := make([]*unstructured.Unstructured, 0, len(engineResult.Resources))
	for _, r := range engineResult.Resources {
		u, convErr := r.ToUnstructured()
		if convErr != nil {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("converting resource %s/%s to unstructured: %w", r.Kind(), r.Name(), convErr),
			}
		}
		resources = append(resources, u)
	}

	return &RenderResult{
		Resources:  resources,
		Release:    *rel.Metadata,
		Module:     *rel.Module.Metadata,
		Components: engineResult.Components,
		MatchPlan:  engineResult.MatchPlan,
		Warnings:   engineResult.Warnings,
	}, nil
}

// RenderFromReleaseFile loads a release file, optionally injects a local module,
// and executes the render pipeline. This is the release-file equivalent of
// RenderRelease() (which does module-directory rendering).
//
// Pipeline:
//  1. parse release file into a barebones release
//  2. optionally inject a local module via --module
//  3. resolve values from file or inline release values
//  4. process the release into concrete components and a render result
//  5. convert resources at the cmdutil boundary
func RenderFromReleaseFile(ctx context.Context, opts RenderFromReleaseFileOpts) (*RenderResult, error) { //nolint:gocyclo // orchestration function; complexity is inherent
	if opts.Config == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("configuration not loaded")}
	}
	if opts.K8sConfig == nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("kubernetes config not resolved")}
	}
	if opts.ReleaseFilePath == "" {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("release file path is required")}
	}

	namespace := opts.K8sConfig.Namespace.Value
	providerName := opts.K8sConfig.Provider.Value

	output.Debug("rendering from release file",
		"file", opts.ReleaseFilePath,
		"namespace", namespace,
		"provider", providerName,
	)

	cueCtx := opts.Config.CueContext
	fileRelease, err := internalreleasefile.GetReleaseFile(opts.ReleaseFilePath)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}
	if fileRelease.Kind == internalreleasefile.KindBundleRelease {
		return nil, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("bundle releases are not yet supported — use a #ModuleRelease file"),
		}
	}
	rel := fileRelease.Module

	// Step 2: Optionally inject local module via --module flag.
	if opts.ModulePath != "" {
		modVal, modErr := loader.LoadModulePackage(cueCtx, opts.ModulePath)
		if modErr != nil {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("loading module from --module: %w", modErr),
			}
		}
		rel.RawCUE = rel.RawCUE.FillPath(cue.MakePath(cue.Def("module")), modVal)
		rel.Module.Raw = modVal
		rel.Module.Config = modVal.LookupPath(cue.ParsePath("#config"))
		rel.Config = rel.RawCUE.LookupPath(cue.ParsePath("#module.#config"))
		if modMeta := rel.Module.Metadata; modMeta == nil {
			rel.Module.Metadata = &pkgmodule.ModuleMetadata{}
			if err := modVal.LookupPath(cue.ParsePath("metadata")).Decode(rel.Module.Metadata); err != nil {
				return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("decoding module metadata from --module: %w", err)}
			}
		}
		if err := rel.RawCUE.Err(); err != nil {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("filling #module from --module: %w", err),
			}
		}
	}

	// Check that #module is filled before proceeding.
	moduleVal := rel.RawCUE.LookupPath(cue.MakePath(cue.Def("module")))
	if !moduleVal.Exists() || moduleVal.Validate(cue.Concrete(true)) != nil {
		if opts.ModulePath == "" {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("#module is not filled in the release file — either import a module or use --module <path>"),
			}
		}
	}

	valuesVals, err := resolveReleaseValues(cueCtx, rel.RawCUE, opts.ReleaseFilePath, opts.ValuesFiles)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Override namespace only when the user explicitly provided one via flag or
	// env var. Config file and built-in default are transparent to the release:
	// the release definition owns its target namespace.
	if s := opts.K8sConfig.Namespace.Source; s == config.SourceFlag || s == config.SourceEnv {
		rel.Metadata.Namespace = namespace
	}

	// Step 3: Load provider.
	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	// Step 4: Process and render.
	engineResult, err := releaseprocess.ProcessModuleRelease(ctx, rel, valuesVals, p)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Step 5: Convert resources to Unstructured.
	resources := make([]*unstructured.Unstructured, 0, len(engineResult.Resources))
	for _, r := range engineResult.Resources {
		u, convErr := r.ToUnstructured()
		if convErr != nil {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("converting resource %s/%s to unstructured: %w", r.Kind(), r.Name(), convErr),
			}
		}
		resources = append(resources, u)
	}

	return &RenderResult{
		Resources:  resources,
		Release:    *rel.Metadata,
		Module:     *rel.Module.Metadata,
		Components: engineResult.Components,
		MatchPlan:  engineResult.MatchPlan,
		Warnings:   engineResult.Warnings,
	}, nil
}

// loadReleaseWithValues resolves which loading strategy to use for a release
// file, based on whether a values file is explicitly given or can be
// auto-discovered, and returns the evaluated CUE value.
//
// Resolution order:
//  1. valuesFile is non-empty → LoadReleaseFileWithValues (error if file missing).
//  2. values.cue exists next to the release file → LoadReleaseFileWithValues.
//  3. Neither → LoadReleaseFile; then check that the loaded package already
//     contains a concrete values field. If not, return an actionable error.
func resolveReleaseValues(cueCtx *cue.Context, rawRelease cue.Value, releaseFilePath string, valuesFiles []string) ([]cue.Value, error) {
	if len(valuesFiles) > 0 {
		return loadValuesFiles(cueCtx, valuesFiles)
	}

	releaseDir, err := resolveReleaseDir(releaseFilePath)
	if err != nil {
		return nil, err
	}
	autoValues := filepath.Join(releaseDir, "values.cue")
	if _, statErr := os.Stat(autoValues); statErr == nil {
		valuesVal, err := loader.LoadValuesFile(cueCtx, autoValues)
		if err != nil {
			return nil, err
		}
		return []cue.Value{valuesVal}, nil
	}

	valuesVal := rawRelease.LookupPath(cue.ParsePath("values"))
	if !valuesVal.Exists() || valuesVal.Validate(cue.Concrete(true)) != nil {
		return nil, fmt.Errorf("release has no concrete values - provide --values <file> or add a values.cue to the release directory")
	}
	return []cue.Value{valuesVal}, nil
}

//nolint:gocyclo // release loading combines debug-values, inline-values, and file-values cases
func loadModuleReleaseForRender(cueCtx *cue.Context, modulePath string, valuesFiles []string, debugValues bool, releaseName string) (*modulerelease.ModuleRelease, []cue.Value, error) {
	fileRelease, err := internalreleasefile.GetReleaseFile(modulePath)
	if err != nil {
		return nil, nil, err
	}
	if fileRelease.Kind != internalreleasefile.KindModuleRelease || fileRelease.Module == nil {
		return nil, nil, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("unsupported release kind %q (use bundle commands for BundleRelease)", fileRelease.Kind),
		}
	}
	rel := fileRelease.Module

	var valuesVals []cue.Value
	switch {
	case debugValues && len(valuesFiles) == 0:
		modVal, modErr := loader.LoadModulePackage(cueCtx, modulePath)
		if modErr != nil {
			return nil, nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading module for debugValues: %w", modErr)}
		}
		debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
		if !debugVal.Exists() {
			return nil, nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("module does not define debugValues - add a debugValues field or provide a values file with -f"),
			}
		}
		if err := debugVal.Validate(cue.Concrete(true)); err != nil {
			PrintValidationError("debugValues not concrete", err)
			return nil, nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: fmt.Errorf("debugValues is not concrete - module must provide complete test values"), Printed: true}
		}
		valuesVals = []cue.Value{debugVal}
	case len(valuesFiles) > 0:
		loadedValues, loadErr := loadValuesFiles(cueCtx, valuesFiles)
		if loadErr != nil {
			return nil, nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: loadErr}
		}
		valuesVals = loadedValues
	default:
		inlineValues := rel.RawCUE.LookupPath(cue.ParsePath("values"))
		if !inlineValues.Exists() || inlineValues.Validate(cue.Concrete(true)) != nil {
			return nil, nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("release has no concrete values - provide -f <values-file> or enable debugValues")}
		}
		valuesVals = []cue.Value{inlineValues}
	}

	if releaseName != "" {
		rel.Metadata.Name = releaseName
	}
	return rel, valuesVals, nil
}

func loadValuesFiles(cueCtx *cue.Context, valuesFiles []string) ([]cue.Value, error) {
	valuesVals := make([]cue.Value, 0, len(valuesFiles))
	for _, valuesFile := range valuesFiles {
		valuesVal, err := loader.LoadValuesFile(cueCtx, valuesFile)
		if err != nil {
			return nil, fmt.Errorf("loading values file %q: %w", valuesFile, err)
		}
		valuesVals = append(valuesVals, valuesVal)
	}
	return valuesVals, nil
}

// resolveReleaseDir returns the directory containing the release file.
// When path is a directory itself, it is returned as-is.
func resolveReleaseDir(path string) (string, error) {
	info, err := os.Stat(path)
	if err != nil {
		if os.IsNotExist(err) {
			// Path may be a .cue file that doesn't exist yet — use its parent.
			return filepath.Dir(path), nil
		}
		return "", fmt.Errorf("stat release path: %w", err)
	}
	if info.IsDir() {
		return path, nil
	}
	return filepath.Dir(path), nil
}

// ShowOutputOpts controls how ShowRenderOutput displays results.
type ShowOutputOpts struct {
	Verbose bool
}

// ShowRenderOutput shows transformer match output and logs warnings.
// It returns an *ExitError if the result has errors.
func ShowRenderOutput(result *RenderResult, opts ShowOutputOpts) error {
	// Show transformer matches
	switch {
	case opts.Verbose:
		WriteVerboseMatchLog(result)
	default:
		WriteTransformerMatches(result)
	}

	// Log warnings
	releaseLog := output.ReleaseLogger(result.Release.Name)
	for _, w := range result.Warnings {
		releaseLog.Warn(w)
	}

	return nil
}
