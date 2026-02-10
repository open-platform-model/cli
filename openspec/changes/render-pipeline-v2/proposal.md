## Why

The render pipeline has two problems that share a root cause: the Go code reimplements logic that CUE already provides, and a single shared `*cue.Context` is used across concurrent goroutines.

1. **ReleaseBuilder reimplements `#ModuleRelease`**. The Go code manually does `FillPath(#config, values)`, `Validate(cue.Concrete(true))`, and extracts metadata field-by-field. The CUE `#ModuleRelease` definition already handles value validation (`close(#module.#config)`), config injection (`_#module: #module & {#config: values}`), component concreteness (`components: _#module.#components`), and identity computation (`uuid.SHA1(...)`). The Go code should construct a `#ModuleRelease` in CUE and extract results — not reimplement the logic.

2. **Executor panics on concurrent CUE evaluation**. Worker goroutines call `FillPath` on CUE values sharing a single `*cue.Context`, triggering a panic in CUE's internal scheduler (`adt/sched.go:483`). CUE values from the same context share mutable internal state. This blocks any module with more than one component from rendering in parallel.

Both problems are addressed by: (1) using a CUE overlay to compute release metadata via the CUE `uuid` package (Phase 3), and (2) running executor jobs sequentially to eliminate the concurrency panic (Phase 5).

## What Changes

- **ReleaseBuilder uses CUE overlay for release metadata**: The ReleaseBuilder uses `load.Config.Overlay` to inject a virtual CUE file (`opm_release_overlay.cue`) that computes release identity (`uuid.SHA1`) and labels in CUE. Config injection (`FillPath(#config, values)`) and concreteness validation remain in Go — a hybrid approach necessitated by a CUE SDK limitation where `close()` panics on pattern constraints.
- **Executor runs jobs sequentially**: The worker pool and goroutines were removed. All transformer jobs run sequentially in a simple loop on the shared `*cue.Context`. CUE value serialization (`Syntax()`) was attempted but panics on transformer values with cross-package references. Sequential execution is the correct fix because CUE evaluation dominates runtime and cannot be parallelized within a single context.
- **ModuleLoader removed**: The `ModuleLoader` type is replaced by a lightweight `extractModuleMetadata` helper on the pipeline and the `ReleaseBuilder.Build()` method which accepts a module path directly.
- **No API/flag/command changes**: `Pipeline`, `Executor`, and all command interfaces remain unchanged. This is an internal refactor and bugfix.

**SemVer: PATCH** — no user-facing API, flag, or behavioral changes. Fixes a runtime panic.

## Capabilities

### New Capabilities

_None — this is a refactor and bugfix, not a new capability._

### Modified Capabilities

- `render-pipeline`: Phase 3 (ReleaseBuilder) uses a CUE overlay for release metadata computation (identity, labels) while keeping Go-side `FillPath` for config injection (hybrid approach). Phase 5 (Executor) changes from parallel worker pool to sequential execution on the shared `*cue.Context`. Phase 2 changes from `ModuleLoader` to lightweight metadata extraction. Pipeline phases: Phase 0 (ConfigLoader), Phase 1 (ProviderLoader), Phase 2 (Metadata Extraction), Phase 3 (ReleaseBuilder), Phase 4 (Matcher), Phase 5 (Executor).
- `build`: The build spec references ReleaseBuilder and execution behavior. Updated to reflect hybrid overlay release building and sequential executor.

## Impact

- **Affected packages**: `internal/build/release_builder.go` (primary — hybrid overlay for metadata, FillPath for config injection), `internal/build/executor.go` (primary — sequential execution, removed worker pool), `internal/build/pipeline.go` (rewritten — removed ModuleLoader dependency, added lightweight metadata extraction), `internal/build/module.go` (cleanup — removed dead ModuleLoader/LoadedModule code, kept LoadedComponent)
- **Affected commands**: All commands using the render pipeline (`mod apply`, `mod build`, `mod diff`) benefit from the fix. No command code changes required.
- **Dependencies**: No new Go dependencies. Uses existing CUE SDK APIs: `load.Config.Overlay`, `cue.FillPath`, `cue.Validate`. Removed: `cue/format`, `cue/cuecontext` (from executor), `sync` (from executor), `runtime` (from pipeline).
- **CUE dependency**: The overlay imports the CUE `uuid` package (stdlib) for identity computation. Modules must import `opmodel.dev/core@v0` (already the case — provides `metadata.fqn`, `metadata.version`).
- **Performance**: Phase 3 adds a `load.Instances` call with overlay (module dependencies cached, minimal cost). Phase 5 changes from parallel to sequential — acceptable because CUE evaluation dominates runtime and typical job counts are 5-15.
- **Risk**: Low — same 4 pre-existing transform errors on jellyfin; no new regressions. All tests pass. No panics.
