package loader

import (
	"fmt"
	"sort"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/experiments/factory/internal/core/bundle"
	"github.com/opmodel/cli/experiments/factory/internal/core/bundlerelease"
	"github.com/opmodel/cli/experiments/factory/internal/core/module"
	"github.com/opmodel/cli/experiments/factory/internal/core/modulerelease"
)

// LoadBundleReleaseFromValue decodes a BundleRelease from an already-loaded
// CUE package value.
//
// The CUE #BundleRelease comprehension has already produced a `releases` map
// of #ModuleRelease entries (one per #BundleInstance). This function:
//
//  1. Decodes bundle release metadata (name, uuid) from the schema value.
//  2. Decodes the bundle metadata from #bundle.metadata.
//  3. Bundle Gate: validates consumer values against #bundle.#config — catches
//     type mismatches and missing required fields at the bundle/consumer boundary
//     with a clear, attributed error before any comprehension output is read.
//  4. Validates the `releases` struct is evaluable (catches structural CUE errors
//     not intercepted by the Bundle Gate).
//  5. Iterates the `releases` struct — for each entry runs the Module Gate
//     (validates instance values against #module.#config), verifies the whole
//     release entry is concrete, then finalizes the `components` field.
//
// The returned BundleRelease.Releases map is keyed by instance name (matching
// the CUE `releases` keys, e.g. "server", "proxy").
func LoadBundleReleaseFromValue(cueCtx *cue.Context, pkg cue.Value) (*bundlerelease.BundleRelease, error) {
	if err := pkg.Err(); err != nil {
		return nil, fmt.Errorf("evaluating bundle release: %w", err)
	}

	// Extract bundle release metadata from the schema value (preserves uuid etc.)
	brMeta, err := extractBundleReleaseMetadata(pkg)
	if err != nil {
		return nil, fmt.Errorf("extracting bundle release metadata: %w", err)
	}

	// Extract bundle metadata from the #bundle hidden field.
	bundleMeta, bundleRaw, err := extractBundleInfo(pkg)
	if err != nil {
		return nil, fmt.Errorf("extracting bundle info: %w", err)
	}

	// Bundle Gate: validate consumer values against #bundle.#config before
	// reading the releases comprehension output. Catches type mismatches and
	// missing required fields at the bundle/consumer boundary — produces a
	// clear error attributed to the bundle release name rather than a deeply
	// nested CUE unification error from the comprehension output.
	bundleConfigVal := pkg.LookupPath(cue.ParsePath("#bundle.#config"))
	bundleValuesVal := pkg.LookupPath(cue.ParsePath("values"))
	if cfgErr := validateConfig(bundleConfigVal, bundleValuesVal, "bundle", brMeta.Name); cfgErr != nil {
		return nil, cfgErr
	}

	// Validate that the releases struct is evaluable before iterating.
	// Catches comprehension-level errors not intercepted by the Bundle Gate
	// (e.g. structural CUE errors in the bundle definition itself).
	releasesVal := pkg.LookupPath(cue.ParsePath("releases"))
	if !releasesVal.Exists() {
		return nil, fmt.Errorf("no releases produced — bundle definition may have errors")
	}
	if err := releasesVal.Err(); err != nil {
		return nil, fmt.Errorf("evaluating bundle releases: %w", err)
	}

	// Iterate the `releases` map with per-release finalization.
	// Each release's components are finalized individually.
	releases, err := extractBundleReleases(cueCtx, pkg)
	if err != nil {
		return nil, fmt.Errorf("extracting bundle releases: %w", err)
	}

	return &bundlerelease.BundleRelease{
		Metadata: brMeta,
		Bundle: bundle.Bundle{
			Metadata: bundleMeta,
			Data:     bundleRaw,
		},
		Releases: releases,
		Schema:   pkg,
	}, nil
}

// extractBundleReleaseMetadata decodes the bundle release metadata struct.
func extractBundleReleaseMetadata(v cue.Value) (*bundlerelease.BundleReleaseMetadata, error) {
	meta := &bundlerelease.BundleReleaseMetadata{}

	metaVal := v.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("metadata field not found in bundle release")
	}
	if err := metaVal.Decode(meta); err != nil {
		return nil, fmt.Errorf("decoding bundle release metadata: %w", err)
	}
	if meta.Name == "" {
		return nil, fmt.Errorf("bundle release metadata.name is empty")
	}

	return meta, nil
}

// extractBundleInfo reads bundle-level metadata from the release's #bundle
// hidden field.
func extractBundleInfo(releaseVal cue.Value) (*bundle.BundleMetadata, cue.Value, error) {
	bundleVal := releaseVal.LookupPath(cue.ParsePath("#bundle"))
	if !bundleVal.Exists() {
		return nil, cue.Value{}, fmt.Errorf("#bundle field not found in bundle release value")
	}
	if err := bundleVal.Err(); err != nil {
		return nil, cue.Value{}, fmt.Errorf("evaluating #bundle: %w", err)
	}

	meta := &bundle.BundleMetadata{}
	metaVal := bundleVal.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, cue.Value{}, fmt.Errorf("metadata field not found in #bundle")
	}
	if err := metaVal.Decode(meta); err != nil {
		return nil, cue.Value{}, fmt.Errorf("decoding bundle metadata: %w", err)
	}

	return meta, bundleVal, nil
}

// extractBundleReleases iterates the `releases` struct and builds a
// *ModuleRelease for each entry. Keys are sorted for deterministic ordering.
//
// For each release entry the Module Gate runs before finalization:
//
//   - Module Gate: validateConfig(#module.#config, values) — validates the
//     instance values against the module's #config schema. Catches type
//     mismatches and missing required fields at the module boundary, producing
//     a clear per-release error before any finalization is attempted.
//   - Concreteness check: schemaEntry.Validate(cue.Concrete(true)) — verifies
//     the whole resolved #ModuleRelease is concrete. Safe to run on the full
//     entry because the Bundle Gate and Module Gate have already validated
//     values at both levels.
//
// Only the `components` field is finalized — the `values` field carries
// #config validators that would be rejected by BuildExpr. Components are
// self-contained Kubernetes resource specs that finalize cleanly.
func extractBundleReleases(cueCtx *cue.Context, schemaPkg cue.Value) (map[string]*modulerelease.ModuleRelease, error) {
	schemaReleasesVal := schemaPkg.LookupPath(cue.ParsePath("releases"))
	if !schemaReleasesVal.Exists() {
		return nil, fmt.Errorf("releases field not found in bundle release")
	}
	if err := schemaReleasesVal.Err(); err != nil {
		return nil, fmt.Errorf("evaluating releases: %w", err)
	}

	// Collect field names first for deterministic iteration.
	iter, err := schemaReleasesVal.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating releases: %w", err)
	}
	var keys []string
	for iter.Next() {
		keys = append(keys, iter.Selector().Unquoted())
	}
	sort.Strings(keys)

	releases := make(map[string]*modulerelease.ModuleRelease, len(keys))
	for _, key := range keys {
		schemaEntry := schemaReleasesVal.LookupPath(cue.MakePath(cue.Str(key)))

		// Catch CUE evaluation errors on the release entry before gate checks.
		if err := schemaEntry.Err(); err != nil {
			return nil, fmt.Errorf("release %q: evaluation error: %w", key, err)
		}

		// Module Gate: validate instance values against #module.#config.
		// This runs the same check as the standalone Module Gate, scoped to this
		// release entry. The instance name is used as the display name.
		moduleConfigVal := schemaEntry.LookupPath(cue.ParsePath("#module.#config"))
		moduleValuesVal := schemaEntry.LookupPath(cue.ParsePath("values"))
		if cfgErr := validateConfig(moduleConfigVal, moduleValuesVal, "module", key); cfgErr != nil {
			return nil, fmt.Errorf("release %q: %w", key, cfgErr)
		}

		// Concreteness check on the whole release entry.
		// Safe here because Bundle Gate + Module Gate have already validated values
		// at both levels. If this fails, it indicates a gap in the bundle wiring
		// that the gates didn't catch (a bug, not a user config error).
		if err := schemaEntry.Validate(cue.Concrete(true)); err != nil {
			return nil, fmt.Errorf("release %q: not fully concrete after gate validation "+
				"(bundle wiring may be incomplete): %w", key, err)
		}

		componentsVal := schemaEntry.LookupPath(cue.ParsePath("components"))
		if !componentsVal.Exists() {
			return nil, fmt.Errorf("release %q: no components field in schema", key)
		}

		dataComponents, err := finalizeValue(cueCtx, componentsVal)
		if err != nil {
			return nil, fmt.Errorf("release %q: finalizing components: %w", key, err)
		}

		rel, err := decodeModuleReleaseEntry(key, schemaEntry, dataComponents)
		if err != nil {
			return nil, fmt.Errorf("release %q: %w", key, err)
		}
		releases[key] = rel
	}

	return releases, nil
}

// decodeModuleReleaseEntry decodes a single ModuleRelease entry from the
// `releases` map. Each entry is a fully resolved #ModuleRelease value
// produced by the CUE #BundleRelease comprehension.
//
// schemaEntry is the original constrained value (for ModuleRenderer matching).
// dataComponents is the finalized, constraint-free components value (for FillPath injection).
func decodeModuleReleaseEntry(name string, schemaEntry cue.Value, dataComponents cue.Value) (*modulerelease.ModuleRelease, error) {
	// Extract release metadata from the schema entry (has all fields).
	relMeta := &modulerelease.ReleaseMetadata{Name: name}
	metaVal := schemaEntry.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("metadata field not found")
	}
	if err := metaVal.Decode(relMeta); err != nil {
		return nil, fmt.Errorf("decoding release metadata: %w", err)
	}

	// Extract module info from the #module hidden field of the schema entry.
	modMeta, modRaw, err := extractModuleInfo(schemaEntry)
	if err != nil {
		return nil, fmt.Errorf("extracting module info: %w", err)
	}

	return &modulerelease.ModuleRelease{
		Metadata: relMeta,
		Module: module.Module{
			Metadata: modMeta,
			Raw:      modRaw,
		},
		DataComponents: dataComponents, // finalized components only — safe for FillPath
		Schema:         schemaEntry,    // original constrained value — for transformer matching
	}, nil
}
