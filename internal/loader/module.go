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
// instance via explicit filtered file list, and performing full CUE evaluation.
//
// LoadModule calls mod.ResolvePath() internally — the returned *module.Module always
// has a validated, absolute ModulePath.
//
// Loading strategy:
//  1. Enumerate all top-level .cue files in the module directory.
//  2. Filter out all values*.cue files from the package load. This is required
//     because #Module is a closed CUE definition — values.cue defines a "values"
//     field that is not present on #Module in v1alpha1. All values*.cue files are
//     excluded from load.Instances; the builder discovers and loads values.cue.
//  3. Load the package from the explicit non-values file list via load.Instances.
//  4. Evaluate the CUE instance and extract metadata, #config, and #components.
//
// All metadata fields (name, defaultNamespace, fqn, version, uuid, labels) are
// extracted from the evaluated value using LookupPath. No AST inspection is performed.
//
// The returned module has Raw set and passes mod.Validate().
func LoadModule(cueCtx *cue.Context, modulePath, registry string) (*module.Module, error) {
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

	// Step 3: Separate module files from values*.cue files.
	// All values*.cue files are excluded from the package load because #Module is
	// a closed CUE definition. Including values.cue (which defines a "values" field
	// not present on #Module) would cause a CUE evaluation error. The builder
	// discovers and loads values.cue separately at build time.
	var moduleFiles []string
	for _, f := range allFiles {
		base := filepath.Base(f)
		if isValuesFile(base) {
			continue // all values*.cue excluded from package load
		}
		rel, relErr := filepath.Rel(mod.ModulePath, f)
		if relErr != nil {
			return nil, fmt.Errorf("computing relative path for %s: %w", f, relErr)
		}
		moduleFiles = append(moduleFiles, "./"+rel)
	}

	if len(moduleFiles) == 0 {
		return nil, fmt.Errorf("no non-values .cue files found in %s", mod.ModulePath)
	}

	// Step 4: Load CUE instance from explicit filtered file list.
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

	// Step 8: Extract schema-level components from #components.
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
		"modulePath", mod.Metadata.ModulePath,
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
func extractModuleMetadata(v cue.Value, meta *module.ModuleMetadata) { //nolint:cyclop // linear field extraction; each branch is a distinct metadata field
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

	if f := v.LookupPath(cue.ParsePath("metadata.modulePath")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.ModulePath = str
		}
	}

	// v1alpha1: metadata.fqn is a computed field (modulePath/name:version for modules,
	// modulePath/name@version for primitives). Extract it directly.
	if f := v.LookupPath(cue.ParsePath("metadata.fqn")); f.Exists() {
		if str, err := f.String(); err == nil {
			meta.FQN = str
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

	if labels, err := extractCUEStringMap(v, "metadata.labels"); err == nil && len(labels) > 0 {
		meta.Labels = labels
	}
}
