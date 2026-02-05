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

// Loader handles module and values loading.
type Loader struct{}

// NewLoader creates a new Loader instance.
func NewLoader() *Loader {
	return &Loader{}
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
func (l *Loader) Load(ctx context.Context, opts RenderOptions) (*LoadedModule, error) {
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

	// Extract module metadata
	module, err := l.extractModule(value, opts)
	if err != nil {
		return nil, fmt.Errorf("extracting module metadata: %w", err)
	}
	module.Path = absPath
	module.Value = value

	// Extract components
	components, err := l.extractComponents(value)
	if err != nil {
		return nil, fmt.Errorf("extracting components: %w", err)
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
func (l *Loader) loadValuesFile(cueCtx *cue.Context, path string) (cue.Value, error) {
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
func (l *Loader) extractModule(value cue.Value, opts RenderOptions) (*LoadedModule, error) {
	module := &LoadedModule{
		Labels: make(map[string]string),
	}

	// Look for module metadata at module.metadata or metadata
	metadataPath := cue.ParsePath("module.metadata")
	metadata := value.LookupPath(metadataPath)
	if !metadata.Exists() {
		metadataPath = cue.ParsePath("metadata")
		metadata = value.LookupPath(metadataPath)
	}

	if metadata.Exists() {
		// Extract name
		if nameVal := metadata.LookupPath(cue.ParsePath("name")); nameVal.Exists() {
			if str, err := nameVal.String(); err == nil {
				module.Name = str
			}
		}

		// Extract version
		if versionVal := metadata.LookupPath(cue.ParsePath("version")); versionVal.Exists() {
			if str, err := versionVal.String(); err == nil {
				module.Version = str
			}
		}

		// Extract defaultNamespace
		if nsVal := metadata.LookupPath(cue.ParsePath("defaultNamespace")); nsVal.Exists() {
			if str, err := nsVal.String(); err == nil {
				module.DefaultNamespace = str
			}
		}

		// Extract labels
		if labelsVal := metadata.LookupPath(cue.ParsePath("labels")); labelsVal.Exists() {
			iter, err := labelsVal.Fields()
			if err == nil {
				for iter.Next() {
					if str, err := iter.Value().String(); err == nil {
						module.Labels[iter.Label()] = str
					}
				}
			}
		}
	}

	// Apply overrides (FR-B-033, FR-B-034)
	// --name takes precedence over module name
	if opts.Name != "" {
		module.Name = opts.Name
	}

	// --namespace takes precedence over defaultNamespace
	if opts.Namespace != "" {
		module.Namespace = opts.Namespace
	} else if module.DefaultNamespace != "" {
		module.Namespace = module.DefaultNamespace
	}

	// Validate required fields
	if module.Namespace == "" {
		return nil, fmt.Errorf("namespace required: set --namespace or define module.metadata.defaultNamespace")
	}

	return module, nil
}

// extractComponents extracts components from the module value.
func (l *Loader) extractComponents(value cue.Value) ([]*LoadedComponent, error) {
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
		name := iter.Label()
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
func (l *Loader) extractComponent(name string, value cue.Value) (*LoadedComponent, error) {
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
				fqn := iter.Label()
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
				fqn := iter.Label()
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
					comp.Labels[iter.Label()] = str
				}
			}
		}
	}

	return comp, nil
}

// extractLabelsFromValue extracts labels from a value (resource or trait).
func (l *Loader) extractLabelsFromValue(value cue.Value, labels map[string]string) {
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
			labels[iter.Label()] = str
		}
	}
}
