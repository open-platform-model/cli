// Package transform provides provider loading, component matching, and transformer execution.
package transform

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"

	"github.com/opmodel/cli/internal/build/component"
	"github.com/opmodel/cli/internal/core"
)

// Job is a unit of work: one transformer applied to one component.
type Job struct {
	Transformer *LoadedTransformer
	Component   *component.Component
	Release     *core.ModuleRelease
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
	Resources []*core.Resource
	Errors    []error
}

// MatchResult is the internal result of matching.
type MatchResult struct {
	// ByTransformer groups components by transformer FQN.
	ByTransformer map[string][]*component.Component

	// Unmatched contains components with no matching transformers.
	Unmatched []*component.Component

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
