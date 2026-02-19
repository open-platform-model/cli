// Package transform provides provider loading, component matching, and transformer execution.
package transform

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/build/module"
	"github.com/opmodel/cli/internal/build/release"
)

// Job is a unit of work: one transformer applied to one component.
type Job struct {
	Transformer *LoadedTransformer
	Component   *module.LoadedComponent
	Release     *release.BuiltRelease
}

// JobResult is the result of executing a job.
type JobResult struct {
	Component   string
	Transformer string
	Resources   []*unstructured.Unstructured
	Error       error
}

// ExecuteResult is the combined result of all jobs.
type ExecuteResult struct {
	Resources []*Resource
	Errors    []error
}

// Resource represents a single rendered platform resource.
type Resource struct {
	Object      *unstructured.Unstructured
	Component   string
	Transformer string
}

// MatchResult is the internal result of matching.
type MatchResult struct {
	// ByTransformer groups components by transformer FQN.
	ByTransformer map[string][]*module.LoadedComponent

	// Unmatched contains components with no matching transformers.
	Unmatched []*module.LoadedComponent

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

// MatchPlan describes the transformer-component matching results.
// Used for verbose output and debugging.
type MatchPlan struct {
	// Matches maps component names to their matched transformers.
	Matches map[string][]TransformerMatch

	// Unmatched lists components with no matching transformers.
	Unmatched []string
}

// TransformerMatch records a single transformer match.
type TransformerMatch struct {
	// TransformerFQN is the fully qualified transformer name.
	TransformerFQN string

	// Reason explains why this transformer matched.
	Reason string
}

// TransformerSummary provides guidance on transformer requirements.
// Used in error messages to help users understand matching.
type TransformerSummary struct {
	FQN               string
	RequiredLabels    map[string]string
	RequiredResources []string
	RequiredTraits    []string
}
