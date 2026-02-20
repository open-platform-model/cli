package release

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core"
	"github.com/opmodel/cli/internal/output"
)

// Builder creates a concrete release from a loaded module.
//
// It uses the module's pre-evaluated CUE value (mod.CUEValue()), selects
// values, fills #config, and extracts concrete components.
type Builder struct {
	cueCtx   *cue.Context
	registry string // CUE_REGISTRY value for module dependency resolution
}

// NewBuilder creates a new Builder.
func NewBuilder(ctx *cue.Context, registry string) *Builder {
	return &Builder{
		cueCtx:   ctx,
		registry: registry,
	}
}

// Build creates a concrete release from a pre-loaded *core.Module.
//
// The build process:
//  1. Precondition: mod.CUEValue() must exist (module was fully loaded)
//  2. Select values: use --values files if provided, else mod.Values
//  3. Inject selected values into #config via FillPath (makes #config concrete)
//  4. Validate the concrete release tree for structural errors
//  5. Extract concrete components from #components
//  6. Construct release metadata using Go values + core.ComputeReleaseUUID()
func (b *Builder) Build(mod *core.Module, opts Options, valuesFiles []string) (*core.ModuleRelease, error) {
	output.Debug("building release",
		"path", mod.ModulePath,
		"name", opts.Name,
		"namespace", opts.Namespace,
	)

	// Step 1: Precondition — module must have been fully loaded
	base := mod.CUEValue()
	if !base.Exists() {
		return nil, fmt.Errorf("module CUE value is not set — ensure the module was fully loaded via module.Load()")
	}

	// Step 2: Select values
	// If --values files are provided, load and unify them (external values take precedence).
	// Otherwise, fall back to mod.Values (the default values.cue).
	selectedValues, err := b.selectValues(mod, valuesFiles)
	if err != nil {
		return nil, err
	}

	// Validate selected values against #config schema via Unify
	if mod.Config.Exists() && selectedValues.Exists() {
		unified := mod.Config.Unify(selectedValues)
		if err := unified.Err(); err != nil {
			return nil, &core.ValidationError{
				Message: "values do not match module config schema",
				Cause:   err,
				Details: formatCUEDetails(err),
			}
		}
	}

	// Step 3: Extract #config (used by ValidateValues on the returned release)
	configDef := base.LookupPath(cue.ParsePath("#config"))

	// Step 4: Inject values into #config to make components concrete
	concreteRelease := base.FillPath(cue.ParsePath("#config"), selectedValues)
	if allErrs := collectAllCUEErrors(concreteRelease); allErrs != nil {
		return nil, &core.ValidationError{
			Message: "failed to inject values into #config",
			Cause:   allErrs,
			Details: formatCUEDetails(allErrs),
		}
	}

	// Step 5: Validate the full module tree for structural errors (fatal loading guard)
	if allErrs := collectAllCUEErrors(concreteRelease); allErrs != nil {
		return nil, &core.ValidationError{
			Message: "release validation failed",
			Cause:   allErrs,
			Details: formatCUEDetails(allErrs),
		}
	}

	// Step 6: Extract concrete components from #components
	componentsValue := concreteRelease.LookupPath(cue.ParsePath("#components"))
	if !componentsValue.Exists() {
		return nil, fmt.Errorf("module missing '#components' field")
	}
	coreComponents, err := core.ExtractComponents(componentsValue)
	if err != nil {
		return nil, err
	}

	// Step 7: Construct release metadata from Go values + ComputeReleaseUUID
	relMeta := extractReleaseMetadata(mod, opts)
	modMeta := *mod.Metadata // copy

	// Collect component names and set on both metadata types
	componentNames := make([]string, 0, len(coreComponents))
	for name := range coreComponents {
		componentNames = append(componentNames, name)
	}
	relMeta.Components = componentNames
	modMeta.Components = append([]string{}, componentNames...)

	// Build the core.Module embedding for the release (preserves values for ValidateValues)
	relMod := core.Module{
		Metadata:   &modMeta,
		ModulePath: mod.ModulePath,
		Config:     configDef,
		Values:     selectedValues,
	}
	relMod.SetPkgName(mod.PkgName())

	return &core.ModuleRelease{
		Metadata:   &relMeta,
		Module:     relMod,
		Components: coreComponents,
		Values:     selectedValues,
	}, nil
}

// selectValues determines the values to use for this build.
//
// If valuesFiles are provided, load and unify them together, then extract the
// `values` field. External files take full precedence over disk values.cue —
// they are NOT unified into the module base to avoid conflicts with on-disk values.
// Otherwise fall back to mod.Values (from values.cue on disk, extracted at load time).
// Returns an error if no values are available.
func (b *Builder) selectValues(mod *core.Module, valuesFiles []string) (cue.Value, error) {
	if len(valuesFiles) > 0 {
		// Load and unify all provided values files together.
		// External files take full precedence — not merged with mod.Values.
		// Each file uses the same `values: { ... }` top-level field convention.
		var unified cue.Value
		for i, path := range valuesFiles {
			v, err := b.loadValuesFile(path)
			if err != nil {
				return cue.Value{}, fmt.Errorf("loading values file %s: %w", path, err)
			}
			if i == 0 {
				unified = v
			} else {
				unified = unified.Unify(v)
			}
		}
		if unified.Err() != nil {
			return cue.Value{}, fmt.Errorf("unifying values files: %w", unified.Err())
		}
		// Extract the `values` field from the unified external files
		values := unified.LookupPath(cue.ParsePath("values"))
		if !values.Exists() {
			return cue.Value{}, &core.ValidationError{
				Message: "no 'values' field found in provided values files — files must define a top-level 'values:' field",
			}
		}
		return values, nil
	}

	// Fall back to mod.Values (from values.cue loaded during module.Load)
	if !mod.Values.Exists() {
		return cue.Value{}, &core.ValidationError{
			Message: "module missing 'values' field — provide values via values.cue or --values flag",
		}
	}
	return mod.Values, nil
}

// loadValuesFile loads a single values file and compiles it.
func (b *Builder) loadValuesFile(path string) (cue.Value, error) {
	absPath, err := filepath.Abs(path)
	if err != nil {
		return cue.Value{}, fmt.Errorf("resolving path: %w", err)
	}

	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return cue.Value{}, fmt.Errorf("file not found: %s", absPath)
	}

	content, err := os.ReadFile(absPath)
	if err != nil {
		return cue.Value{}, fmt.Errorf("reading file: %w", err)
	}

	value := b.cueCtx.CompileBytes(content, cue.Filename(absPath))
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling values: %w", value.Err())
	}

	return value, nil
}
