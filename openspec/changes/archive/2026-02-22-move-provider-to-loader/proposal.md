## Why

`internal/provider` is a thin adapter package whose only job is to parse CUE values into `*core.Transformer` slices — but `internal/loader` already does exactly this kind of work for modules. The package introduces an intermediate type (`LoadedProvider`) that forces the pipeline to manually convert a slice into a map just to construct a `*core.Provider`, creating unnecessary indirection. Consolidating provider loading into `internal/loader` removes a package, eliminates `LoadedProvider`, and gives the pipeline a ready-to-use `*core.Provider` directly.

## What Changes

- **New function** `loader.LoadProvider()` in `internal/loader/provider.go` — returns `*core.Provider` with all metadata fields, transformers, and `CueCtx` populated
- **Full metadata extraction** — `ProviderMetadata` fields (`name`, `description`, `version`, `minVersion`, `labels`) and root-level `apiVersion`/`kind` are now extracted from the CUE value (currently only `name` is set)
- **`internal/provider` package removed** — `LoadedProvider`, `types.go`, and `provider.go` deleted
- **Pipeline simplified** — manual `[]*core.Transformer` → `map[string]*core.Transformer` conversion bridge removed; Phase 2 and Phase 4 setup collapse into one call
- SemVer: **PATCH** — internal refactoring, no user-facing behavior changes

## Capabilities

### New Capabilities

- `provider-loader`: Loading a provider CUE value into a fully-populated `*core.Provider` via `loader.LoadProvider()`, including complete metadata extraction following the same pattern as `loader.Load()` for modules

### Modified Capabilities

- `provider-loading`: The loading API moves from `provider.Load() → *LoadedProvider` to `loader.LoadProvider() → *core.Provider`; all auto-selection and error behavior is preserved

## Impact

- `internal/provider/` — deleted entirely
- `internal/loader/` — new `provider.go` file added
- `internal/pipeline/pipeline.go` — import updated, 6-line conversion bridge removed
- `internal/core/provider.go` — untouched (types stay in place)
- All existing provider loading behavior (auto-select, error messages, debug logging) is preserved
