package loader

import (
	"fmt"
	"os"
	"path/filepath"
	"strings"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/core/component"
	"github.com/opmodel/cli/internal/core/module"
	"github.com/opmodel/cli/internal/output"
)

// LoadModule constructs a *module.Module by resolving the module path, loading the CUE
// instance via Approach A (explicit filtered file list), and performing full CUE
// evaluation.
//
// LoadModule calls mod.ResolvePath() internally — the returned *module.Module always
// has a validated, absolute ModulePath.
//
// Loading strategy (Approach A):
//  1. Enumerate all top-level .cue files in the module directory.
//  2. Filter out all values*.cue files from the package load. Any values*.cue
//     files other than values.cue are silently ignored and reported via DEBUG.
//  3. Load the package from the explicit non-values file list via load.Instances.
//  4. If values.cue is present, load it separately via ctx.CompileBytes and
//     extract the "values" field into mod.Values (Pattern A).
//  5. If no values.cue, fall back to mod.Raw.LookupPath("values") for inline
//     values defined in module.cue (Pattern B).
//
// Extra values*.cue files (e.g. values_prod.cue) in the module directory are
// filtered out and never unified into mod.Raw. They can be passed explicitly
// via --values at build time.
//
// All metadata fields (name, defaultNamespace, fqn, version, uuid, labels) are
// extracted from the evaluated value using LookupPath. No AST inspection is performed.
//
// The returned module has Raw set and passes mod.Validate().
func LoadModule(cueCtx *cue.Context, modulePath, registry string) (*module.Module, error) { //nolint:gocyclo // sequential module loading; each branch handles a distinct load step
	mod := &module.Module{ModulePath: modulePath}

	// Step 1: Resolve and validate the module path.
	if err := mod.ResolvePath(); err != nil {
		return nil, err
	}

	// Step 2: Enumerate all top-level .cue files in the module directory.
	allFiles, err := cueFilesInDir(mod.ModulePath)
	if err != nil {
		return nil, fmt.Errorf("enumerating module files: %w", err)
	}

	// Step 3: Separate module files from values*.cue files (Approach A filter).
	// All values*.cue files are excluded from the package load. values.cue is
	// loaded separately in Step 9. Any other values*.cue files (e.g. values_prod.cue)
	// are silently skipped and reported via DEBUG so the user understands why
	// they have no effect on the loaded module defaults.
	var moduleFiles []string
	var valuesFilePath string
	var skippedValuesFiles []string
	for _, f := range allFiles {
		base := filepath.Base(f)
		if isValuesFile(base) {
			if base == "values.cue" {
				valuesFilePath = f
			} else {
				skippedValuesFiles = append(skippedValuesFiles, base)
			}
			continue // all values*.cue excluded from package load
		}
		rel, relErr := filepath.Rel(mod.ModulePath, f)
		if relErr != nil {
			return nil, fmt.Errorf("computing relative path for %s: %w", f, relErr)
		}
		moduleFiles = append(moduleFiles, "./"+rel)
	}
	mod.SkippedValuesFiles = skippedValuesFiles
	mod.HasValuesCue = valuesFilePath != ""

	if len(moduleFiles) == 0 {
		return nil, fmt.Errorf("no non-values .cue files found in %s", mod.ModulePath)
	}

	// Step 4: Load CUE instance from explicit filtered file list (Approach A).
	if registry != "" {
		_ = os.Setenv("CUE_REGISTRY", registry)
		defer func() { _ = os.Unsetenv("CUE_REGISTRY") }()
	}

	cfg := &load.Config{Dir: mod.ModulePath}
	instances := load.Instances(moduleFiles, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", mod.ModulePath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading module: %w", inst.Err)
	}

	// Step 5: Full CUE evaluation via BuildInstance.
	baseValue := cueCtx.BuildInstance(inst)
	if err := baseValue.Err(); err != nil {
		return nil, fmt.Errorf("evaluating module CUE: %w", err)
	}

	// Step 6: Extract all metadata from the evaluated value.
	mod.Metadata = &module.ModuleMetadata{}
	extractModuleMetadata(baseValue, mod.Metadata)

	// Step 7: Extract #config (zero value if absent — no error).
	if configDef := baseValue.LookupPath(cue.ParsePath("#config")); configDef.Exists() {
		mod.Config = configDef
	}

	// Step 8: Populate mod.Values.
	//
	// Pattern A: values.cue present — load it separately via CompileBytes so it
	// is never unified into mod.Raw. Extract the "values" field.
	//
	// Pattern B: no values.cue — fall back to inline values in mod.Raw (from
	// module.cue). Zero cue.Value if neither source exists.
	if valuesFilePath != "" {
		content, readErr := os.ReadFile(valuesFilePath)
		if readErr != nil {
			return nil, fmt.Errorf("reading values.cue: %w", readErr)
		}
		compiled := cueCtx.CompileBytes(content, cue.Filename(valuesFilePath))
		if err := compiled.Err(); err != nil {
			return nil, fmt.Errorf("compiling values.cue: %w", err)
		}
		if v := compiled.LookupPath(cue.ParsePath("values")); v.Exists() {
			mod.Values = v
		}
	} else {
		// Pattern B: inline values from module.cue (already in mod.Raw).
		if v := baseValue.LookupPath(cue.ParsePath("values")); v.Exists() {
			mod.Values = v
		}
	}

	// Step 9: Extract schema-level components from #components.
	if componentsValue := baseValue.LookupPath(cue.ParsePath("#components")); componentsValue.Exists() {
		components, err := component.ExtractComponents(componentsValue)
		if err != nil {
			return nil, fmt.Errorf("extracting components: %w", err)
		}
		mod.Components = components
	}

	// Step 10: Store the fully evaluated CUE value — required for BUILD phase FillPath injection.
	mod.Raw = baseValue

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

// cueFilesInDir returns the absolute paths of all top-level .cue files in dir,
// excluding any files inside cue.mod/ subdirectory.
func cueFilesInDir(dir string) ([]string, error) {
	entries, err := os.ReadDir(dir)
	if err != nil {
		return nil, err
	}
	var files []string
	for _, e := range entries {
		if e.IsDir() {
			continue
		}
		if strings.HasSuffix(e.Name(), ".cue") {
			files = append(files, filepath.Join(dir, e.Name()))
		}
	}
	return files, nil
}

// isValuesFile reports whether a filename matches the values*.cue pattern:
// any .cue file whose base name starts with "values" and ends with ".cue".
func isValuesFile(name string) bool {
	base := filepath.Base(name)
	return strings.HasPrefix(base, "values") && strings.HasSuffix(base, ".cue")
}

// extractModuleMetadata extracts all scalar metadata fields from the
// CUE-evaluated module value into the provided ModuleMetadata struct.
func extractModuleMetadata(v cue.Value, meta *module.ModuleMetadata) { //nolint:gocyclo // linear field extraction; each branch is a distinct metadata field
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
