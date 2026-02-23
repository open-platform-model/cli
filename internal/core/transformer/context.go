package transformer

import (
	"github.com/opmodel/cli/internal/core/component"
	"github.com/opmodel/cli/internal/core/module"
	"github.com/opmodel/cli/internal/core/modulerelease"
)

// TransformerContext holds the context data passed to transformers during execution.
// This matches the CUE #TransformerContext definition.
type TransformerContext struct {
	// Name is the release name (from --name or module.metadata.name)
	Name string `json:"name"`

	// Namespace is the target namespace (from --namespace or defaultNamespace)
	Namespace string `json:"namespace"`

	// ModuleMetadata contains module-level identity metadata.
	ModuleMetadata *module.ModuleMetadata `json:"#moduleMetadata"`

	// ReleaseMetadata contains release-level identity metadata.
	ReleaseMetadata *modulerelease.ReleaseMetadata `json:"#releaseMetadata"`

	// ComponentMetadata contains component-level metadata.
	// Uses component.ComponentMetadata directly — no separate wrapper type needed.
	ComponentMetadata *component.ComponentMetadata `json:"#componentMetadata"`
}

// NewTransformerContext constructs the context for a transformer execution.
func NewTransformerContext(rel *modulerelease.ModuleRelease, comp *component.Component) *TransformerContext {
	compMeta := &component.ComponentMetadata{
		Name:        "",
		Labels:      map[string]string{},
		Annotations: map[string]string{},
	}
	if comp.Metadata != nil {
		compMeta.Name = comp.Metadata.Name
		if comp.Metadata.Labels != nil {
			compMeta.Labels = comp.Metadata.Labels
		}
		if comp.Metadata.Annotations != nil {
			compMeta.Annotations = comp.Metadata.Annotations
		}
	}
	return &TransformerContext{
		Name:              rel.Metadata.Name,
		Namespace:         rel.Metadata.Namespace,
		ModuleMetadata:    rel.Module.Metadata,
		ReleaseMetadata:   rel.Metadata,
		ComponentMetadata: compMeta,
	}
}

// ToMap converts TransformerContext to a map for CUE encoding.
// The output shape matches the CUE #context definition:
// #moduleReleaseMetadata contains name, namespace, fqn, version, uuid, labels, annotations.
// The uuid value is the release UUID (from ReleaseMetadata.UUID).
// The fqn and version values come from ModuleMetadata.
func (c *TransformerContext) ToMap() map[string]any {
	moduleReleaseMetadata := map[string]any{
		"name":      c.ReleaseMetadata.Name,
		"namespace": c.ReleaseMetadata.Namespace,
		"fqn":       c.ModuleMetadata.FQN,
		"version":   c.ModuleMetadata.Version,
		"uuid":      c.ReleaseMetadata.UUID,
	}
	if len(c.ReleaseMetadata.Labels) > 0 {
		moduleReleaseMetadata["labels"] = c.ReleaseMetadata.Labels
	}
	if len(c.ReleaseMetadata.Annotations) > 0 {
		moduleReleaseMetadata["annotations"] = c.ReleaseMetadata.Annotations
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
