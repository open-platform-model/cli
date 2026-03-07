package engine

import (
	"context"
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/experiments/factory/pkg/bundlerelease"
	"github.com/opmodel/cli/experiments/factory/pkg/core"
	"github.com/opmodel/cli/experiments/factory/pkg/provider"
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
//
// cueModuleDir and cueCtx are passed through to the underlying ModuleRenderer.
func NewBundleRenderer(p *provider.Provider, cueModuleDir string, cueCtx *cue.Context) *BundleRenderer {
	return &BundleRenderer{
		moduleRenderer: NewModuleRenderer(p, cueModuleDir, cueCtx),
	}
}

// Render executes the full OPM pipeline for each module release in the bundle.
//
// Releases are processed in sorted key order for deterministic output.
// Each release is rendered independently via ModuleRenderer.Render().
// If any single release fails, rendering stops and the error is returned
// with context about which release failed.
func (br *BundleRenderer) Render(ctx context.Context, rel *bundlerelease.BundleRelease) (*BundleRenderResult, error) {
	// Sort release keys for deterministic ordering.
	keys := make([]string, 0, len(rel.Releases))
	for k := range rel.Releases {
		keys = append(keys, k)
	}
	sort.Strings(keys)

	result := &BundleRenderResult{
		ReleaseOrder: keys,
	}

	for _, key := range keys {
		select {
		case <-ctx.Done():
			return result, ctx.Err()
		default:
		}

		modRel := rel.Releases[key]

		modResult, err := br.moduleRenderer.Render(ctx, modRel)
		if err != nil {
			return nil, fmt.Errorf("rendering release %q (module %s): %w",
				key, modRel.Module.Metadata.FQN, err)
		}

		result.Resources = append(result.Resources, modResult.Resources...)
		result.Warnings = append(result.Warnings, modResult.Warnings...)
	}

	return result, nil
}
