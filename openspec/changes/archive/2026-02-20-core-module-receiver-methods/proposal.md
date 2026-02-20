## Why

The `core.Module` type is a passive data container today — construction, path resolution, and validation are all scattered across standalone functions in `internal/build/module/`. Moving this behavior onto receiver methods makes the type self-describing and gives the build pipeline a cleaner, more readable orchestration flow. An investigation of the CUE `#Module` schema also revealed that the existing `ExtractMetadata` CUE fallback is unnecessary, since `metadata.name!` is a mandatory field.

## What Changes

- Add `ResolvePath() error` receiver method to `core.Module` — validates and resolves `ModulePath` in-place (abs path, directory exists, `cue.mod/` present); filesystem-only, no CUE evaluation
- Add `Validate() error` receiver method to `core.Module` — structural validation only (non-nil `Metadata`, non-empty `Name`/`FQN`, non-empty `ModulePath`); does not enforce CUE concreteness
- Introduce `module.Load(cueCtx, modulePath, registry) (*core.Module, error)` in `internal/build/module/` — free-standing constructor that calls `mod.ResolvePath()` then AST inspection, returns a populated `*core.Module`
- Remove `module.ExtractMetadata()` — the CUE evaluation fallback for `metadata.name` is unnecessary; `name!` is mandatory in the CUE schema and a computed name is an unsupported authoring pattern
- Remove `module.MetadataPreview` type — no longer needed once `Load()` returns `*core.Module` directly
- Remove standalone `module.ResolvePath(string)` function — superseded by the `Module.ResolvePath()` receiver method
- `build/pipeline.go` PREPARATION phase simplified to: `module.Load()` → `mod.Validate()`

## Capabilities

### New Capabilities

- `module-receiver-methods`: Receiver methods on `core.Module` for path resolution and structural validation; `module.Load()` constructor in `internal/build/module/`

### Modified Capabilities

- `render-pipeline`: The PREPARATION phase of `pipeline.Render()` delegates path resolution and module validation to receiver methods on `core.Module`; the `ExtractMetadata` CUE fallback is removed

## Impact

- `internal/core/module.go` — new `ResolvePath()` and `Validate()` methods; gains `os`/`filepath` stdlib imports
- `internal/build/module/loader.go` — new `Load()` function returning `*core.Module`; `ResolvePath()`, `ExtractMetadata()`, and `InspectModule()` standalone functions removed or consolidated; `MetadataPreview` type removed
- `internal/build/pipeline.go` — PREPARATION phase updated; `resolveNamespace()` and related metadata bridging code simplified
- Related change: `namespace-resolution-precedence` depends on `module.Load()` surfacing `Metadata.DefaultNamespace`
- SemVer: **PATCH** — internal refactor, no change to CLI behavior or public-facing pipeline interface
