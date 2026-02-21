package core

import (
	"fmt"
	"sort"
	"strings"

	"cuelang.org/go/cue"
)

type Provider struct {
	ApiVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Metadata *ProviderMetadata `json:"metadata"`

	Transformers map[string]*Transformer `json:"transformers,omitempty"`

	DeclaredResources   []cue.Value `json:"#declaredResources,omitempty"`
	DeclaredTraits      []cue.Value `json:"#declaredTraits,omitempty"`
	DeclaredDefinitions []cue.Value `json:"#declaredDefinitions,omitempty"`
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

// Requirements returns all transformers as a []TransformerRequirements slice,
// sorted by transformer name for deterministic output.
func (p *Provider) Requirements() []TransformerRequirements {
	names := make([]string, 0, len(p.Transformers))
	for name := range p.Transformers {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]TransformerRequirements, 0, len(names))
	for _, name := range names {
		result = append(result, p.Transformers[name])
	}
	return result
}

// Match evaluates all transformers in the provider against all components and
// returns a *TransformerMatchPlan describing matched and unmatched pairs.
//
// Matching rules:
//   - Required labels: All must exist with matching values
//   - Required resources: All FQNs must exist in component.Resources
//   - Required traits: All FQNs must exist in component.Traits
//   - Multiple transformers CAN match one component
//   - Zero matches causes component to be listed in Unmatched
//
// Component names and transformer names are both sorted before iteration to
// produce a deterministic match plan regardless of map iteration order.
func (p *Provider) Match(components map[string]*Component) *TransformerMatchPlan {
	plan := &TransformerMatchPlan{
		Matches:   make([]*TransformerMatch, 0),
		Unmatched: make([]string, 0),
	}

	// Sort component names for deterministic output.
	compNames := make([]string, 0, len(components))
	for name := range components {
		compNames = append(compNames, name)
	}
	sort.Strings(compNames)

	// Sort transformer names for deterministic output.
	tfNames := make([]string, 0, len(p.Transformers))
	for name := range p.Transformers {
		tfNames = append(tfNames, name)
	}
	sort.Strings(tfNames)

	for _, compName := range compNames {
		comp := components[compName]
		matched := false

		for _, tfName := range tfNames {
			tf := p.Transformers[tfName]
			m := p.evaluateMatch(comp, tf)
			plan.Matches = append(plan.Matches, m)
			if m.Matched {
				matched = true
			}
		}

		if !matched {
			name := compName
			if comp.Metadata != nil {
				name = comp.Metadata.Name
			}
			plan.Unmatched = append(plan.Unmatched, name)
		}
	}

	return plan
}

// evaluateMatch checks whether a single transformer matches a single component.
func (p *Provider) evaluateMatch(comp *Component, tf *Transformer) *TransformerMatch {
	name := ""
	if comp.Metadata != nil {
		name = comp.Metadata.Name
	}
	fqn := ""
	if tf.Metadata != nil {
		fqn = tf.Metadata.FQN
	}

	labels := map[string]string{}
	if comp.Metadata != nil && comp.Metadata.Labels != nil {
		labels = comp.Metadata.Labels
	}

	detail := &TransformerMatchDetail{
		ComponentName:  name,
		TransformerFQN: fqn,
	}
	matched := true

	// Check required labels.
	for label, expectedValue := range tf.RequiredLabels {
		actualValue, exists := labels[label]
		if !exists || actualValue != expectedValue {
			matched = false
			if !exists {
				detail.MissingLabels = append(detail.MissingLabels, label)
			} else {
				detail.MissingLabels = append(detail.MissingLabels,
					fmt.Sprintf("%s (expected %q, got %q)", label, expectedValue, actualValue))
			}
		}
	}

	// Check required resources.
	for resourceFQN := range tf.RequiredResources {
		if _, exists := comp.Resources[resourceFQN]; !exists {
			matched = false
			detail.MissingResources = append(detail.MissingResources, resourceFQN)
		}
	}

	// Check required traits.
	for traitFQN := range tf.RequiredTraits {
		if _, exists := comp.Traits[traitFQN]; !exists {
			matched = false
			detail.MissingTraits = append(detail.MissingTraits, traitFQN)
		}
	}

	// Track unhandled traits — only meaningful when the transformer matched.
	if matched {
		handledTraits := make(map[string]bool)
		for t := range tf.RequiredTraits {
			handledTraits[t] = true
		}
		for t := range tf.OptionalTraits {
			handledTraits[t] = true
		}
		for traitFQN := range comp.Traits {
			if !handledTraits[traitFQN] {
				detail.UnhandledTraits = append(detail.UnhandledTraits, traitFQN)
			}
		}
	}

	detail.Reason = buildMatchReason(matched, detail, tf)

	return &TransformerMatch{
		Matched:     matched,
		Transformer: tf,
		Component:   comp,
		Detail:      detail,
	}
}

// buildMatchReason creates a human-readable match reason string.
func buildMatchReason(matched bool, detail *TransformerMatchDetail, tf *Transformer) string {
	if matched {
		var parts []string

		if len(tf.RequiredLabels) > 0 {
			labels := make([]string, 0, len(tf.RequiredLabels))
			for k, v := range tf.RequiredLabels {
				labels = append(labels, fmt.Sprintf("%s=%s", k, v))
			}
			sort.Strings(labels)
			parts = append(parts, "requiredLabels["+strings.Join(labels, ", ")+"]")
		}

		if len(tf.RequiredResources) > 0 {
			keys := make([]string, 0, len(tf.RequiredResources))
			for k := range tf.RequiredResources {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts = append(parts, "requiredResources["+strings.Join(keys, ", ")+"]")
		}

		if len(tf.RequiredTraits) > 0 {
			keys := make([]string, 0, len(tf.RequiredTraits))
			for k := range tf.RequiredTraits {
				keys = append(keys, k)
			}
			sort.Strings(keys)
			parts = append(parts, "requiredTraits["+strings.Join(keys, ", ")+"]")
		}

		if len(parts) == 0 {
			return "Matched: no requirements"
		}
		return "Matched: " + strings.Join(parts, ", ")
	}

	// Not matched — explain why.
	var reasons []string
	if len(detail.MissingLabels) > 0 {
		reasons = append(reasons, "missing labels: "+strings.Join(detail.MissingLabels, ", "))
	}
	if len(detail.MissingResources) > 0 {
		reasons = append(reasons, "missing resources: "+strings.Join(detail.MissingResources, ", "))
	}
	if len(detail.MissingTraits) > 0 {
		reasons = append(reasons, "missing traits: "+strings.Join(detail.MissingTraits, ", "))
	}
	return "Not matched: " + strings.Join(reasons, "; ")
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

// TransformerRequirements is the interface satisfied by types that expose
// transformer matching requirements. Used for error messages and diagnostics.
type TransformerRequirements interface {
	GetFQN() string
	GetRequiredLabels() map[string]string
	GetRequiredResources() []string
	GetRequiredTraits() []string
}

// ToLegacyMatchPlan converts a TransformerMatchPlan to a MatchPlan for use
// by RenderResult consumers (cmdutil/output.go) until those are migrated to
// read TransformerMatchPlan directly.
func (p *TransformerMatchPlan) ToLegacyMatchPlan() MatchPlan {
	matches := make(map[string][]TransformerMatchOld)
	for _, m := range p.Matches {
		if !m.Matched || m.Detail == nil {
			continue
		}
		compName := m.Detail.ComponentName
		matches[compName] = append(matches[compName], TransformerMatchOld{
			TransformerFQN: m.Detail.TransformerFQN,
			Reason:         m.Detail.Reason,
		})
	}
	return MatchPlan{
		Matches:   matches,
		Unmatched: p.Unmatched,
	}
}

// MatchPlan describes the transformer-component matching results.
// Retained as the shape of RenderResult.MatchPlan; consumers use the
// map-keyed-by-component-name view. Will be replaced by TransformerMatchPlan
// when cmdutil/output.go and its tests are migrated.
type MatchPlan struct {
	// Matches maps component names to their matched transformers.
	Matches map[string][]TransformerMatchOld

	// Unmatched lists components with no matching transformers.
	Unmatched []string
}

// TransformerMatchOld records a single transformer match for MatchPlan.
type TransformerMatchOld struct {
	// TransformerFQN is the fully qualified transformer name.
	TransformerFQN string

	// Reason explains why this transformer matched.
	Reason string
}
