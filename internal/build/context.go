package build

// TransformerContext holds the context data passed to transformers.
// This matches the CUE #TransformerContext definition.
type TransformerContext struct {
	// Name is the release name (from --name or module.metadata.name)
	Name string `json:"name"`

	// Namespace is the target namespace (from --namespace or defaultNamespace)
	Namespace string `json:"namespace"`

	// ModuleMetadata contains module-level metadata
	ModuleMetadata *TransformerModuleMetadata `json:"#moduleMetadata"`

	// ComponentMetadata contains component-level metadata
	ComponentMetadata *TransformerComponentMetadata `json:"#componentMetadata"`
}

// TransformerModuleMetadata contains module metadata for transformers.
type TransformerModuleMetadata struct {
	Name    string            `json:"name"`
	Version string            `json:"version"`
	Labels  map[string]string `json:"labels,omitempty"`
}

// TransformerComponentMetadata contains component metadata for transformers.
type TransformerComponentMetadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
	Resources   []string          `json:"resources,omitempty"`
	Traits      []string          `json:"traits,omitempty"`
}

// NewTransformerContext constructs the context for a transformer execution.
// This context is unified with the transformer's #transform function.
func NewTransformerContext(release *BuiltRelease, component *LoadedComponent) *TransformerContext {
	// Extract resource FQNs
	resourceFQNs := make([]string, 0, len(component.Resources))
	for fqn := range component.Resources {
		resourceFQNs = append(resourceFQNs, fqn)
	}

	// Extract trait FQNs
	traitFQNs := make([]string, 0, len(component.Traits))
	for fqn := range component.Traits {
		traitFQNs = append(traitFQNs, fqn)
	}

	return &TransformerContext{
		Name:      release.Metadata.Name,
		Namespace: release.Metadata.Namespace,
		ModuleMetadata: &TransformerModuleMetadata{
			Name:    release.Metadata.Name,
			Version: release.Metadata.Version,
			Labels:  release.Metadata.Labels,
		},
		ComponentMetadata: &TransformerComponentMetadata{
			Name:        component.Name,
			Labels:      component.Labels,
			Annotations: component.Annotations,
			Resources:   resourceFQNs,
			Traits:      traitFQNs,
		},
	}
}

// ToMap converts TransformerContext to a map for CUE encoding.
func (c *TransformerContext) ToMap() map[string]any {
	moduleMetadata := map[string]any{
		"name":    c.ModuleMetadata.Name,
		"version": c.ModuleMetadata.Version,
	}
	if len(c.ModuleMetadata.Labels) > 0 {
		moduleMetadata["labels"] = c.ModuleMetadata.Labels
	}

	componentMetadata := map[string]any{
		"name": c.ComponentMetadata.Name,
	}
	if len(c.ComponentMetadata.Labels) > 0 {
		componentMetadata["labels"] = c.ComponentMetadata.Labels
	}
	if len(c.ComponentMetadata.Resources) > 0 {
		componentMetadata["resources"] = c.ComponentMetadata.Resources
	}
	if len(c.ComponentMetadata.Traits) > 0 {
		componentMetadata["traits"] = c.ComponentMetadata.Traits
	}
	if len(c.ComponentMetadata.Annotations) > 0 {
		componentMetadata["annotations"] = c.ComponentMetadata.Annotations
	}

	return map[string]any{
		"name":               c.Name,
		"namespace":          c.Namespace,
		"#moduleMetadata":    moduleMetadata,
		"#componentMetadata": componentMetadata,
	}
}
