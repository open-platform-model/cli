# Design: CLI Build Command

## Technical Approach

### Package Organization

```text
internal/
├── build/                  # Core render pipeline (implements Pipeline interface)
│   ├── pipeline.go             # Pipeline struct, Render() method
│   ├── loader.go               # Module and values loading
│   ├── provider.go             # Provider loading and transformer indexing
│   ├── matcher.go              # Component-transformer matching
│   ├── executor.go             # Parallel transformer execution
│   ├── context.go              # TransformerContext construction
│   ├── types.go                # Internal types (Worker, Job)
│   └── errors.go               # Error implementations (from render-pipeline-v1)
│
├── output/                 # Output formatting (CLI-specific)
│   ├── manifest.go             # YAML/JSON formatting
│   ├── split.go                # Split file output
│   └── verbose.go              # Verbose logging
│
└── cmd/mod/
    └── build.go                # CLI command (uses build + output)
```

### Key Design Decisions

#### 1. Pipeline Implementation

**Decision**: Implement `Pipeline` interface from render-pipeline-v1.

```go
// internal/build/pipeline.go
type pipeline struct {
    config   *config.OPMConfig
    loader   *Loader
    provider *ProviderLoader
    matcher  *Matcher
    executor *Executor
}

func NewPipeline(cfg *config.OPMConfig) Pipeline {
    return &pipeline{config: cfg, ...}
}

func (p *pipeline) Render(ctx context.Context, opts RenderOptions) (*RenderResult, error) {
    // Phase 1: Load module and values
    module, err := p.loader.Load(ctx, opts)
    if err != nil {
        return nil, err // Fatal error
    }
    
    // Phase 2: Load provider
    provider, err := p.provider.Load(ctx, opts.Provider)
    if err != nil {
        return nil, err // Fatal error
    }
    
    // Phase 3: Match components to transformers
    matchPlan := p.matcher.Match(module.Components, provider.Transformers)
    
    // Phase 4: Execute transformers (parallel)
    resources, errors := p.executor.Execute(ctx, matchPlan, module)
    
    // Phase 5: Build result
    return &RenderResult{
        Resources: resources,
        Module:    module.Metadata(),
        MatchPlan: matchPlan,
        Errors:    errors,
        Warnings:  collectWarnings(matchPlan, opts.Strict),
    }, nil
}
```

#### 2. Separation from Output Formatting

**Decision**: Output formatting is NOT part of Pipeline. CLI command handles formatting.

**Rationale**:
- Pipeline returns `RenderResult` with structured data
- CLI command uses `output` package to format as YAML/JSON
- Other consumers (apply, diff) don't need formatting

```go
// internal/cmd/mod/build.go
func runBuild(cmd *cobra.Command, args []string) error {
    pipeline := build.NewPipeline(cfg)
    
    result, err := pipeline.Render(ctx, buildOpts)
    if err != nil {
        return err
    }
    
    if result.HasErrors() {
        output.PrintErrors(result.Errors)
        return ErrRenderFailed
    }
    
    // CLI-specific: format and output
    return output.WriteManifests(result.Resources, outputOpts)
}
```

#### 3. Parallel Execution with Isolated Contexts

**Decision**: Each worker has its own `cue.Context` to prevent memory sharing issues.

```go
// internal/build/executor.go
type Executor struct {
    workers int
}

func (e *Executor) Execute(ctx context.Context, plan MatchPlan, module *LoadedModule) ([]*Resource, []error) {
    jobs := make(chan Job)
    results := make(chan Result)
    
    // Start workers with isolated CUE contexts
    for i := 0; i < e.workers; i++ {
        go func() {
            cueCtx := cuecontext.New() // Fresh context per worker
            for job := range jobs {
                result := e.executeJob(cueCtx, job, module)
                results <- result
            }
        }()
    }
    
    // ... distribute jobs, collect results
}
```

#### 4. Fail-on-End Error Aggregation

**Decision**: Execute all transformers, collect all errors, return in RenderResult.

**Rationale**: Users see all problems at once, not just the first one.

## Render Pipeline Phases

```text
+-------------------------------------------------------------------+
|                       Render Pipeline                              |
+-------------------------------------------------------------------+
|  Phase 1: Module Loading & Validation                        [Go]  |
|           - Load CUE module via cue/load                          |
|           - Unify values.cue + --values files                     |
|           - Construct ModuleRelease                               |
|           - Build base TransformerContext                          |
+-------------------------------------------------------------------+
|  Phase 2: Provider Loading                                   [Go]  |
|           - Load provider from config                             |
|           - Index transformers by FQN                             |
+-------------------------------------------------------------------+
|  Phase 3: Component Matching                                [CUE]  |
|           - For each component, evaluate #Matches against each    |
|             transformer                                            |
|           - Build MatchPlan grouping components by transformer    |
|           - Identify unmatched components                          |
+-------------------------------------------------------------------+
|  Phase 4: Parallel Transformer Execution                     [Go]  |
|           - Create jobs from MatchPlan                            |
|           - Workers execute #transform with isolated cue.Context  |
|           - Collect resources and errors                          |
+-------------------------------------------------------------------+
|  Phase 5: Result Construction                                [Go]  |
|           - Order resources by weight (for apply)                 |
|           - Aggregate errors                                       |
|           - Return RenderResult                                    |
+-------------------------------------------------------------------+
```

## Data Flow

```text
RenderOptions
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Loader                                                                   │
│                                                                          │
│   ModulePath ──▶ Load cue.mod/ ──▶ Evaluate module.cue                  │
│                                          │                               │
│   values.cue (required) ─────────────────┤                               │
│   --values files (optional) ─────────────┴──▶ Unify values              │
│                                                      │                   │
│   --name, --namespace ───────────────────────────────┴──▶ LoadedModule  │
└─────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ ProviderLoader                                                           │
│                                                                          │
│   --provider or config.defaultProvider ──▶ Load from config.providers   │
│                                                      │                   │
│   provider.transformers ─────────────────────────────┴──▶ LoadedProvider│
└─────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Matcher                                                                  │
│                                                                          │
│   For each component:                                                   │
│     For each transformer:                                               │
│       Evaluate #Matches in CUE                                          │
│       If match: record in MatchPlan.Matches                             │
│                                                                          │
│   Components with no matches ──▶ MatchPlan.Unmatched                    │
└─────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Executor (parallel)                                                      │
│                                                                          │
│   For each (transformer, components) in MatchPlan:                      │
│     For each component:                                                 │
│       Build TransformerContext                                          │
│       Unify transformer.#transform + component + context                │
│       Decode output to *unstructured.Unstructured                       │
│       Wrap as *Resource                                                 │
└─────────────────────────────────────────────────────────────────────────┘
    │
    ▼
RenderResult {
    Resources: []*Resource (ordered by weight)
    Module: ModuleMetadata
    MatchPlan: MatchPlan
    Errors: []error
    Warnings: []string
}
```

## TransformerContext Construction

```go
// internal/build/context.go
func buildTransformerContext(module *LoadedModule, component *Component, opts RenderOptions) map[string]any {
    return map[string]any{
        "name":               opts.Name,
        "namespace":          opts.Namespace,
        "#moduleMetadata":    module.Metadata,
        "#componentMetadata": component.Metadata,
        // labels computed in CUE from metadata
    }
}
```

The CUE-side `#TransformerContext` (defined in transformer.cue) derives labels from the injected metadata:

```cue
#TransformerContext: close({
    #moduleMetadata:    _
    #componentMetadata: _
    name:               string
    namespace:          string
    
    labels: {
        "app.kubernetes.io/managed-by": "open-platform-model"
        "app.kubernetes.io/name":       #componentMetadata.name
        "app.kubernetes.io/version":    #moduleMetadata.version
        "app.kubernetes.io/instance":   "\(name)-\(namespace)"
        // ... more computed labels
    }
})
```

## Transformer Matching Logic

A transformer matches a component if ALL conditions are met:

1. **Required Labels**: All labels in `transformer.requiredLabels` exist in component's effective labels with matching values
2. **Required Resources**: All FQNs in `transformer.requiredResources` exist in `component.#resources`
3. **Required Traits**: All FQNs in `transformer.requiredTraits` exist in `component.#traits`

Multiple transformers CAN match a single component (producing multiple resources). Zero matches is an error.

## CLI Command

```text
opm mod build [path] [flags]

Arguments:
  path    Path to module directory (default: current directory)

Flags:
  -f, --values strings      Additional values files (can be repeated)
  -n, --namespace string    Target namespace (required if not in module)
      --name string         Release name (default: module name)
      --provider string     Provider to use (default: from config)
  -o, --output string       Output format: yaml, json (default: yaml)
      --split               Write separate files per resource
      --out-dir string      Directory for split output (default: ./manifests)
      --strict              Error on unhandled traits
  -v, --verbose             Show matching decisions
      --verbose-json        Structured JSON verbose output
```

## File Changes

| File | Purpose |
|------|---------|
| `internal/build/pipeline.go` | Pipeline implementation |
| `internal/build/loader.go` | Module loading |
| `internal/build/provider.go` | Provider loading |
| `internal/build/matcher.go` | Component-transformer matching |
| `internal/build/executor.go` | Parallel execution |
| `internal/build/context.go` | TransformerContext construction |
| `internal/build/types.go` | Internal types (Worker, Job, etc.) |
| `internal/build/errors.go` | Error type implementations |
| `internal/output/manifest.go` | YAML/JSON formatting |
| `internal/output/split.go` | Split file output |
| `internal/output/verbose.go` | Verbose output |
| `internal/cmd/mod/build.go` | CLI command |
