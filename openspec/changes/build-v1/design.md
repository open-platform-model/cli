# Design: CLI Build Command

## Technical Approach

### Render Pipeline Architecture

```text
+-------------------------------------------------------------------+
|                       Hybrid Render Pipeline                       |
+-------------------------------------------------------------------+
|  Phase 1: Module Loading & Validation                        [Go]  |
|           - Load CUE via cue/load                                  |
|           - Extract release metadata                               |
|           - Build base TransformerContext                          |
+-------------------------------------------------------------------+
|  Phase 2: Provider Loading                                   [Go]  |
|           - Access provider.transformers from CUE                  |
+-------------------------------------------------------------------+
|  Phase 3: Component Matching                                [CUE]  |
|           - CUE evaluates #Matches predicate                       |
|           - CUE computes #matchedTransformers map                  |
|           - Go reads back the computed matching plan               |
+-------------------------------------------------------------------+
|  Phase 4: Parallel Transformer Execution                     [Go]  |
|           - Iterate CUE-computed matches                           |
|           - Unify transformer.#transform + inputs                  |
|           - Workers: isolated cue.Context -> Decode output         |
+-------------------------------------------------------------------+
|  Phase 5: Aggregation & Output                               [Go]  |
|           - Collect results from workers                           |
|           - Aggregate errors (fail-on-end)                         |
|           - Output YAML manifests                                  |
+-------------------------------------------------------------------+
```

### Key Design Decisions

#### CLI-only rendering

Rendering logic resides in Go CLI, not CUE schemas. CUE handles definition and matching; Go handles orchestration and I/O.

#### ModuleRelease construction

`mod build` constructs `#ModuleRelease` internally from local module + values files. No separate release file needed for local builds.

#### Parallel execution

Transformer execution uses goroutine workers with isolated CUE contexts to prevent memory sharing issues.

#### Fail-on-end error handling

Collect all errors during rendering, aggregate and display together, then exit with non-zero status.

## Data Flow

```text
Module + Values → ModuleRelease → Components → Transformers → Resources → YAML
```

### Values Resolution

1. Load `values.cue` from module root (required, fail if not found)
2. Load additional files from `--values` flags in order
3. Unify all values using CUE (fail on conflict with native error)
4. `--namespace` flag takes precedence over `#Module.metadata.defaultNamespace`
5. `--name` flag takes precedence over `#Module.metadata.name`

### ModuleRelease Construction

```text
Local Module Path + Values Files → #ModuleRelease (in-memory)
     ↓
- name: from --name or module.metadata.name
- namespace: from --namespace or module.metadata.defaultNamespace
- version: from module.metadata.version
- components: from module with values unified
```

## Transformer Matching Logic

**Concept: Capability vs. Intent**

1. **Capability (Resources & Traits)**: Does the component have the necessary data?
2. **Intent (Labels)**: Does the component have the specific label to disambiguate?

A transformer matches if ALL conditions are met:

1. **Required Labels**: Present in effective labels with matching values
2. **Required Resources**: FQNs exist in component.#resources
3. **Required Traits**: FQNs exist in component.#traits

Multiple transformers can match a single component. If no transformers match, it's an error.

### Match Evaluation (CUE)

```cue
#Matches: {
    transformer: #Transformer
    component:   #Component

    // 1. Check Required Labels
    // Logic: All labels in transformer.requiredLabels must exist in component.metadata.labels with same value
    _reqLabels: *transformer.requiredLabels | {}
    _missingLabels: [
        for k, v in _reqLabels
        if len([for lk, lv in component.metadata.labels if lk == k && (lv & v) != _|_ {true}]) == 0 {
            k
        },
    ]

    // 2. Check Required Resources
    // Logic: All keys in transformer.requiredResources must exist in component.#resources
    _reqResources: *transformer.requiredResources | {}
    _missingResources: [
        for k, v in _reqResources
        if len([for rk, rv in component.#resources if rk == k && (rv & v) != _|_ {true}]) == 0 {
            k
        },
    ]

    // 3. Check Required Traits
    // Logic: All keys in transformer.requiredTraits must exist in component.#traits
    _reqTraits: *transformer.requiredTraits | {}
    _missingTraits: [
        for k, v in _reqTraits
        if component.#traits == _|_ || len([for tk, tv in component.#traits if tk == k && (tv & v) != _|_ {true}]) == 0 {
            k
        },
    ]

    // Result: true if no requirements are missing
    result: len(_missingLabels) == 0 && len(_missingResources) == 0 && len(_missingTraits) == 0
}
```

## TransformerContext

Context injected into every transformer execution.

### CUE Definition

```cue
#TransformerContext: close({
    #moduleMetadata:    _ // Injected during rendering
    #componentMetadata: _ // Injected during rendering
    name:               string // Injected during rendering (release name)
    namespace:          string // Injected during rendering (target namespace)

    moduleLabels: {
        if #moduleMetadata.labels != _|_ {#moduleMetadata.labels}
    }

    componentLabels: {
        "app.kubernetes.io/instance": "\(name)-\(namespace)"

        if #componentMetadata.labels != _|_ {#componentMetadata.labels}
    }

    controllerLabels: {
        "app.kubernetes.io/managed-by": "open-platform-model"
        "app.kubernetes.io/name":       #componentMetadata.name
        "app.kubernetes.io/version":    #moduleMetadata.version
    }

    labels: {[string]: string}
    labels: {
        for k, v in moduleLabels {
            (k): "\(v)"
        }
        for k, v in componentLabels {
            (k): "\(v)"
        }
        for k, v in controllerLabels {
            (k): "\(v)"
        }
        ...
    }
})
```

### Go Equivalent

```go
type TransformerContext struct {
    Name      string            `json:"name"`
    Namespace string            `json:"namespace"`
    
    // Hidden fields (injected via CUE)
    ModuleMetadata    map[string]any
    ComponentMetadata map[string]any
    
    // Computed fields (derived in CUE)
    Labels map[string]string `json:"labels"`
}
```

## Go Data Structures

### Pipeline

Central orchestrator for the render process.

```go
type Pipeline struct {
    Loader   *Loader
    Provider *Provider
    Renderer *Renderer
    Logger   Logger
}
```

### Provider

Loaded OPM provider (e.g., `kubernetes`).

```go
type Provider struct {
    Name         string
    Version      string
    Transformers map[string]*Transformer
}
```

### Transformer

Single transformation unit.

```go
type Transformer struct {
    Name              string
    ID                string // Full CUE path/FQN
    RequiredLabels    map[string]string
    RequiredResources []string
    RequiredTraits    []string
    Source            cue.Value // Raw CUE value for unification
}
```

### Match

Decision to apply transformer to component.

```go
type MatchGroup struct {
    Transformer *Transformer
    Components  []*Component
}

type MatchedMap map[string]*MatchGroup
```

### Component

Wrapper around component data.

```go
type Component struct {
    Name      string
    Labels    map[string]string // Effective labels
    Resources []Resource
    Traits    []Trait
    Source    cue.Value
}
```

### Worker

Parallel execution unit with isolated CUE context.

```go
type Worker struct {
    ID      int
    Context *cue.Context // Isolated context per worker
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

## Error Handling

### Fail-on-End Pattern

```text
1. Execute all transformers
2. Collect all errors into []error
3. After all complete:
   - If errors: aggregate and display together, exit non-zero
   - If success: output manifests
```

### Error Types

| Error | Handling |
|-------|----------|
| Unmatched component | List available transformers and their requirements |
| Unhandled trait (strict) | Error with trait name and component |
| Unhandled trait (normal) | Warning logged, continue |
| Multiple exact matches | Error with conflicting transformer names |
| Values conflict | CUE native error |
| Invalid values file | CUE parse/validation error |

## Output Formatting

### File Naming (--split)

Pattern: `<lowercase-kind>-<resource-name>.yaml`

Examples:

- `deployment-frontend.yaml`
- `service-frontend.yaml`
- `configmap-app-config.yaml`

### Sensitive Data

Redact secrets in verbose logging. Never log:

- Secret data values
- Environment variable values from secrets
- Credential strings

## File Changes

- `cli/internal/cmd/mod/build.go` - Command implementation
- `cli/internal/render/` - Render pipeline package
- `cli/internal/render/pipeline.go` - Pipeline orchestration
- `cli/internal/render/loader.go` - Module and values loading
- `cli/internal/render/matcher.go` - Component-transformer matching
- `cli/internal/render/worker.go` - Parallel transformer workers
- `cli/internal/render/output.go` - YAML/JSON output formatting
- `cli/internal/render/context.go` - TransformerContext construction
