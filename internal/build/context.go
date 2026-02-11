package build

// TransformerContext holds the context data passed to transformers.
// This matches the CUE #TransformerContext definition.
type TransformerContext struct {
	// Name is the release name (from --name or module.metadata.name)
	Name string `json:"name"`

	// Namespace is the target namespace (from --namespace or defaultNamespace)
	Namespace string `json:"namespace"`

	// ModuleReleaseMetadata contains module release metadata.
	// Matches CUE #TransformerContext.#moduleReleaseMetadata (#ModuleRelease.metadata).
	ModuleReleaseMetadata *TransformerModuleReleaseMetadata `json:"#moduleReleaseMetadata"`

	// ComponentMetadata contains component-level metadata
	ComponentMetadata *TransformerComponentMetadata `json:"#componentMetadata"`
}

// TransformerModuleReleaseMetadata contains module release metadata for transformers.
// Fields match #ModuleRelease.metadata (closed struct).
type TransformerModuleReleaseMetadata struct {
	Name      string            `json:"name"`
	Namespace string            `json:"namespace"`
	FQN       string            `json:"fqn"`
	Version   string            `json:"version"`
	Identity  string            `json:"identity"`
	Labels    map[string]string `json:"labels,omitempty"`
}

// TransformerComponentMetadata contains component metadata for transformers.
type TransformerComponentMetadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NewTransformerContext constructs the context for a transformer execution.
// This context is unified with the transformer's #transform function.
func NewTransformerContext(release *BuiltRelease, component *LoadedComponent) *TransformerContext {
	return &TransformerContext{
		Name:      release.Metadata.Name,
		Namespace: release.Metadata.Namespace,
		ModuleReleaseMetadata: &TransformerModuleReleaseMetadata{
			Name:      release.Metadata.Name,
			Namespace: release.Metadata.Namespace,
			FQN:       release.Metadata.FQN,
			Version:   release.Metadata.Version,
			Identity:  release.Metadata.ReleaseIdentity,
			Labels:    release.Metadata.Labels,
		},
		ComponentMetadata: &TransformerComponentMetadata{
			Name:        component.Name,
			Labels:      component.Labels,
			Annotations: component.Annotations,
		},
	}
}

// ToMap converts TransformerContext to a map for CUE encoding.
func (c *TransformerContext) ToMap() map[string]any {
	moduleReleaseMetadata := map[string]any{
		"name":      c.ModuleReleaseMetadata.Name,
		"namespace": c.ModuleReleaseMetadata.Namespace,
		"fqn":       c.ModuleReleaseMetadata.FQN,
		"version":   c.ModuleReleaseMetadata.Version,
		"identity":  c.ModuleReleaseMetadata.Identity,
	}
	if len(c.ModuleReleaseMetadata.Labels) > 0 {
		moduleReleaseMetadata["labels"] = c.ModuleReleaseMetadata.Labels
	}

	componentMetadata := map[string]any{
		"name": c.ComponentMetadata.Name,
	}
	if len(c.ComponentMetadata.Labels) > 0 {
		componentMetadata["labels"] = c.ComponentMetadata.Labels
	}
	if len(c.ComponentMetadata.Annotations) > 0 {
		componentMetadata["annotations"] = c.ComponentMetadata.Annotations
	}

	return map[string]any{
		"name":                   c.Name,
		"namespace":              c.Namespace,
		"#moduleReleaseMetadata": moduleReleaseMetadata,
		"#componentMetadata":     componentMetadata,
	}
}
