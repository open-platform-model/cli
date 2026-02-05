# Data Model: CLI Build Implementation

## Overview

This document defines the **internal** data types for the build command implementation. Shared types (Pipeline, RenderResult, Resource, etc.) are defined in render-pipeline-v1.

## Internal Types

### Pipeline Implementation

```go
// internal/build/pipeline.go

// pipeline implements the Pipeline interface from render-pipeline-v1.
// This is the internal implementation, not exposed to consumers.
type pipeline struct {
    config         *config.OPMConfig
    module         *ModuleLoader
    releaseBuilder *ReleaseBuilder    // Makes #config concrete
    provider       *ProviderLoader
    matcher        *Matcher
    executor       *Executor
}

// NewPipeline creates a new Pipeline implementation.
func NewPipeline(cfg *config.OPMConfig) Pipeline {
    return &pipeline{
        config:         cfg,
        module:         NewModuleLoader(cfg.CueContext),
        releaseBuilder: NewReleaseBuilder(cfg.CueContext, cfg.Registry),
        provider:       NewProviderLoader(cfg),
        matcher:        NewMatcher(),
        executor:       NewExecutor(runtime.NumCPU()),
    }
}
```

### ModuleLoader

```go
// internal/build/module.go

// ModuleLoader handles module and values loading.
type ModuleLoader struct {
    cueCtx *cue.Context
}

// NewModuleLoader creates a new ModuleLoader instance.
func NewModuleLoader(ctx *cue.Context) *ModuleLoader

// LoadedModule is the result of loading a module.
// Note: Components are NOT extracted here - they come from ReleaseBuilder.
type LoadedModule struct {
    // Path to the module directory
    Path string
    
    // CUE value of the unified module (may have non-concrete #config refs)
    Value cue.Value
    
    // Extracted metadata
    Name             string
    Namespace        string
    Version          string
    DefaultNamespace string
    Labels           map[string]string
}

// LoadedComponent is a component with extracted metadata.
// Used by both ReleaseBuilder (extraction) and Executor (transformation).
type LoadedComponent struct {
    Name      string
    Labels    map[string]string    // Effective labels (from metadata.labels)
    Resources map[string]cue.Value // FQN -> resource value
    Traits    map[string]cue.Value // FQN -> trait value
    Value     cue.Value            // Full component value (concrete after release building)
}
```

### ReleaseBuilder

```go
// internal/build/release_builder.go

// ReleaseBuilder creates a concrete release from a #Module.
// It injects values into #config to resolve all configuration references.
type ReleaseBuilder struct {
    cueCtx   *cue.Context
    registry string
}

// NewReleaseBuilder creates a new ReleaseBuilder.
func NewReleaseBuilder(ctx *cue.Context, registry string) *ReleaseBuilder

// ReleaseOptions configures release building.
type ReleaseOptions struct {
    Name      string // Release name (defaults to module name)
    Namespace string // Required: target namespace
}

// BuiltRelease is the result of building a release.
type BuiltRelease struct {
    // Value is the concrete module value (with #config injected)
    Value cue.Value
    
    // Components are concrete components by name
    Components map[string]*LoadedComponent
    
    // Metadata contains release-level metadata
    Metadata ReleaseMetadata
}

// ReleaseMetadata contains release-level metadata.
type ReleaseMetadata struct {
    Name      string
    Namespace string
    Version   string
    FQN       string
    Labels    map[string]string
}

// Build creates a concrete release from a loaded module.
//
// Process:
//   1. Extract values from module.values
//   2. FillPath(#config, values) to make #config concrete
//   3. Extract components from #components
//   4. Validate components are concrete
//   5. Extract metadata
func (b *ReleaseBuilder) Build(moduleValue cue.Value, opts ReleaseOptions) (*BuiltRelease, error)
```

### Provider Loader

```go
// internal/build/provider.go

// ProviderLoader loads providers from config.
type ProviderLoader struct {
    config *config.OPMConfig
}

// LoadedProvider is the result of loading a provider.
type LoadedProvider struct {
    Name         string
    Version      string
    Transformers []*LoadedTransformer
    
    // Index for fast lookup
    byFQN map[string]*LoadedTransformer
}

// TransformerMap returns a map of transformers by FQN.
func (p *LoadedProvider) TransformerMap() map[string]*LoadedTransformer

// ToSummaries returns transformer summaries for error messages.
func (p *LoadedProvider) ToSummaries() []TransformerSummary

// LoadedTransformer is a transformer with extracted requirements.
type LoadedTransformer struct {
    Name              string
    FQN               string  // Fully qualified name
    RequiredLabels    map[string]string
    RequiredResources []string
    RequiredTraits    []string
    OptionalLabels    map[string]string
    OptionalResources []string
    OptionalTraits    []string
    Value             cue.Value  // Full transformer value for execution
}
```

### Matcher

```go
// internal/build/matcher.go

// Matcher evaluates transformer-component matching.
type Matcher struct{}

// MatchResult is the internal result of matching.
type MatchResult struct {
    // ByTransformer groups components by transformer FQN
    // Key: transformer FQN
    // Value: list of components that matched
    ByTransformer map[string][]*LoadedComponent
    
    // Unmatched components
    Unmatched []*LoadedComponent
    
    // Details for verbose output
    Details []MatchDetail
}

// MatchDetail records why a transformer did/didn't match a component.
type MatchDetail struct {
    ComponentName    string
    TransformerFQN   string
    Matched          bool
    
    // If not matched, why
    MissingLabels    []string
    MissingResources []string
    MissingTraits    []string
    
    // Traits not handled by this transformer
    UnhandledTraits  []string
}

// ToMatchPlan converts to the shared MatchPlan type.
func (r *MatchResult) ToMatchPlan() MatchPlan {
    matches := make(map[string][]TransformerMatch)
    for tfqn, components := range r.ByTransformer {
        for _, comp := range components {
            matches[comp.Name] = append(matches[comp.Name], TransformerMatch{
                TransformerFQN: tfqn,
                Reason:         buildMatchReason(r.Details, comp.Name, tfqn),
            })
        }
    }
    
    unmatched := make([]string, len(r.Unmatched))
    for i, c := range r.Unmatched {
        unmatched[i] = c.Name
    }
    
    return MatchPlan{
        Matches:   matches,
        Unmatched: unmatched,
    }
}
```

### Executor

```go
// internal/build/executor.go

// Executor runs transformers in parallel.
type Executor struct {
    workers int
}

// NewExecutor creates a new Executor with the specified worker count.
func NewExecutor(workers int) *Executor

// Job is a unit of work for a worker.
type Job struct {
    Transformer *LoadedTransformer
    Component   *LoadedComponent
    Release     *BuiltRelease  // For context construction
}

// JobResult is the result of executing a job.
type JobResult struct {
    Component   string
    Transformer string
    Resources   []*unstructured.Unstructured  // May produce multiple resources
    Error       error
}

// ExecuteResult is the combined result of all jobs.
type ExecuteResult struct {
    Resources []*Resource
    Errors    []error
}

// ExecuteWithTransformers runs transformations with explicit transformer map.
func (e *Executor) ExecuteWithTransformers(
    ctx context.Context,
    match *MatchResult,
    release *BuiltRelease,
    transformers map[string]*LoadedTransformer,
) *ExecuteResult
```

### TransformerContext

```go
// internal/build/context.go

// TransformerContext holds the context data passed to transformers.
// This matches the CUE #TransformerContext definition.
type TransformerContext struct {
    Name      string `json:"name"`
    Namespace string `json:"namespace"`
    
    ModuleMetadata    *TransformerModuleMetadata    `json:"#moduleMetadata"`
    ComponentMetadata *TransformerComponentMetadata `json:"#componentMetadata"`
}

// TransformerModuleMetadata contains module metadata for transformers.
type TransformerModuleMetadata struct {
    Name    string            `json:"name"`
    Version string            `json:"version"`
    Labels  map[string]string `json:"labels,omitempty"`
}

// TransformerComponentMetadata contains component metadata for transformers.
type TransformerComponentMetadata struct {
    Name      string            `json:"name"`
    Labels    map[string]string `json:"labels,omitempty"`
    Resources []string          `json:"resources,omitempty"`
    Traits    []string          `json:"traits,omitempty"`
}

// NewTransformerContext constructs the context for a transformer execution.
// Uses release metadata and component data.
func NewTransformerContext(release *BuiltRelease, component *LoadedComponent) *TransformerContext {
    return &TransformerContext{
        Name:      release.Metadata.Name,
        Namespace: release.Metadata.Namespace,
        ModuleMetadata: &TransformerModuleMetadata{
            Name:    release.Metadata.Name,
            Version: release.Metadata.Version,
            Labels:  release.Metadata.Labels,
        },
        ComponentMetadata: &TransformerComponentMetadata{
            Name:      component.Name,
            Labels:    component.Labels,
            Resources: extractFQNs(component.Resources),
            Traits:    extractFQNs(component.Traits),
        },
    }
}

// ToMap converts TransformerContext to a map for CUE encoding.
func (c *TransformerContext) ToMap() map[string]any
```

---

## Error Types

```go
// internal/build/errors.go

// RenderError is a base interface for render errors.
type RenderError interface {
    error
    Component() string
}

// UnmatchedComponentError indicates no transformer matched a component.
type UnmatchedComponentError struct {
    ComponentName string
    Available     []TransformerSummary
}

// UnhandledTraitError indicates a trait was not handled by any transformer.
type UnhandledTraitError struct {
    ComponentName string
    TraitFQN      string
    Strict        bool
}

// TransformError indicates transformer execution failed.
type TransformError struct {
    ComponentName  string
    TransformerFQN string
    Cause          error
}

// TransformerSummary provides guidance on transformer requirements.
type TransformerSummary struct {
    FQN               string
    RequiredLabels    map[string]string
    RequiredResources []string
    RequiredTraits    []string
}

// NamespaceRequiredError indicates namespace was not provided and module has no default.
type NamespaceRequiredError struct {
    ModuleName string
}

// ModuleValidationError indicates the module failed CUE validation.
type ModuleValidationError struct {
    Message       string
    ComponentName string
    FieldPath     string
    Cause         error
}

// ReleaseValidationError indicates the release failed validation.
// This typically happens when values are incomplete or non-concrete.
type ReleaseValidationError struct {
    Message string
    Cause   error
}

func (e *ReleaseValidationError) Error() string {
    if e.Cause != nil {
        return fmt.Sprintf("release validation failed: %s: %v", e.Message, e.Cause)
    }
    return fmt.Sprintf("release validation failed: %s", e.Message)
}

func (e *ReleaseValidationError) Unwrap() error {
    return e.Cause
}
```

---

## Output Types

These types are in `internal/output/` and are CLI-specific.

```go
// internal/output/manifest.go

// ManifestOptions controls manifest output formatting.
type ManifestOptions struct {
    Format string  // "yaml" or "json"
    Writer io.Writer
}

// WriteManifests writes resources to the writer.
func WriteManifests(resources []*Resource, opts ManifestOptions) error

// SplitOptions controls split file output.
type SplitOptions struct {
    OutDir string
    Format string
}

// WriteSplitManifests writes each resource to a separate file.
func WriteSplitManifests(resources []*Resource, opts SplitOptions) error

// VerboseOptions controls verbose output.
type VerboseOptions struct {
    JSON   bool
    Writer io.Writer
}

// WriteMatchPlan writes matching decisions for verbose output.
func WriteMatchPlan(plan MatchPlan, details []MatchDetail, opts VerboseOptions) error
```

---

## Resource Ordering

Resources are ordered by weight for sequential apply. Weights are defined in `pkg/weights/`.

```go
// pkg/weights/weights.go

// GetWeight returns the weight for a GVK.
// Lower weights are applied first.
func GetWeight(gvk schema.GroupVersionKind) int

// Default weights (same as deploy-v1):
// -100: CRDs
//    0: Namespaces
//    5: ClusterRole, ClusterRoleBinding
//   10: ServiceAccount, Role, RoleBinding
//   15: Secret, ConfigMap
//   20: StorageClass, PV, PVC
//   50: Service
//  100: Deployment, StatefulSet, DaemonSet
//  110: Job, CronJob
//  150: Ingress, NetworkPolicy
//  200: HPA
//  500: Webhooks
```

---

## Type Relationships

```text
pipeline (internal)
    │
    ├──▶ ModuleLoader ──▶ LoadedModule
    │                        │
    │                        └──▶ cue.Value (raw, may have #config refs)
    │
    ├──▶ ReleaseBuilder ──▶ BuiltRelease
    │                           │
    │                           ├──▶ cue.Value (concrete, #config filled)
    │                           ├──▶ map[string]*LoadedComponent (concrete)
    │                           └──▶ ReleaseMetadata
    │
    ├──▶ ProviderLoader ──▶ LoadedProvider
    │                           │
    │                           └──▶ []*LoadedTransformer
    │
    ├──▶ Matcher ──▶ MatchResult
    │                    │
    │                    ├──▶ ByTransformer map[string][]*LoadedComponent
    │                    ├──▶ Unmatched []*LoadedComponent
    │                    └──▶ Details []MatchDetail
    │
    └──▶ Executor ──▶ ExecuteResult
             │
             └──▶ workers ──▶ Job ──▶ JobResult
                    │                      │
                    ├─ Transformer *LoadedTransformer
                    ├─ Component *LoadedComponent
                    └─ Release *BuiltRelease
                                 │
                                 └──▶ NewTransformerContext(release, component)
```

---

## Data Flow Through Types

```text
RenderOptions
       │
       ▼
   ModuleLoader.Load()
       │
       ▼
   LoadedModule { Value: cue.Value (with #config refs), metadata }
       │
       ▼
   ReleaseBuilder.Build()
       │
       ├─ values := Value.LookupPath("values")
       ├─ concreteModule := Value.FillPath("#config", values)
       ├─ components := extractComponents(concreteModule)
       └─ validate each component is concrete
       │
       ▼
   BuiltRelease { Value: concrete, Components: map[string]*LoadedComponent }
       │
       ▼
   Matcher.Match(release.Components, provider.Transformers)
       │
       ▼
   MatchResult { ByTransformer: map[fqn][]*LoadedComponent }
       │
       ▼
   Executor.ExecuteWithTransformers(matchResult, release, transformerMap)
       │
       ├─ For each (transformer, component) pair:
       │    ├─ job := Job{Transformer, Component, Release}
       │    ├─ ctx := NewTransformerContext(release, component)
       │    ├─ #transform.FillPath("#component", component.Value)
       │    ├─ #transform.FillPath("#context.*", ctx fields)
       │    └─ decode output to []*unstructured.Unstructured
       │
       ▼
   ExecuteResult { Resources: []*Resource, Errors: []error }
       │
       ▼
   RenderResult { Resources (sorted), Module, MatchPlan, Errors, Warnings }
```
