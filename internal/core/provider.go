package core

import "cuelang.org/go/cue"

type Provider struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Metadata *ProviderMetadata `json:"metadata"`

	Transformers map[string]*Transformer `json:"transformers,omitempty"`

	DeclaredResources   []cue.Value `json:"#declaredResources,omitempty"`
	DeclaredTraits      []cue.Value `json:"#declaredTraits,omitempty"`
	DeclaredDefinitions []cue.Value `json:"#declaredDefinitions,omitempty"`

	// CueCtx is the CUE evaluation context used to compile this provider's values.
	// Set by the loader (transform.LoadProvider) from GlobalConfig.CueContext.
	// Required for transformer execution in TransformerMatchPlan.Execute().
	CueCtx *cue.Context `json:"-"`
}

type ProviderMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version"`
	MinVersion  string            `json:"minVersion,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

type Transformer struct {
	ApiVersion string `json:"apiVersion"`
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

type TransformerMetadata struct {
	ApiVersion  string            `json:"apiVersion"`
	Name        string            `json:"name"`
	FQN         string            `json:"fqn"`
	Description string            `json:"description,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
	Annotations map[string]string `json:"annotations,omitempty"`
}

type TransformerMatchPlan struct {
	Matches   []*TransformerMatch `json:"matches,omitempty"`
	Unmatched []string            `json:"unmatched,omitempty"`
}

type TransformerMatch struct {
	// Whether the transformer matched the component.
	Matched bool `json:"matched"`
	// The matched transformer
	Transformer *Transformer `json:"transformer,omitempty"`
	// The matched component
	Component *Component `json:"component,omitempty"`

	Detail *TransformerMatchDetail `json:"detail,omitempty"`
}

type TransformerMatchDetail struct {
	// ComponentName is the name of the matched component, for verbose output and debugging.
	ComponentName string `json:"componentName"`
	// The fully qualified name of the matched transformer.
	TransformerFQN string `json:"transformerFQN"`
	// Reason explains why this transformer matched or did not match, for verbose output and debugging.
	Reason string `json:"reason,omitempty"`

	MissingLabels      []string `json:"missingLabels,omitempty"`
	MissingResources   []string `json:"missingResources,omitempty"`
	MissingTraits      []string `json:"missingTraits,omitempty"`
	UnhandledResources []string `json:"unhandledResources,omitempty"`
	UnhandledTraits    []string `json:"unhandledTraits,omitempty"`
}

// LEGACY TYPES

// MatchPlan describes the transformer-component matching results.
// Used for verbose output and debugging; not part of the core render contract.
type MatchPlan struct {
	// Matches maps component names to their matched transformers.
	Matches map[string][]TransformerMatchOld

	// Unmatched lists components with no matching transformers.
	Unmatched []string
}

// TransformerMatch records a single transformer match.
type TransformerMatchOld struct {
	// TransformerFQN is the fully qualified transformer name.
	TransformerFQN string

	// Reason explains why this transformer matched.
	Reason string
}

// TransformerRequirements is the interface satisfied by LoadedTransformer.
// It exposes the minimum set of fields needed for error messages and
// matching diagnostics, without copying data into a separate struct.
type TransformerRequirements interface {
	GetFQN() string
	GetRequiredLabels() map[string]string
	GetRequiredResources() []string
	GetRequiredTraits() []string
}
