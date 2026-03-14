## Why

`module.Release` currently serves as both parsed input and mutable processing state. The `RawCUE` field changes meaning mid-pipeline — it starts as the raw parsed release CUE value, then gets overwritten with values-filled concrete CUE during processing. Fields like `DataComponents` and `Values` start as zero values and are populated later, so the struct is partially initialized for most of its lifetime. The `Config` field duplicates `Module.Config`. There is no clear boundary between "preparing a release" and "rendering a release" — `ProcessModuleRelease` does both config validation and transform execution in one call.

## What Changes

- **Simplify `module.Release`**: Remove `RawCUE`, `DataComponents`, `Config`. Replace with `Spec` (concrete, values-filled release definition) and `Values` (concrete merged values). Access config schema via `Module.Config`.
- **Introduce `ParseModuleRelease`**: New public function in `pkg/module` that validates values against `Module.Config`, merges them, fills them into the release spec, ensures concreteness, decodes metadata, and constructs a fully prepared `*module.Release`.
- **Simplify `ProcessModuleRelease`**: Assumes a prepared `*module.Release` (values already validated and filled). Derives finalized components transiently, matches against the provider, executes transforms, and returns `*render.ModuleResult`. No longer validates config or manages values.
- **Remove `ExecuteComponents()` method**: Finalized components are transient local variables inside `ProcessModuleRelease`, not stored on `Release`.
- **Keep `MatchComponents()` method**: Still reads schema-preserving components from `Release.Spec` via `LookupPath`.
- **Update `internal/releasefile`**: `GetReleaseFile` returns raw release parse data; it no longer constructs a fully prepared `*module.Release`.
- **Simplify `internal/workflow/render`**: Orchestration calls `ParseModuleRelease` then `ProcessModuleRelease` — clear preparation/rendering boundary.
- **Update all consumers**: `internal/releasefile`, `internal/workflow/render`, `pkg/render`, `pkg/bundle` all adopt the simplified type and pipeline.

## Capabilities

### New Capabilities

- `module-release-parsing`: Introduces `ParseModuleRelease` — the public preparation API that validates, merges, fills, and constructs a fully prepared `*module.Release`.

### Modified Capabilities

- `module-release-type`: `module.Release` is simplified to four fields (`Metadata`, `Module`, `Spec`, `Values`) with a clear invariant: `Spec` is concrete and complete, `Values` is concrete and merged.
- `module-release-processing`: `ProcessModuleRelease` accepts a prepared `*module.Release` and a `*provider.Provider`, returns `*render.ModuleResult`. Config validation is no longer its responsibility.
- `engine-rendering`: `Module.Execute()` and internal execution functions (`executeTransforms`, `executePair`, `injectContext`) continue to accept `*module.Release`. Field access changes from `rel.RawCUE` to `rel.Spec`. Finalized components are passed as local arguments, not read from the release.
- `release-file-loading`: `GetReleaseFile` returns raw release parse data suitable for input to `ParseModuleRelease`, not a fully constructed `*module.Release`.

## Impact

- **`pkg/module/release.go`**: Simplified `Release` struct, new `ParseModuleRelease` function, removed `NewRelease` constructor, removed `ExecuteComponents()`.
- **`pkg/render/process_modulerelease.go`**: Signature changes — accepts `*module.Release` (prepared), returns `*ModuleResult`. No longer calls `ValidateConfig`.
- **`pkg/render/module_renderer.go`**: `Module.Execute()` field access updates (`RawCUE` → `Spec`). Finalized components passed as argument rather than read from `rel.ExecuteComponents()`.
- **`pkg/render/execute.go`**: `executeTransforms`, `executePair`, `injectContext` field access updates.
- **`pkg/bundle/release.go`**: `Releases` map type stays `map[string]*module.Release`.
- **`internal/releasefile/get_release_file.go`**: Returns raw parse result instead of constructing `*module.Release`.
- **`internal/workflow/render/render.go`**: Orchestration restructured — calls `ParseModuleRelease` then `ProcessModuleRelease`.
- **Tests**: Release construction sites across test files need updating.
- **SemVer**: MINOR — `pkg/module` is a public package but not yet consumed externally.
- **No behavior change**: Render output, value precedence, matching, and apply behavior are all unchanged.
