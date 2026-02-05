package build

import (
	"fmt"
	"strings"
)

// Matcher evaluates transformer-component matching.
type Matcher struct{}

// NewMatcher creates a new Matcher instance.
func NewMatcher() *Matcher {
	return &Matcher{}
}

// MatchResult is the internal result of matching.
type MatchResult struct {
	// ByTransformer groups components by transformer FQN.
	// Key: transformer FQN
	// Value: list of components that matched
	ByTransformer map[string][]*LoadedComponent

	// Unmatched contains components with no matching transformers.
	Unmatched []*LoadedComponent

	// Details records matching decisions for verbose output.
	Details []MatchDetail
}

// MatchDetail records why a transformer did/didn't match a component.
type MatchDetail struct {
	ComponentName    string
	TransformerFQN   string
	Matched          bool
	MissingLabels    []string
	MissingResources []string
	MissingTraits    []string
	UnhandledTraits  []string
	Reason           string
}

// Match evaluates all transformers against all components.
//
// Matching rules (per FR-B-040 to FR-B-043):
//   - Required labels: All must exist with matching values
//   - Required resources: All FQNs must exist in component.#resources
//   - Required traits: All FQNs must exist in component.#traits
//   - Multiple transformers CAN match one component
//   - Zero matches causes component to be unmatched
func (m *Matcher) Match(components []*LoadedComponent, transformers []*LoadedTransformer) *MatchResult {
	result := &MatchResult{
		ByTransformer: make(map[string][]*LoadedComponent),
		Unmatched:     make([]*LoadedComponent, 0),
		Details:       make([]MatchDetail, 0),
	}

	for _, comp := range components {
		matched := false

		for _, tf := range transformers {
			detail := m.evaluateMatch(comp, tf)
			result.Details = append(result.Details, detail)

			if detail.Matched {
				matched = true
				result.ByTransformer[tf.FQN] = append(result.ByTransformer[tf.FQN], comp)
			}
		}

		if !matched {
			result.Unmatched = append(result.Unmatched, comp)
		}
	}

	return result
}

// evaluateMatch checks if a transformer matches a component.
func (m *Matcher) evaluateMatch(comp *LoadedComponent, tf *LoadedTransformer) MatchDetail {
	detail := MatchDetail{
		ComponentName:  comp.Name,
		TransformerFQN: tf.FQN,
		Matched:        true,
	}

	// Check required labels
	for label, expectedValue := range tf.RequiredLabels {
		actualValue, exists := comp.Labels[label]
		if !exists || actualValue != expectedValue {
			detail.Matched = false
			if !exists {
				detail.MissingLabels = append(detail.MissingLabels, label)
			} else {
				detail.MissingLabels = append(detail.MissingLabels,
					fmt.Sprintf("%s (expected %q, got %q)", label, expectedValue, actualValue))
			}
		}
	}

	// Check required resources
	for _, resourceFQN := range tf.RequiredResources {
		if _, exists := comp.Resources[resourceFQN]; !exists {
			detail.Matched = false
			detail.MissingResources = append(detail.MissingResources, resourceFQN)
		}
	}

	// Check required traits
	for _, traitFQN := range tf.RequiredTraits {
		if _, exists := comp.Traits[traitFQN]; !exists {
			detail.Matched = false
			detail.MissingTraits = append(detail.MissingTraits, traitFQN)
		}
	}

	// Track unhandled traits (traits not in optional or required)
	if detail.Matched {
		handledTraits := make(map[string]bool)
		for _, t := range tf.RequiredTraits {
			handledTraits[t] = true
		}
		for _, t := range tf.OptionalTraits {
			handledTraits[t] = true
		}

		for traitFQN := range comp.Traits {
			if !handledTraits[traitFQN] {
				detail.UnhandledTraits = append(detail.UnhandledTraits, traitFQN)
			}
		}
	}

	// Build reason string
	detail.Reason = m.buildReason(detail, tf)

	return detail
}

// buildReason creates a human-readable match reason.
func (m *Matcher) buildReason(detail MatchDetail, tf *LoadedTransformer) string {
	if detail.Matched {
		var parts []string

		if len(tf.RequiredLabels) > 0 {
			labels := make([]string, 0, len(tf.RequiredLabels))
			for k, v := range tf.RequiredLabels {
				labels = append(labels, fmt.Sprintf("%s=%s", k, v))
			}
			parts = append(parts, "requiredLabels["+strings.Join(labels, ", ")+"]")
		}

		if len(tf.RequiredResources) > 0 {
			parts = append(parts, "requiredResources["+strings.Join(tf.RequiredResources, ", ")+"]")
		}

		if len(tf.RequiredTraits) > 0 {
			parts = append(parts, "requiredTraits["+strings.Join(tf.RequiredTraits, ", ")+"]")
		}

		if len(parts) == 0 {
			return "Matched: no requirements"
		}
		return "Matched: " + strings.Join(parts, ", ")
	}

	// Not matched - explain why
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

// ToMatchPlan converts MatchResult to the shared MatchPlan type.
func (r *MatchResult) ToMatchPlan() MatchPlan {
	matches := make(map[string][]TransformerMatch)

	// Build matches by component name
	for tfFQN, components := range r.ByTransformer {
		for _, comp := range components {
			// Find the detail for this match
			reason := ""
			for _, detail := range r.Details {
				if detail.ComponentName == comp.Name && detail.TransformerFQN == tfFQN && detail.Matched {
					reason = detail.Reason
					break
				}
			}

			matches[comp.Name] = append(matches[comp.Name], TransformerMatch{
				TransformerFQN: tfFQN,
				Reason:         reason,
			})
		}
	}

	// Build unmatched list
	unmatched := make([]string, len(r.Unmatched))
	for i, comp := range r.Unmatched {
		unmatched[i] = comp.Name
	}

	return MatchPlan{
		Matches:   matches,
		Unmatched: unmatched,
	}
}

// GetUnmatchedDetails returns details for unmatched components.
// Useful for error messages showing why nothing matched.
func (r *MatchResult) GetUnmatchedDetails(componentName string) []MatchDetail {
	var details []MatchDetail
	for _, detail := range r.Details {
		if detail.ComponentName == componentName && !detail.Matched {
			details = append(details, detail)
		}
	}
	return details
}
