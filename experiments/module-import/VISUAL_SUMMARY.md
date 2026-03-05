# Visual Summary: Flattened Modules + CUE Imports

## The Question

```
┌─────────────────────────────────────────────────────────────┐
│  Can we keep the flattened authoring style:                 │
│                                                             │
│    package jellyfin                                         │
│                                                             │
│    core.#Module        ← embedded at root, not nested       │
│    metadata: { ... }                                        │
│    #config: { ... }                                         │
│    #components: { ... }                                     │
│                                                             │
│  ...and ALSO make it importable via CUE modules?            │
└─────────────────────────────────────────────────────────────┘
```

## The Answer: YES ✓

```
┌───────────────────────────────────────────────────────────────────┐
│                     WHAT WORKS                                    │
├───────────────────────────────────────────────────────────────────┤
│                                                                   │
│  ✓ Flattened embedding works                                     │
│  ✓ Hidden fields (#config, #components) are accessible           │
│  ✓ #ModuleRelease integration works                              │
│  ✓ Values flow through to components correctly                   │
│                                                                   │
│  REQUIREMENT: Package must not have extra fields beyond #Module  │
│                                                                   │
└───────────────────────────────────────────────────────────────────┘
```

## The Constraint: `values.cue`

```
┌─────────────────────────────────────────────────────────────────────┐
│                        THE PROBLEM                                  │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  Package scope after loading:                                      │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │  [FROM core.#Module]                                      │    │
│  │  ├─ apiVersion                                            │    │
│  │  ├─ kind                                                  │    │
│  │  ├─ metadata                                              │    │
│  │  ├─ #config                                               │    │
│  │  ├─ #components                                           │    │
│  │  ├─ #policies?                                            │    │
│  │  └─ debugValues                                           │    │
│  │                                                           │    │
│  │  [FROM values.cue - EXTRA FIELD]                          │    │
│  │  └─ values: { ... }   ← NOT in #Module definition!       │    │
│  └───────────────────────────────────────────────────────────┘    │
│                                                                     │
│  When assigning to #module field:                                  │
│  ┌───────────────────────────────────────────────────────────┐    │
│  │  release: core.#ModuleRelease & {                         │    │
│  │      #module: importedPackage   ← Has "values" field      │    │
│  │                ▼                                           │    │
│  │        ERROR: #Module.values: field not allowed           │    │
│  │               (closedness violation)                       │    │
│  │  }                                                         │    │
│  └───────────────────────────────────────────────────────────┘    │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Why This Happens

```
┌──────────────────────────────────────────────────────────────────┐
│                   CUE CLOSEDNESS SEMANTICS                        │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  1. core.#Module is a DEFINITION (starts with #)                │
│     → Closed when referenced                                    │
│                                                                  │
│  2. values.cue writes to PACKAGE ROOT:                          │
│     package jellyfin                                            │
│     values: { ... }      ← Outside #Module scope               │
│                                                                  │
│  3. On import, package contains:                                │
│     #Module fields + extra "values" field                       │
│                                                                  │
│  4. When unified with #Module schema:                           │
│     Extra field rejected due to closedness                      │
│                                                                  │
│  EMBEDDING EXCEPTION applies at AUTHORING time, not IMPORT time │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## The Solution: `@if(dev)` Build Tag

```
┌──────────────────────────────────────────────────────────────────┐
│              USE @if(dev) TO CONDITIONALLY INCLUDE values.cue     │
├──────────────────────────────────────────────────────────────────┤
│                                                                  │
│  Module structure:                                              │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  jellyfin/                                             │    │
│  │  ├── cue.mod/module.cue                                │    │
│  │  ├── module.cue          ← package jellyfin; #Module   │    │
│  │  ├── components.cue      ← package jellyfin            │    │
│  │  └── values.cue          ← @if(dev) at top             │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  values.cue content:                                            │
│  ┌────────────────────────────────────────────────────────┐    │
│  │  @if(dev)                   ← CUE build attribute      │    │
│  │                                                         │    │
│  │  package jellyfin                                      │    │
│  │                                                         │    │
│  │  values: {                                             │    │
│  │      image: { tag: "latest" }                          │    │
│  │      replicas: 1                                       │    │
│  │  }                                                     │    │
│  └────────────────────────────────────────────────────────┘    │
│                                                                  │
│  LOCAL DEV (opm CLI with -t dev):                              │
│    → values.cue INCLUDED, validated against #config            │
│                                                                  │
│  IMPORTED AS DEPENDENCY (no tags):                             │
│    → values.cue EXCLUDED, clean #Module unification            │
│                                                                  │
│  Why this is the best solution:                                │
│  • Values validated during development (compile-time safety)   │
│  • Automatic exclusion on import (no manual steps)             │
│  • Works with pure CUE tooling (not just Go SDK)               │
│  • CUE-idiomatic (exactly what @if() was designed for)         │
│  • One line of ceremony for authors                            │
│                                                                  │
└──────────────────────────────────────────────────────────────────┘
```

## Comparison: Before vs After

```
┌────────────────────────────────────┬────────────────────────────────────┐
│          BEFORE (Current)          │        AFTER (Importable)          │
├────────────────────────────────────┼────────────────────────────────────┤
│                                    │                                    │
│  package main                      │  package jellyfin                  │
│  ↑ not importable                  │  ↑ importable                      │
│                                    │                                    │
│  core.#Module                      │  core.#Module                      │
│  metadata: { ... }                 │  metadata: { ... }                 │
│  #config: { ... }                  │  #config: { ... }                  │
│  #components: { ... }              │  #components: { ... }              │
│                                    │                                    │
│  values.cue:                       │  values.cue:                       │
│    values: { ... }                 │    @if(dev)          ← ADD THIS    │
│  ↑ no conditional loading          │    values: { ... }                 │
│                                    │  ↑ included in dev, excluded on    │
│                                    │    import (automatic)              │
│                                    │                                    │
│  Used by: CLI only                 │  Used by: CLI + CUE imports        │
│  Distribution: Manual/git          │  Distribution: CUE registry        │
│  values.cue always loaded          │  values.cue conditional on -t dev  │
│                                    │                                    │
└────────────────────────────────────┴────────────────────────────────────┘
```
┌────────────────────────────────────┬────────────────────────────────────┐
│          BEFORE (Current)          │        AFTER (Importable)          │
├────────────────────────────────────┼────────────────────────────────────┤
│                                    │                                    │
│  package main                      │  package jellyfin                  │
│  ↑ not importable                  │  ↑ importable                      │
│                                    │                                    │
│  core.#Module                      │  core.#Module                      │
│  metadata: { ... }                 │  metadata: { ... }                 │
│  #config: { ... }                  │  #config: { ... }                  │
│  #components: { ... }              │  #components: { ... }              │
│                                    │                                    │
│  values.cue:                       │  values.cue:                       │
│    values: { ... }                 │    values: { ... }                 │
│  ↑ published with module           │  ↑ dev-only (not published)        │
│                                    │                                    │
│  Used by: CLI only                 │  Used by: CLI + CUE imports        │
│  Distribution: Manual/git          │  Distribution: CUE registry        │
│                                    │                                    │
└────────────────────────────────────┴────────────────────────────────────┘
```

## Module Import Flow

```
┌─────────────────────────────────────────────────────────────────────┐
│                   CONSUMER USING IMPORTED MODULE                    │
├─────────────────────────────────────────────────────────────────────┤
│                                                                     │
│  1. Import the module                                              │
│     ┌──────────────────────────────────────────────────────┐      │
│     │  import "opmodel.dev/modules/jellyfin@v0"            │      │
│     └──────────────────────────────────────────────────────┘      │
│                                                                     │
│  2. Create a ModuleRelease                                         │
│     ┌──────────────────────────────────────────────────────┐      │
│     │  release: core.#ModuleRelease & {                    │      │
│     │      metadata: {                                     │      │
│     │          name:      "my-jellyfin"                    │      │
│     │          namespace: "media"                          │      │
│     │      }                                               │      │
│     │      #module: jellyfin  ← imported module            │      │
│     │      values: {          ← user config                │      │
│     │          port:     8096                              │      │
│     │          replicas: 1                                 │      │
│     │          media: {                                    │      │
│     │              tvshows: { size: "100Gi" }              │      │
│     │              movies:  { size: "200Gi" }              │      │
│     │          }                                           │      │
│     │      }                                               │      │
│     │  }                                                   │      │
│     └──────────────────────────────────────────────────────┘      │
│                                                                     │
│  3. CUE evaluates:                                                 │
│     • Unifies #module.#config with user values                    │
│     • Resolves components with concrete values                    │
│     • Computes UUID, labels                                       │
│                                                                     │
│  4. Result: Concrete K8s resources ready to apply                 │
│                                                                     │
└─────────────────────────────────────────────────────────────────────┘
```

## Summary

| Aspect | Status | Notes |
|--------|--------|-------|
| Flattened style | ✅ Works | Keep `core.#Module` at root |
| Package naming | ✅ Change needed | `package main` → `package <name>` |
| Hidden fields | ✅ Works | `#config`, `#components` accessible |
| `values.cue` | ✅ Use `@if(dev)` | Include in dev, exclude on import |
| #ModuleRelease | ✅ Works | Full integration tested |
| CUE registry | ✅ Compatible | Ready for distribution |

**Bottom line**: You can have your cake and eat it too. Flattened authoring + CUE imports + validated defaults = ✅

**The winning solution**: Add one line (`@if(dev)`) to `values.cue` and you get:
- Local dev: values.cue loaded and validated ✓
- On import: values.cue excluded automatically ✓
- No changes to #Module definition ✓
- Works with pure CUE tooling ✓
