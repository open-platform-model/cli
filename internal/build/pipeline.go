// Package build provides the render pipeline implementation.
package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"

	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/transform"
	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/pkg/weights"
)

// pipeline implements the Pipeline interface.
// It orchestrates module loading, release building, provider loading,
// component matching, and transformer execution.
type pipeline struct {
	config         *config.OPMConfig
	releaseBuilder *ReleaseBuilder
	provider       *ProviderLoader
	matcher        *Matcher
	executor       *Executor
}

// NewPipeline creates a new Pipeline implementation.
// The pipeline uses the provided configuration for provider resolution.
func NewPipeline(cfg *config.OPMConfig) Pipeline {
	return &pipeline{
		config:         cfg,
		releaseBuilder: NewReleaseBuilder(cfg.CueContext, cfg.Registry),
		provider:       NewProviderLoader(cfg),
		matcher:        NewMatcher(),
		executor:       NewExecutor(),
	}
}

// Render executes the pipeline and returns results.
//
// The render process follows these phases:
//  1. Resolve module path and inspect metadata via AST (InspectModule — no CUE evaluation)
//  2. Build #ModuleRelease via AST overlay (loads module with overlay + values)
//  3. Load provider and transformers (ProviderLoader)
//  4. Match components to transformers (Matcher)
//  5. Execute transformers (Executor)
//  6. Build and return RenderResult
//
// Fatal errors (module not found, provider missing, incomplete values) return error.
// Render errors (unmatched components, transform failures) are in RenderResult.Errors.
func (p *pipeline) Render(ctx context.Context, opts RenderOptions) (*RenderResult, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// Phase 1: Resolve module path and inspect module metadata via AST
	modulePath, err := module.ResolvePath(opts.ModulePath)
	if err != nil {
		return nil, err
	}

	// When no --values flags are provided, values.cue must exist on disk.
	// When --values flags ARE provided, values.cue on disk is ignored (stubbed
	// out during Build) so the external values take full precedence.
	if len(opts.Values) == 0 {
		valuesPath := filepath.Join(modulePath, "values.cue")
		if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
			return nil, fmt.Errorf("values.cue not found in %s — provide values via values.cue or --values flag", modulePath)
		}
	}

	// Log which values files are being used
	if len(opts.Values) > 0 {
		valuesFileNames := make([]string, len(opts.Values))
		for i, vf := range opts.Values {
			valuesFileNames[i] = filepath.Base(vf)
		}
		output.Debug("using values files", "files", strings.Join(valuesFileNames, ", "))
	} else {
		output.Debug("using default values.cue")
	}

	inspection, err := p.releaseBuilder.InspectModule(modulePath)
	if err != nil {
		return nil, err
	}

	// If AST inspection returned empty name, fall back to BuildInstance + LookupPath
	// to handle computed metadata expressions.
	moduleMeta := &module.MetadataPreview{
		Name:             inspection.Name,
		DefaultNamespace: inspection.DefaultNamespace,
	}
	if moduleMeta.Name == "" {
		fallbackMeta, err := module.ExtractMetadata(p.releaseBuilder.CueContext(), modulePath, opts.Registry)
		if err != nil {
			return nil, err
		}
		moduleMeta = fallbackMeta
	}

	// Phase 2: Build #ModuleRelease (loads module with AST overlay, unifies values)
	releaseName := opts.Name
	if releaseName == "" {
		releaseName = moduleMeta.Name
	}
	namespace := p.resolveNamespace(opts.Namespace, moduleMeta.DefaultNamespace)
	if namespace == "" {
		return nil, &NamespaceRequiredError{ModuleName: moduleMeta.Name}
	}

	release, err := p.releaseBuilder.Build(modulePath, ReleaseOptions{
		Name:      releaseName,
		Namespace: namespace,
		PkgName:   inspection.PkgName,
	}, opts.Values)
	if err != nil {
		return nil, err // Fatal: release building failed (likely incomplete values)
	}

	output.Debug("release built",
		"name", release.Metadata.Name,
		"namespace", release.Metadata.Namespace,
		"components", len(release.Components),
	)

	// Phase 3: Load provider
	providerName := opts.Provider
	// Default to the only configured provider if not specified
	if providerName == "" && p.config != nil && len(p.config.Providers) == 1 {
		for name := range p.config.Providers {
			providerName = name
			break
		}
	}
	provider, err := p.provider.Load(ctx, providerName)
	if err != nil {
		return nil, err // Fatal: provider loading failed
	}

	// Phase 4: Match components to transformers
	components := p.componentsToSlice(release.Components)
	matchResult := p.matcher.Match(components, provider.Transformers)
	matchPlan := convertMatchPlan(matchResult.ToMatchPlan())

	// Collect errors for unmatched components
	var errors []error
	for _, comp := range matchResult.Unmatched {
		errors = append(errors, &UnmatchedComponentError{
			ComponentName: comp.Name,
			Available:     provider.ToSummaries(),
		})
	}

	// Phase 5: Execute transformers (only for matched components)
	var resources []*Resource
	if len(matchResult.ByTransformer) > 0 {
		// Build transformer map for executor
		transformerMap := make(map[string]*LoadedTransformer)
		for _, tf := range provider.Transformers {
			transformerMap[tf.FQN] = tf
		}
		execResult := p.executor.ExecuteWithTransformers(ctx, matchResult, release, transformerMap)
		resources = convertResources(execResult.Resources)
		errors = append(errors, execResult.Errors...)
	}

	// Phase 6: Build result
	// Sort resources with deterministic 5-key total ordering:
	// weight → group → kind → namespace → name
	// This matches the digest sort in internal/inventory, making opm mod build
	// output deterministic for equal-weight resources.
	sort.SliceStable(resources, func(i, j int) bool {
		ri, rj := resources[i], resources[j]
		wi := weights.GetWeight(ri.GVK())
		wj := weights.GetWeight(rj.GVK())
		if wi != wj {
			return wi < wj
		}
		gi, gj := ri.GVK().Group, rj.GVK().Group
		if gi != gj {
			return gi < gj
		}
		ki, kj := ri.GVK().Kind, rj.GVK().Kind
		if ki != kj {
			return ki < kj
		}
		nsi, nsj := ri.Namespace(), rj.Namespace()
		if nsi != nsj {
			return nsi < nsj
		}
		return ri.Name() < rj.Name()
	})

	// Collect warnings (e.g., unhandled traits)
	warnings := collectWarnings(matchResult)

	return &RenderResult{
		Resources: resources,
		Release:   p.releaseToModuleMetadata(release, moduleMeta.Name),
		MatchPlan: matchPlan,
		Errors:    errors,
		Warnings:  warnings,
	}, nil
}

// resolveNamespace resolves the target namespace using precedence:
// 1. --namespace flag (highest)
// 2. module.metadata.defaultNamespace
func (p *pipeline) resolveNamespace(flagValue, defaultNamespace string) string {
	if flagValue != "" {
		return flagValue
	}
	return defaultNamespace
}

// componentsToSlice converts component map to slice for matcher
func (p *pipeline) componentsToSlice(m map[string]*LoadedComponent) []*LoadedComponent {
	result := make([]*LoadedComponent, 0, len(m))
	for _, comp := range m {
		result = append(result, comp)
	}
	return result
}

// releaseToModuleMetadata converts release metadata to ModuleReleaseMetadata for API compatibility.
// moduleName is the canonical module name from module.metadata.name (e.g. "minecraft"),
// which may differ from release.Metadata.Name when --release-name overrides the default.
func (p *pipeline) releaseToModuleMetadata(release *BuiltRelease, moduleName string) ModuleReleaseMetadata {
	names := make([]string, 0, len(release.Components))
	for name := range release.Components {
		names = append(names, name)
	}
	return ModuleReleaseMetadata{
		Name:            release.Metadata.Name,
		ModuleName:      moduleName,
		Namespace:       release.Metadata.Namespace,
		Version:         release.Metadata.Version,
		Labels:          release.Metadata.Labels,
		Components:      names,
		Identity:        release.Metadata.Identity,
		ReleaseIdentity: release.Metadata.ReleaseIdentity,
	}
}

// collectWarnings gathers non-fatal warnings from the match result.
//
// A trait is considered "unhandled" only if NO matched transformer handles it.
// This means if ServiceTransformer requires Expose trait and DeploymentTransformer
// doesn't, the Expose trait is still considered handled (by ServiceTransformer).
func collectWarnings(result *MatchResult) []string {
	var warnings []string

	// Step 1: Count how many transformers matched each component
	componentMatchCount := make(map[string]int)
	for i := range result.Details {
		detail := &result.Details[i]
		if detail.Matched {
			componentMatchCount[detail.ComponentName]++
		}
	}

	// Step 2: Count how many matched transformers consider each trait unhandled
	// Key: component name, Value: map of trait -> count of transformers that don't handle it
	traitUnhandledCount := make(map[string]map[string]int)
	for i := range result.Details {
		detail := &result.Details[i]
		if detail.Matched {
			if traitUnhandledCount[detail.ComponentName] == nil {
				traitUnhandledCount[detail.ComponentName] = make(map[string]int)
			}
			for _, trait := range detail.UnhandledTraits {
				traitUnhandledCount[detail.ComponentName][trait]++
			}
		}
	}

	// Step 3: A trait is truly unhandled only if ALL matched transformers
	// consider it unhandled (i.e., no transformer handles it)
	for componentName, traitCounts := range traitUnhandledCount {
		matchCount := componentMatchCount[componentName]
		for trait, unhandledCount := range traitCounts {
			if unhandledCount == matchCount {
				warnings = append(warnings,
					"component "+componentName+": unhandled trait "+trait)
			}
		}
	}

	return warnings
}

// convertResources converts transform.Resource slice to build.Resource slice.
// Both types have the same fields; this bridges the internal and public types.
func convertResources(in []*transform.Resource) []*Resource {
	out := make([]*Resource, len(in))
	for i, r := range in {
		out[i] = &Resource{
			Object:      r.Object,
			Component:   r.Component,
			Transformer: r.Transformer,
		}
	}
	return out
}

// convertMatchPlan converts transform.MatchPlan to build.MatchPlan.
// Both types have the same fields; this bridges the internal and public types.
func convertMatchPlan(in transform.MatchPlan) MatchPlan {
	matches := make(map[string][]TransformerMatch, len(in.Matches))
	for comp, tms := range in.Matches {
		converted := make([]TransformerMatch, len(tms))
		for i, tm := range tms {
			converted[i] = TransformerMatch{
				TransformerFQN: tm.TransformerFQN,
				Reason:         tm.Reason,
			}
		}
		matches[comp] = converted
	}
	return MatchPlan{
		Matches:   matches,
		Unmatched: in.Unmatched,
	}
}
