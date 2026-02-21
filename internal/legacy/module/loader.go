package module

import (
	"fmt"
	"os"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/output"
)

// Load constructs a *core.Module by resolving the module path, loading the CUE
// instance, and performing full CUE evaluation.
//
// Load calls mod.ResolvePath() internally — the returned *core.Module always
// has a validated, absolute ModulePath.
//
// After full CUE evaluation via cueCtx.BuildInstance(), all metadata fields
// (name, defaultNamespace, fqn, version, uuid, labels) are extracted from the
// evaluated value using LookupPath. No AST inspection is performed.
//
// The returned module has CUEValue() set and passes mod.Validate().
func Load(cueCtx *cue.Context, modulePath, registry string) (*core.Module, error) {
	mod := &core.Module{ModulePath: modulePath}

	// Step 1: Resolve and validate the module path
	if err := mod.ResolvePath(); err != nil {
		return nil, err
	}

	// Step 2: Load the CUE instance
	if registry != "" {
		os.Setenv("CUE_REGISTRY", registry)
		defer os.Unsetenv("CUE_REGISTRY")
	}

	cfg := &load.Config{Dir: mod.ModulePath}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", mod.ModulePath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading module: %w", inst.Err)
	}

	// PkgName is only available from the build.Instance, not from the evaluated value
	mod.SetPkgName(inst.PkgName)

	// Step 3: Full CUE evaluation via BuildInstance
	baseValue := cueCtx.BuildInstance(inst)
	if err := baseValue.Err(); err != nil {
		return nil, fmt.Errorf("evaluating module CUE: %w", err)
	}

	// Step 4: Extract all metadata from the evaluated value
	mod.Metadata = &core.ModuleMetadata{}
	extractModuleMetadata(baseValue, mod.Metadata)

	// Step 5: Extract #config and values (zero value if absent — no error)
	if configDef := baseValue.LookupPath(cue.ParsePath("#config")); configDef.Exists() {
		mod.Config = configDef
	}
	if valuesField := baseValue.LookupPath(cue.ParsePath("values")); valuesField.Exists() {
		mod.Values = valuesField
	}

	// Step 6: Extract schema-level components from #components
	if componentsValue := baseValue.LookupPath(cue.ParsePath("#components")); componentsValue.Exists() {
		components, err := core.ExtractComponents(componentsValue)
		if err != nil {
			return nil, fmt.Errorf("extracting components: %w", err)
		}
		mod.Components = components
	}

	// Step 7: Store the evaluated CUE value on the module
	mod.SetCUEValue(baseValue)

	output.Debug("loaded module",
		"path", mod.ModulePath,
		"name", mod.Metadata.Name,
		"fqn", mod.Metadata.FQN,
		"version", mod.Metadata.Version,
		"defaultNamespace", mod.Metadata.DefaultNamespace,
		"components", len(mod.Components),
	)

	return mod, nil
}

// extractModuleMetadata extracts all scalar metadata fields from the
// CUE-evaluated module value into the provided ModuleMetadata struct.
func extractModuleMetadata(v cue.Value, meta *core.ModuleMetadata) { //nolint:gocyclo // linear field extraction; each branch is a distinct metadata field
	if f := v.LookupPath(cue.ParsePath("metadata.name")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.Name = str
		}
	}

	if f := v.LookupPath(cue.ParsePath("metadata.defaultNamespace")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.DefaultNamespace = str
		}
	}

	if f := v.LookupPath(cue.ParsePath("metadata.fqn")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.FQN = str
		}
	}
	// Fallback: use metadata.apiVersion as FQN if fqn is absent
	if meta.FQN == "" {
		if f := v.LookupPath(cue.ParsePath("metadata.apiVersion")); f.Exists() {
			if str, err := f.String(); err == nil {
				meta.FQN = str
			}
		}
	}

	if f := v.LookupPath(cue.ParsePath("metadata.version")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.Version = str
		}
	}

	if f := v.LookupPath(cue.ParsePath("metadata.uuid")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.UUID = str
		}
	}

	if labelsVal := v.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		labels := make(map[string]string)
		if iter, err := labelsVal.Fields(); err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					labels[iter.Selector().Unquoted()] = str
				}
			}
		}
		if len(labels) > 0 {
			meta.Labels = labels
		}
	}
}
