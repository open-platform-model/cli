package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/output"
)

// ModuleLoader handles module and values loading.
type ModuleLoader struct {
	cueCtx *cue.Context
}

// NewModuleLoader creates a new ModuleLoader instance.
func NewModuleLoader(ctx *cue.Context) *ModuleLoader {
	return &ModuleLoader{cueCtx: ctx}
}

// LoadedModule is the result of loading a module.
type LoadedModule struct {
	// Path to the module directory
	Path string

	// CUE value of the unified module
	Value cue.Value

	// Extracted metadata
	Name             string
	Namespace        string
	Version          string
	DefaultNamespace string
	Labels           map[string]string
}

// LoadedComponent is a component with extracted metadata.
// Components are now extracted by ReleaseBuilder, but this type is still used.
type LoadedComponent struct {
	Name        string
	Labels      map[string]string    // Effective labels (merged from resources/traits)
	Annotations map[string]string    // Annotations from metadata.annotations
	Resources   map[string]cue.Value // FQN -> resource value
	Traits      map[string]cue.Value // FQN -> trait value
	Value       cue.Value            // Full component value
}

// Load loads a module with its values.
//
// Loading process:
//  1. Load the CUE module from ModulePath
//  2. Auto-discover and load values.cue (required)
//  3. Unify with --values files in order
//  4. Apply --namespace and --name overrides
//  5. Return raw module value for release building
func (l *ModuleLoader) Load(ctx context.Context, opts RenderOptions) (*LoadedModule, error) {
	// Resolve absolute path
	absPath, err := filepath.Abs(opts.ModulePath)
	if err != nil {
		return nil, fmt.Errorf("resolving module path: %w", err)
	}

	// Verify module directory exists
	if _, err := os.Stat(absPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("module directory not found: %s", absPath)
	}

	// Verify cue.mod exists (it's a CUE module)
	cueModPath := filepath.Join(absPath, "cue.mod")
	if _, err := os.Stat(cueModPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("not a CUE module: missing cue.mod/ directory in %s", absPath)
	}

	// Verify values.cue exists (required per FR-B-030)
	valuesPath := filepath.Join(absPath, "values.cue")
	if _, err := os.Stat(valuesPath); os.IsNotExist(err) {
		return nil, fmt.Errorf("values.cue required but not found in %s", absPath)
	}

	output.Debug("loading module",
		"path", absPath,
		"values_files", opts.Values,
	)

	// Set CUE_REGISTRY if provided
	if opts.Registry != "" {
		os.Setenv("CUE_REGISTRY", opts.Registry)
		defer os.Unsetenv("CUE_REGISTRY")
	}

	// Use the shared CUE context
	cueCtx := l.cueCtx

	// Load the module
	cfg := &load.Config{
		Dir: absPath,
	}

	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", absPath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading module: %w", inst.Err)
	}

	value := cueCtx.BuildInstance(inst)
	if value.Err() != nil {
		return nil, fmt.Errorf("building module: %w", value.Err())
	}

	// Load and unify additional values files
	for _, valuesFile := range opts.Values {
		valuesValue, err := l.loadValuesFile(cueCtx, valuesFile)
		if err != nil {
			return nil, fmt.Errorf("loading values file %s: %w", valuesFile, err)
		}
		value = value.Unify(valuesValue)
		if value.Err() != nil {
			return nil, fmt.Errorf("unifying values from %s: %w", valuesFile, value.Err())
		}
	}

	// Extract module metadata
	module, err := l.extractModule(value, opts)
	if err != nil {
		return nil, fmt.Errorf("extracting module metadata: %w", err)
	}
	module.Path = absPath
	module.Value = value

	output.Debug("loaded module",
		"name", module.Name,
		"namespace", module.Namespace,
		"version", module.Version,
	)

	return module, nil
}

// loadValuesFile loads a single values file.
func (l *ModuleLoader) loadValuesFile(cueCtx *cue.Context, path string) (cue.Value, error) {
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

	value := cueCtx.CompileBytes(content, cue.Filename(absPath))
	if value.Err() != nil {
		return cue.Value{}, fmt.Errorf("compiling values: %w", value.Err())
	}

	return value, nil
}

// extractModule extracts module metadata from the CUE value.
func (l *ModuleLoader) extractModule(value cue.Value, opts RenderOptions) (*LoadedModule, error) {
	module := &LoadedModule{
		Labels: make(map[string]string),
	}

	// Look for module metadata at module.metadata or metadata
	metadata := l.findMetadata(value)
	if metadata.Exists() {
		l.extractMetadataFields(metadata, module)
	}

	// Apply overrides (FR-B-033, FR-B-034)
	l.applyOverrides(module, opts)

	// Validate required fields
	if module.Namespace == "" {
		return nil, &NamespaceRequiredError{ModuleName: module.Name}
	}

	return module, nil
}

// findMetadata locates the metadata value in the module.
func (l *ModuleLoader) findMetadata(value cue.Value) cue.Value {
	metadata := value.LookupPath(cue.ParsePath("module.metadata"))
	if !metadata.Exists() {
		metadata = value.LookupPath(cue.ParsePath("metadata"))
	}
	return metadata
}

// extractMetadataFields extracts name, version, namespace, and labels from metadata.
func (l *ModuleLoader) extractMetadataFields(metadata cue.Value, module *LoadedModule) {
	module.Name = l.extractStringField(metadata, "name")
	module.Version = l.extractStringField(metadata, "version")
	module.DefaultNamespace = l.extractStringField(metadata, "defaultNamespace")
	l.extractLabels(metadata.LookupPath(cue.ParsePath("labels")), module.Labels)
}

// extractStringField extracts a string field from a CUE value.
func (l *ModuleLoader) extractStringField(value cue.Value, field string) string {
	if fieldVal := value.LookupPath(cue.ParsePath(field)); fieldVal.Exists() {
		if str, err := fieldVal.String(); err == nil {
			return str
		}
	}
	return ""
}

// extractLabels extracts labels from a CUE value into a map.
func (l *ModuleLoader) extractLabels(labelsVal cue.Value, labels map[string]string) {
	if !labelsVal.Exists() {
		return
	}
	iter, err := labelsVal.Fields()
	if err != nil {
		return
	}
	for iter.Next() {
		if str, err := iter.Value().String(); err == nil {
			labels[iter.Selector().Unquoted()] = str
		}
	}
}

// applyOverrides applies command-line overrides to the module.
func (l *ModuleLoader) applyOverrides(module *LoadedModule, opts RenderOptions) {
	// --name takes precedence over module name
	if opts.Name != "" {
		module.Name = opts.Name
	}

	// Resolve namespace using precedence: flag > defaultNamespace
	module.Namespace = l.resolveNamespace(opts.Namespace, module.DefaultNamespace)
}

// resolveNamespace resolves the target namespace using precedence:
// 1. --namespace flag (highest)
// 2. module.metadata.defaultNamespace
// Returns empty string if neither is set (caller should validate).
func (l *ModuleLoader) resolveNamespace(flagValue, defaultNamespace string) string {
	if flagValue != "" {
		return flagValue
	}
	return defaultNamespace
}
