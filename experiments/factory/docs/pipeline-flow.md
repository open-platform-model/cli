# Pipeline Flow

## Overview

The render pipeline has two input shapes — a standalone `ModuleRelease` and a
`BundleRelease` — and a common rendering core. Both shapes produce `*core.Resource`
values carrying Kubernetes manifests with provenance metadata (release, component,
transformer).

The pipeline is split across two packages:

- **`pkg/loader`** — CUE evaluation and Go struct extraction (load phase)
- **`pkg/engine`** — CUE matching and transform execution (render phase)

The entry point `cmd/main.go` loads the CUE package, detects the kind, and
dispatches to the appropriate loader and renderer.

---

## ModuleRelease Flow

```text
 Input files
 ┌───────────────────┐  ┌────────────────┐
 │    release.cue    │  │   values.cue   │
 │  (applies        │  │  (consumer-    │
 │  #ModuleRelease) │  │   supplied)    │
 └────────┬──────────┘  └───────┬────────┘
          │                     │
          └──────────┬──────────┘
                     │ load.Instances + cueCtx.BuildInstance
                     ▼
          ┌──────────────────────┐
          │   CUE package value  │
          │   (single instance   │
          │    of both files)    │
          └──────────┬───────────┘
                     │ LoadReleasePackage
                     │ DetectReleaseKind → "ModuleRelease"
                     │
                     │ LoadModuleReleaseFromValue
                     ▼
          ┌──────────────────────┐
          │  pkg.Err()           │  ← top-level CUE evaluation error
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │   Module Gate        │  ← validateConfig(#module.#config, values)
          │                      │    catches type mismatches + missing fields
          └──────────┬───────────┘
                     │ pass
                     ▼
          ┌──────────────────────┐
          │ releaseVal.Validate  │  ← whole-release concreteness check
          │ (cue.Concrete(true)) │    catches derived fields not in #config
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │ extractReleaseMetadata│ ← decode metadata.{name,namespace,uuid,...}
          │ extractModuleInfo    │ ← decode #module.metadata (FQN, version)
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │ finalizeValue        │  ← Syntax(cue.Final()) + BuildExpr
          │ (whole release)      │    strips matchN validators, close(),
          └──────────┬───────────┘    definition fields; takes defaults
                     │
                     ▼
          ┌──────────────────────────────────────────┐
          │  ModuleRelease{                           │
          │    Schema:         releaseVal   (original │ ← used by engine for matching
          │                    constrained)           │   (#resources, #traits present)
          │    DataComponents: finalized components   │ ← used by engine for FillPath
          │    Metadata:       ReleaseMetadata        │   (constraint-free)
          │    Module:         ModuleMetadata         │
          │  }                                        │
          └──────────────────────┬───────────────────┘
                                 │
                                 │ engine.NewModuleRenderer.Render
                                 ▼
          ┌──────────────────────────────────────────┐
          │  Phase 1 — Match                          │
          │                                           │
          │  buildMatchPlan:                          │
          │    load ./core/matcher CUE package        │
          │    fill #MatchPlan.#provider ← provider   │
          │    fill #MatchPlan.#components             │
          │         ← rel.Schema["components"]        │ ← schema value: has #resources,
          │    evaluate → decode matchPlanResult      │   #traits for matching logic
          │                                           │
          │  MatchPlan{                               │
          │    Matches:         comp → tf → result    │
          │    Unmatched:       []string              │
          │    UnhandledTraits: comp → []fqn          │
          │  }                                        │
          └──────────────────────┬───────────────────┘
                                 │ error if any Unmatched
                                 │
                                 ▼
          ┌──────────────────────────────────────────┐
          │  Phase 2 — Execute (per matched pair)     │
          │                                           │
          │  executePair:                             │
          │    look up transformer #transform         │
          │    look up dataComp ← DataComponents      │ ← finalized: safe for FillPath
          │    look up schemaComp ← Schema            │ ← preserved: metadata.labels etc.
          │    FillPath #component ← dataComp         │
          │    injectContext:                         │
          │      FillPath #context.#moduleRelease...  │
          │      FillPath #context.#componentMetadata │
          │    evaluate unified → look up output      │
          │    decode output → []*core.Resource       │
          └──────────────────────┬───────────────────┘
                                 │
                                 ▼
          ┌──────────────────────────────────────────┐
          │  RenderResult{                            │
          │    Resources: []*core.Resource            │ ← each has Value, Release,
          │    Warnings:  []string                    │   Component, Transformer
          │  }                                        │
          └──────────────────────────────────────────┘
```

---

## BundleRelease Flow

```text
 Input files
 ┌───────────────────┐  ┌────────────────┐
 │    release.cue    │  │   values.cue   │
 │  (applies        │  │  (bundle-level │
 │  #BundleRelease) │  │   consumer     │
 │                   │  │   values)      │
 └────────┬──────────┘  └───────┬────────┘
          │                     │
          └──────────┬──────────┘
                     │ load.Instances + cueCtx.BuildInstance
                     ▼
          ┌──────────────────────┐
          │   CUE package value  │
          │                      │
          │   #BundleRelease     │
          │   comprehension      │
          │   already evaluated: │
          │   releases: {        │
          │     server: #ModRel  │
          │     proxy:  #ModRel  │
          │   }                  │
          └──────────┬───────────┘
                     │ LoadReleasePackage
                     │ DetectReleaseKind → "BundleRelease"
                     │
                     │ LoadBundleReleaseFromValue
                     ▼
          ┌──────────────────────┐
          │  pkg.Err()           │  ← top-level CUE evaluation error
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │ extractBundleRelease │
          │   Metadata           │  ← decode metadata.{name,uuid}
          │ extractBundleInfo    │  ← decode #bundle.metadata (FQN, version)
          └──────────┬───────────┘
                     │
                     ▼
          ┌──────────────────────┐
          │   Bundle Gate        │  ← validateConfig(#bundle.#config, values)
          │                      │    consumer values vs. bundle schema
          └──────────┬───────────┘
                     │ pass
                     ▼
          ┌──────────────────────┐
          │ releasesVal.Err()    │  ← comprehension-level CUE errors
          └──────────┬───────────┘
                     │
                     │ extractBundleReleases
                     │ (per release entry, sorted)
                     ▼
          ┌──────────────────────────────────────────┐
          │  for each release key ("server", "proxy")  │
          │                                           │
          │  schemaEntry.Err()                        │ ← per-entry CUE errors
          │                                           │
          │  ┌─────────────────────────────────────┐  │
          │  │  Module Gate                         │  │ ← validateConfig(
          │  │  (per release)                       │  │     #module.#config,
          │  └─────────────────────────────────────┘  │     values)
          │                 │ pass                     │   instance values vs.
          │                 ▼                         │   module schema
          │  schemaEntry.Validate(cue.Concrete(true)) │ ← whole-release concreteness
          │                 │                         │
          │                 ▼                         │
          │  finalizeValue(components)                │ ← finalize components only
          │                 │                         │   (values carries #config
          │                 ▼                         │    validators; skip it)
          │  decodeModuleReleaseEntry                 │ ← decode Go struct
          └──────────────────────┬───────────────────┘
                                 │ (repeat for each release)
                                 ▼
          ┌──────────────────────────────────────────┐
          │  BundleRelease{                           │
          │    Metadata:  BundleReleaseMetadata       │
          │    Bundle:    Bundle{Metadata, Raw}       │
          │    Schema:    pkg (whole CUE value)       │
          │    Releases: {                            │
          │      "server": *ModuleRelease{...}        │
          │      "proxy":  *ModuleRelease{...}        │
          │    }                                      │
          │  }                                        │
          └──────────────────────┬───────────────────┘
                                 │
                                 │ engine.NewBundleRenderer.Render
                                 ▼
          ┌──────────────────────────────────────────┐
          │  for each release key (sorted)            │
          │                                           │
          │  moduleRenderer.Render(modRel)            │ ← same flow as standalone
          │    Phase 1 — Match (buildMatchPlan)       │   ModuleRelease rendering
          │    Phase 2 — Execute (executeTransforms)  │
          │                                           │
          │  collect resources + warnings             │
          └──────────────────────┬───────────────────┘
                                 │
                                 ▼
          ┌──────────────────────────────────────────┐
          │  BundleRenderResult{                      │
          │    Resources:    []*core.Resource         │ ← all releases combined
          │    Warnings:     []string                 │
          │    ReleaseOrder: []string                 │ ← sorted keys for display
          │  }                                        │
          └──────────────────────────────────────────┘
```

---

## Schema vs. Data Components

Every `ModuleRelease` (whether standalone or from a bundle) carries components in
two forms:

```text
ModuleRelease
├── Schema         (original constrained CUE value — the full release)
│   └── components
│       └── server
│           ├── #resources: ["Deployment", "Service"]   ← definition field
│           ├── #traits:    ["opm.dev/observable"]      ← definition field
│           └── spec: { ... }                           ← regular fields
│
└── DataComponents (finalized, constraint-free CUE value)
    └── server
        └── spec: { ... }                               ← no definition fields
```

The distinction is required because the two phases of the engine need different things:

| Phase   | Uses           | Why                                                              |
|---------|----------------|------------------------------------------------------------------|
| Match   | Schema         | `#resources` and `#traits` must be present for CUE matching      |
| Execute | DataComponents | `FillPath` on `#component` fails with schema constraints present |
| Context | Schema         | `metadata.labels` and `metadata.annotations` are on schema comp  |

`finalizeValue` (using `Syntax(cue.Final()) + BuildExpr`) is what produces
`DataComponents` from the original value. It strips `matchN` validators, `close()`
enforcement, and definition fields, and takes defaults.

---

## Gate Positions in Context

The two gates are positioned to catch the most common user errors as early as
possible, before any expensive finalization or rendering work:

```text
BundleRelease path:
  pkg.Err()                           — structural CUE evaluation error
  extractMetadata                     — safe: reads concrete fields only
  Bundle Gate  ◄──────────────────── consumer values vs. #bundle.#config
  releasesVal.Err()                   — comprehension errors
  per release:
    schemaEntry.Err()                 — entry-level CUE error
    Module Gate  ◄────────────────── instance values vs. #module.#config
    schemaEntry.Validate(Concrete)    — remaining open fields
    finalizeValue + decode

ModuleRelease path:
  releaseVal.Err()                    — structural CUE evaluation error
  Module Gate  ◄────────────────────  consumer values vs. #module.#config
  releaseVal.Validate(Concrete)       — remaining open fields
  extractMetadata
  finalizeValue
```

See [validation-gates.md](validation-gates.md) for the full gate specification.

---

## Key Packages

| Package                    | Role                                              |
|----------------------------|---------------------------------------------------|
| `cmd/main.go`              | Entry point: load, detect kind, dispatch          |
| `pkg/loader`               | CUE load, gate validation, Go struct extraction   |
| `pkg/loader/validate.go`   | `validateConfig`, `ConfigError`                   |
| `pkg/loader/module_release.go` | `LoadReleasePackage`, `LoadModuleReleaseFromValue` |
| `pkg/loader/bundle_release.go` | `LoadBundleReleaseFromValue`, `extractBundleReleases` |
| `pkg/engine/module_renderer.go` | `ModuleRenderer.Render`                      |
| `pkg/engine/bundle_renderer.go` | `BundleRenderer.Render`                      |
| `pkg/engine/matchplan.go`  | `buildMatchPlan`, `MatchPlan`, `MatchedPairs`     |
| `pkg/engine/execute.go`    | `executeTransforms`, `executePair`, `injectContext` |
