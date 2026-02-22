// Package transformer provides the core transformer types, matching logic,
// and execution engine used by the render pipeline.
package transformer

import (
	"sort"

	"cuelang.org/go/cue"
)

// Transformer holds the parsed representation of a transformer definition
// from a provider CUE module.
type Transformer struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Metadata *TransformerMetadata `json:"metadata"`

	RequiredLabels    map[string]string    `json:"requiredLabels,omitempty"`
	RequiredResources map[string]cue.Value `json:"requiredResources,omitempty"`
	RequiredTraits    map[string]cue.Value `json:"requiredTraits,omitempty"`

	OptionalLabels    map[string]string    `json:"optionalLabels,omitempty"`
	OptionalResources map[string]cue.Value `json:"optionalResources,omitempty"`
	OptionalTraits    map[string]cue.Value `json:"optionalTraits,omitempty"`

	Transform cue.Value `json:"#transform,omitempty"`
}

// TransformerMetadata holds identity metadata for a transformer.
type TransformerMetadata struct {
	APIVersion  string            `json:"apiVersion"`
	Name        string            `json:"name"`
	FQN         string            `json:"fqn"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

// GetFQN returns the transformer's fully qualified name.
func (t *Transformer) GetFQN() string {
	if t.Metadata == nil {
		return ""
	}
	return t.Metadata.FQN
}

// GetRequiredLabels returns the transformer's required labels.
func (t *Transformer) GetRequiredLabels() map[string]string {
	return t.RequiredLabels
}

// GetRequiredResources returns the FQNs of the transformer's required resources.
func (t *Transformer) GetRequiredResources() []string {
	keys := make([]string, 0, len(t.RequiredResources))
	for k := range t.RequiredResources {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// GetRequiredTraits returns the FQNs of the transformer's required traits.
func (t *Transformer) GetRequiredTraits() []string {
	keys := make([]string, 0, len(t.RequiredTraits))
	for k := range t.RequiredTraits {
		keys = append(keys, k)
	}
	sort.Strings(keys)
	return keys
}

// TransformerRequirements is the interface satisfied by types that expose
// transformer matching requirements. Used for error messages and diagnostics.
type TransformerRequirements interface {
	GetFQN() string
	GetRequiredLabels() map[string]string
	GetRequiredResources() []string
	GetRequiredTraits() []string
}
