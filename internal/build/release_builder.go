package build

import (
	"fmt"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/output"
)

// LabelReleaseID is the label key for release identity.
// Duplicated here to avoid circular dependency with kubernetes package.
const LabelReleaseID = "module-release.opmodel.dev/uuid"

// ReleaseBuilder creates a concrete release from a #Module
type ReleaseBuilder struct {
	cueCtx   *cue.Context
	registry string // CUE_REGISTRY value (for future use)
}

// NewReleaseBuilder creates a new ReleaseBuilder
func NewReleaseBuilder(ctx *cue.Context, registry string) *ReleaseBuilder {
	return &ReleaseBuilder{
		cueCtx:   ctx,
		registry: registry,
	}
}

// ReleaseOptions configures release building
type ReleaseOptions struct {
	Name      string // Release name (defaults to module name)
	Namespace string // Required: target namespace
}

// BuiltRelease is the result of building a release
type BuiltRelease struct {
	Value      cue.Value                   // The concrete module value (with #config injected)
	Components map[string]*LoadedComponent // Concrete components by name
	Metadata   ReleaseMetadata
}

// ReleaseMetadata contains release-level metadata
type ReleaseMetadata struct {
	Name      string
	Namespace string
	Version   string
	FQN       string
	Labels    map[string]string
	// Identity is the module identity UUID (from #Module.metadata.identity).
	Identity string
	// ReleaseIdentity is the release identity UUID, extracted from metadata.labels.
	// This is computed by the CUE catalog schemas and injected as a label.
	ReleaseIdentity string
}

// Build creates a concrete release from a loaded #Module
//
// The build process:
//  1. Extract values from module.values
//  2. Inject values into #config via FillPath (makes #config concrete)
//  3. Extract components from #components (now concrete due to #config references)
//  4. Validate all components are fully concrete
//  5. Extract metadata from the module (including identity)
func (b *ReleaseBuilder) Build(moduleValue cue.Value, opts ReleaseOptions) (*BuiltRelease, error) {
	output.Debug("building release", "name", opts.Name, "namespace", opts.Namespace)

	// Step 1: Extract values from module
	values := moduleValue.LookupPath(cue.ParsePath("values"))
	if !values.Exists() {
		return nil, &ReleaseValidationError{
			Message: "module missing 'values' field - ensure module uses #config pattern",
		}
	}

	// Step 2: Inject values into #config to make it concrete
	// This is the key step: filling #config with values resolves all #config references
	concreteModule := moduleValue.FillPath(cue.ParsePath("#config"), values)
	if concreteModule.Err() != nil {
		return nil, &ReleaseValidationError{
			Message: "failed to inject values into #config",
			Cause:   concreteModule.Err(),
		}
	}

	// Step 3: Extract concrete components from #components
	components, err := b.extractComponents(concreteModule)
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

	// Step 5: Extract metadata from the module (including identity)
	metadata := b.extractMetadata(concreteModule, opts)

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

// extractComponents extracts concrete components from the module
func (b *ReleaseBuilder) extractComponents(concreteModule cue.Value) (map[string]*LoadedComponent, error) {
	// Look for #components in the concrete module
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

// extractComponent extracts a single component with its metadata
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

// extractMetadata extracts release metadata from the module
func (b *ReleaseBuilder) extractMetadata(concreteModule cue.Value, opts ReleaseOptions) ReleaseMetadata {
	metadata := ReleaseMetadata{
		Name:      opts.Name,
		Namespace: opts.Namespace,
		Labels:    make(map[string]string),
	}

	// Extract version from module.metadata.version
	if v := concreteModule.LookupPath(cue.ParsePath("metadata.version")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Version = str
		}
	}

	// Extract FQN from module.metadata.fqn (computed field)
	// Fallback to apiVersion if fqn is not available
	if v := concreteModule.LookupPath(cue.ParsePath("metadata.fqn")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.FQN = str
		}
	}
	if metadata.FQN == "" {
		// Fallback: use apiVersion as FQN if fqn field is not present
		if v := concreteModule.LookupPath(cue.ParsePath("metadata.apiVersion")); v.Exists() {
			if str, err := v.String(); err == nil {
				metadata.FQN = str
			}
		}
	}

	// Extract identity from module.metadata.identity (computed field from catalog)
	if v := concreteModule.LookupPath(cue.ParsePath("metadata.identity")); v.Exists() {
		if str, err := v.String(); err == nil {
			metadata.Identity = str
		}
	}

	// Extract labels from module.metadata.labels
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

	// Extract release identity from labels (computed by CUE catalog schemas)
	// The release-id label is set by the catalog schema transformer
	if rid, ok := metadata.Labels[LabelReleaseID]; ok {
		metadata.ReleaseIdentity = rid
	}

	return metadata
}
