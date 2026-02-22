// Package component provides the Component domain type and extraction helpers,
// mirroring catalog/v0/core/component.cue.
package component

import (
	"fmt"

	"cuelang.org/go/cue"
)

// Component is a component with extracted metadata.
// Components are extracted by the release builder during the build phase.
type Component struct {
	APIVersion string             `json:"apiVersion"`
	Kind       string             `json:"kind"`
	Metadata   *ComponentMetadata `json:"metadata"`

	Resources  map[string]cue.Value `json:"#resources"`  // FQN -> resource value
	Traits     map[string]cue.Value `json:"#traits"`     // FQN -> trait value
	Blueprints map[string]cue.Value `json:"#blueprints"` // FQN -> blueprint value
	Spec       cue.Value            `json:"spec"`        // OpenAPI schema for the component spec
	Value      cue.Value            `json:"value"`       // Full component value
}

// ComponentMetadata holds the identity and routing metadata for a component.
type ComponentMetadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}

// Validate checks that the component has the minimum required fields populated.
// Returns an error describing the first missing required field.
func (c *Component) Validate() error {
	if c.Metadata == nil {
		return fmt.Errorf("component metadata is nil")
	}
	if c.Metadata.Name == "" {
		return fmt.Errorf("component metadata.name is empty")
	}
	if len(c.Resources) == 0 {
		return fmt.Errorf("component %q has no resources", c.Metadata.Name)
	}
	if !c.Value.Exists() {
		return fmt.Errorf("component %q has no CUE value", c.Metadata.Name)
	}
	return nil
}

// IsConcrete reports whether the component's CUE value is fully concrete
// (i.e., all fields have concrete values, no unresolved constraints).
func (c *Component) IsConcrete() bool {
	return c.Value.Validate(cue.Concrete(true)) == nil
}

// ExtractComponents extracts components from a CUE value representing #components.
// It iterates all fields of the provided value, extracts each component, and validates it.
// Returns an error if the value has no fields or a component fails Validate().
func ExtractComponents(v cue.Value) (map[string]*Component, error) {
	components := make(map[string]*Component)

	iter, err := v.Fields()
	if err != nil {
		return nil, fmt.Errorf("iterating components: %w", err)
	}

	for iter.Next() {
		name := iter.Selector().Unquoted()
		compValue := iter.Value()

		comp := extractComponent(name, compValue)
		if err := comp.Validate(); err != nil {
			return nil, fmt.Errorf("component %q: %w", name, err)
		}
		components[name] = comp
	}

	return components, nil
}

// extractComponent extracts a single component from a CUE value.
func extractComponent(name string, value cue.Value) *Component { //nolint:gocyclo // linear field extraction; each branch is a distinct component field
	comp := &Component{
		Metadata: &ComponentMetadata{
			Name:        name,
			Labels:      make(map[string]string),
			Annotations: make(map[string]string),
		},
		Resources:  make(map[string]cue.Value),
		Traits:     make(map[string]cue.Value),
		Blueprints: make(map[string]cue.Value),
		Value:      value,
	}

	// Override name from metadata.name if present
	if metaName := value.LookupPath(cue.ParsePath("metadata.name")); metaName.Exists() {
		if str, err := metaName.String(); err == nil {
			comp.Metadata.Name = str
		}
	}

	// Extract #resources
	if resourcesValue := value.LookupPath(cue.ParsePath("#resources")); resourcesValue.Exists() {
		if iter, err := resourcesValue.Fields(); err == nil {
			for iter.Next() {
				comp.Resources[iter.Selector().Unquoted()] = iter.Value()
			}
		}
	}

	// Extract #traits
	if traitsValue := value.LookupPath(cue.ParsePath("#traits")); traitsValue.Exists() {
		if iter, err := traitsValue.Fields(); err == nil {
			for iter.Next() {
				comp.Traits[iter.Selector().Unquoted()] = iter.Value()
			}
		}
	}

	// Extract #blueprints
	if blueprintsValue := value.LookupPath(cue.ParsePath("#blueprints")); blueprintsValue.Exists() {
		if iter, err := blueprintsValue.Fields(); err == nil {
			for iter.Next() {
				comp.Blueprints[iter.Selector().Unquoted()] = iter.Value()
			}
		}
	}

	// Extract spec
	if specValue := value.LookupPath(cue.ParsePath("spec")); specValue.Exists() {
		comp.Spec = specValue
	}

	// Extract metadata.labels
	if labelsValue := value.LookupPath(cue.ParsePath("metadata.labels")); labelsValue.Exists() {
		if iter, err := labelsValue.Fields(); err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					comp.Metadata.Labels[iter.Selector().Unquoted()] = str
				}
			}
		}
	}

	// Extract metadata.annotations
	if annotationsValue := value.LookupPath(cue.ParsePath("metadata.annotations")); annotationsValue.Exists() {
		if iter, err := annotationsValue.Fields(); err == nil {
			for iter.Next() {
				if str, err := iter.Value().String(); err == nil {
					comp.Metadata.Annotations[iter.Selector().Unquoted()] = str
				}
			}
		}
	}

	return comp
}
