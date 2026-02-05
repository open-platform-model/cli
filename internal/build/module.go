package build

import (
	"context"
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/cuecontext"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/output"
)

// ModuleLoader handles module and values loading.
type ModuleLoader struct{}

// NewModuleLoader creates a new ModuleLoader instance.
func NewModuleLoader() *ModuleLoader {
	return &ModuleLoader{}
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

	// Components extracted from module
	Components []*LoadedComponent
}

// LoadedComponent is a component with extracted metadata.
type LoadedComponent struct {
	Name      string
	Labels    map[string]string    // Effective labels (merged from resources/traits)
	Resources map[string]cue.Value // FQN -> resource value
	Traits    map[string]cue.Value // FQN -> trait value
	Value     cue.Value            // Full component value
}

// Metadata returns ModuleMetadata for RenderResult.
func (m *LoadedModule) Metadata() ModuleMetadata {
	names := make([]string, len(m.Components))
	for i, c := range m.Components {
		names[i] = c.Name
	}
	return ModuleMetadata{
		Name:       m.Name,
		Namespace:  m.Namespace,
		Version:    m.Version,
		Labels:     m.Labels,
		Components: names,
	}
}

// Load loads a module with its values.
//
// Loading process:
//  1. Load the CUE module from ModulePath
//  2. Auto-discover and load values.cue (required)
//  3. Unify with --values files in order
//  4. Apply --namespace and --name overrides
//  5. Extract components with metadata
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

	// Create fresh CUE context
	cueCtx := cuecontext.New()

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

	// Check if this is a ModuleRelease (has concrete components) or Module (has #components)
	isRelease := l.isModuleRelease(value)
	if isRelease {
		output.Debug("module release detected, using concrete components")
	} else {
		output.Debug("module detected, using #components definition")
	}

	// Extract module metadata
	module, err := l.extractModule(value, opts)
	if err != nil {
		return nil, fmt.Errorf("extracting module metadata: %w", err)
	}
	module.Path = absPath
	module.Value = value

	// Extract components from the appropriate field based on module type
	components, err := l.extractComponentsFromModule(value, isRelease)
	if err != nil {
		return nil, err // Error already has context
	}
	module.Components = components

	output.Debug("loaded module",
		"name", module.Name,
		"namespace", module.Namespace,
		"version", module.Version,
		"components", len(module.Components),
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

// extractComponents extracts components from the module value.
func (l *ModuleLoader) extractComponents(value cue.Value) ([]*LoadedComponent, error) {
	var components []*LoadedComponent

	// Look for components at module.components or components
	componentsPath := cue.ParsePath("module.components")
	componentsValue := value.LookupPath(componentsPath)
	if !componentsValue.Exists() {
		componentsPath = cue.ParsePath("components")
		componentsValue = value.LookupPath(componentsPath)
	}

	if !componentsValue.Exists() {
		// No components is valid (empty module)
		return components, nil
	}

	// Iterate over components (struct fields)
	iter, err := componentsValue.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating components: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		compValue := iter.Value()

		comp, err := l.extractComponent(name, compValue)
		if err != nil {
			return nil, fmt.Errorf("extracting component %s: %w", name, err)
		}
		components = append(components, comp)
	}

	return components, nil
}

// extractComponent extracts a single component's metadata.
//
//nolint:unparam // error return allows for future validation
func (l *ModuleLoader) extractComponent(name string, value cue.Value) (*LoadedComponent, error) {
	comp := &LoadedComponent{
		Name:      name,
		Labels:    make(map[string]string),
		Resources: make(map[string]cue.Value),
		Traits:    make(map[string]cue.Value),
		Value:     value,
	}

	// Extract #resources
	resourcesValue := value.LookupPath(cue.ParsePath("#resources"))
	if resourcesValue.Exists() {
		iter, err := resourcesValue.Fields()
		if err == nil {
			for iter.Next() {
				fqn := iter.Selector().Unquoted()
				comp.Resources[fqn] = iter.Value()

				// Extract labels from resource
				l.extractLabelsFromValue(iter.Value(), comp.Labels)
			}
		}
	}

	// Extract #traits
	traitsValue := value.LookupPath(cue.ParsePath("#traits"))
	if traitsValue.Exists() {
		iter, err := traitsValue.Fields()
		if err == nil {
			for iter.Next() {
				fqn := iter.Selector().Unquoted()
				comp.Traits[fqn] = iter.Value()

				// Extract labels from trait
				l.extractLabelsFromValue(iter.Value(), comp.Labels)
			}
		}
	}

	// Extract component-level labels
	labelsValue := value.LookupPath(cue.ParsePath("labels"))
	if labelsValue.Exists() {
		iter, err := labelsValue.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					comp.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}

	return comp, nil
}

// extractLabelsFromValue extracts labels from a value (resource or trait).
func (l *ModuleLoader) extractLabelsFromValue(value cue.Value, labels map[string]string) {
	labelsValue := value.LookupPath(cue.ParsePath("labels"))
	if !labelsValue.Exists() {
		return
	}

	iter, err := labelsValue.Fields()
	if err != nil {
		return
	}

	for iter.Next() {
		if str, err := iter.Value().String(); err == nil {
			labels[iter.Selector().Unquoted()] = str
		}
	}
}
