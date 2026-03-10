package engine

import (
	"context"
	"errors"
	"fmt"
	"sort"

	"github.com/opmodel/cli/pkg/bundlerelease"
	"github.com/opmodel/cli/pkg/core"
	"github.com/opmodel/cli/pkg/match"
	"github.com/opmodel/cli/pkg/provider"
)

// BundleRenderer drives the OPM render pipeline for a BundleRelease.
//
// It iterates the releases map (produced by CUE's #BundleRelease comprehension,
// one entry per #BundleInstance) and calls ModuleRenderer.Render() for each
// ModuleRelease entry. Resources from all releases are collected into a single
// result with provenance tracking via the Resource.Release field.
//
// A BundleRenderer is not safe for concurrent use (CUE context is single-threaded).
type BundleRenderer struct {
	moduleRenderer *ModuleRenderer
}

// BundleRenderResult holds the output of a successful BundleRenderer.Render call.
type BundleRenderResult struct {
	// Resources is the ordered list of rendered Kubernetes resources from all
	// module releases in the bundle. Each resource carries Release, Component,
	// and Transformer provenance.
	Resources []*core.Resource

	// Warnings is a list of human-readable advisory messages aggregated from
	// all module release renders.
	Warnings []string

	// ReleaseOrder is the sorted list of release instance names, preserving
	// the order in which releases were rendered. Useful for display.
	ReleaseOrder []string
}

// NewBundleRenderer creates a BundleRenderer for the given provider.
func NewBundleRenderer(p *provider.Provider) *BundleRenderer {
	return &BundleRenderer{
		moduleRenderer: NewModuleRenderer(p),
	}
}

// Render executes the full OPM pipeline for each module release in the bundle.
//
// Releases are processed in sorted key order for deterministic output.
// Each release is rendered independently via ModuleRenderer.Render().
//
// Fix for DEBT.md #5: Fail-slow at bundle level — all releases are attempted even
// if earlier ones fail. Errors from all failed releases are collected and returned
// together so the operator can see all failures in one pass. This matches the
// fail-slow behavior of executeTransforms at the pair level.
func (br *BundleRenderer) Render(ctx context.Context, rel *bundlerelease.BundleRelease) (*BundleRenderResult, error) {
	// Sort release keys for deterministic ordering.
	keys := make([]string, 0, len(rel.Releases))
	for k := range rel.Releases {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := &BundleRenderResult{
		Resources:    []*core.Resource{},
		Warnings:     []string{},
		ReleaseOrder: keys,
	}

	var errs []error

	for _, key := range keys {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		modRel := rel.Releases[key]

		plan, err := match.Match(modRel.MatchComponents(), br.moduleRenderer.provider)
		if err != nil {
			errs = append(errs, fmt.Errorf("matching release %q (module %s): %w",
				key, modRel.Module.Metadata.FQN, err))
			continue
		}

		modResult, err := br.moduleRenderer.Render(ctx, modRel, plan)
		if err != nil {
			// Collect the error and continue — fail-slow so all releases are attempted.
			errs = append(errs, fmt.Errorf("rendering release %q (module %s): %w",
				key, modRel.Module.Metadata.FQN, err))
			continue
		}

		result.Resources = append(result.Resources, modResult.Resources...)
		result.Warnings = append(result.Warnings, modResult.Warnings...)
	}

	if len(errs) > 0 {
		return result, errors.Join(errs...)
	}

	return result, nil
}
