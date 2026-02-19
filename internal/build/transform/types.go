// Package transform provides provider loading, component matching, and transformer execution.
package transform

import (
	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"

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

// GVK returns the GroupVersionKind of the resource.
func (r *Resource) GVK() schema.GroupVersionKind {
	return r.Object.GroupVersionKind()
}

// Kind returns the resource kind (e.g., "Deployment").
func (r *Resource) Kind() string {
	return r.Object.GetKind()
}

// Name returns the resource name from metadata.
func (r *Resource) Name() string {
	return r.Object.GetName()
}

// Namespace returns the resource namespace from metadata.
// Empty string for cluster-scoped resources.
func (r *Resource) Namespace() string {
	return r.Object.GetNamespace()
}

// Labels returns the resource labels.
func (r *Resource) Labels() map[string]string {
	return r.Object.GetLabels()
}

// GetObject returns the underlying unstructured object.
func (r *Resource) GetObject() *unstructured.Unstructured {
	return r.Object
}

// GetGVK returns the GroupVersionKind.
func (r *Resource) GetGVK() schema.GroupVersionKind {
	return r.GVK()
}

// GetKind returns the resource kind.
func (r *Resource) GetKind() string {
	return r.Kind()
}

// GetName returns the resource name.
func (r *Resource) GetName() string {
	return r.Name()
}

// GetNamespace returns the resource namespace.
func (r *Resource) GetNamespace() string {
	return r.Namespace()
}

// GetComponent returns the source component name.
func (r *Resource) GetComponent() string {
	return r.Component
}

// GetTransformer returns the transformer FQN.
func (r *Resource) GetTransformer() string {
	return r.Transformer
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
// Used for verbose output and debugging; not part of the core render contract.
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

// TransformerRequirements is the interface satisfied by LoadedTransformer.
// It exposes the minimum set of fields needed for error messages and
// matching diagnostics, without copying data into a separate struct.
type TransformerRequirements interface {
	GetFQN() string
	GetRequiredLabels() map[string]string
	GetRequiredResources() []string
	GetRequiredTraits() []string
}
