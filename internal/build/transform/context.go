package transform

import (
	"github.com/opmodel/cli/internal/build/component"
	"github.com/opmodel/cli/internal/core"
)

// TransformerContext holds the context data passed to transformers.
// This matches the CUE #TransformerContext definition.
type TransformerContext struct {
	// Name is the release name (from --name or module.metadata.name)
	Name string `json:"name"`

	// Namespace is the target namespace (from --namespace or defaultNamespace)
	Namespace string `json:"namespace"`

	// ModuleMetadata contains module-level identity metadata.
	ModuleMetadata *core.ModuleMetadata `json:"#moduleMetadata"`

	// ReleaseMetadata contains release-level identity metadata.
	ReleaseMetadata *core.ReleaseMetadata `json:"#releaseMetadata"`

	// ComponentMetadata contains component-level metadata
	ComponentMetadata *TransformerComponentMetadata `json:"#componentMetadata"`
}

// TransformerComponentMetadata contains component metadata for transformers.
type TransformerComponentMetadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// NewTransformerContext constructs the context for a transformer execution.
func NewTransformerContext(rel *core.ModuleRelease, comp *component.Component) *TransformerContext {
	return &TransformerContext{
		Name:            rel.Metadata.Name,
		Namespace:       rel.Metadata.Namespace,
		ModuleMetadata:  rel.Module.Metadata,
		ReleaseMetadata: rel.Metadata,
		ComponentMetadata: &TransformerComponentMetadata{
			Name:        comp.Name,
			Labels:      comp.Labels,
			Annotations: comp.Annotations,
		},
	}
}

// ToMap converts TransformerContext to a map for CUE encoding.
// The output shape is identical to the previous implementation:
// #moduleReleaseMetadata contains name, namespace, fqn, version, identity, labels.
// The identity value is the release UUID (from ReleaseMetadata.UUID).
// The fqn and version values come from ModuleMetadata.
func (c *TransformerContext) ToMap() map[string]any {
	moduleReleaseMetadata := map[string]any{
		"name":      c.ReleaseMetadata.Name,
		"namespace": c.ReleaseMetadata.Namespace,
		"fqn":       c.ModuleMetadata.FQN,
		"version":   c.ModuleMetadata.Version,
		"identity":  c.ReleaseMetadata.UUID,
	}
	if len(c.ReleaseMetadata.Labels) > 0 {
		moduleReleaseMetadata["labels"] = c.ReleaseMetadata.Labels
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
