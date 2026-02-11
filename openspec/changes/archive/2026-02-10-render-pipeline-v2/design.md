## Context

The render pipeline (`internal/build/`) orchestrates module loading, release building, transformer matching, and parallel execution. It was designed with six phases (see `archive/2026-02-05-build-v1/design.md`), but the current Go implementation has two architectural issues:

1. **Phase 3 (ReleaseBuilder)** reimplements CUE logic in Go. It manually calls `FillPath(#config, values)`, validates concreteness with `Validate(cue.Concrete(true))`, and extracts metadata field-by-field. The CUE `#ModuleRelease` definition (`opmodel.dev/core@v0`) already handles all of this declaratively.

2. **Phase 5 (Executor)** shares a single `*cue.Context` across worker goroutines. When multiple workers call `FillPath` concurrently, CUE's internal `adt.Vertex` graph (which is not goroutine-safe) panics at `sched.go:483`.

### Current CUE Context Flow

```text
cuecontext.New()  [config/loader.go:179]  ← THE ONE CONTEXT
    ├── ctx.BuildInstance(configInst)      ← config/provider tree
    │     └── Providers map[string]cue.Value
    │           └── LoadedTransformer.Value
    └── ctx.BuildInstance(moduleInst)      ← module tree
          └── FillPath(#config, values)
                └── LoadedComponent.Value (CONCRETE)

Both trees share the same *cue.Context.
Executor goroutines do cross-tree FillPath:
  transformValue.FillPath(#component, component.Value)
This triggers concurrent evaluation on the shared runtime → PANIC
```

### `#ModuleRelease` CUE Definition (from `opmodel.dev/core@v0`)

```cue
#ModuleRelease: close({
    metadata: {
        name!:      #NameType
        namespace!: string
        fqn:        #moduleMetadata.fqn
        version:    #moduleMetadata.version
        identity:   #UUIDType & uuid.SHA1(OPMNamespace, "\(fqn):\(name):\(namespace)")
        labels: {if #moduleMetadata.labels != _|_ {#moduleMetadata.labels}} & {
            "module-release.opmodel.dev/name":    "\(name)"
            "module-release.opmodel.dev/version": "\(version)"
            "module-release.opmodel.dev/uuid":    "\(identity)"
        }
    }
    #module!:         #Module
    #moduleMetadata:  #module.metadata
    _#module:         #module & {#config: values}   // ← injects values into #config
    components:       _#module.#components           // ← concrete from _#module
    values:           close(#module.#config)         // ← validates against schema
})
```

This definition handles value validation, config injection, component concreteness, and identity computation — all in CUE. The Go code should construct this, not reimplement it.

## Goals / Non-Goals

**Goals:**

- Use `#ModuleRelease` CUE definition in Phase 3 instead of manual Go logic
- Fix the CUE concurrency panic in Phase 5 by isolating `*cue.Context` per job
- Preserve the existing `Pipeline` interface and all command-level behavior
- Document the full 6-phase pipeline with clear CUE context boundaries

**Non-Goals:**

- Changing the `Pipeline` interface or `RenderResult` contract
- Adding new CLI flags or commands
- Changing the matching algorithm (Phase 4)
- Optimizing CUE evaluation performance beyond fixing the panic
- Moving transformer execution into CUE (transformers still execute via Go `FillPath`)

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
│ Phase 1: ProviderLoader      │ │ Phase 2: Module Metadata Extraction        │
│ (global CUE context)         │ │ (global CUE context)                       │
│                              │ │                                            │
│ config.Providers[name]       │ │ Lightweight load — just for name/namespace │
│        │                     │ │                                            │
│        ▼                     │ │ ModulePath ──▶ load.Instances()           │
│ provider.transformers        │ │                       │                    │
│        │                     │ │ ctx.BuildInstance() ──┘                    │
│        ▼                     │ │        │                                   │
│ For each transformer:        │ │ Extract metadata.name                      │
│   Extract #requirements      │ │ Extract metadata.defaultNamespace          │
│   - requiredLabels           │ │        │                                   │
│   - requiredResources        │ │        ▼                                   │
│   - requiredTraits           │ │ moduleMetadataPreview {                    │
│   - optionalLabels           │ │   name:             string                 │
│   - optionalResources        │ │   defaultNamespace:  string                │
│   - optionalTraits           │ │ }                                          │
│   Store cue.Value            │ │                                            │
│                              │ │ Used to resolve --name and --namespace     │
│ Result: LoadedProvider {     │ │ defaults before the full overlay build.    │
│   Transformers: []*LT        │ │                                            │
│   each LT.Value in global ctx│ │ Note: No ModuleLoader type — this is a     │
│ }                            │ │ helper method on pipeline (pipeline.go).   │
└──────────────────────────────┘ └────────────────────────────────────────────┘
           │                            │
           └────────────┬───────────────┘
                        │
                        ▼
┌─────────────────────────────────────────────────────────────────────────────┐
│ Phase 3: ReleaseBuilder — Hybrid Overlay (global CUE context)               │
│                                                                             │
│   Inputs:                                                                   │
│     modulePath     (absolute path to module directory)                      │
│     --values files (loaded and unified during build)                        │
│     --name, --namespace   (release identity)                                │
│                                                                             │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │ Step 1: Detect CUE package name from module directory            │      │
│   │                                                                  │      │
│   │   load.Instances(["."]) → inst.PkgName                           │      │
│   └──────────────────────────┬───────────────────────────────────────┘      │
│                              │                                              │
│                              ▼                                              │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │ Step 2: Generate metadata overlay via CUE uuid package           │      │
│   │                                                                  │      │
│   │ Use load.Config.Overlay to add a virtual CUE file that computes  │      │
│   │ release metadata (identity, labels) in CUE:                      │      │
│   │                                                                  │      │
│   │   // virtual: <modulePath>/opm_release_overlay.cue               │      │
│   │   package <pkgName>                                              │      │
│   │   import "uuid"                                                  │      │
│   │   #opmReleaseMeta: {                                             │      │
│   │       name:      "<releaseName>"                                 │      │
│   │       namespace: "<namespace>"                                   │      │
│   │       fqn:       metadata.fqn                                    │      │
│   │       version:   metadata.version                                │      │
│   │       identity:  uuid.SHA1(OPMNamespace, "fqn:name:namespace")   │      │
│   │       labels: metadata.labels & {                                │      │
│   │           "module-release.opmodel.dev/name":    name             │      │
│   │           "module-release.opmodel.dev/version": version          │      │
│   │           "module-release.opmodel.dev/uuid":    identity         │      │
│   │       }                                                          │      │
│   │   }                                                              │      │
│   │                                                                  │      │
│   │ KEY: Uses #opmReleaseMeta (definition) — NOT hidden field.       │      │
│   │ Definitions don't violate close() on #Module. Hidden fields and  │      │
│   │ regular fields do.                                               │      │
│   │                                                                  │      │
│   │ KEY: File named opm_release_overlay.cue — NOT _opm_release.cue.  │      │
│   │ CUE excludes files starting with _ from the loader.              │      │
│   └──────────────────────────┬───────────────────────────────────────┘      │
│                              │                                              │
│                              ▼                                              │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │ Step 3: Load module with overlay + unify values                  │      │
│   │                                                                  │      │
│   │ load.Instances(["."]) with Overlay → ctx.BuildInstance()         │      │
│   │        │                                                         │      │
│   │ Unify with --values files: CompileBytes + Unify                  │      │
│   └──────────────────────────┬───────────────────────────────────────┘      │
│                              │                                              │
│                              ▼                                              │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │ Step 4: Inject values into #config (Go FillPath — hybrid)        │      │
│   │                                                                  │      │
│   │ concreteModule = value.FillPath(#config, values)                 │      │
│   │                                                                  │      │
│   │ WHY HYBRID: The original plan was to use core.#ModuleRelease     │      │
│   │ which does _#module: #module & {#config: values} in CUE.         │      │
│   │ This FAILED because #ModuleRelease uses:                         │      │
│   │   values: close(#module.#config)                                 │      │
│   │ which panics with "struct argument must be concrete" when        │      │
│   │ #config contains CUE pattern constraints like [Name=string]:{}   │      │
│   │ (as in the jellyfin module). This is a CUE SDK limitation.       │      │
│   │                                                                  │      │
│   │ So: overlay computes METADATA only. Config injection still       │      │
│   │ uses Go-side FillPath (which works with pattern constraints).    │      │
│   └──────────────────────────┬───────────────────────────────────────┘      │
│                              │                                              │
│                              ▼                                              │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │ Step 5: Extract components + validate concreteness               │      │
│   │                                                                  │      │
│   │ concreteModule.LookupPath("#components") ──▶ iterate            │      │
│   │   For each: extract name, labels, annotations,                   │      │
│   │             #resources, #traits, Value                           │      │
│   │   Validate: comp.Value.Validate(cue.Concrete(true))              │      │
│   └──────────────────────────┬───────────────────────────────────────┘      │
│                              │                                              │
│                              ▼                                              │
│   ┌──────────────────────────────────────────────────────────────────┐      │
│   │ Step 6: Extract release metadata from overlay                    │      │
│   │                                                                  │      │
│   │ concreteModule.LookupPath("#opmReleaseMeta") ──▶ extract        │      │
│   │   name, namespace, version, fqn, identity, labels                │      │
│   │   (identity + labels computed by CUE uuid.SHA1)                  │      │
│   │                                                                  │      │
│   │ concreteModule.LookupPath("metadata.identity") ──▶ module ID    │      │
│   └──────────────────────────┬───────────────────────────────────────┘      │
│                              │                                              │
│                              ▼                                              │
│   Result: BuiltRelease {                                                    │
│     Value:      cue.Value (module with #config injected)                    │
│     Components: map[string]*LoadedComponent   (CONCRETE, validated)         │
│     Metadata:   ReleaseMetadata {                                           │
│       Name, Namespace, Version, FQN                                         │
│       Identity:        from module metadata (uuid computed by #Module)      │
│       ReleaseIdentity: from overlay (uuid.SHA1 of fqn:name:namespace)       │
│       Labels:          merged module labels + release labels                │
│     }                                                                       │
│   }                                                                         │
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
│ Phase 5: Executor (SEQUENTIAL — shared CUE context)                         │
│                                                                             │
│   CUE's *cue.Context is not safe for concurrent use. FillPath triggers      │
│   adt.Vertex evaluation which mutates shared internal state. Serialization  │
│   (Syntax() → CompileString) was attempted but panics on transformer        │
│   values with complex cross-package references (CUE exporter limitation).   │
│                                                                             │
│   Sequential execution is correct because:                                  │
│     • CUE evaluation (FillPath + Decode) dominates job runtime              │
│     • True parallelism requires isolated *cue.Context per worker,           │
│       which requires re-loading entire CUE module graphs (expensive)        │
│     • For typical modules (5-15 jobs), sequential is fast enough            │
│                                                                             │
│ ┌─ Sequential loop ──────────────────────────────────────────────────────┐  │
│ │                                                                        │  │
│ │  For each (transformer, component) pair in MatchResult.ByTransformer:  │  │
│ │                                                                        │  │
│ │  ┌─ Per Job (executeJob) ────────────────────────────────────────────┐ │  │
│ │  │                                                                   │ │  │
│ │  │  1. Get shared CUE context from transformer:                      │ │  │
│ │  │     cueCtx := job.Transformer.Value.Context()                     │ │  │
│ │  │                                                                   │ │  │
│ │  │  2. Look up #transform from transformer value:                    │ │  │
│ │  │     transformValue := tf.Value.LookupPath("#transform")           │ │  │
│ │  │                                                                   │ │  │
│ │  │  3. Inject component into transformer:                            │ │  │
│ │  │     unified := transformValue.FillPath(#component, comp.Value)    │ │  │
│ │  │                                                                   │ │  │
│ │  │  4. Build & inject #context:                                      │ │  │
│ │  │     tfCtx := NewTransformerContext(release, component)            │ │  │
│ │  │     unified.FillPath(#context.name, ...)                          │ │  │
│ │  │     unified.FillPath(#context.namespace, ...)                     │ │  │
│ │  │     unified.FillPath(#context.#moduleMetadata, ...)               │ │  │
│ │  │     unified.FillPath(#context.#componentMetadata, ...)            │ │  │
│ │  │                                                                   │ │  │
│ │  │  5. Extract output:                                               │ │  │
│ │  │     unified.LookupPath("output")                                  │ │  │
│ │  │       ├── ListKind    → iterate, decode each                      │ │  │
│ │  │       ├── Single resource (has apiVersion) → decode               │ │  │
│ │  │       └── Map of resources → iterate fields, decode each          │ │  │
│ │  │     Each → Decode to map[string]any                               │ │  │
│ │  │          → normalizeK8sResource()                                 │ │  │
│ │  │          → *unstructured.Unstructured                             │ │  │
│ │  │                                                                   │ │  │
│ │  │  6. Return JobResult{Resources, Error}                            │ │  │
│ │  │                                                                   │ │  │
│ │  │  ⚠ All CUE operations use the shared global context.              │ │  │
│ │  │    Safe because execution is single-threaded.                     │ │  │
│ │  └───────────────────────────────────────────────────────────────────┘ │  │
│ │                                                                        │  │
│ └────────────────────────────────────────────────────────────────────────┘  │
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

## CUE Context Boundaries

```text
┌─────────────────────────────────────────────────────────────┐
│ GLOBAL *cue.Context (created once in Phase 0)               │
│                                                             │
│   Phase 0: ConfigLoader     — BuildInstance (config.cue)    │
│   Phase 1: ProviderLoader   — read from config values       │
│   Phase 2: Metadata Extract — BuildInstance (module dir)    │
│   Phase 3: ReleaseBuilder   — BuildInstance (overlay)       │
│                                FillPath (#config injection) │
│                                Validate (concreteness)      │
│   Phase 4: Matcher          — read-only label comparison    │
│   Phase 5: Executor         — FillPath (inject #component)  │
│                                FillPath (inject #context)   │
│                                Decode   (extract output)    │
│                                                             │
│   ALL SINGLE-THREADED. No concurrency risk.                 │
│                                                             │
│   The entire pipeline runs on a single goroutine.           │
│   Parallelism was abandoned because:                        │
│   • CUE Syntax() panics on transformer values with          │
│     cross-package references (exporter limitation)          │
│   • CUE Decode triggers lazy evaluation, so even read-only  │
│     operations mutate shared adt.Vertex state               │
│   • True isolation requires re-loading module graphs per    │
│     worker — too expensive for typical job counts (5-15)    │
└─────────────────────────────────────────────────────────────┘
```

## Decisions

### Decision 1: Use `load.Config.Overlay` for `#ModuleRelease` construction

**Context**: Phase 3 needs to evaluate `core.#ModuleRelease` which requires import resolution (`opmodel.dev/core@v0`). `CompileString` cannot resolve imports. `BuildInstance` can, but needs a `*build.Instance` from `load.Instances`.

**Options considered**:

1. `CompileString` with inlined `#ModuleRelease` — fails because it cannot resolve the `uuid` import used in identity computation.
2. `load.Instances` on the core package separately — requires knowing the registry path and managing a second module load. Complex.
3. `load.Config.Overlay` to inject a virtual file into the module directory — the module already imports `opmodel.dev/core@v0`, so the overlay file can reference `core.#ModuleRelease` naturally.

**Decision**: Option 3 — Overlay.

**Rationale**: The overlay file lives in the same CUE package as the module. It has access to all the module's imports, fields, and values. No separate module loading required. The CUE SDK handles caching, so the second `load.Instances` call reuses the already-loaded module and dependencies.

### Decision 2: Sequential executor instead of parallel with serialization

**Context**: The original design called for serializing CUE values to source text (`format.Node(value.Syntax())`) and re-compiling in fresh `*cue.Context` instances per worker. This would enable safe parallel execution.

**Options considered**:

1. CUE source text serialization (`Syntax()` → `CompileString`) — preserves CUE constructs but **panics** on transformer values with cross-package references. The CUE exporter hits an "unreachable" path in `internal/core/export/self.go:379` when handling complex `adt.Vertex` references from multi-package module graphs (e.g., `core.#Transformer`, `workload_resources.#ContainerResource`). Tested with `cue.All()`, `cue.Concrete(false)`, `cue.Docs(false)` options.
2. JSON round-trip (`MarshalJSON` → `CompileString`) — loses CUE definitions (`#transform`, `#component`). Transformers contain `_|_` guards and comprehensions that don't survive JSON.
3. Sequential execution — run all jobs in a simple loop on the shared context. No concurrency = no panic. Simplest correct fix.
4. Per-worker module re-loading — re-load the entire provider CUE module graph in a fresh context per worker. Correct but expensive and complex.

**Decision**: Option 3 — Sequential execution.

**Rationale**: CUE evaluation (FillPath + Decode) dominates job runtime. The only pure-Go work (normalizeK8sResource) is trivially fast. There is no meaningful work to parallelize. For typical modules (5-15 transformer jobs), sequential execution completes in under a second. The serialization approach was attempted and abandoned due to a CUE SDK limitation in the exporter. Per-worker re-loading adds significant complexity for marginal benefit. Sequential is the simplest correct approach.

### Decision 3: Hybrid overlay (metadata only, not full #ModuleRelease)

**Context**: The original design called for using `core.#ModuleRelease` in the overlay to handle value validation, config injection, component concreteness, and identity computation — all in CUE. This would eliminate Go-side `FillPath(#config, values)` and `Validate(cue.Concrete(true))`.

**Options considered**:

1. Full `#ModuleRelease` in overlay — the CUE definition uses `values: close(#module.#config)` which panics with "struct argument must be concrete" when `#config` contains CUE pattern constraints like `[Name=string]: {...}`. This is a CUE SDK limitation with `close()` + pattern constraints.
2. Metadata-only overlay — the overlay computes release identity and labels via `uuid.SHA1`. Config injection (`FillPath`) and concreteness validation remain in Go. Hybrid approach.
3. No overlay — compute everything in Go, including uuid-based identity. Requires importing a Go UUID library.

**Decision**: Option 2 — Metadata-only overlay (hybrid).

**Rationale**: The overlay lets CUE compute `uuid.SHA1` for release identity (which requires the CUE `uuid` package — not available in Go without reimplementing the same hash). Config injection via Go `FillPath` works correctly with pattern constraints. The hybrid approach gets the best of both worlds: CUE for identity/labels, Go for config injection. The overlay file uses `#opmReleaseMeta` (a definition) to avoid violating `close()` constraints on `#Module`.

### Decision 4: Overlay uses `#opmReleaseMeta` definition and `opm_release_overlay.cue` filename

**Context**: The overlay file adds a field to the module's CUE package. It must not conflict with module-defined fields and must be accessible from Go.

**Key learnings discovered during implementation**:

1. Files starting with `_` are excluded by CUE's loader → overlay file named `opm_release_overlay.cue` (not `_opm_release.cue`)
2. Hidden fields (`_foo`) can't be accessed via `cue.ParsePath()` → used `#opmReleaseMeta` (public definition)
3. Definitions (`#foo`) don't violate `close()` constraints on `#Module`, regular fields and hidden fields do

**Decision**: Use `#opmReleaseMeta` as the definition name in `opm_release_overlay.cue`.

**Rationale**: Definitions are CUE type-level constructs that don't contribute regular struct fields, so they don't violate `close()` on `#Module`. They're accessible via `cue.ParsePath("#opmReleaseMeta")` from Go. The `opm_release_overlay.cue` filename avoids the `_` prefix exclusion.

### Decision 5: `#module` injection uses package-level references

**Context**: The overlay file needs to inject the loaded module's definition into `#ModuleRelease.#module`. The module's fields (metadata, #config, #components, values) are already in scope because the overlay is in the same CUE package.

**Decision**: The overlay references the module's top-level fields directly:

```cue
_opmRelease: core.#ModuleRelease & {
    metadata: name:      "<releaseName>"
    metadata: namespace: "<namespace>"
    // #module gets the top-level module definition
    // All fields (metadata, #config, #components, values)
    // are in scope from module.cue + components.cue + values.cue
    #module: {
        apiVersion: apiVersion
        kind:       kind
        metadata:   metadata
        #config:    #config
        #components: #components
        values:     values
    }
}
```

**Rationale**: The overlay is part of the same CUE package (same `package main`). All top-level fields from `module.cue`, `components.cue`, and `values.cue` are visible. The `#module` field explicitly maps these to the `#Module` structure expected by `#ModuleRelease`.

## Risks / Trade-offs

**[Risk] Overlay re-evaluation cost** → The `load.Instances` + `BuildInstance` call with overlay re-evaluates the entire module. Mitigation: CUE SDK caches loaded instances and dependencies. The overhead is the overlay metadata evaluation on top of already-parsed modules. For typical modules (1-5 components), this is sub-second.

**[Risk] Overlay file name collision** → A module author could create a file named `opm_release_overlay.cue`. Mitigation: Use a highly unlikely name with internal prefix. If collision occurs, `load.Instances` will fail with a duplicate field error — detectable and recoverable.

**[Risk] Two-phase module loading** → Phase 2 loads the module for metadata, Phase 3 re-loads with overlay. This means two `load.Instances` calls. Mitigation: The first load is lightweight (just for name/namespace). The second load includes the overlay. CUE SDK caches make this acceptable. A future optimization could merge both phases into one overlay-based load and extract metadata from `#opmReleaseMeta`.

**[Risk] Sequential execution throughput** → For modules with many components (>20), sequential execution may become a bottleneck. Mitigation: Typical OPM modules have 1-5 components. Even 20 components × 12 transformers = 240 jobs completes in seconds. If parallelism is needed in the future, the correct approach is per-worker module re-loading (re-build provider `*build.Instance` in fresh contexts), not value serialization.

**[Resolved Risk] `Syntax()` serialization** → The original design planned to serialize CUE values via `format.Node(value.Syntax())`. This **panics** on transformer values with cross-package references — a CUE SDK limitation in `internal/core/export/self.go`. This risk materialized during implementation and was resolved by switching to sequential execution.

**[Resolved Risk] `close()` + pattern constraints** → The original design planned to use `core.#ModuleRelease` which calls `close(#module.#config)`. This panics when `#config` contains CUE pattern constraints like `[Name=string]: {...}`. Resolved by using a hybrid overlay that only computes metadata, with Go-side `FillPath` for config injection.
