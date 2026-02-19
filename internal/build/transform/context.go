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

	// ModuleMetadata contains module-level identity metadata.
	ModuleMetadata *module.ModuleMetadata `json:"#moduleMetadata"`

	// ReleaseMetadata contains release-level identity metadata.
	ReleaseMetadata *release.ReleaseMetadata `json:"#releaseMetadata"`

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
func NewTransformerContext(rel *release.BuiltRelease, component *module.LoadedComponent) *TransformerContext {
	relMeta := rel.ToReleaseMetadata()
	// ModuleMetadata in transformer context: use release name as module name since
	// we only have the release-level name here. DefaultNamespace is not available
	// at this point; callers that need the canonical module name use the pipeline.
	modMeta := rel.ToModuleMetadata(rel.Metadata.Name, "")
	return &TransformerContext{
		Name:            rel.Metadata.Name,
		Namespace:       rel.Metadata.Namespace,
		ModuleMetadata:  &modMeta,
		ReleaseMetadata: &relMeta,
		ComponentMetadata: &TransformerComponentMetadata{
			Name:        component.Name,
			Labels:      component.Labels,
			Annotations: component.Annotations,
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
