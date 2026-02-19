package transform

import (
	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
)

// TransformerContext holds the context data passed to transformers.
// This matches the CUE #TransformerContext definition.
type TransformerContext struct {
	// Name is the release name (from --name or module.metadata.name)
	Name string `json:"name"`

	// Namespace is the target namespace (from --namespace or defaultNamespace)
	Namespace string `json:"namespace"`

	// ModuleReleaseMetadata contains module release metadata.
	ModuleReleaseMetadata *TransformerModuleReleaseMetadata `json:"#moduleReleaseMetadata"`

	// ComponentMetadata contains component-level metadata
	ComponentMetadata *TransformerComponentMetadata `json:"#componentMetadata"`
}

// TransformerModuleReleaseMetadata contains module release metadata for transformers.
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
func NewTransformerContext(rel *release.BuiltRelease, component *module.LoadedComponent) *TransformerContext {
	return &TransformerContext{
		Name:      rel.Metadata.Name,
		Namespace: rel.Metadata.Namespace,
		ModuleReleaseMetadata: &TransformerModuleReleaseMetadata{
			Name:      rel.Metadata.Name,
			Namespace: rel.Metadata.Namespace,
			FQN:       rel.Metadata.FQN,
			Version:   rel.Metadata.Version,
			Identity:  rel.Metadata.ReleaseIdentity,
			Labels:    rel.Metadata.Labels,
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
