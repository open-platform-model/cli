package transformer

import (
	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core/component"
)

// TransformerMatchPlan holds the result of matching all transformers against
// all components.
type TransformerMatchPlan struct {
	Matches   []*TransformerMatch `json:"matches,omitempty"`
	Unmatched []string            `json:"unmatched,omitempty"`

	// cueCtx is the CUE evaluation context used by Execute() for encoding operations.
	// Set by Provider.Match() from the Provider's own CueCtx field.
	cueCtx *cue.Context
}

// NewMatchPlan creates a new TransformerMatchPlan with the given CUE context.
// Used by Provider.Match() to construct the plan before populating it.
func NewMatchPlan(cueCtx *cue.Context) *TransformerMatchPlan {
	return &TransformerMatchPlan{
		Matches:   make([]*TransformerMatch, 0),
		Unmatched: make([]string, 0),
		cueCtx:    cueCtx,
	}
}

// AppendMatch adds a match result to the plan.
func (p *TransformerMatchPlan) AppendMatch(m *TransformerMatch) {
	p.Matches = append(p.Matches, m)
}

// AppendUnmatched records a component name that had no matching transformer.
func (p *TransformerMatchPlan) AppendUnmatched(componentName string) {
	p.Unmatched = append(p.Unmatched, componentName)
}

// TransformerMatch records a single (transformer, component) match result.
type TransformerMatch struct {
	// Whether the transformer matched the component.
	Matched bool `json:"matched"`
	// The matched transformer
	Transformer *Transformer `json:"transformer,omitempty"`
	// The matched component
	Component *component.Component `json:"component,omitempty"`

	Detail *TransformerMatchDetail `json:"detail,omitempty"`
}

// TransformerMatchDetail holds diagnostic information about a single match attempt.
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
