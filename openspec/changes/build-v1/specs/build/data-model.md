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
    config   *config.OPMConfig
    module   *ModuleLoader
    provider *ProviderLoader
    matcher  *Matcher
    executor *Executor
    logger   *output.Logger
}

// NewPipeline creates a new Pipeline implementation.
func NewPipeline(cfg *config.OPMConfig) Pipeline {
    return &pipeline{
        config:   cfg,
        module:   NewModuleLoader(),
        provider: NewProviderLoader(cfg),
        matcher:  NewMatcher(),
        executor: NewExecutor(runtime.NumCPU()),
    }
}
```

### ModuleLoader

```go
// internal/build/module.go

// ModuleLoader handles module and values loading.
type ModuleLoader struct{}

// LoadedModule is the result of loading a module.
type LoadedModule struct {
    // Path to the module directory
    Path string
    
    // CUE value of the unified module
    Value cue.Value
    
    // Extracted metadata
    Name             string
    Namespace        string
    Version          string
    DefaultNamespace string
    Labels           map[string]string
    
    // Components extracted from module
    Components []*LoadedComponent
}

// LoadedComponent is a component with extracted metadata.
type LoadedComponent struct {
    Name      string
    Labels    map[string]string  // Effective labels (merged from resources/traits)
    Resources map[string]cue.Value  // FQN -> resource value
    Traits    map[string]cue.Value  // FQN -> trait value
    Value     cue.Value             // Full component value
}

// Metadata returns ModuleMetadata for RenderResult.
func (m *LoadedModule) Metadata() ModuleMetadata {
    names := make([]string, len(m.Components))
    for i, c := range m.Components {
        names[i] = c.Name
    }
    return ModuleMetadata{
        Name:       m.Name,
        Namespace:  m.Namespace,
        Version:    m.Version,
        Labels:     m.Labels,
        Components: names,
    }
}
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

// MatchResult is the internal result of matching (converted to MatchPlan for RenderResult).
type MatchResult struct {
    // Matches groups components by transformer
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
    ComponentName  string
    TransformerFQN string
    Matched        bool
    
    // If not matched, why
    MissingLabels    []string
    MissingResources []string
    MissingTraits    []string
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

// Job is a unit of work for a worker.
type Job struct {
    Transformer *LoadedTransformer
    Component   *LoadedComponent
    Context     map[string]any  // TransformerContext data
}

// JobResult is the result of executing a job.
type JobResult struct {
    Component   string
    Transformer string
    Resource    *unstructured.Unstructured  // nil if error
    Error       error
}

// ExecuteResult is the combined result of all jobs.
type ExecuteResult struct {
    Resources []*Resource
    Errors    []error
}
```

### Worker

```go
// internal/build/executor.go

// worker executes jobs with an isolated CUE context.
type worker struct {
    id     int
    ctx    *cue.Context  // Isolated context
    jobs   <-chan Job
    results chan<- JobResult
}

func (w *worker) run() {
    for job := range w.jobs {
        result := w.executeJob(job)
        w.results <- result
    }
}

func (w *worker) executeJob(job Job) JobResult {
    // Build CUE value for transformation
    // transformer.#transform & {#component: component, #context: context}
    transformValue := job.Transformer.Value.LookupPath(cue.ParsePath("#transform"))
    
    // Unify with component and context
    unified := transformValue.Unify(w.ctx.Encode(map[string]any{
        "#component": job.Component.Value,
        "#context":   job.Context,
    }))
    
    if unified.Err() != nil {
        return JobResult{
            Component:   job.Component.Name,
            Transformer: job.Transformer.FQN,
            Error:       &TransformError{
                ComponentName:  job.Component.Name,
                TransformerFQN: job.Transformer.FQN,
                Cause:          unified.Err(),
            },
        }
    }
    
    // Extract output
    outputValue := unified.LookupPath(cue.ParsePath("output"))
    
    var obj map[string]any
    if err := outputValue.Decode(&obj); err != nil {
        return JobResult{
            Component:   job.Component.Name,
            Transformer: job.Transformer.FQN,
            Error:       &TransformError{
                ComponentName:  job.Component.Name,
                TransformerFQN: job.Transformer.FQN,
                Cause:          err,
            },
        }
    }
    
    return JobResult{
        Component:   job.Component.Name,
        Transformer: job.Transformer.FQN,
        Resource:    &unstructured.Unstructured{Object: obj},
    }
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
    │                    │
    │                    └──▶ []*LoadedComponent
    │
    ├──▶ ProviderLoader ──▶ LoadedProvider
    │                           │
    │                           └──▶ []*LoadedTransformer
    │
    ├──▶ Matcher ──▶ MatchResult
    │                    │
    │                    └──▶ []MatchDetail
    │
    └──▶ Executor ──▶ ExecuteResult
             │
             └──▶ workers ──▶ Job ──▶ JobResult
                                          │
                                          └──▶ *Resource (shared type)
```
