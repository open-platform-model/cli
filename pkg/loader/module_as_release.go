package loader

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
)

// SynthesizeModuleRelease constructs a *modulerelease.ModuleRelease from a
// loaded module CUE value and a concrete values CUE value, without requiring
// a release.cue file.
//
// This enables "opm mod build ." and "opm mod apply ." on bare module
// directories (no release.cue), using debugValues or an explicit -f values
// file as the values source.
//
// The synthesis follows the same CUE insight as the release pipeline:
//
//	_#module:   #module & {#config: values}   // fill #config with concrete values
//	components: _#module.#components          // components evaluate with concrete config
//
// Steps:
//  1. Module Gate: validate valuesVal against #config
//  2. Fill #config with values to produce a fully evaluable module
//  3. Extract #components from the filled module
//  4. Wrap #components under a regular "components" field for MatchComponents()
//  5. Finalize components for constraint-free execution
//  6. Decode module metadata from the module value's "metadata" field
//  7. Construct ReleaseMetadata (UUID left empty — no release file to compute it)
//  8. Return NewModuleRelease
func SynthesizeModuleRelease(cueCtx *cue.Context, modVal, valuesVal cue.Value, releaseName, namespace string) (*modulerelease.ModuleRelease, error) {
	// Step 1: Module Gate — validate consumer values against #config.
	// Mirrors the Module Gate in LoadModuleReleaseFromValue, but uses modVal
	// directly instead of reaching through a release value.
	moduleConfigVal := modVal.LookupPath(cue.ParsePath("#config"))
	if cfgErr := validateConfig(moduleConfigVal, valuesVal, "module", releaseName); cfgErr != nil {
		return nil, cfgErr
	}

	// Step 2: Fill #config with the provided values.
	// This resolves all #config references throughout the module's #components.
	filledMod := modVal.FillPath(cue.ParsePath("#config"), valuesVal)
	if err := filledMod.Err(); err != nil {
		return nil, fmt.Errorf("filling #config with values: %w", err)
	}

	// Step 3: Extract schema components from #components.
	// This is a definition field (# prefix), so it preserves #resources, #traits,
	// and #blueprints sub-definitions needed for the CUE match plan evaluator.
	schemaComps := filledMod.LookupPath(cue.ParsePath("#components"))
	if !schemaComps.Exists() {
		return nil, fmt.Errorf("module has no #components field — synthesis requires a standard #Module with #components")
	}
	if err := schemaComps.Err(); err != nil {
		return nil, fmt.Errorf("evaluating #components: %w", err)
	}

	// Step 4: Wrap components under a regular "components" field.
	// MatchComponents() does schema.LookupPath("components") — a regular field,
	// not a definition. We create a synthetic schema value so the engine can find
	// "components" at the expected path.
	syntheticSchema := cueCtx.CompileString("{}")
	syntheticSchema = syntheticSchema.FillPath(cue.ParsePath("components"), schemaComps)
	if err := syntheticSchema.Err(); err != nil {
		return nil, fmt.Errorf("building synthetic schema: %w", err)
	}

	// Step 5: Finalize components for constraint-free FillPath injection.
	// finalizeValue strips matchN validators, close() enforcement, and definition
	// fields, leaving a plain data value suitable for transformer injection.
	dataComponents, err := finalizeValue(cueCtx, schemaComps)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	// Step 6: Decode module metadata from the module value's "metadata" field.
	// In synthesis mode we read directly from modVal.metadata (not #module.metadata
	// as in the release path — there is no release wrapper here).
	modMeta := &module.ModuleMetadata{}
	metaVal := modVal.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("module has no metadata field")
	}
	if err := metaVal.Decode(modMeta); err != nil {
		return nil, fmt.Errorf("decoding module metadata: %w", err)
	}

	// Step 7: Construct ReleaseMetadata.
	// UUID is intentionally left empty — the release UUID is normally computed by
	// CUE using the module UUID and release coordinates. Replicating that in Go
	// would couple us to a CUE catalog implementation detail.
	// For "build": UUID is only metadata in manifest labels — not critical.
	// For "apply": apply.go guards on releaseID != "" before inventory work,
	// so an empty UUID causes inventory tracking to be skipped gracefully.
	relMeta := &modulerelease.ReleaseMetadata{
		Name:      releaseName,
		Namespace: namespace,
	}

	// Step 8: Assemble and return the ModuleRelease.
	return modulerelease.NewModuleRelease(relMeta, module.Module{
		Metadata: modMeta,
		Raw:      modVal,
	}, syntheticSchema, dataComponents), nil
}
