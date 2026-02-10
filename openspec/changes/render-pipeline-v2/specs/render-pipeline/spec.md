## ADDED Requirements

### Requirement: Pipeline phases are numbered and documented

The render pipeline SHALL be organized into six sequential phases with clear CUE context boundaries. This establishes the canonical phase numbering for all pipeline documentation and debugging.

#### Scenario: Pipeline executes all six phases in order

- **WHEN** `Pipeline.Render()` is called with valid `RenderOptions`
- **THEN** the pipeline SHALL execute the following phases in order:
  - Phase 0: ConfigLoader (load and validate configuration)
  - Phase 1: ProviderLoader (load provider and extract transformer requirements)
  - Phase 2: ModuleLoader (load module, unify values, extract metadata)
  - Phase 3: ReleaseBuilder (construct `#ModuleRelease` via CUE, extract concrete components)
  - Phase 4: Matcher (match components to transformers by requirements)
  - Phase 5: Executor (execute transformers in parallel with isolated CUE contexts)

### Requirement: CUE context isolation boundary

Phases 0-4 SHALL use a single shared `*cue.Context` (the global context created in Phase 0). Phase 5 SHALL create a fresh `*cue.Context` per executor job. No CUE state SHALL be shared across goroutines.

#### Scenario: Phases 0-4 share a single CUE context

- **WHEN** the pipeline executes Phases 0 through 4
- **THEN** all CUE operations (BuildInstance, LookupPath, FillPath, field iteration) SHALL use the global `*cue.Context` created during config loading
- **AND** all operations SHALL be single-threaded (no concurrent CUE evaluation)

#### Scenario: Phase 5 isolates CUE contexts per job

- **WHEN** the executor dispatches a job to a worker goroutine
- **THEN** the worker SHALL create a fresh `*cue.Context` via `cuecontext.New()`
- **AND** all CUE operations within that job (CompileString, FillPath, Decode) SHALL use only the fresh context
- **AND** the fresh context SHALL be discarded after the job completes

#### Scenario: Module with multiple components renders without panic

- **WHEN** a module has N components (N > 1) that match transformers
- **THEN** the executor SHALL process all component-transformer pairs in parallel without runtime panics
- **AND** the `RenderResult` SHALL contain resources from all successful jobs

### Requirement: Pre-execution serialization for executor jobs

Before dispatching jobs to worker goroutines, the executor SHALL serialize transformer and component CUE values to source text in a single-threaded pre-execution phase. Serialization SHALL use CUE-native representation, not JSON.

#### Scenario: Transformer values serialized to CUE source text

- **WHEN** the executor prepares jobs for execution
- **THEN** each unique transformer value SHALL be serialized to CUE source text via `format.Node(value.Syntax())`
- **AND** the serialized text SHALL be cached by transformer FQN (serialized once, used by all jobs referencing that transformer)

#### Scenario: Component values serialized to CUE source text

- **WHEN** the executor prepares jobs for execution
- **THEN** each unique component value SHALL be serialized to CUE source text via `format.Node(value.Syntax())`
- **AND** the serialized text SHALL be cached by component name (serialized once, used by all jobs referencing that component)

#### Scenario: Serialized values re-materialize in fresh context

- **WHEN** a worker job re-materializes a transformer or component from cached CUE source text
- **THEN** the worker SHALL use `freshCtx.CompileString(cachedSource)` to produce a new `cue.Value` in the isolated context
- **AND** the re-materialized value SHALL be functionally equivalent to the original (same structure, same constraints, same concrete data)

## MODIFIED Requirements

### Requirement: ReleaseBuilder uses CUE #ModuleRelease definition

The ReleaseBuilder SHALL construct a `core.#ModuleRelease` instance in CUE using `load.Config.Overlay` to inject a virtual file into the module directory. The CUE evaluator SHALL handle value validation, config injection, component concreteness, and metadata computation. Go SHALL only extract results from the evaluated CUE value.

#### Scenario: ReleaseBuilder constructs #ModuleRelease via overlay

- **WHEN** the ReleaseBuilder receives a `LoadedModule` and `ReleaseOptions`
- **THEN** it SHALL create a `load.Config` with an `Overlay` map containing a virtual CUE file (e.g., `_opm_release.cue`) in the module directory
- **AND** the virtual file SHALL import `opmodel.dev/core@v0` and construct `_opmRelease: core.#ModuleRelease & { ... }`
- **AND** it SHALL call `load.Instances` and `BuildInstance` with the overlay to produce the evaluated release

#### Scenario: Values validated by CUE schema

- **WHEN** the ReleaseBuilder evaluates `#ModuleRelease` with values that do not satisfy `#module.#config`
- **THEN** CUE SHALL return a validation error with file path, line, and column information
- **AND** the ReleaseBuilder SHALL propagate this error as a fatal pipeline error

#### Scenario: Components are concrete after #ModuleRelease evaluation

- **WHEN** the `#ModuleRelease` evaluation succeeds
- **THEN** the `components` field SHALL contain fully concrete component values (all `#config` references resolved)
- **AND** Go SHALL extract `LoadedComponent` structs by iterating `_opmRelease.components`

#### Scenario: Release metadata computed by CUE

- **WHEN** the `#ModuleRelease` evaluation succeeds
- **THEN** `_opmRelease.metadata` SHALL contain the release identity UUID (computed by `uuid.SHA1` in CUE)
- **AND** `_opmRelease.metadata.labels` SHALL contain the standard release labels (`module-release.opmodel.dev/name`, `module-release.opmodel.dev/version`, `module-release.opmodel.dev/uuid`)
- **AND** Go SHALL extract `ReleaseMetadata` from `_opmRelease.metadata`

#### Scenario: Overlay uses hidden field to avoid conflicts

- **WHEN** the overlay file is injected into the module directory
- **THEN** the release field SHALL be named `_opmRelease` (CUE hidden field with underscore prefix)
- **AND** the field SHALL NOT conflict with any module-authored fields

## Pipeline Data Flow

```text
                              ╔═══════════════════╗
                              ║   CLI Invocation  ║
                              ║  flags, env vars  ║
                              ╚═════════╤═════════╝
                                        │
                                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Phase 0: ConfigLoader                                                       │
│                                                                             │
│   ~/.opm/config.cue ──────────┐                                             │
│   --config flag ──────────────┤                                             │
│   OPM_REGISTRY env ───────────┤                                             │
│   --registry flag ────────────┤                                             │
│                               ▼                                             │
│   ┌─────────────────────────────────────────┐                               │
│   │ 1. Resolve config path (flag > default) │                               │
│   │ 2. Bootstrap registry (regex extract)   │                               │
│   │ 3. Resolve registry (flag > env > cfg)  │                               │
│   │ 4. cuecontext.New() ◄── GLOBAL CONTEXT  │                               │
│   │ 5. Load config.cue with CUE_REGISTRY    │                               │
│   │ 6. Extract providers map[string]Value   │                               │
│   │ 7. Extract log, kubernetes config       │                               │
│   └─────────────────────────────────────────┘                               │
│                                                                             │
│   Result: OPMConfig {                                                       │
│     Config:     *Config          (registry, k8s, log settings)              │
│     Providers:  map[string]cue.Value  ◄── in global CUE context             │
│     CueContext: *cue.Context     ◄── THE ONE GLOBAL CONTEXT                 │
│     Registry:   string                                                      │
│   }                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
                                        │
           ┌────────────────────────────┤
           │                            │
           ▼                            ▼
┌──────────────────────────────┐ ┌────────────────────────────────────────────┐
│ Phase 1: ProviderLoader      │ │ Phase 2: ModuleLoader                      │
│ (global CUE context)         │ │ (global CUE context)                       │
│                              │ │                                            │
│ config.Providers[name]       │ │ ModulePath ──▶ load.Instances()           │
│        │                     │ │                      │                     │
│        ▼                     │ │ ctx.BuildInstance() ──┘                    │
│ provider.transformers        │ │        │                                   │
│        │                     │ │ values.cue is auto-loaded as part of       │
│        ▼                     │ │ the CUE instance (same directory/package)  │
│ For each transformer:        │ │        │                                   │
│   Extract #requirements      │ │ --values files ──▶ CompileBytes + Unify   │
│   - requiredLabels           │ │        │                                   │
│   - requiredResources        │ │ --name, --namespace ──▶ override metadata │
│   - requiredTraits           │ │        │                                   │
│   - optionalLabels           │ │        ▼                                   │
│   - optionalResources        │ │ LoadedModule {                             │
│   - optionalTraits           │ │   Value:     cue.Value  ◄── IS #Module     │
│   Store cue.Value            │ │   Name, Namespace, Version, Labels         │
│                              │ │   Path: string (absolute module dir)       │
│ Result: LoadedProvider {     │ │ }                                          │
│   Transformers: []*LT        │ │                                            │
│   each LT.Value in global ctx│ │ The module imports opmodel.dev/core@v0     │
│ }                            │ │ so #ModuleRelease is in the eval context   │
└──────────────────────────────┘ └────────────────────────────────────────────┘
           │                            │
           └────────────┬───────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Phase 3: ReleaseBuilder (global CUE context)                                │
│                                                                             │
│   Inputs:                                                                   │
│     LoadedModule.Value    (raw #Module — #config unresolved)                │
│     LoadedModule.Path     (absolute path to module directory)               │
│     --values files        (already unified into LoadedModule.Value)         │
│     --name, --namespace   (release identity)                                │
│                                                                             │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │ Step 1: Construct #ModuleRelease via Overlay                     │      │
│   │                                                                  │      │
│   │ The module's CUE context already has core.#ModuleRelease         │      │
│   │ loaded (it's a dependency of every OPM module).                  │      │
│   │                                                                  │      │
│   │ Use load.Config.Overlay to add a virtual CUE file                │      │
│   │ in the module directory that constructs the release:             │      │
│   │                                                                  │      │
│   │   // virtual: <modulePath>/_opm_release.cue                      │      │
│   │   package main                                                   │      │
│   │   import "opmodel.dev/core@v0"                                   │      │
│   │   _opmRelease: core.#ModuleRelease & {                           │      │
│   │       metadata: name:      "<releaseName>"                       │      │
│   │       metadata: namespace: "<namespace>"                         │      │
│   │       #module: {                                                 │      │
│   │           // reference to the loaded module (same package)       │      │
│   │           // fields are available because overlay is in the      │      │
│   │           // same CUE package as module.cue + components.cue     │      │
│   │       }                                                          │      │
│   │       values: values  // from values.cue in the same package     │      │
│   │   }                                                              │      │
│   │                                                                  │      │
│   │ load.Instances(["."]) with Overlay → BuildInstance               │      │
│   │                                                                  │      │
│   │ CUE evaluator takes over:                                        │      │
│   │                                                                  │      │
│   │   values: close(#module.#config)                                 │      │
│   │     └── validates values against schema                          │      │
│   │         If invalid → CUE error with file:line:col                │      │
│   │                                                                  │      │
│   │   _#module: #module & { #config: values }                        │      │
│   │     └── unifies module with concrete values                      │      │
│   │         #config resolves → all #config refs become concrete      │      │
│   │                                                                  │      │
│   │   components: _#module.#components                               │      │
│   │     └── components are CONCRETE (inherited from _#module)        │      │
│   │                                                                  │      │
│   │   metadata.identity: uuid.SHA1(...)                              │      │
│   │   metadata.labels: { "module-release.opmodel.dev/uuid": ... }    │      │
│   │     └── computed fields resolve automatically                    │      │
│   │                                                                  │      │
│   └──────────────────────────┬───────────────────────────────────────┘      │
│                              │                                              │
│                              ▼                                              │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │ Step 2: Extract result (Go reads from evaluated CUE)             │      │
│   │                                                                  │      │
│   │   release.LookupPath("_opmRelease.components") ──▶ iterate      │      │
│   │     For each: extract name, labels, annotations,                 │      │
│   │               #resources, #traits, Value                         │      │
│   │                                                                  │      │
│   │   release.LookupPath("_opmRelease.metadata") ──▶ extract        │      │
│   │     name, namespace, version, fqn, identity, labels              │      │
│   └──────────────────────────┬───────────────────────────────────────┘      │
│                              │                                              │
│                              ▼                                              │
│   Result: BuiltRelease {                                                    │
│     Value:      cue.Value (#ModuleRelease, fully evaluated)                 │
│     Components: map[string]*LoadedComponent   (CONCRETE, from CUE)          │
│     Metadata:   ReleaseMetadata               (from CUE computed fields)    │
│   }                                                                         │
│                                                                             │
│   GUARANTEE: CUE enforces concreteness. Go only extracts.                   │
└─────────────────────────────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Phase 4: Matcher (pure Go, no CUE evaluation)                               │
│                                                                             │
│   For each component in BuiltRelease.Components:                            │
│     For each transformer in LoadedProvider.Transformers:                    │
│                                                                             │
│       ┌──────────────────────────────────────────┐                          │
│       │ Check requiredLabels    ∈ comp.Labels    │                          │
│       │ Check requiredResources ∈ comp.Resources │                          │
│       │ Check requiredTraits    ∈ comp.Traits    │                          │
│       │                                          │                          │
│       │ ALL required match? ──▶ MATCHED         │                          │
│       │ Any required miss?  ──▶ skip            │                          │
│       └──────────────────────────────────────────┘                          │
│                                                                             │
│   Result: MatchResult {                                                     │
│     ByTransformer: map[tfFQN][]*LoadedComponent  (matched pairs)            │
│     Unmatched:     []*LoadedComponent            (no transformer found)     │
│     Details:       []MatchDetail                 (per-pair reasoning)       │
│   }                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Phase 5: Executor (PARALLEL — isolated CUE contexts)                        │
│                                                                             │
│ ┌─ Pre-execution (single-threaded) ───────────────────────────────────────┐ │
│ │                                                                         │ │
│ │  For each unique transformer in matched set:                            │ │
│ │    tfSource[fqn] = format.Node(transformer.Value.Syntax())              │ │
│ │    ──▶ serialize to CUE source text, cached once per transformer       │ │
│ │                                                                         │ │
│ │  For each unique component in matched set:                              │ │
│ │    compSource[name] = format.Node(component.Value.Syntax())             │ │
│ │    ──▶ serialize to CUE source text, cached once per component         │ │
│ │    (components are CONCRETE — but we keep them as CUE, not JSON)        │ │
│ │                                                                         │ │
│ │  Build jobs: []Job{TransformerFQN, ComponentName, Release}              │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│ ┌─ Worker pool (N goroutines) ────────────────────────────────────────────┐ │
│ │                                                                         │ │
│ │  ┌─ Per Job ──────────────────────────────────────────────────────────┐ │ │
│ │  │                                                                    │ │ │
│ │  │  1. freshCtx := cuecontext.New()    ◄── ISOLATED CONTEXT           │ │ │
│ │  │                                                                    │ │ │
│ │  │  2. Re-materialize transformer (CUE → CUE):                        │ │ │
│ │  │     tfValue := freshCtx.CompileString(tfSource[job.TfFQN])         │ │ │
│ │  │     transformValue := tfValue.LookupPath("#transform")             │ │ │
│ │  │                                                                    │ │ │
│ │  │  3. Re-materialize component (CUE → CUE):                          │ │ │
│ │  │     compValue := freshCtx.CompileString(compSource[job.CompName])  │ │ │
│ │  │                                                                    │ │ │
│ │  │  4. Inject component into transformer:                             │ │ │
│ │  │     unified := transformValue.FillPath(#component, compValue)      │ │ │
│ │  │                                                                    │ │ │
│ │  │  5. Build & inject #context:                                       │ │ │
│ │  │     tfCtx := NewTransformerContext(release, component)             │ │ │
│ │  │     unified.FillPath(#context.name, ...)                           │ │ │
│ │  │     unified.FillPath(#context.namespace, ...)                      │ │ │
│ │  │     unified.FillPath(#context.#moduleMetadata, ...)                │ │ │
│ │  │     unified.FillPath(#context.#componentMetadata, ...)             │ │ │
│ │  │                                                                    │ │ │
│ │  │  6. Extract output:                                                │ │ │
│ │  │     unified.LookupPath("output")                                   │ │ │
│ │  │       ├── ListKind    → iterate, decode each                       │ │ │
│ │  │       ├── Single resource (has apiVersion) → decode                │ │ │
│ │  │       └── Map of resources → iterate fields, decode each           │ │ │
│ │  │     Each → Decode to map[string]any                                │ │ │
│ │  │          → normalizeK8sResource()                                  │ │ │
│ │  │          → *unstructured.Unstructured                              │ │ │
│ │  │                                                                    │ │ │
│ │  │  7. Return JobResult{Resources, Error}                             │ │ │
│ │  │                                                                    │ │ │
│ │  │  ⚠ All CUE operations (2-6) use freshCtx ONLY.                    │ │ │
│ │  │    Zero shared CUE state between goroutines.                       │ │ │
│ │  └────────────────────────────────────────────────────────────────────┘ │ │
│ │                                                                         │ │
│ └─────────────────────────────────────────────────────────────────────────┘ │
│                                                                             │
│   Result: ExecuteResult {                                                   │
│     Resources: []*Resource{Object, Component, Transformer}                  │
│     Errors:    []error (fail-on-end aggregation)                            │
│   }                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Result Assembly                                                             │
│                                                                             │
│   Resources ──▶ sort by weight (weights.GetWeight per GVK)                 │
│   Unmatched ──▶ UnmatchedComponentError per component                      │
│   Warnings  ──▶ unhandled traits (non-strict mode only)                    │
│                                                                             │
│   RenderResult {                                                            │
│     Resources: []*Resource        (ordered for sequential apply)            │
│     Module:    ModuleMetadata     (name, ns, ver, identity, releaseID)      │
│     MatchPlan: MatchPlan          (for verbose/debug output)                │
│     Errors:    []error            (aggregated: unmatched + transform errs)  │
│     Warnings:  []string           (unhandled traits in non-strict mode)     │
│   }                                                                         │
└─────────────────────────────────────────────────────────────────────────────┘
```

### CUE Context Boundaries

```text
┌─────────────────────────────────────────────────────────────┐
│ GLOBAL *cue.Context (created once in Phase 0)               │
│                                                             │
│   Phase 0: ConfigLoader     — BuildInstance (config.cue)    │
│   Phase 1: ProviderLoader   — read from config values       │
│   Phase 2: ModuleLoader     — BuildInstance (module dir)    │
│   Phase 3: ReleaseBuilder   — BuildInstance (overlay)       │
│   Phase 4: Matcher          — read-only label comparison    │
│                                                             │
│   ALL SINGLE-THREADED. No concurrency risk.                 │
├─────────────────────────────────────────────────────────────┤
│ FRESH *cue.Context (created per job in Phase 5)             │
│                                                             │
│   Per worker job:                                           │
│     CompileString  — re-materialize transformer from CUE    │
│     CompileString  — re-materialize component from CUE      │
│     FillPath       — inject #component, #context            │
│     Decode         — extract output resources               │
│                                                             │
│   PARALLEL. Each goroutine owns its context exclusively.    │
│   Zero shared CUE state across goroutines.                  │
└─────────────────────────────────────────────────────────────┘
```
