package cmdutil

import (
	"context"
	"errors"
	"fmt"
	"path/filepath"

	"cuelang.org/go/cue"
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	internalreleasefile "github.com/opmodel/cli/internal/releasefile"
	oerrors "github.com/opmodel/cli/pkg/errors"
	"github.com/opmodel/cli/pkg/loader"
	pkgmodule "github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
	"github.com/opmodel/cli/pkg/releaseprocess"
)

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
	fileRelease, err := internalreleasefile.GetReleaseFile(cueCtx, opts.ReleaseFilePath)
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
