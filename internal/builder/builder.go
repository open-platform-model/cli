// Package builder implements the BUILD phase of the OPM render pipeline.
//
// It uses Approach C: load #ModuleRelease from opmodel.dev/core@v0 (resolved
// from the module's own pinned dependency cache), inject the module and user
// values via FillPath, and let CUE evaluate UUID, labels, components, and
// metadata natively. Go only reads back the resulting concrete values.
//
// Requires OPM_REGISTRY / CUE_REGISTRY to be set for registry resolution.
package builder

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/output"
)

// Options holds the release-level options supplied by the caller.
type Options struct {
	// Name is the release name (from --name flag or module default).
	Name string

	// Namespace is the target Kubernetes namespace.
	Namespace string
}

// Build creates a concrete *core.ModuleRelease from a pre-loaded *core.Module.
//
// The build process (Approach C):
//  1. Load opmodel.dev/core@v0 from the module's pinned dependency cache
//  2. Extract #ModuleRelease schema from the core value
//  3. Select values: use valuesFiles if provided, else mod.Values
//  4. Validate selected values against mod's #config schema
//  5. FillPath chain: #module → metadata.name → metadata.namespace → values
//  6. Validate the result is fully concrete
//  7. Read back metadata (uuid, version, labels) and components from CUE
//  8. Construct and return *core.ModuleRelease
//
// The ctx must be the same context used to load the module (mod.Raw was built
// with it). Passing a different context will cause FillPath to fail.
func Build(ctx *cue.Context, mod *core.Module, opts Options, valuesFiles []string) (*core.ModuleRelease, error) {
	output.Debug("building release (Approach C)",
		"path", mod.ModulePath,
		"name", opts.Name,
		"namespace", opts.Namespace,
	)

	// Step 1: Load opmodel.dev/core@v0 from the module's pinned dep cache.
	// Using Dir: mod.ModulePath tells CUE to resolve the import against the
	// module's own cue.mod/module.cue — no separate registry call needed.
	coreInstances := load.Instances([]string{"opmodel.dev/core@v0"}, &load.Config{
		Dir: mod.ModulePath,
	})
	if len(coreInstances) == 0 {
		return nil, fmt.Errorf("loading opmodel.dev/core@v0: no instances returned")
	}
	if coreInstances[0].Err != nil {
		return nil, fmt.Errorf("loading opmodel.dev/core@v0: %w", coreInstances[0].Err)
	}

	coreVal := ctx.BuildInstance(coreInstances[0])
	if err := coreVal.Err(); err != nil {
		return nil, fmt.Errorf("building opmodel.dev/core@v0: %w", err)
	}

	// Step 2: Extract #ModuleRelease schema.
	releaseSchema := coreVal.LookupPath(cue.ParsePath("#ModuleRelease"))
	if !releaseSchema.Exists() {
		return nil, fmt.Errorf("#ModuleRelease not found in opmodel.dev/core@v0")
	}
	if err := releaseSchema.Err(); err != nil {
		return nil, fmt.Errorf("#ModuleRelease is invalid: %w", err)
	}

	// Step 3: Select values (external files or module defaults).
	selectedValues, err := selectValues(ctx, mod, valuesFiles)
	if err != nil {
		return nil, err
	}

	// Step 4: Validate selected values against module's #config schema.
	if mod.Config.Exists() && selectedValues.Exists() {
		unified := mod.Config.Unify(selectedValues)
		if err := unified.Err(); err != nil {
			return nil, &core.ValidationError{
				Message: "values do not match module #config schema",
				Cause:   err,
			}
		}
	}

	// Step 5: FillPath chain — order matters (see design Decision 4).
	// #module must be filled before values because _#module: #module & {#config: values}
	// depends on #module being present.
	result := releaseSchema.
		FillPath(cue.MakePath(cue.Def("module")), mod.Raw).
		FillPath(cue.ParsePath("metadata.name"), ctx.CompileString(`"`+opts.Name+`"`)).
		FillPath(cue.ParsePath("metadata.namespace"), ctx.CompileString(`"`+opts.Namespace+`"`)).
		FillPath(cue.ParsePath("values"), selectedValues)

	if err := result.Err(); err != nil {
		return nil, &core.ValidationError{
			Message: "FillPath injection failed",
			Cause:   err,
		}
	}

	// Step 6: Validate full concreteness of the #ModuleRelease result.
	if err := result.Validate(cue.Concrete(true)); err != nil {
		return nil, &core.ValidationError{
			Message: "release is not fully concrete after value injection — check that all required values are provided",
			Cause:   err,
		}
	}

	// Step 7a: Read back release metadata from CUE.
	relMeta, err := extractReleaseMetadata(result, opts)
	if err != nil {
		return nil, fmt.Errorf("reading back release metadata: %w", err)
	}

	// Step 7b: Read back components.
	componentsVal := result.LookupPath(cue.ParsePath("components"))
	if !componentsVal.Exists() {
		return nil, fmt.Errorf("#ModuleRelease is missing 'components' field")
	}
	components, err := core.ExtractComponents(componentsVal)
	if err != nil {
		return nil, fmt.Errorf("extracting components: %w", err)
	}

	// Collect component names for metadata.
	componentNames := make([]string, 0, len(components))
	for name := range components {
		componentNames = append(componentNames, name)
	}
	relMeta.Components = componentNames

	// Step 8: Construct the module embed for the release.
	modCopy := *mod
	if modCopy.Metadata != nil {
		metaCopy := *mod.Metadata
		metaCopy.Components = append([]string{}, componentNames...)
		modCopy.Metadata = &metaCopy
	}
	modCopy.Values = selectedValues

	output.Debug("release built",
		"uuid", relMeta.UUID,
		"components", len(components),
	)

	return &core.ModuleRelease{
		Metadata:   relMeta,
		Module:     modCopy,
		Components: components,
		Values:     selectedValues,
	}, nil
}

// extractReleaseMetadata reads back scalar release metadata fields from the
// fully-concrete #ModuleRelease CUE value.
func extractReleaseMetadata(result cue.Value, opts Options) (*core.ReleaseMetadata, error) {
	meta := &core.ReleaseMetadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
	}

	if v := result.LookupPath(cue.ParsePath("metadata.uuid")); v.Exists() {
		if s, err := v.String(); err == nil {
			meta.UUID = s
		} else {
			return nil, fmt.Errorf("metadata.uuid: %w", err)
		}
	}

	if v := result.LookupPath(cue.ParsePath("metadata.version")); v.Exists() {
		if s, err := v.String(); err == nil {
			_ = s // version is on module metadata; stored separately if needed
		}
	}

	if labelsVal := result.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		labels := make(map[string]string)
		if iter, err := labelsVal.Fields(); err == nil {
			for iter.Next() {
				if s, err := iter.Value().String(); err == nil {
					labels[iter.Selector().Unquoted()] = s
				}
			}
		}
		if len(labels) > 0 {
			meta.Labels = labels
		}
	}

	return meta, nil
}
