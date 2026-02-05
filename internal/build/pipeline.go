// Package build provides the render pipeline implementation.
package build

import (
	"context"
	"runtime"
	"sort"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/internal/output"
	"github.com/opmodel/cli/pkg/weights"
)

// pipeline implements the Pipeline interface.
// It orchestrates module loading, release building, provider loading,
// component matching, and parallel transformer execution.
type pipeline struct {
	config         *config.OPMConfig
	module         *ModuleLoader
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
		module:         NewModuleLoader(cfg.CueContext),
		releaseBuilder: NewReleaseBuilder(cfg.CueContext, cfg.Registry),
		provider:       NewProviderLoader(cfg),
		matcher:        NewMatcher(),
		executor:       NewExecutor(runtime.NumCPU()),
	}
}

// Render executes the pipeline and returns results.
//
// The render process follows these phases:
//  1. Load module and values (ModuleLoader)
//  2. Build #ModuleRelease with concrete components (ReleaseBuilder)
//  3. Load provider and transformers (ProviderLoader)
//  4. Match components to transformers (Matcher)
//  5. Execute transformers in parallel (Executor)
//  6. Build and return RenderResult
//
// Fatal errors (module not found, provider missing, incomplete values) return error.
// Render errors (unmatched components, transform failures) are in RenderResult.Errors.
func (p *pipeline) Render(ctx context.Context, opts RenderOptions) (*RenderResult, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// Phase 1: Load raw module and values
	module, err := p.module.Load(ctx, opts)
	if err != nil {
		return nil, err // Fatal: module loading failed
	}

	// Phase 2: Build #ModuleRelease (makes components concrete)
	releaseName := opts.Name
	if releaseName == "" {
		releaseName = module.Name
	}
	release, err := p.releaseBuilder.Build(module.Value, ReleaseOptions{
		Name:      releaseName,
		Namespace: module.Namespace,
	})
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
	matchPlan := matchResult.ToMatchPlan()

	// Collect errors for unmatched components
	var errors []error
	for _, comp := range matchResult.Unmatched {
		errors = append(errors, &UnmatchedComponentError{
			ComponentName: comp.Name,
			Available:     provider.ToSummaries(),
		})
	}

	// Phase 5: Execute transformers in parallel (only for matched components)
	var resources []*Resource
	if len(matchResult.ByTransformer) > 0 {
		// Build transformer map for executor
		transformerMap := make(map[string]*LoadedTransformer)
		for _, tf := range provider.Transformers {
			transformerMap[tf.FQN] = tf
		}
		execResult := p.executor.ExecuteWithTransformers(ctx, matchResult, release, transformerMap)
		resources = execResult.Resources
		errors = append(errors, execResult.Errors...)
	}

	// Phase 6: Build result
	// Sort resources by weight
	sort.Slice(resources, func(i, j int) bool {
		wi := weights.GetWeight(resources[i].GVK())
		wj := weights.GetWeight(resources[j].GVK())
		return wi < wj
	})

	// Collect warnings (e.g., unhandled traits in non-strict mode)
	warnings := collectWarnings(matchResult, opts.Strict)

	return &RenderResult{
		Resources: resources,
		Module:    p.releaseToModuleMetadata(release),
		MatchPlan: matchPlan,
		Errors:    errors,
		Warnings:  warnings,
	}, nil
}

// componentsToSlice converts component map to slice for matcher
func (p *pipeline) componentsToSlice(m map[string]*LoadedComponent) []*LoadedComponent {
	result := make([]*LoadedComponent, 0, len(m))
	for _, comp := range m {
		result = append(result, comp)
	}
	return result
}

// releaseToModuleMetadata converts release metadata to ModuleMetadata for API compatibility
func (p *pipeline) releaseToModuleMetadata(release *BuiltRelease) ModuleMetadata {
	names := make([]string, 0, len(release.Components))
	for name := range release.Components {
		names = append(names, name)
	}
	return ModuleMetadata{
		Name:       release.Metadata.Name,
		Namespace:  release.Metadata.Namespace,
		Version:    release.Metadata.Version,
		Labels:     release.Metadata.Labels,
		Components: names,
	}
}

// collectWarnings gathers non-fatal warnings from the match result.
// In strict mode, unhandled traits are errors, not warnings.
//
// A trait is considered "unhandled" only if NO matched transformer handles it.
// This means if ServiceTransformer requires Expose trait and DeploymentTransformer
// doesn't, the Expose trait is still considered handled (by ServiceTransformer).
func collectWarnings(result *MatchResult, strict bool) []string {
	var warnings []string

	if !strict {
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
				// If unhandledCount equals matchCount, no transformer handled this trait
				if unhandledCount == matchCount {
					warnings = append(warnings,
						"component "+componentName+": unhandled trait "+trait)
				}
			}
		}
	}

	return warnings
}
