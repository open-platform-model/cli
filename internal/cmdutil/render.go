package cmdutil

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/pkg/engine"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/loader"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
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
	// ValuesFile is the path to the values CUE file (optional, from --values/-f).
	// When empty, values.cue next to the release file is used if it exists.
	// If neither is found and the release package has no concrete values field,
	// an error is returned.
	ValuesFile string
	// ModulePath is the path to a local module directory (optional, from --module).
	ModulePath string
	// K8sConfig is the pre-resolved Kubernetes configuration.
	K8sConfig *config.ResolvedKubernetesConfig
	// Config is the fully loaded global configuration.
	Config *config.GlobalConfig
}

// RenderRelease executes the common render pipeline shared by build, vet, apply,
// and diff commands. It loads the release package, detects its kind, loads the
// module release, loads the provider, and runs the engine renderer.
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

	// Load the CUE release package.
	// When DebugValues is set and no -f flag is provided, extract the module's
	// debugValues field and inject it as the values source.
	var pkg cue.Value
	if opts.DebugValues && len(opts.Values) == 0 {
		// Load module package to extract debugValues.
		modVal, modErr := loader.LoadModulePackage(cueCtx, modulePath)
		if modErr != nil {
			return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading module for debugValues: %w", modErr)}
		}
		debugVal := modVal.LookupPath(cue.ParsePath("debugValues"))
		if !debugVal.Exists() {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("module does not define debugValues — add a debugValues field or provide a values file with -f"),
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
		var loadErr error
		pkg, _, loadErr = loader.LoadReleasePackageWithValue(cueCtx, modulePath, debugVal)
		if loadErr != nil {
			PrintValidationError("render failed", loadErr)
			return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: loadErr, Printed: true}
		}
	} else {
		// Resolve values file: use first -f flag value if provided, else empty
		// (loader will fall back to values.cue in the module directory).
		var valuesFile string
		if len(opts.Values) > 0 {
			valuesFile = opts.Values[0]
		}
		var loadErr error
		pkg, _, loadErr = loader.LoadReleasePackage(cueCtx, modulePath, valuesFile)
		if loadErr != nil {
			PrintValidationError("render failed", loadErr)
			return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: loadErr, Printed: true}
		}
	}

	// Detect release kind (ModuleRelease or BundleRelease).
	kind, err := loader.DetectReleaseKind(pkg)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}
	if kind != "ModuleRelease" {
		return nil, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("unsupported release kind %q (use bundle commands for BundleRelease)", kind),
		}
	}

	// Load the ModuleRelease from the evaluated package.
	fallbackName := filepath.Base(modulePath)
	if opts.ReleaseName != "" {
		fallbackName = opts.ReleaseName
	}
	rel, err := loader.LoadModuleReleaseFromValue(cueCtx, pkg, fallbackName)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Override namespace if provided (flag takes precedence over module default).
	if namespace != "" {
		rel.Metadata.Namespace = namespace
	}

	// Load provider from config providers map.
	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	// Run the engine renderer.
	renderer := engine.NewModuleRenderer(p, opts.Config.Matcher)
	engineResult, err := renderer.Render(ctx, rel)
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
//
//  1. loader.LoadReleaseFile()              load .cue file with import resolution
//  2. loader.DetectReleaseKind()            error on BundleRelease (not yet supported)
//  3. [optional] loader.LoadModulePackage() + FillPath for --module flag
//  4. loader.LoadModuleReleaseFromValue()   validate + extract *ModuleRelease
//  5. loader.LoadProvider()                 wrap provider CUE value
//  6. engine.ModuleRenderer.Render()        CUE-native match + execute
//  7. Resource.ToUnstructured()             convert at cmdutil boundary
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
	registry := opts.Config.Registry

	// Step 1: Load the release file, merging a values file when appropriate.
	//
	// Resolution order:
	//   a. --values flag provided → load release + that file.
	//   b. values.cue exists next to the release file → load release + values.cue.
	//   c. Neither → load the release file alone and check for an inline
	//      concrete values field; error if none found.
	releaseVal, _, err := loadReleaseWithValues(cueCtx, opts.ReleaseFilePath, opts.ValuesFile, registry)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Step 2: Detect release kind — reject BundleRelease (not yet supported).
	kind, err := loader.DetectReleaseKind(releaseVal)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: err}
	}
	if kind == "BundleRelease" {
		return nil, &oerrors.ExitError{
			Code: oerrors.ExitGeneralError,
			Err:  fmt.Errorf("bundle releases are not yet supported — use a #ModuleRelease file"),
		}
	}

	// Step 3: Optionally inject local module via --module flag.
	if opts.ModulePath != "" {
		modVal, modErr := loader.LoadModulePackage(cueCtx, opts.ModulePath)
		if modErr != nil {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("loading module from --module: %w", modErr),
			}
		}
		releaseVal = releaseVal.FillPath(cue.MakePath(cue.Def("module")), modVal)
		if err := releaseVal.Err(); err != nil {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("filling #module from --module: %w", err),
			}
		}
	}

	// Check that #module is filled before proceeding.
	moduleVal := releaseVal.LookupPath(cue.MakePath(cue.Def("module")))
	if !moduleVal.Exists() || moduleVal.Validate(cue.Concrete(true)) != nil {
		if opts.ModulePath == "" {
			return nil, &oerrors.ExitError{
				Code: oerrors.ExitGeneralError,
				Err:  fmt.Errorf("#module is not filled in the release file — either import a module or use --module <path>"),
			}
		}
	}

	// Step 4: Load the ModuleRelease from the evaluated value.
	fallbackName := filepath.Base(opts.ReleaseFilePath)
	rel, err := loader.LoadModuleReleaseFromValue(cueCtx, releaseVal, fallbackName)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Override namespace if provided.
	if namespace != "" {
		rel.Metadata.Namespace = namespace
	}

	// Step 5: Load provider.
	p, err := loader.LoadProvider(providerName, opts.Config.Providers)
	if err != nil {
		return nil, &oerrors.ExitError{Code: oerrors.ExitGeneralError, Err: fmt.Errorf("loading provider: %w", err)}
	}

	// Step 6: Run the engine renderer.
	renderer := engine.NewModuleRenderer(p, opts.Config.Matcher)
	engineResult, err := renderer.Render(ctx, rel)
	if err != nil {
		PrintValidationError("render failed", err)
		return nil, &oerrors.ExitError{Code: oerrors.ExitValidationError, Err: err, Printed: true}
	}

	// Step 7: Convert resources to Unstructured.
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
func loadReleaseWithValues(cueCtx *cue.Context, releaseFilePath, valuesFile, registry string) (cue.Value, string, error) {
	// Case 1: explicit --values flag.
	if valuesFile != "" {
		return loader.LoadReleaseFileWithValues(cueCtx, releaseFilePath, valuesFile, registry)
	}

	// Case 2: auto-discover values.cue next to the release file.
	// resolveReleaseFile handles directory → release.cue expansion; we replicate
	// the directory detection here to find the sibling values.cue.
	releaseDir, err := resolveReleaseDir(releaseFilePath)
	if err != nil {
		return cue.Value{}, "", err
	}
	autoValues := filepath.Join(releaseDir, "values.cue")
	if _, statErr := os.Stat(autoValues); statErr == nil {
		return loader.LoadReleaseFileWithValues(cueCtx, releaseFilePath, autoValues, registry)
	}

	// Case 3: no values file — load release file alone and verify inline values.
	val, dir, loadErr := loader.LoadReleaseFile(cueCtx, releaseFilePath, registry)
	if loadErr != nil {
		return cue.Value{}, "", loadErr
	}

	valuesVal := val.LookupPath(cue.ParsePath("values"))
	if !valuesVal.Exists() || valuesVal.Validate(cue.Concrete(true)) != nil {
		return cue.Value{}, "", fmt.Errorf(
			"release has no concrete values — provide --values <file> or add a values.cue to the release directory",
		)
	}

	return val, dir, nil
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
