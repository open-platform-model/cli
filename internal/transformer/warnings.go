// Package transformer provides utilities for working with transformer match plans,
// including warning collection for unhandled component traits.
package transformer

import (
	"github.com/opmodel/cli/internal/core"
)

// CollectWarnings gathers non-fatal warnings from the match plan.
//
// A trait is considered "unhandled" only if NO matched transformer handles it.
// This means if ServiceTransformer requires Expose trait and DeploymentTransformer
// doesn't, the Expose trait is still considered handled (by ServiceTransformer).
func CollectWarnings(plan *core.TransformerMatchPlan) []string {
	var warnings []string

	// Step 1: Count how many transformers matched each component.
	componentMatchCount := make(map[string]int)
	for _, m := range plan.Matches {
		if m.Matched && m.Detail != nil {
			componentMatchCount[m.Detail.ComponentName]++
		}
	}

	// Step 2: Count how many matched transformers consider each trait unhandled.
	// Key: component name, Value: map of trait -> count of transformers that don't handle it.
	traitUnhandledCount := make(map[string]map[string]int)
	for _, m := range plan.Matches {
		if m.Matched && m.Detail != nil {
			if traitUnhandledCount[m.Detail.ComponentName] == nil {
				traitUnhandledCount[m.Detail.ComponentName] = make(map[string]int)
			}
			for _, trait := range m.Detail.UnhandledTraits {
				traitUnhandledCount[m.Detail.ComponentName][trait]++
			}
		}
	}

	// Step 3: A trait is truly unhandled only if ALL matched transformers
	// consider it unhandled (i.e., no transformer handles it).
	for componentName, traitCounts := range traitUnhandledCount {
		matchCount := componentMatchCount[componentName]
		for trait, unhandledCount := range traitCounts {
			if unhandledCount == matchCount {
				warnings = append(warnings,
					"component "+componentName+": unhandled trait "+trait)
			}
		}
	}

	return warnings
}
