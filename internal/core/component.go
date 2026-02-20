package core

import (
	"cuelang.org/go/cue"
)

// Component is a component with extracted metadata.
// Components are extracted by the release builder during the build phase.
type Component struct {
	ApiVersion string               `json:"apiVersion"`
	Kind       string               `json:"kind"`
	Metadata   *ComponentMetadata   `json:"metadata"`

	Resources  map[string]cue.Value `json:"#resources"`  // FQN -> resource value
	Traits     map[string]cue.Value `json:"#traits"`     // FQN -> trait value
	Blueprints map[string]cue.Value `json:"#blueprints"` // FQN -> blueprint value
	Spec       cue.Value            `json:"spec"`        // OpenAPI schema for the component spec
	Value      cue.Value            `json:"value"`       // Full component value
}

type ComponentMetadata struct {
	Name        string            `json:"name"`
	Labels      map[string]string `json:"labels"`
	Annotations map[string]string `json:"annotations"`
}
