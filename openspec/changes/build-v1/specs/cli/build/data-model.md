# Data Model: CLI Render System

## Core Entities

### Pipeline

The central orchestrator.

```go
type Pipeline struct {
    Loader    *Loader
    Provider  *Provider
    Renderer  *Renderer
    Logger    Logger
}
```

### Provider

Represents a loaded OPM provider (e.g., `kubernetes`).

```go
type Provider struct {
    Name         string
    Version      string
    Transformers map[string]*Transformer // Registry of available transformers
}
```

### Transformer

A single transformation unit.

```go
type Transformer struct {
    Name              string
    ID                string // Full CUE path/ID
    RequiredLabels    map[string]string
    RequiredResources []string // Resource types (e.g. "Container")
    RequiredTraits    []string // Trait names (e.g. "Expose")
    
    // The raw CUE value of the transformer (for unification)
    Source cue.Value 
}
```

### Match

Represents the decision to apply a specific transformer to a specific component.

```go
type MatchGroup struct {
    Transformer *Transformer
    Components  []*Component
}

type MatchedMap map[string]*MatchGroup
```

### Component

A wrapper around a component's data.

```go
type Component struct {
    Name      string
    Labels    map[string]string // Effective labels
    Resources []Resource
    Traits    []Trait
    
    Source    cue.Value
}
```

### TransformerContext

Data injected into the transformation.

**Note**: The actual implementation uses CUE's hidden field pattern for metadata injection, giving transformers access to the full module and component metadata structs rather than pre-selected fields. This design is more flexible and idiomatic to CUE.

**CUE Definition**: openspec/changes/build/specs/cli/build/definitions/transformer.cue

**Go Equivalent** (for CLI runtime context construction):

```go
type TransformerContext struct {
    Name      string            `json:"name"`      // Release name
    Namespace string            `json:"namespace"` // Target namespace
    
    // Hidden fields (injected via CUE, not directly accessible in Go)
    ModuleMetadata    map[string]any // Full module metadata struct
    ComponentMetadata map[string]any // Full component metadata struct
    
    // Computed fields (derived in CUE from metadata)
    Labels map[string]string `json:"labels"` // Merged labels
}
```

## Parallel Execution Model

### Worker

A self-contained execution unit.

```go
type Worker struct {
    ID      int
    Context *cue.Context // Isolated context
}

type Job struct {
    TransformerID string
    ComponentData []byte // Marshaled component
    ContextData   []byte // Marshaled context
}

type Result struct {
    Resource *unstructured.Unstructured
    Error    error
}
```

## Error Types

```go
// UnmatchedComponentError indicates no transformer matched a component
type UnmatchedComponentError struct {
    ComponentName       string
    AvailableTransformers []TransformerSummary
}

// MultipleMatchError indicates multiple transformers matched exactly
type MultipleMatchError struct {
    ComponentName  string
    MatchedTransformers []string
}

// UnhandledTraitError indicates a trait was not handled by any transformer
type UnhandledTraitError struct {
    ComponentName string
    TraitName     string
    StrictMode    bool // If true, this is an error; if false, warning
}

// ValuesConflictError wraps CUE unification errors for values files
type ValuesConflictError struct {
    Files   []string
    CUEError error
}

// TransformerSummary provides guidance on what a transformer requires
type TransformerSummary struct {
    Name              string
    RequiredLabels    map[string]string
    RequiredResources []string
    RequiredTraits    []string
}
```

## Output Types

```go
// RenderResult contains the complete output of a render operation
type RenderResult struct {
    Resources []Resource
    Warnings  []string
    Errors    []error
}

// Resource represents a single rendered Kubernetes resource
type Resource struct {
    APIVersion string
    Kind       string
    Name       string
    Namespace  string
    Data       map[string]any
    
    // Metadata for output
    SourceComponent  string
    SourceTransformer string
}

// OutputOptions controls how results are written
type OutputOptions struct {
    Format   string // "yaml" or "json"
    Split    bool   // Write separate files
    OutDir   string // Directory for split output
    Verbose  bool   // Enable verbose logging
    VerboseJSON bool // Structured JSON verbose output
}
```
