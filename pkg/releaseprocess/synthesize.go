package releaseprocess

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/pkg/module"
	"github.com/opmodel/cli/pkg/modulerelease"
)

func SynthesizeModuleRelease(cueCtx *cue.Context, modVal, valuesVal cue.Value, releaseName, namespace string) (*modulerelease.ModuleRelease, error) {
	moduleConfigVal := modVal.LookupPath(cue.ParsePath("#config"))
	mergedValues, cfgErr := ValidateConfig(moduleConfigVal, []cue.Value{valuesVal}, "module", releaseName)
	if cfgErr != nil {
		return nil, cfgErr
	}

	filledMod := modVal.FillPath(cue.ParsePath("#config"), mergedValues)
	if err := filledMod.Err(); err != nil {
		return nil, fmt.Errorf("filling #config with values: %w", err)
	}

	schemaComps := filledMod.LookupPath(cue.ParsePath("#components"))
	if !schemaComps.Exists() {
		return nil, fmt.Errorf("module has no #components field - synthesis requires a standard #Module with #components")
	}
	if err := schemaComps.Err(); err != nil {
		return nil, fmt.Errorf("evaluating #components: %w", err)
	}

	rawCUE := cueCtx.CompileString("{}")
	rawCUE = rawCUE.FillPath(cue.ParsePath("components"), schemaComps)
	if err := rawCUE.Err(); err != nil {
		return nil, fmt.Errorf("building synthetic schema: %w", err)
	}

	dataComponents, err := finalizeValue(cueCtx, schemaComps)
	if err != nil {
		return nil, fmt.Errorf("finalizing components: %w", err)
	}

	modMeta := &module.ModuleMetadata{}
	metaVal := modVal.LookupPath(cue.ParsePath("metadata"))
	if !metaVal.Exists() {
		return nil, fmt.Errorf("module has no metadata field")
	}
	if err := metaVal.Decode(modMeta); err != nil {
		return nil, fmt.Errorf("decoding module metadata: %w", err)
	}

	relMeta := &modulerelease.ReleaseMetadata{
		Name:      releaseName,
		Namespace: namespace,
	}

	return modulerelease.NewModuleRelease(relMeta, module.Module{
		Metadata: modMeta,
		Config:   moduleConfigVal,
		Raw:      modVal,
	}, rawCUE, dataComponents, moduleConfigVal, mergedValues), nil
}
