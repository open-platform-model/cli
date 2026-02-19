// Package build provides the render pipeline interface and shared types.
// This package defines the contract between rendering operations and their consumers
// (build, apply, diff commands).
package build

import (
	"context"
	"errors"

	"github.com/opmodel/cli/internal/build/transform"
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

	// Name is the release name.
	// Optional. If empty, uses module.metadata.name as the release name.
	// This value becomes the module-release.opmodel.dev/name label on resources.
	Name string

	// Namespace overrides module.metadata.defaultNamespace.
	// Required if module doesn't define defaultNamespace.
	Namespace string

	// Provider selects which provider to use.
	// Optional. If empty, uses default provider from config.
	Provider string

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

	// Release contains release-level metadata (name, namespace, release UUID, labels).
	Release ReleaseMetadata

	// Module contains module-level metadata (canonical name, FQN, version, module UUID, labels).
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

// Resource is a type alias for transform.Resource.
// All methods are defined on transform.Resource and are accessible here transparently.
type Resource = transform.Resource

// MatchPlan is a type alias for transform.MatchPlan.
type MatchPlan = transform.MatchPlan

// TransformerMatch is a type alias for transform.TransformerMatch.
type TransformerMatch = transform.TransformerMatch
