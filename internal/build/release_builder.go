package build

import (
	"fmt"
	"os"
	"path/filepath"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"

	"github.com/opmodel/cli/internal/output"
)

// ReleaseBuilder creates a concrete release from a module directory.
//
// It uses a CUE overlay to compute release metadata (identity, labels)
// via the CUE uuid package, and FillPath to make components concrete.
type ReleaseBuilder struct {
	cueCtx   *cue.Context
	registry string // CUE_REGISTRY value for module dependency resolution
}

// NewReleaseBuilder creates a new ReleaseBuilder.
func NewReleaseBuilder(ctx *cue.Context, registry string) *ReleaseBuilder {
	return &ReleaseBuilder{
		cueCtx:   ctx,
		registry: registry,
	}
}

// ReleaseOptions configures release building.
type ReleaseOptions struct {
	Name      string // Release name (defaults to module name)
	Namespace string // Required: target namespace
}

// BuiltRelease is the result of building a release.
type BuiltRelease struct {
	Value      cue.Value                   // The concrete module value (with #config injected)
	Components map[string]*LoadedComponent // Concrete components by name
	Metadata   ReleaseMetadata
}

// ReleaseMetadata contains release-level metadata.
type ReleaseMetadata struct {
	Name      string
	Namespace string
	Version   string
	FQN       string
	Labels    map[string]string
	// Identity is the module identity UUID (from #Module.metadata.identity).
	Identity string
	// ReleaseIdentity is the release identity UUID.
	// Computed by the CUE overlay via uuid.SHA1(OPMNamespace, "fqn:name:namespace").
	ReleaseIdentity string
}

// Build creates a concrete release by loading the module with a CUE overlay
// that computes release metadata (identity, labels) via the uuid package.
//
// The build process:
//  1. Load the module directory with an overlay file that computes release metadata
//  2. Unify with additional values files (--values)
//  3. Inject values into #config via FillPath (makes #config concrete)
//  4. Extract concrete components from #components
//  5. Validate all components are fully concrete
//  6. Extract release metadata from the overlay-computed #opmReleaseMeta
func (b *ReleaseBuilder) Build(modulePath string, opts ReleaseOptions, valuesFiles []string) (*BuiltRelease, error) {
	output.Debug("building release",
		"path", modulePath,
		"name", opts.Name,
		"namespace", opts.Namespace,
	)

	// Set CUE_REGISTRY if configured
	if b.registry != "" {
		os.Setenv("CUE_REGISTRY", b.registry)
		defer os.Unsetenv("CUE_REGISTRY")
	}

	// Step 1: Detect the CUE package name from the module directory
	pkgName, err := b.detectPackageName(modulePath)
	if err != nil {
		return nil, fmt.Errorf("detecting package name: %w", err)
	}

	// Step 2: Generate the overlay CUE content
	overlayCUE := b.generateOverlayCUE(pkgName, opts)

	// Step 3: Load the module with the overlay
	overlayPath := filepath.Join(modulePath, "opm_release_overlay.cue")
	cfg := &load.Config{
		Dir: modulePath,
		Overlay: map[string]load.Source{
			overlayPath: load.FromBytes(overlayCUE),
		},
	}

	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("no CUE instances found in %s", modulePath)
	}

	inst := instances[0]
	if inst.Err != nil {
		return nil, fmt.Errorf("loading module with overlay: %w", inst.Err)
	}

	value := b.cueCtx.BuildInstance(inst)
	if value.Err() != nil {
		return nil, fmt.Errorf("building module with overlay: %w", value.Err())
	}

	// Step 4: Unify with additional values files
	for _, valuesFile := range valuesFiles {
		valuesValue, err := b.loadValuesFile(valuesFile)
		if err != nil {
			return nil, fmt.Errorf("loading values file %s: %w", valuesFile, err)
		}
		value = value.Unify(valuesValue)
		if value.Err() != nil {
			return nil, fmt.Errorf("unifying values from %s: %w", valuesFile, value.Err())
		}
	}

	// Step 5: Inject values into #config to make components concrete
	values := value.LookupPath(cue.ParsePath("values"))
	if !values.Exists() {
		return nil, &ReleaseValidationError{
			Message: "module missing 'values' field - ensure module uses #config pattern",
		}
	}

	concreteModule := value.FillPath(cue.ParsePath("#config"), values)
	if concreteModule.Err() != nil {
		return nil, &ReleaseValidationError{
			Message: "failed to inject values into #config",
			Cause:   concreteModule.Err(),
		}
	}

	// Step 6: Extract concrete components from #components
	components, err := b.extractComponentsFromDefinition(concreteModule)
	if err != nil {
		return nil, err
	}

	// Step 7: Validate components are concrete
	for name, comp := range components {
		if err := comp.Value.Validate(cue.Concrete(true)); err != nil {
			return nil, &ReleaseValidationError{
				Message: fmt.Sprintf("component %q has non-concrete values - check that all required values are provided", name),
				Cause:   err,
			}
		}
	}

	// Step 8: Extract release metadata from overlay-computed #opmReleaseMeta
	metadata := b.extractReleaseMetadata(concreteModule, opts)

	output.Debug("release built successfully",
		"name", metadata.Name,
		"namespace", metadata.Namespace,
		"components", len(components),
	)

	return &BuiltRelease{
		Value:      concreteModule,
		Components: components,
		Metadata:   metadata,
	}, nil
}

// BuildFromValue creates a concrete release from an already-loaded module value.
// This is the legacy path for modules that don't import opmodel.dev/core@v0
// (e.g., test fixtures with inline CUE).
//
// The build process:
//  1. Extract values from module.values
//  2. Inject values into #config via FillPath (makes #config concrete)
//  3. Extract components from #components
//  4. Validate all components are fully concrete
//  5. Extract metadata from the module (including identity)
func (b *ReleaseBuilder) BuildFromValue(moduleValue cue.Value, opts ReleaseOptions) (*BuiltRelease, error) {
	output.Debug("building release (legacy)", "name", opts.Name, "namespace", opts.Namespace)

	// Step 1: Extract values from module
	values := moduleValue.LookupPath(cue.ParsePath("values"))
	if !values.Exists() {
		return nil, &ReleaseValidationError{
			Message: "module missing 'values' field - ensure module uses #config pattern",
		}
	}

	// Step 2: Inject values into #config to make it concrete
	concreteModule := moduleValue.FillPath(cue.ParsePath("#config"), values)
	if concreteModule.Err() != nil {
		return nil, &ReleaseValidationError{
			Message: "failed to inject values into #config",
			Cause:   concreteModule.Err(),
		}
	}

	// Step 3: Extract concrete components from #components
	components, err := b.extractComponentsFromDefinition(concreteModule)
	if err != nil {
		return nil, err
	}

	// Step 4: Validate components are concrete
	for name, comp := range components {
		if err := comp.Value.Validate(cue.Concrete(true)); err != nil {
			return nil, &ReleaseValidationError{
				Message: fmt.Sprintf("component %q has non-concrete values - check that all required values are provided", name),
				Cause:   err,
			}
		}
	}

	// Step 5: Extract metadata from the module
	metadata := b.extractMetadataFromModule(concreteModule, opts)

	output.Debug("release built successfully",
		"name", metadata.Name,
		"namespace", metadata.Namespace,
		"components", len(components),
	)

	return &BuiltRelease{
		Value:      concreteModule,
		Components: components,
		Metadata:   metadata,
	}, nil
}

// detectPackageName loads the module directory minimally to determine the CUE package name.
func (b *ReleaseBuilder) detectPackageName(modulePath string) (string, error) {
	cfg := &load.Config{Dir: modulePath}
	instances := load.Instances([]string{"."}, cfg)
	if len(instances) == 0 {
		return "", fmt.Errorf("no CUE instances found in %s", modulePath)
	}
	inst := instances[0]
	if inst.Err != nil {
		return "", fmt.Errorf("loading module for package detection: %w", inst.Err)
	}
	if inst.PkgName == "" {
		return "", fmt.Errorf("module has no package name: %s", modulePath)
	}
	return inst.PkgName, nil
}

// generateOverlayCUE generates the virtual overlay CUE content that computes
// release metadata (identity, labels) using the CUE uuid package.
//
// The overlay adds a #opmReleaseMeta definition to the module's CUE package.
// This definition computes:
//   - Release identity (uuid.SHA1 of fqn:name:namespace)
//   - Standard release labels (module-release.opmodel.dev/*)
//   - Module labels (inherited from module.metadata.labels)
//
// The overlay references top-level fields from the module (metadata)
// which are in scope because the overlay shares the same CUE package.
func (b *ReleaseBuilder) generateOverlayCUE(pkgName string, opts ReleaseOptions) []byte {
	overlay := fmt.Sprintf(`package %s

import "uuid"

#opmReleaseMeta: {
	name:      %q
	namespace: %q
	fqn:       metadata.fqn
	version:   metadata.version
	identity:  string & uuid.SHA1("c1cbe76d-5687-5a47-bfe6-83b081b15413", "\(fqn):\(name):\(namespace)")
	labels: metadata.labels & {
		"module-release.opmodel.dev/name":    name
		"module-release.opmodel.dev/version": version
		"module-release.opmodel.dev/uuid":    identity
	}
}
`, pkgName, opts.Name, opts.Namespace)

	return []byte(overlay)
}

// loadValuesFile loads a single values file and compiles it.
func (b *ReleaseBuilder) loadValuesFile(path string) (cue.Value, error) {
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

// extractComponentsFromDefinition extracts components from #components (definition).
func (b *ReleaseBuilder) extractComponentsFromDefinition(concreteModule cue.Value) (map[string]*LoadedComponent, error) {
	componentsValue := concreteModule.LookupPath(cue.ParsePath("#components"))
	if !componentsValue.Exists() {
		return nil, fmt.Errorf("module missing '#components' field")
	}

	components := make(map[string]*LoadedComponent)

	iter, err := componentsValue.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating components: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		compValue := iter.Value()

		comp := b.extractComponent(name, compValue)
		components[name] = comp
	}

	return components, nil
}

// extractComponent extracts a single component with its metadata.
func (b *ReleaseBuilder) extractComponent(name string, value cue.Value) *LoadedComponent {
	comp := &LoadedComponent{
		Name:        name,
		Labels:      make(map[string]string),
		Annotations: make(map[string]string),
		Resources:   make(map[string]cue.Value),
		Traits:      make(map[string]cue.Value),
		Value:       value,
	}

	// Extract metadata.name if present, otherwise use field name
	metaName := value.LookupPath(cue.ParsePath("metadata.name"))
	if metaName.Exists() {
		if str, err := metaName.String(); err == nil {
			comp.Name = str
		}
	}

	// Extract #resources
	resourcesValue := value.LookupPath(cue.ParsePath("#resources"))
	if resourcesValue.Exists() {
		iter, err := resourcesValue.Fields()
		if err == nil {
			for iter.Next() {
				fqn := iter.Selector().Unquoted()
				comp.Resources[fqn] = iter.Value()
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
			}
		}
	}

	// Extract annotations from metadata
	b.extractAnnotations(value, comp.Annotations)

	// Extract labels from metadata
	labelsValue := value.LookupPath(cue.ParsePath("metadata.labels"))
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

	return comp
}

// extractAnnotations extracts annotations from component metadata into the target map.
// CUE annotation values (bool, string) are converted to strings.
func (b *ReleaseBuilder) extractAnnotations(value cue.Value, annotations map[string]string) {
	annotationsValue := value.LookupPath(cue.ParsePath("metadata.annotations"))
	if !annotationsValue.Exists() {
		return
	}
	iter, err := annotationsValue.Fields()
	if err != nil {
		return
	}
	for iter.Next() {
		key := iter.Selector().Unquoted()
		v := iter.Value()
		switch v.Kind() {
		case cue.BoolKind:
			if b, err := v.Bool(); err == nil {
				if b {
					annotations[key] = "true"
				} else {
					annotations[key] = "false"
				}
			}
		default:
			if str, err := v.String(); err == nil {
				annotations[key] = str
			}
		}
	}
}

// extractReleaseMetadata extracts release metadata from the overlay-computed
// #opmReleaseMeta definition and module metadata.
//
// The overlay computes identity, labels, fqn, and version via CUE.
// Module identity comes from metadata.identity (computed by #Module).
func (b *ReleaseBuilder) extractReleaseMetadata(concreteModule cue.Value, opts ReleaseOptions) ReleaseMetadata {
	metadata := ReleaseMetadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
		Labels:    make(map[string]string),
	}

	// Extract from overlay-computed #opmReleaseMeta
	relMeta := concreteModule.LookupPath(cue.ParsePath("#opmReleaseMeta"))
	if relMeta.Exists() && relMeta.Err() == nil {
		// Version
		if v := relMeta.LookupPath(cue.ParsePath("version")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.Version = str
			}
		}

		// FQN
		if v := relMeta.LookupPath(cue.ParsePath("fqn")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}

		// Release identity (computed by CUE uuid.SHA1)
		if v := relMeta.LookupPath(cue.ParsePath("identity")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.ReleaseIdentity = str
			}
		}

		// Labels (includes module labels + standard release labels)
		if labelsVal := relMeta.LookupPath(cue.ParsePath("labels")); labelsVal.Exists() {
			iter, err := labelsVal.Fields()
			if err == nil {
				for iter.Next() {
					if str, err := iter.Value().String(); err == nil {
						metadata.Labels[iter.Selector().Unquoted()] = str
					}
				}
			}
		}
	} else {
		// Fallback: extract from module metadata directly (no overlay)
		b.extractMetadataFallback(concreteModule, &metadata)
	}

	// Extract module identity from metadata.identity (always from module, not release)
	if v := concreteModule.LookupPath(cue.ParsePath("metadata.identity")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Identity = str
		}
	}

	return metadata
}

// extractMetadataFallback extracts metadata from module fields when overlay is not available.
func (b *ReleaseBuilder) extractMetadataFallback(concreteModule cue.Value, metadata *ReleaseMetadata) {
	if v := concreteModule.LookupPath(cue.ParsePath("metadata.version")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Version = str
		}
	}

	if v := concreteModule.LookupPath(cue.ParsePath("metadata.fqn")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.FQN = str
		}
	}
	if metadata.FQN == "" {
		if v := concreteModule.LookupPath(cue.ParsePath("metadata.apiVersion")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}
	}

	if labelsVal := concreteModule.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		iter, err := labelsVal.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					metadata.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}

}

// extractMetadataFromModule extracts metadata from a module value (legacy path).
// Used by BuildFromValue for modules without overlay support.
func (b *ReleaseBuilder) extractMetadataFromModule(concreteModule cue.Value, opts ReleaseOptions) ReleaseMetadata {
	metadata := ReleaseMetadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
		Labels:    make(map[string]string),
	}

	if v := concreteModule.LookupPath(cue.ParsePath("metadata.version")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Version = str
		}
	}

	if v := concreteModule.LookupPath(cue.ParsePath("metadata.fqn")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.FQN = str
		}
	}
	if metadata.FQN == "" {
		if v := concreteModule.LookupPath(cue.ParsePath("metadata.apiVersion")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}
	}

	if v := concreteModule.LookupPath(cue.ParsePath("metadata.identity")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Identity = str
		}
	}

	if labelsVal := concreteModule.LookupPath(cue.ParsePath("metadata.labels")); labelsVal.Exists() {
		iter, err := labelsVal.Fields()
		if err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					metadata.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}

	return metadata
}
