## Why

`internal/loader` now hosts both the module loader and the provider loader (after the provider loader was moved from `internal/provider`). Both files are in the same package, and the module loader's top-level function is named `Load`, which conflicts with naming clarity and would have conflicted with `LoadProvider` at the package level. Renaming `Load` to `LoadModule` makes the API self-documenting and creates a consistent parallel: `LoadModule` / `LoadProvider`.

## What Changes

- `func Load(...)` in `internal/loader/module.go` is renamed to `func LoadModule(...)`
- All call sites updated: `loader.Load(...)` → `loader.LoadModule(...)`
- Doc comment and inline references updated to match

## Capabilities

### New Capabilities
<!-- None — this is a pure rename refactor -->

### Modified Capabilities

- `loader-api`: The exported function name changes from `Load` to `LoadModule`. **BREAKING** at the package API level (no external consumers; all callers are within this repository).

## Impact

- `internal/loader/module.go` — function signature (renamed from `loader.go`)
- `internal/loader/module_test.go` — 13 call sites (renamed from `loader_test.go`)
- `internal/pipeline/pipeline.go` — 1 call site + 1 comment
- `internal/builder/builder_test.go` — 4 call sites
- `tests/integration/values-flow/main.go` — 1 call site
- `experiments/values-flow/helpers_test.go` — 2 comment references
- SemVer: PATCH (internal refactor; no user-facing behavior change)
