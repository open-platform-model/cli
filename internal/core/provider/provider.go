// Package provider provides the Provider type and component-transformer matching logic.
package provider

import (
	"fmt"
	"sort"
	"strings"

	"cuelang.org/go/cue"

	"github.com/opmodel/cli/internal/core/component"
	"github.com/opmodel/cli/internal/core/transformer"
)

// Provider holds the parsed representation of a provider CUE module,
// including all transformer definitions and CUE evaluation context.
type Provider struct {
	APIVersion string `json:"apiVersion"`
	Kind       string `json:"kind"`

	Metadata *ProviderMetadata `json:"metadata"`

	Transformers map[string]*transformer.Transformer `json:"transformers,omitempty"`

	DeclaredResources   []cue.Value `json:"#declaredResources,omitempty"`
	DeclaredTraits      []cue.Value `json:"#declaredTraits,omitempty"`
	DeclaredDefinitions []cue.Value `json:"#declaredDefinitions,omitempty"`

	// CueCtx is the CUE evaluation context used to compile this provider's values.
	// Set by the loader (transform.LoadProvider) from GlobalConfig.CueContext.
	// Propagated into TransformerMatchPlan.cueCtx by Match() for use in Execute().
	CueCtx *cue.Context `json:"-"`
}

// ProviderMetadata holds identity metadata for a provider.
type ProviderMetadata struct {
	Name        string            `json:"name"`
	Description string            `json:"description,omitempty"`
	Version     string            `json:"version"`
	MinVersion  string            `json:"minVersion,omitempty"`
	Labels      map[string]string `json:"labels,omitempty"`
}

// Requirements returns all transformers as a []TransformerRequirements slice,
// sorted by transformer name for deterministic output.
func (p *Provider) Requirements() []transformer.TransformerRequirements {
	names := make([]string, 0, len(p.Transformers))
	for name := range p.Transformers {
		names = append(names, name)
	}
	sort.Strings(names)

	result := make([]transformer.TransformerRequirements, 0, len(names))
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
func (p *Provider) Match(components map[string]*component.Component) *transformer.TransformerMatchPlan {
	plan := transformer.NewMatchPlan(p.CueCtx)

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
			m := evaluateMatch(comp, tf)
			plan.AppendMatch(m)
			if m.Matched {
				matched = true
			}
		}

		if !matched {
			name := compName
			if comp.Metadata != nil {
				name = comp.Metadata.Name
			}
			plan.AppendUnmatched(name)
		}
	}

	return plan
}

// evaluateMatch checks whether a single transformer matches a single component.
func evaluateMatch(comp *component.Component, tf *transformer.Transformer) *transformer.TransformerMatch { //nolint:gocyclo // linear per-field match evaluation; each branch is a distinct matching criterion
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

	detail := &transformer.TransformerMatchDetail{
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

	return &transformer.TransformerMatch{
		Matched:     matched,
		Transformer: tf,
		Component:   comp,
		Detail:      detail,
	}
}

// buildMatchReason creates a human-readable match reason string.
func buildMatchReason(matched bool, detail *transformer.TransformerMatchDetail, tf *transformer.Transformer) string {
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
