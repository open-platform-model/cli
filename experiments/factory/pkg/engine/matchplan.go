package engine

import (
	"fmt"

	"cuelang.org/go/cue"
	"cuelang.org/go/cue/load"
)

// MatchResult is the Go representation of a single (component, transformer) match outcome.
type MatchResult struct {
	Matched          bool     `json:"matched"`
	MissingLabels    []string `json:"missingLabels"`
	MissingResources []string `json:"missingResources"`
	MissingTraits    []string `json:"missingTraits"`
}

// MatchPlan holds the decoded result of matching all components against all transformers.
//
// Matches is a two-level map: component name → transformer FQN → MatchResult.
// Unmatched lists component names for which no transformer matched.
// UnhandledTraits maps component name to trait FQNs not handled by any matched transformer.
type MatchPlan struct {
	// Matches[compName][tfFQN] = result of evaluating the (comp, tf) pair.
	Matches map[string]map[string]MatchResult

	// Unmatched holds component names with zero matching transformers (error condition).
	Unmatched []string

	// UnhandledTraits[compName] = trait FQNs present on the component but handled
	// by no matched transformer (warning condition).
	UnhandledTraits map[string][]string
}

// MatchedPair is a resolved (component, transformer) pair ready for transform execution.
type MatchedPair struct {
	ComponentName  string
	TransformerFQN string
}

// MatchedPairs returns all (component, transformer) pairs where matched == true,
// sorted deterministically: by component name first, then transformer FQN.
func (p *MatchPlan) MatchedPairs() []MatchedPair {
	pairs := make([]MatchedPair, 0)
	for compName, tfResults := range p.Matches {
		for tfFQN, result := range tfResults {
			if result.Matched {
				pairs = append(pairs, MatchedPair{
					ComponentName:  compName,
					TransformerFQN: tfFQN,
				})
			}
		}
	}
	sortMatchedPairs(pairs)
	return pairs
}

// Warnings returns human-readable warning strings for unhandled traits.
func (p *MatchPlan) Warnings() []string {
	var warnings []string
	for compName, traits := range p.UnhandledTraits {
		for _, fqn := range traits {
			warnings = append(warnings, fmt.Sprintf(
				"component %q: trait %q is not handled by any matched transformer (values will be ignored)",
				compName, fqn,
			))
		}
	}
	return warnings
}

// buildMatchPlan evaluates #MatchPlan from the CUE matcher package by filling in
// #provider and #components, then decodes the result into a Go MatchPlan using
// Decode() directly into tagged structs.
//
// schemaComponents must come from rel.Schema (not rel.DataComponents) so that
// definition fields (#resources, #traits) are present for the CUE matching logic.
func buildMatchPlan(cueCtx *cue.Context, cueModuleDir string, providerVal, schemaComponents cue.Value) (*MatchPlan, error) {
	// Load the matcher package from the CUE module.
	cfg := &load.Config{Dir: cueModuleDir}
	instances := load.Instances([]string{"./core/matcher"}, cfg)
	if len(instances) == 0 {
		return nil, fmt.Errorf("matcher package not found in %s", cueModuleDir)
	}
	if instances[0].Err != nil {
		return nil, fmt.Errorf("loading matcher package: %w", instances[0].Err)
	}

	matcherPkg := cueCtx.BuildInstance(instances[0])
	if err := matcherPkg.Err(); err != nil {
		return nil, fmt.Errorf("building matcher package: %w", err)
	}

	// Look up the #MatchPlan definition and fill its inputs.
	matchPlanDef := matcherPkg.LookupPath(cue.MakePath(cue.Def("MatchPlan")))
	if !matchPlanDef.Exists() {
		return nil, fmt.Errorf("#MatchPlan not found in matcher package")
	}

	filled := matchPlanDef.FillPath(cue.MakePath(cue.Def("provider")), providerVal)
	filled = filled.FillPath(cue.MakePath(cue.Def("components")), schemaComponents)

	if err := filled.Err(); err != nil {
		return nil, fmt.Errorf("evaluating #MatchPlan: %w", err)
	}

	// matchPlanResult mirrors the three output fields of #MatchPlan.
	// Decode() uses json tags to map CUE field names to Go struct fields.
	type matchPlanResult struct {
		Matches         map[string]map[string]MatchResult `json:"matches"`
		Unmatched       []string                          `json:"unmatched"`
		UnhandledTraits map[string][]string               `json:"unhandledTraits"`
	}

	var result matchPlanResult
	if err := filled.Decode(&result); err != nil {
		return nil, fmt.Errorf("decoding match plan: %w", err)
	}

	return &MatchPlan{
		Matches:         result.Matches,
		Unmatched:       result.Unmatched,
		UnhandledTraits: result.UnhandledTraits,
	}, nil
}

// sortMatchedPairs sorts pairs in-place: component name ascending, then transformer FQN ascending.
func sortMatchedPairs(pairs []MatchedPair) {
	for i := 1; i < len(pairs); i++ {
		for j := i; j > 0; j-- {
			a, b := pairs[j-1], pairs[j]
			if a.ComponentName > b.ComponentName ||
				(a.ComponentName == b.ComponentName && a.TransformerFQN > b.TransformerFQN) {
				pairs[j-1], pairs[j] = pairs[j], pairs[j-1]
			} else {
				break
			}
		}
	}
}
