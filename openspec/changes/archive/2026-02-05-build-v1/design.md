# Design: CLI Build Command

## Technical Approach

### Package Organization

```text
internal/
├── build/                  # Core render pipeline (implements Pipeline interface)
│   ├── pipeline.go             # Pipeline struct, Render() method
│   ├── module.go               # Module and values loading
│   ├── release_builder.go      # Release building (concrete components from #config)
│   ├── provider.go             # Provider loading and transformer indexing
│   ├── matcher.go              # Component-transformer matching
│   ├── executor.go             # Parallel transformer execution
│   ├── context.go              # TransformerContext construction
│   ├── types.go                # Shared types (Pipeline, RenderResult, Resource)
│   └── errors.go               # Error implementations
│
├── output/                 # Output formatting (CLI-specific)
│   ├── manifest.go             # YAML/JSON formatting
│   ├── split.go                # Split file output
│   └── verbose.go              # Verbose logging
│
└── cmd/
    └── mod_build.go                # CLI command (uses build + output)
```

### Key Design Decisions

#### 1. Pipeline Implementation

**Decision**: Implement `Pipeline` interface with 6-phase architecture.

```go
// internal/build/pipeline.go
type pipeline struct {
    config         *config.OPMConfig
    module         *ModuleLoader
    releaseBuilder *ReleaseBuilder    // Makes #config concrete
    provider       *ProviderLoader
    matcher        *Matcher
    executor       *Executor
}

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

func (p *pipeline) Render(ctx context.Context, opts RenderOptions) (*RenderResult, error) {
    // Phase 1: Load module and values (raw, may have #config references)
    module, err := p.module.Load(ctx, opts)
    if err != nil {
        return nil, err // Fatal error
    }
    
    // Phase 2: Build release (makes #config concrete, extracts components)
    release, err := p.releaseBuilder.Build(module.Value, ReleaseOptions{
        Name:      resolveName(opts.Name, module.Name),
        Namespace: module.Namespace,
    })
    if err != nil {
        return nil, err // Fatal error (likely incomplete values)
    }
    
    // Phase 3: Load provider
    provider, err := p.provider.Load(ctx, opts.Provider)
    if err != nil {
        return nil, err // Fatal error
    }
    
    // Phase 4: Match components to transformers
    matchResult := p.matcher.Match(release.Components, provider.Transformers)
    
    // Phase 5: Execute transformers (parallel)
    execResult := p.executor.ExecuteWithTransformers(ctx, matchResult, release, provider.TransformerMap())
    
    // Phase 6: Build result
    return &RenderResult{
        Resources: sortByWeight(execResult.Resources),
        Module:    release.ToModuleMetadata(),
        MatchPlan: matchResult.ToMatchPlan(),
        Errors:    append(unmatchedErrors(matchResult), execResult.Errors...),
        Warnings:  collectWarnings(matchResult, opts.Strict),
    }, nil
}
```

#### 2. Release Building with #config Pattern

**Decision**: Use `ReleaseBuilder` to inject concrete values into `#config` before component extraction.

**Rationale**:

- Modules define `#config` as schema with constraints (no defaults)
- Modules define `values: #config` with concrete defaults
- Components reference `#config` (e.g., `image: #config.web.image`)
- At build time, `FillPath(#config, values)` makes `#config` concrete
- Components then resolve to concrete values

```go
// internal/build/release_builder.go
func (b *ReleaseBuilder) Build(moduleValue cue.Value, opts ReleaseOptions) (*BuiltRelease, error) {
    // Step 1: Extract values from module.values
    values := moduleValue.LookupPath(cue.ParsePath("values"))
    
    // Step 2: Inject values into #config (KEY STEP)
    concreteModule := moduleValue.FillPath(cue.ParsePath("#config"), values)
    
    // Step 3: Extract concrete components from #components
    components := b.extractComponents(concreteModule)
    
    // Step 4: Validate components are fully concrete
    for name, comp := range components {
        if err := comp.Value.Validate(cue.Concrete(true)); err != nil {
            return nil, &ReleaseValidationError{...}
        }
    }
    
    // Step 5: Extract metadata and return
    return &BuiltRelease{Value: concreteModule, Components: components, Metadata: ...}, nil
}
```

#### 3. Separation from Output Formatting

**Decision**: Output formatting is NOT part of Pipeline. CLI command handles formatting.

**Rationale**:

- Pipeline returns `RenderResult` with structured data
- CLI command uses `output` package to format as YAML/JSON
- Other consumers (apply, diff) don't need formatting

```go
// internal/cmd/mod_build.go
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

#### 4. Parallel Execution with Isolated Contexts

**Decision**: Workers use the transformer's existing CUE context.

```go
// internal/build/executor.go
func (e *Executor) executeJob(job Job) JobResult {
    // Get context from transformer value (already loaded)
    cueCtx := job.Transformer.Value.Context()
    
    // Get #transform and inject component + context via FillPath
    transformValue := job.Transformer.Value.LookupPath(cue.ParsePath("#transform"))
    unified := transformValue.FillPath(cue.ParsePath("#component"), job.Component.Value)
    unified = unified.FillPath(cue.ParsePath("#context.name"), cueCtx.Encode(ctx.Name))
    // ... fill other context fields
    
    // Extract output
    outputValue := unified.LookupPath(cue.ParsePath("output"))
    // ... decode to unstructured
}
```

#### 5. Fail-on-End Error Aggregation

**Decision**: Execute all transformers, collect all errors, return in RenderResult.

**Rationale**: Users see all problems at once, not just the first one.

## Render Pipeline Phases

```text
+--------------------------------------------------------------------+
|                       Render Pipeline                              |
+--------------------------------------------------------------------+
|  Phase 1: Module Loading                                     [Go]  |
|           - Load CUE module via cue/load                           |
|           - Verify values.cue exists (required)                    |
|           - Unify values.cue + --values files                      |
|           - Extract metadata (name, namespace, version)            |
|           - Apply --namespace and --name overrides                 |
+--------------------------------------------------------------------+
|  Phase 2: Release Building                                   [Go]  |
|           - Extract values from module.values                      |
|           - FillPath(#config, values) to make #config concrete     |
|           - Extract concrete components from #components           |
|           - Validate all components are fully concrete             |
|           - Extract release metadata                               |
+--------------------------------------------------------------------+
|  Phase 3: Provider Loading                                   [Go]  |
|           - Load provider from config                              |
|           - Index transformers by FQN                              |
+--------------------------------------------------------------------+
|  Phase 4: Component Matching                                 [Go]  |
|           - For each component, evaluate against transformers      |
|           - Check required labels, resources, traits               |
|           - Build MatchResult grouping components by transformer   |
|           - Identify unmatched components                          |
+--------------------------------------------------------------------+
|  Phase 5: Parallel Transformer Execution                     [Go]  |
|           - Create jobs from MatchResult                           |
|           - Workers execute #transform via FillPath injection      |
|           - FillPath(#component, componentValue)                   |
|           - FillPath(#context.*, contextFields)                    |
|           - Collect resources and errors                           |
+--------------------------------------------------------------------+
|  Phase 6: Result Construction                                [Go]  |
|           - Order resources by weight (for apply)                  |
|           - Aggregate errors from matching and execution           |
|           - Collect warnings (unhandled traits in non-strict)      |
|           - Return RenderResult                                    |
+--------------------------------------------------------------------+
```

## Data Flow

```text
RenderOptions
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ ModuleLoader                                                            │
│                                                                         │
│   ModulePath ──▶ Load cue.mod/ ──▶ Evaluate module.cue                │
│                                          │                              │
│   values.cue (required) ─────────────────┤                              │
│   --values files (optional) ─────────────┴──▶ Unify values             │
│                                                      │                  │
│   --name, --namespace ───────────────────────────────┴──▶ LoadedModule │
│                                          (raw Value with #config refs)  │
└─────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ ReleaseBuilder                                                          │
│                                                                         │
│   module.values ──────────────▶ Extract values                         │
│                                       │                                 │
│   FillPath(#config, values) ──────────┴──▶ Concrete module             │
│                                                  │                      │
│   #components ───────────────────────────────────┴──▶ BuiltRelease     │
│                                          (concrete Components map)      │
└─────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌───────────────────────────────────────────────────────────────────────────┐
│ ProviderLoader                                                            │
│                                                                           │
│   --provider or config.defaultProvider ──▶ Load from config.providers    │
│                                                      │                    │
│   provider.transformers ─────────────────────────────┴──▶ LoadedProvider │
└───────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Matcher                                                                 │
│                                                                         │
│   For each component in release.Components:                             │
│     For each transformer:                                               │
│       Check requiredLabels, requiredResources, requiredTraits           │
│       If all match: record in MatchResult.ByTransformer                 │
│                                                                         │
│   Components with no matches ──▶ MatchResult.Unmatched                 │
└─────────────────────────────────────────────────────────────────────────┘
    │
    ▼
┌─────────────────────────────────────────────────────────────────────────┐
│ Executor (parallel)                                                     │
│                                                                         │
│   For each (transformer, components) in MatchResult:                    │
│     For each component:                                                 │
│       Build Job{Transformer, Component, Release}                        │
│       FillPath(#component, component.Value)                             │
│       FillPath(#context.*, contextFields)                               │
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

## Module Configuration Pattern

Modules use a schema/values separation for type-safe configuration:

### Pattern Structure

```cue
// module.cue - Schema definition (constraints only)
#config: {
    web: {
        image:    string
        replicas: int & >=1
        port:     int & >0 & <=65535
    }
}

// Declares that values must satisfy #config
values: #config

// values.cue - Concrete defaults
values: {
    web: {
        image:    "nginx:1.25"
        replicas: 2
        port:     8080
    }
}

// components.cue - References #config (not values)
#components: {
    web: {
        spec: {
            container: {
                name:  "web"
                image: #config.web.image  // References #config
            }
            replicas: #config.web.replicas
        }
    }
}
```

### How It Works

1. **At definition time**: `#components` references `#config` (abstract)
2. **At build time**: `ReleaseBuilder.Build()` calls `FillPath(#config, values)`
3. **Result**: `#config` becomes concrete, `#components` resolves to concrete values
4. **Extraction**: Components extracted from concrete `#components`

### Benefits

- **Type safety**: Users get validation against `#config` schema
- **Separation**: Schema constraints separate from default values
- **Composability**: Multiple values files can override defaults
- **Clear errors**: CUE reports which constraint was violated

## TransformerContext Construction

```go
// internal/build/context.go
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
| `internal/build/pipeline.go` | Pipeline implementation (6-phase orchestration) |
| `internal/build/module.go` | Module loading (raw values, metadata extraction) |
| `internal/build/release_builder.go` | Release building (#config injection, component extraction) |
| `internal/build/provider.go` | Provider loading |
| `internal/build/matcher.go` | Component-transformer matching |
| `internal/build/executor.go` | Parallel execution (FillPath injection) |
| `internal/build/context.go` | TransformerContext construction |
| `internal/build/types.go` | Shared types (Pipeline, RenderResult, Resource) |
| `internal/build/errors.go` | Error types (ReleaseValidationError, etc.) |
| `internal/output/manifest.go` | YAML/JSON formatting |
| `internal/output/split.go` | Split file output |
| `internal/output/verbose.go` | Verbose output |
| `internal/cmd/mod_build.go` | CLI command |
