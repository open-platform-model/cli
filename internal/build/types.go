// Package build provides the render pipeline interface and shared types.
// This package defines the contract between rendering operations and their consumers
// (build, apply, diff commands).
package build

import (
	"context"
	"errors"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/schema"
)

// Pipeline defines the contract for render pipelines.
// Implemented by the build pipeline in this package.
type Pipeline interface {
	// Render executes the pipeline and returns results.
	//
	// Fatal errors (module not found, provider missing) return error.
	// Render errors (unmatched components, transform failures) are in RenderResult.Errors.
	//
	// The context is used for cancellation. Long-running operations should
	// check ctx.Done() and return ctx.Err() if canceled.
	Render(ctx context.Context, opts RenderOptions) (*RenderResult, error)
}

// RenderOptions configures a render operation.
type RenderOptions struct {
	// ModulePath is the path to the module directory.
	// Required. Must contain cue.mod/ and module.cue.
	ModulePath string

	// Values are paths to additional values files to unify (in order).
	// Optional. Files are unified after values.cue from module root.
	Values []string

	// Name overrides module.metadata.name for the release.
	// Optional. If empty, uses module.metadata.name.
	Name string

	// Namespace overrides module.metadata.defaultNamespace.
	// Required if module doesn't define defaultNamespace.
	Namespace string

	// Provider selects which provider to use.
	// Optional. If empty, uses default provider from config.
	Provider string

	// Strict enables strict trait handling.
	// When true, unhandled traits cause errors instead of warnings.
	Strict bool

	// Registry overrides the CUE registry URL.
	// Optional. If empty, uses resolved registry from config.
	Registry string
}

// Validate checks that required options are set.
func (o RenderOptions) Validate() error {
	if o.ModulePath == "" {
		return errors.New("ModulePath is required")
	}
	return nil
}

// RenderResult is the output of a render operation.
// This is the contract between rendering and consumers.
type RenderResult struct {
	// Resources are the rendered platform resources.
	// Ordered for sequential apply (respecting resource weights/dependencies).
	// Empty slice (not nil) if no resources were rendered.
	Resources []*Resource

	// Module contains metadata about the source module.
	Module ModuleMetadata

	// MatchPlan describes which transformers matched which components.
	// Used for verbose output and debugging.
	MatchPlan MatchPlan

	// Errors contains aggregated render errors (fail-on-end pattern).
	// Empty slice if all components rendered successfully.
	Errors []error

	// Warnings contains non-fatal warnings.
	// Examples: deprecated transformer used, unused values.
	Warnings []string
}

// HasErrors returns true if there are render errors.
func (r *RenderResult) HasErrors() bool {
	return len(r.Errors) > 0
}

// HasWarnings returns true if there are warnings.
func (r *RenderResult) HasWarnings() bool {
	return len(r.Warnings) > 0
}

// ResourceCount returns the number of rendered resources.
func (r *RenderResult) ResourceCount() int {
	return len(r.Resources)
}

// Resource represents a single rendered platform resource.
type Resource struct {
	// Object is the rendered resource as unstructured data.
	// Includes all metadata, labels, and spec fields.
	Object *unstructured.Unstructured

	// Component is the name of the source component.
	// Matches a key in module.components.
	Component string

	// Transformer is the FQN of the transformer that produced this resource.
	// Example: "opmodel.dev/transformers/kubernetes@v0#DeploymentTransformer"
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

// ModuleMetadata contains information about the source module.
// This metadata is used for labeling resources and verbose output.
type ModuleMetadata struct {
	// Name is the module name.
	// May be overridden by RenderOptions.Name.
	Name string

	// Namespace is the target namespace.
	// May be overridden by RenderOptions.Namespace.
	Namespace string

	// Version is the module version (semver).
	Version string

	// Labels from the module definition.
	// These are propagated to resources via TransformerContext.
	Labels map[string]string

	// Components lists the component names in the module.
	// Useful for understanding scope of render.
	Components []string
}

// MatchPlan describes the transformer-component matching results.
// Used for verbose output and debugging; not part of the core render contract.
type MatchPlan struct {
	// Matches maps component names to their matched transformers.
	// Key: component name
	// Value: list of transformers that matched (multiple allowed)
	Matches map[string][]TransformerMatch

	// Unmatched lists components with no matching transformers.
	// These will appear in RenderResult.Errors as UnmatchedComponentError.
	Unmatched []string
}

// TransformerMatch records a single transformer match.
type TransformerMatch struct {
	// TransformerFQN is the fully qualified transformer name.
	// Example: "opmodel.dev/transformers/kubernetes@v0#DeploymentTransformer"
	TransformerFQN string

	// Reason explains why this transformer matched.
	// Human-readable, for verbose output.
	// Example: "Matched: requiredLabels[workload-type=stateless], requiredResources[Container]"
	Reason string
}

// ResourceInfo interface implementation methods.
// These methods allow Resource to be used with the output package.

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
