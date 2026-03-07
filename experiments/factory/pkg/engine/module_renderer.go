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
//	cueCtx := cuecontext.New()
//	r := engine.NewModuleRenderer(provider, cueModuleDir, cueCtx)
//
//	result, err := r.Render(ctx, release)
//	// result.Resources — decoded Kubernetes resources with provenance
//	// result.Warnings  — unhandled trait warnings
package engine

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/experiments/factory/pkg/core"
	"github.com/opmodel/cli/experiments/factory/pkg/modulerelease"
	"github.com/opmodel/cli/experiments/factory/pkg/provider"
)

// ModuleRenderer drives the OPM render pipeline for a single ModuleRelease.
//
// A ModuleRenderer is constructed once per provider and reused across multiple
// Render calls. It is not safe for concurrent use (CUE context is single-threaded).
type ModuleRenderer struct {
	provider     *provider.Provider
	cueModuleDir string // absolute path to the factory v1alpha1/ CUE module root
	cueCtx       *cue.Context
}

// RenderResult holds the output of a successful Render call.
type RenderResult struct {
	// Resources is the ordered list of rendered Kubernetes resources.
	// Each resource carries Component and Transformer provenance for inventory tracking.
	Resources []*core.Resource

	// Warnings is a list of human-readable advisory messages (e.g. unhandled traits).
	// A non-empty Warnings slice does NOT indicate failure.
	Warnings []string
}

// NewModuleRenderer creates a ModuleRenderer for the given provider.
//
// cueModuleDir must be the absolute path to the factory v1alpha1 CUE module
// directory (the one containing cue.mod/). It is used to load the matcher package.
// cueCtx must be the same context used to build the provider and release values.
func NewModuleRenderer(p *provider.Provider, cueModuleDir string, cueCtx *cue.Context) *ModuleRenderer {
	return &ModuleRenderer{
		provider:     p,
		cueModuleDir: cueModuleDir,
		cueCtx:       cueCtx,
	}
}

// Render executes the full OPM pipeline for the given module release:
//  1. CUE evaluates #MatchPlan to determine which transformers apply to which components.
//  2. Go checks for unmatched components — returns an error if any are found.
//  3. Go executes each matched pair: injects #context, evaluates #transform, decodes output.
//  4. Warnings for unhandled traits are collected and returned.
//
// The returned error summarises all per-pair execution failures; execution continues
// past individual pair errors so all matches are attempted.
func (r *ModuleRenderer) Render(ctx context.Context, rel *modulerelease.ModuleRelease) (*RenderResult, error) {
	// Extract the components CUE value from Schema (preserves #resources, #traits
	// definition fields required by the match plan). DataComponents is already the
	// finalized, constraint-free components value — safe for FillPath injection.
	schemaComponents := rel.Schema.LookupPath(cue.ParsePath("components"))
	if !schemaComponents.Exists() {
		return nil, fmt.Errorf("release %q: no components field in schema CUE value", rel.Metadata.Name)
	}
	dataComponents := rel.DataComponents
	if !dataComponents.Exists() {
		return nil, fmt.Errorf("release %q: no finalized data components value", rel.Metadata.Name)
	}

	// Phase 1 — matching (CUE #MatchPlan evaluation).
	// Uses schemaComponents so that definition fields (#resources, #traits) are
	// preserved for the CUE matcher's comprehension logic.
	plan, err := buildMatchPlan(r.cueCtx, r.cueModuleDir, r.provider.Data, schemaComponents)
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
	// (from rel.DataComponents — already finalized, no materialize() needed).
	resources, errs := executeTransforms(ctx, r.cueCtx, plan, r.provider.Data, schemaComponents, dataComponents, rel)
	if len(errs) > 0 {
		return nil, fmt.Errorf("executing transforms: %w", joinErrors(errs))
	}

	return &RenderResult{
		Resources: resources,
		Warnings:  plan.Warnings(),
	}, nil
}

// UnmatchedComponentsError is returned when one or more components have no
// matching transformer. It includes diagnostics for each unmatched component.
type UnmatchedComponentsError struct {
	// Components is the list of component names with no matching transformer.
	Components []string

	// Matches is the full match result matrix, used to build per-component diagnostics.
	Matches map[string]map[string]MatchResult
}

func (e *UnmatchedComponentsError) Error() string {
	var sb strings.Builder
	fmt.Fprintf(&sb, "%d component(s) have no matching transformer: %v\n",
		len(e.Components), e.Components)

	for _, compName := range e.Components {
		tfResults, ok := e.Matches[compName]
		if !ok {
			continue
		}
		fmt.Fprintf(&sb, "  component %q:\n", compName)
		for tfFQN, result := range tfResults {
			if result.Matched {
				continue
			}
			fmt.Fprintf(&sb, "    transformer %q did not match:\n", tfFQN)
			if len(result.MissingLabels) > 0 {
				fmt.Fprintf(&sb, "      missing labels:    %v\n", result.MissingLabels)
			}
			if len(result.MissingResources) > 0 {
				fmt.Fprintf(&sb, "      missing resources: %v\n", result.MissingResources)
			}
			if len(result.MissingTraits) > 0 {
				fmt.Fprintf(&sb, "      missing traits:    %v\n", result.MissingTraits)
			}
		}
	}

	return sb.String()
}

// joinErrors combines multiple errors into one unwrappable error using errors.Join.
// Each constituent error remains accessible via errors.Is / errors.As.
func joinErrors(errs []error) error {
	return errors.Join(errs...)
}
