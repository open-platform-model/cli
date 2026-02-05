# Design: Render Pipeline Interface

## Technical Approach

### Package Organization

The render pipeline is split into distinct packages with clear responsibilities:

```text
internal/
├── build/              # Core render pipeline (implements Pipeline interface)
│   ├── pipeline.go         # Pipeline struct and orchestration
│   ├── loader.go           # Module and values loading
│   ├── provider.go         # Provider loading and transformer indexing
│   ├── matcher.go          # Component-transformer matching
│   ├── executor.go         # Parallel transformer execution
│   ├── context.go          # TransformerContext construction
│   ├── types.go            # RenderResult, Resource (from this spec)
│   └── errors.go           # Shared error types
│
├── output/             # Output formatting (CLI-specific, NOT part of interface)
│   ├── manifest.go         # YAML/JSON manifest formatting
│   ├── split.go            # Split file output
│   └── verbose.go          # Verbose logging helpers
│
└── kubernetes/         # K8s operations (deploy-v1)
    ├── apply.go            # Server-side apply using RenderResult
    ├── diff.go             # Diff using RenderResult
    └── ...
```

### Key Design Decisions

#### 1. RenderResult as the Contract

**Decision**: All consumers receive a `RenderResult` struct, not raw YAML/JSON.

**Rationale**:

- Enables type-safe access to rendered resources
- Consumers can add their own labels, filter resources, etc.
- Supports both CLI output and Kubernetes apply without transformation
- MatchPlan enables debugging without exposing matcher internals

#### 2. Pipeline Interface

**Decision**: Define a `Pipeline` interface, not just the struct.

**Rationale**:

- Enables testing with mock pipelines
- Future: Bundle rendering can implement same interface with different internals
- Consumers depend on interface, not implementation

#### 3. Errors in RenderResult

**Decision**: RenderResult contains `Errors []error` rather than returning error from Render().

**Rationale**:

- Supports fail-on-end pattern (render all components, aggregate errors)
- Consumers can still process partial results (useful for diff)
- Fatal errors (can't load module) still return error from Render()

#### 4. Resources are Unstructured

**Decision**: Use `*unstructured.Unstructured` for rendered resources.

**Rationale**:

- Provider-agnostic: works for Kubernetes and future providers
- No need to know resource schemas at compile time
- Kubernetes client-go already uses this pattern

## Data Flow

```text
┌─────────────────────────────────────────────────────────────────────────┐
│                Consumer (build cmd, apply cmd, etc.)                    │
│                                                                         │
│   BuildOptions {                                                        │
│     ModulePath: "./my-module"                                           │
│     Values: ["values.cue", "prod.cue"]                                  │
│     Namespace: "production"                                             │
│     Provider: "kubernetes"                                              │
│   }                                                                     │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                         Pipeline.Render(ctx, opts)                      │
│                                                                         │
│   ┌─────────┐    ┌──────────┐    ┌─────────┐    ┌──────────┐            │
│   │ Loader  │──▶│ Provider │──▶│ Matcher │──▶│ Executor │            │
│   └─────────┘    └──────────┘    └─────────┘    └──────────┘            │
│                                                                         │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                                    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│                              RenderResult                               │
│                                                                         │
│   Resources: []*Resource        // Ordered, ready for apply/output      │
│   Module: ModuleMetadata        // Source module info                   │
│   MatchPlan: MatchPlan          // Debugging: what matched what         │
│   Errors: []error               // Aggregated render errors             │
│   Warnings: []string            // Non-fatal warnings                   │
└─────────────────────────────────────────────────────────────────────────┘
                                    │
                    ┌───────────────┼───────────────┐
                    ▼               ▼               ▼
              ┌─────────┐    ┌───────────┐    ┌──────────┐
              │ Output  │    │ K8s Apply │    │ K8s Diff │
              │ (YAML)  │    │           │    │          │
              └─────────┘    └───────────┘    └──────────┘
```

## Interface Definitions

### Pipeline Interface

```go
// Pipeline defines the contract for render pipelines.
// Implemented by internal/build.Pipeline.
type Pipeline interface {
    // Render executes the pipeline and returns results.
    // Fatal errors (module not found, provider missing) return error.
    // Render errors (unmatched components, transform failures) are in RenderResult.Errors.
    Render(ctx context.Context, opts RenderOptions) (*RenderResult, error)
}
```

### RenderOptions

```go
// RenderOptions configures a render operation.
type RenderOptions struct {
    // ModulePath is the path to the module directory.
    ModulePath string

    // Values are additional values files to unify (in order).
    Values []string

    // Name overrides module.metadata.name for the release.
    Name string

    // Namespace overrides module.metadata.defaultNamespace.
    Namespace string

    // Provider selects which provider to use (default: from config).
    Provider string

    // Strict enables strict trait handling (unhandled traits are errors).
    Strict bool
}
```

### RenderResult

```go
// RenderResult is the output of a render operation.
// This is the contract between rendering and consumers.
type RenderResult struct {
    // Resources are the rendered platform resources, ordered for apply.
    Resources []*Resource

    // Module contains metadata about the source module.
    Module ModuleMetadata

    // MatchPlan describes which transformers matched which components.
    // Used for verbose output and debugging.
    MatchPlan MatchPlan

    // Errors contains aggregated render errors (fail-on-end pattern).
    // Empty if all components rendered successfully.
    Errors []error

    // Warnings contains non-fatal warnings.
    Warnings []string
}

// HasErrors returns true if there are render errors.
func (r *RenderResult) HasErrors() bool {
    return len(r.Errors) > 0
}
```

### Resource

```go
// Resource represents a single rendered platform resource.
type Resource struct {
    // Object is the rendered resource as unstructured data.
    Object *unstructured.Unstructured

    // Component is the name of the source component.
    Component string

    // Transformer is the FQN of the transformer that produced this resource.
    Transformer string
}

// GVK returns the GroupVersionKind of the resource.
func (r *Resource) GVK() schema.GroupVersionKind {
    return r.Object.GroupVersionKind()
}

// Name returns the resource name.
func (r *Resource) Name() string {
    return r.Object.GetName()
}

// Namespace returns the resource namespace.
func (r *Resource) Namespace() string {
    return r.Object.GetNamespace()
}
```

### ModuleMetadata

```go
// ModuleMetadata contains information about the source module.
type ModuleMetadata struct {
    // Name is the module name (may be overridden by RenderOptions.Name).
    Name string

    // Namespace is the target namespace (may be overridden by RenderOptions.Namespace).
    Namespace string

    // Version is the module version.
    Version string

    // Labels from the module definition.
    Labels map[string]string

    // Components lists the component names in the module.
    Components []string
}
```

### MatchPlan

```go
// MatchPlan describes the transformer-component matching results.
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
```

## Error Types

```go
// RenderError is a base interface for render errors.
type RenderError interface {
    error
    Component() string  // Which component failed
}

// UnmatchedComponentError indicates no transformer matched a component.
type UnmatchedComponentError struct {
    ComponentName string
    Available     []TransformerSummary
}

// UnhandledTraitError indicates a trait was not handled.
type UnhandledTraitError struct {
    ComponentName string
    TraitFQN      string
    Strict        bool  // If true, this is fatal
}

// TransformError indicates transformer execution failed.
type TransformError struct {
    ComponentName   string
    TransformerFQN  string
    Cause           error
}

// TransformerSummary provides guidance on transformer requirements.
type TransformerSummary struct {
    FQN               string
    RequiredLabels    map[string]string
    RequiredResources []string
    RequiredTraits    []string
}
```

## Ownership Boundaries

| Owner | Responsibility |
|-------|----------------|
| render-pipeline-v1 (this spec) | Interface definitions, shared types |
| build-v1 | Pipeline implementation, CLI command |
| deploy-v1 | Consuming RenderResult, K8s operations |
| platform-adapter-spec | Transformer definitions, Provider structure |

## File Changes

This is an interface specification. Implementation files are in build-v1.

Shared types will be in: `internal/build/types.go`
