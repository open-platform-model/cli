// Package build provides the render pipeline implementation.
package build

import (
	"context"
	"runtime"
	"sort"

	"github.com/opmodel/cli/internal/config"
	"github.com/opmodel/cli/pkg/weights"
)

// pipeline implements the Pipeline interface.
// It orchestrates module loading, provider loading, component matching,
// and parallel transformer execution.
type pipeline struct {
	config   *config.OPMConfig
	loader   *Loader
	provider *ProviderLoader
	matcher  *Matcher
	executor *Executor
}

// NewPipeline creates a new Pipeline implementation.
// The pipeline uses the provided configuration for provider resolution.
func NewPipeline(cfg *config.OPMConfig) Pipeline {
	return &pipeline{
		config:   cfg,
		loader:   NewLoader(),
		provider: NewProviderLoader(cfg),
		matcher:  NewMatcher(),
		executor: NewExecutor(runtime.NumCPU()),
	}
}

// Render executes the pipeline and returns results.
//
// The render process follows these phases:
//  1. Load module and values (Loader)
//  2. Load provider and transformers (ProviderLoader)
//  3. Match components to transformers (Matcher)
//  4. Execute transformers in parallel (Executor)
//  5. Build and return RenderResult
//
// Fatal errors (module not found, provider missing) return error.
// Render errors (unmatched components, transform failures) are in RenderResult.Errors.
func (p *pipeline) Render(ctx context.Context, opts RenderOptions) (*RenderResult, error) {
	// Validate options
	if err := opts.Validate(); err != nil {
		return nil, err
	}

	// Phase 1: Load module and values
	module, err := p.loader.Load(ctx, opts)
	if err != nil {
		return nil, err // Fatal: module loading failed
	}

	// Phase 2: Load provider
	providerName := opts.Provider
	if providerName == "" && p.config != nil && p.config.Config != nil {
		// Use default provider from config if not specified
		// TODO: Add defaultProvider to config
	}
	provider, err := p.provider.Load(ctx, providerName)
	if err != nil {
		return nil, err // Fatal: provider loading failed
	}

	// Phase 3: Match components to transformers
	matchResult := p.matcher.Match(module.Components, provider.Transformers)
	matchPlan := matchResult.ToMatchPlan()

	// Collect errors for unmatched components
	var errors []error
	for _, comp := range matchResult.Unmatched {
		errors = append(errors, &UnmatchedComponentError{
			ComponentName: comp.Name,
			Available:     provider.ToSummaries(),
		})
	}

	// Phase 4: Execute transformers in parallel (only for matched components)
	var resources []*Resource
	if len(matchResult.ByTransformer) > 0 {
		execResult := p.executor.Execute(ctx, matchResult, module)
		resources = execResult.Resources
		errors = append(errors, execResult.Errors...)
	}

	// Phase 5: Build result
	// Sort resources by weight for sequential apply
	sort.Slice(resources, func(i, j int) bool {
		wi := weights.GetWeight(resources[i].GVK())
		wj := weights.GetWeight(resources[j].GVK())
		return wi < wj
	})

	// Collect warnings (e.g., unhandled traits in non-strict mode)
	warnings := collectWarnings(matchResult, opts.Strict)

	return &RenderResult{
		Resources: resources,
		Module:    module.Metadata(),
		MatchPlan: matchPlan,
		Errors:    errors,
		Warnings:  warnings,
	}, nil
}

// collectWarnings gathers non-fatal warnings from the match result.
// In strict mode, unhandled traits are errors, not warnings.
func collectWarnings(result *MatchResult, strict bool) []string {
	var warnings []string

	if !strict {
		// In non-strict mode, report unhandled traits as warnings
		for _, detail := range result.Details {
			if detail.Matched && len(detail.UnhandledTraits) > 0 {
				for _, trait := range detail.UnhandledTraits {
					warnings = append(warnings,
						"component "+detail.ComponentName+": unhandled trait "+trait)
				}
			}
		}
	}

	return warnings
}
