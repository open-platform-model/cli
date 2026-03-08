// Package engine implements the OPM render pipeline using CUE-native matching.
//
// The pipeline has two phases:
//  1. Match — CUE evaluates matcher.#MatchPlan to determine which transformers
//     apply to which components. Go decodes the structured result.
//  2. Execute — For each matched (component, transformer) pair, Go calls back into
//     CUE to run the #transform function and decodes the output resources.
//
// This package is intentionally minimal: no Kubernetes apply logic, no CLI output.
// It is designed to be embedded in other tools that need OPM rendering.
//
// Basic usage:
//
//	r := engine.NewModuleRenderer(provider, matcherDef)
//
//	result, err := r.Render(ctx, release)
//	// result.Resources   — decoded Kubernetes resources with provenance
//	// result.MatchPlan   — which transformers matched which components
//	// result.Components  — per-component summary for verbose output
//	// result.Warnings    — unhandled trait warnings
package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/core"
	"github.com/opmodel/cli/pkg/modulerelease"
	"github.com/opmodel/cli/pkg/provider"
)

// ComponentSummary contains display-oriented summary data extracted from a component
// after the render pipeline. It captures the key properties useful for verbose output
// without exposing cue.Value fields.
type ComponentSummary struct {
	// Name is the component name.
	Name string

	// Labels are the component-level labels from metadata.labels.
	// Example: {"core.opmodel.dev/workload-type": "stateless"}
	Labels map[string]string

	// ResourceFQNs are the FQNs of resource types declared by the component.
	// Sorted for deterministic output.
	// Example: ["opmodel.dev/resources/workload/container@v1"]
	ResourceFQNs []string

	// TraitFQNs are the FQNs of traits declared by the component.
	// Sorted for deterministic output.
	// Example: ["opmodel.dev/traits/network/expose@v1"]
	TraitFQNs []string
}

// ModuleRenderer drives the OPM render pipeline for a single ModuleRelease.
//
// A ModuleRenderer is constructed once per provider and reused across multiple
// Render calls. It is not safe for concurrent use (CUE context is single-threaded).
type ModuleRenderer struct {
	provider   *provider.Provider
	matcherDef cue.Value // pre-evaluated #MatchPlan definition from config
}

// RenderResult holds the output of a successful Render call.
type RenderResult struct {
	// Resources is the ordered list of rendered Kubernetes resources.
	// Each resource carries Component and Transformer provenance for inventory tracking.
	Resources []*core.Resource

	// MatchPlan is the decoded result of matching components against transformers.
	// Nil if matching was not performed (e.g. no components).
	MatchPlan *MatchPlan

	// Components is a per-component summary for verbose output, sorted by name.
	Components []ComponentSummary

	// Warnings is a list of human-readable advisory messages (e.g. unhandled traits).
	// A non-empty Warnings slice does NOT indicate failure.
	Warnings []string
}

// NewModuleRenderer creates a ModuleRenderer for the given provider.
//
// matcherDef must be the pre-evaluated #MatchPlan cue.Value loaded from config
// (via internal/config.extractMatcher). The CUE context is derived from
// provider.Data.Context() — no separate cueCtx parameter is needed.
func NewModuleRenderer(p *provider.Provider, matcherDef cue.Value) *ModuleRenderer {
	return &ModuleRenderer{
		provider:   p,
		matcherDef: matcherDef,
	}
}

// Render executes the full OPM pipeline for the given module release:
//  1. CUE evaluates #MatchPlan to determine which transformers apply to which components.
//  2. Go checks for unmatched components — returns an error if any are found.
//  3. Go executes each matched pair: injects #context, evaluates #transform, decodes output.
//  4. Warnings for unhandled traits are collected and returned.
//
// The returned error summarizes all per-pair execution failures; execution continues
// past individual pair errors so all matches are attempted.
func (r *ModuleRenderer) Render(ctx context.Context, rel *modulerelease.ModuleRelease) (*RenderResult, error) {
	// Extract the components CUE values from the ModuleRelease.
	// MatchComponents() returns the schema value (preserves #resources, #traits
	// definition fields required by the match plan).
	// ExecuteComponents() returns the finalized, constraint-free components value
	// safe for FillPath injection into transformer #transform definitions.
	schemaComponents := rel.MatchComponents()
	if !schemaComponents.Exists() {
		return nil, fmt.Errorf("release %q: no components field in schema CUE value", rel.Metadata.Name)
	}
	dataComponents := rel.ExecuteComponents()
	if !dataComponents.Exists() {
		return nil, fmt.Errorf("release %q: no finalized data components value", rel.Metadata.Name)
	}

	// The CUE context lives on each cue.Value — extract it from the provider.
	cueCtx := r.provider.Data.Context()

	// Phase 1 — matching (CUE #MatchPlan evaluation).
	// Uses schemaComponents so that definition fields (#resources, #traits) are
	// preserved for the CUE matcher's comprehension logic.
	plan, err := buildMatchPlan(r.matcherDef, r.provider.Data, schemaComponents)
	if err != nil {
		return nil, fmt.Errorf("building match plan: %w", err)
	}

	// Error on unmatched components — these cannot be rendered.
	if len(plan.Unmatched) > 0 {
		return nil, &UnmatchedComponentsError{
			Components: plan.Unmatched,
			Matches:    plan.Matches,
		}
	}

	// Phase 2 — execution (CUE #transform per pair).
	// Passes both schemaComponents (for metadata extraction) and dataComponents
	// (from rel.ExecuteComponents() — already finalized, no materialize() needed).
	resources, errs := executeTransforms(ctx, cueCtx, plan, r.provider.Data, schemaComponents, dataComponents, rel)
	if len(errs) > 0 {
		return nil, fmt.Errorf("executing transforms: %w", errors.Join(errs...))
	}

	return &RenderResult{
		Resources:  resources,
		MatchPlan:  plan,
		Components: extractComponentSummaries(schemaComponents),
		Warnings:   plan.Warnings(),
	}, nil
}

// extractComponentSummaries iterates the schemaComponents CUE value and builds
// a sorted []ComponentSummary for verbose output.
//
// schemaComponents is the value from rel.MatchComponents() which preserves the
// definition fields (#resources, #traits) that carry FQN keys.
func extractComponentSummaries(schemaComponents cue.Value) []ComponentSummary {
	iter, err := schemaComponents.Fields()
	if err != nil {
		return nil
	}

	var summaries []ComponentSummary
	for iter.Next() {
		compName := iter.Selector().Unquoted()
		compVal := iter.Value()

		summary := ComponentSummary{Name: compName}

		// Extract metadata.labels (optional field).
		if labelsVal := compVal.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
			var labels map[string]string
			if err := labelsVal.Decode(&labels); err == nil {
				summary.Labels = labels
			}
		}

		// Extract #resources keys (definition field — FQN keys).
		if resourcesVal := compVal.LookupPath(cue.MakePath(cue.Def("resources"))); resourcesVal.Exists() {
			var fqns []string
			ri, err := resourcesVal.Fields()
			if err == nil {
				for ri.Next() {
					fqns = append(fqns, ri.Selector().Unquoted())
				}
			}
			sort.Strings(fqns)
			summary.ResourceFQNs = fqns
		}

		// Extract #traits keys (definition field — FQN keys). Optional.
		if traitsVal := compVal.LookupPath(cue.MakePath(cue.Def("traits"))); traitsVal.Exists() {
			var fqns []string
			ti, err := traitsVal.Fields()
			if err == nil {
				for ti.Next() {
					fqns = append(fqns, ti.Selector().Unquoted())
				}
			}
			sort.Strings(fqns)
			summary.TraitFQNs = fqns
		}

		summaries = append(summaries, summary)
	}

	sort.Slice(summaries, func(i, j int) bool {
		return summaries[i].Name < summaries[j].Name
	})
	return summaries
}
