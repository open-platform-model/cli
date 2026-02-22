## Context

`internal/provider` currently owns provider loading and exports `*LoadedProvider{Name, []*core.Transformer}`. The pipeline (Phase 2) calls `provider.Load()` to get a `*LoadedProvider`, then manually converts the transformer slice into a map to construct a `*core.Provider` for Phase 4 matching. This two-step conversion is the only reason `LoadedProvider` exists.

`internal/loader` already fills the same architectural role for modules: it converts raw CUE values into fully-populated domain types (`*core.Module`). Both packages import `core` and `output` and operate on `*cue.Context` inputs. The only reason `LoadProvider()` couldn't live there previously was a non-issue: there was no prior attempt to unify them.

A subpackage approach (`internal/core/provider/`) was considered but rejected: `internal/output` imports `internal/core`, so any `core/*` subpackage importing `output` would create a cycle. Placing the logic in `internal/loader` has no such issue since `loader` already imports `output`.

## Goals / Non-Goals

**Goals:**
- Eliminate `internal/provider` and `LoadedProvider`
- `loader.LoadProvider()` returns `*core.Provider` with `CueCtx`, `Transformers`, and all `ProviderMetadata` fields populated
- Extract all provider metadata fields from CUE value using the same `LookupPath` pattern as `extractModuleMetadata`
- Simplify the pipeline: Phase 2 + Phase 4 setup collapse into one call
- Migrate tests from `internal/provider/` to `internal/loader/`

**Non-Goals:**
- Changing matching logic (`core.Provider.Match()` is untouched)
- Changing transformer parsing behavior or FQN format
- Changing pipeline output or user-facing error messages
- Supporting provider loading from paths (still CUE value map only)

## Decisions

### 1. `LoadProvider()` lives in `internal/loader/`, not a new subpackage

`internal/core/provider/` was considered. Rejected: `internal/output → internal/core` creates a cycle if any `core/*` subpackage imports `output`. `internal/loader` already imports both `core` and `output` with no cycle.

Additionally, `loader` is the natural home — it's already the place that converts raw CUE inputs into domain types. `Load()` → `*core.Module` and `LoadProvider()` → `*core.Provider` are a symmetric pair.

### 2. Return `*core.Provider` directly, not a new wrapper type

`LoadedProvider` exists solely to bridge between `provider.Load()` (returns slice) and `core.Provider` (needs map). Eliminating the wrapper and returning `*core.Provider` directly removes the conversion step and the intermediate type. The pipeline can call `.Match()` immediately.

### 3. Use config key as `Metadata.Name` fallback

Provider CUE values may not always have a `metadata.name` field (the test fixtures in `provider_test.go` don't). The config map key (the name used to look up the provider) is used as the fallback, matching how module loading handles missing metadata.

### 4. `CueCtx` is set inside `LoadProvider()`

Previously, the caller (pipeline) set `CueCtx` after constructing `*core.Provider` manually. `LoadProvider()` now sets it directly on the returned value, consistent with how `*core.Module` gets its context. Callers no longer need to know this field exists.

## Risks / Trade-offs

**Tests reference `Transformers` as a slice** → Mitigation: tests in `provider_test.go` access `lp.Transformers[0]` and `lp.Name`; these must be updated to map lookups (`lp.Transformers["deployment"]`) and `lp.Metadata.Name`.

**Provider CUE values without `metadata.*`** → Mitigation: config key is always used as name fallback; other fields default to zero values (`""`, `nil`). No error is returned for absent metadata.

**`internal/provider` import path disappears** → This is purely internal; no external callers exist. Only `internal/pipeline/pipeline.go` imports the package.

## Migration Plan

1. Add `internal/loader/provider.go` with `LoadProvider()` and all helper functions
2. Add `internal/loader/provider_test.go` (ported from `internal/provider/provider_test.go`, updated for map access and `Metadata.Name`)
3. Update `internal/pipeline/pipeline.go`: swap import, remove 6-line conversion bridge
4. Delete `internal/provider/` entirely
5. Run `task test` and `task lint` to verify
