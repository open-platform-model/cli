# Data Model: Render Pipeline Interface

## Overview

This document defines the shared data types for the render pipeline interface. These types form the contract between the rendering implementation (build-v1) and consumers (deploy-v1, future bundle support).

## Core Types

### Pipeline

The main interface for render operations.

```go
// Pipeline defines the contract for render pipelines.
type Pipeline interface {
    // Render executes the pipeline and returns results.
    //
    // Fatal errors (module not found, provider missing) return error.
    // Render errors (unmatched components, transform failures) are in RenderResult.Errors.
    //
    // The context is used for cancellation. Long-running operations should
    // check ctx.Done() and return ctx.Err() if cancelled.
    Render(ctx context.Context, opts RenderOptions) (*RenderResult, error)
}
```

### RenderOptions

Configuration for a render operation.

```go
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
```

### RenderResult

The output of a render operation.

```go
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
```

### Resource

A single rendered platform resource.

```go
import (
    "k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
    "k8s.io/apimachinery/pkg/runtime/schema"
)

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
```

### ModuleMetadata

Information about the source module.

```go
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
```

### MatchPlan

Describes transformer-component matching results.

```go
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
```

---

## Error Types

### RenderError Interface

```go
// RenderError is a base interface for render errors.
// All render-specific errors implement this interface.
type RenderError interface {
    error
    
    // Component returns the component name where the error occurred.
    Component() string
}
```

### UnmatchedComponentError

```go
// UnmatchedComponentError indicates no transformer matched a component.
type UnmatchedComponentError struct {
    // ComponentName is the name of the unmatched component.
    ComponentName string
    
    // Available lists transformers and their requirements.
    // Helps users understand what's needed to match.
    Available []TransformerSummary
}

func (e *UnmatchedComponentError) Error() string {
    return fmt.Sprintf("component %q: no matching transformer", e.ComponentName)
}

func (e *UnmatchedComponentError) Component() string {
    return e.ComponentName
}
```

### UnhandledTraitError

```go
// UnhandledTraitError indicates a trait was not handled by any transformer.
type UnhandledTraitError struct {
    // ComponentName is the component with the unhandled trait.
    ComponentName string
    
    // TraitFQN is the fully qualified trait name.
    TraitFQN string
    
    // Strict indicates if this was treated as an error (strict mode)
    // or warning (normal mode).
    Strict bool
}

func (e *UnhandledTraitError) Error() string {
    return fmt.Sprintf("component %q: unhandled trait %q", e.ComponentName, e.TraitFQN)
}

func (e *UnhandledTraitError) Component() string {
    return e.ComponentName
}
```

### TransformError

```go
// TransformError indicates transformer execution failed.
type TransformError struct {
    // ComponentName is the component being transformed.
    ComponentName string
    
    // TransformerFQN is the transformer that failed.
    TransformerFQN string
    
    // Cause is the underlying error.
    Cause error
}

func (e *TransformError) Error() string {
    return fmt.Sprintf("component %q, transformer %q: %v", 
        e.ComponentName, e.TransformerFQN, e.Cause)
}

func (e *TransformError) Component() string {
    return e.ComponentName
}

func (e *TransformError) Unwrap() error {
    return e.Cause
}
```

### TransformerSummary

```go
// TransformerSummary provides guidance on transformer requirements.
// Used in error messages to help users understand matching.
type TransformerSummary struct {
    // FQN is the fully qualified transformer name.
    FQN string
    
    // RequiredLabels that components must have.
    RequiredLabels map[string]string
    
    // RequiredResources (FQNs) that components must have.
    RequiredResources []string
    
    // RequiredTraits (FQNs) that components must have.
    RequiredTraits []string
}
```

---

## Type Relationships

```text
Pipeline
    │
    └──▶ Render(ctx, RenderOptions) ──▶ (*RenderResult, error)
                                              │
                                              ├──▶ Resources []*Resource
                                              │         │
                                              │         └──▶ Object (*unstructured.Unstructured)
                                              │         └──▶ Component (string)
                                              │         └──▶ Transformer (string)
                                              │
                                              ├──▶ Module (ModuleMetadata)
                                              │
                                              ├──▶ MatchPlan
                                              │         │
                                              │         ├──▶ Matches (map[string][]TransformerMatch)
                                              │         └──▶ Unmatched ([]string)
                                              │
                                              ├──▶ Errors []error
                                              │         │
                                              │         └──▶ (UnmatchedComponentError | UnhandledTraitError | TransformError)
                                              │
                                              └──▶ Warnings []string
```

---

## Usage Examples

### Build Command

```go
func runBuild(opts BuildCmdOptions) error {
    pipeline := build.NewPipeline(config)
    
    result, err := pipeline.Render(ctx, build.RenderOptions{
        ModulePath: opts.ModulePath,
        Values:     opts.Values,
        Namespace:  opts.Namespace,
        Strict:     opts.Strict,
    })
    if err != nil {
        return err // Fatal error
    }
    
    if result.HasErrors() {
        output.PrintErrors(result.Errors)
        return errors.New("render failed")
    }
    
    return output.WriteManifests(result.Resources, opts.Format)
}
```

### Apply Command

```go
func runApply(opts ApplyCmdOptions) error {
    pipeline := build.NewPipeline(config)
    
    result, err := pipeline.Render(ctx, build.RenderOptions{
        ModulePath: opts.ModulePath,
        Values:     opts.Values,
        Namespace:  opts.Namespace,
    })
    if err != nil {
        return err
    }
    
    if result.HasErrors() {
        return aggregateError(result.Errors)
    }
    
    // Pass resources to Kubernetes client
    return kubernetes.Apply(ctx, result.Resources, kubernetes.ApplyOptions{
        DryRun: opts.DryRun,
        Wait:   opts.Wait,
    })
}
```

### Diff Command

```go
func runDiff(opts DiffCmdOptions) error {
    pipeline := build.NewPipeline(config)
    
    result, err := pipeline.Render(ctx, build.RenderOptions{
        ModulePath: opts.ModulePath,
        Values:     opts.Values,
        Namespace:  opts.Namespace,
    })
    if err != nil {
        return err
    }
    
    // Can still diff partial results
    for _, warning := range result.Warnings {
        output.Warn(warning)
    }
    
    return kubernetes.Diff(ctx, result.Resources)
}
```
