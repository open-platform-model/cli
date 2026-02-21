## Why

Provider and transformer definition loading is currently embedded inside `internal/legacy/build/transform/provider.go`, tightly coupled with component matching logic. A dedicated `internal/provider/` package gives it a clean, single responsibility: take a provider name and its CUE values from GlobalConfig, parse transformer definitions, and return a structured `*LoadedProvider`. This makes provider loading independently testable and reusable by both the matcher and the new pipeline.

## What Changes

- Create `internal/provider/provider.go` with `Load(name string, providers map[string]cue.Value) (*LoadedProvider, error)`
- Create `internal/provider/types.go` with `LoadedProvider`, `LoadedTransformer`, and transformer requirement types
- Supersedes provider loading logic from `internal/legacy/build/transform/provider.go`

## Capabilities

### New Capabilities

- `provider-loading`: Loading and parsing transformer definitions from provider CUE values in GlobalConfig, producing a structured `*LoadedProvider` ready for use in component matching

### Modified Capabilities

_None._

## Impact

- New package `internal/provider/` — no existing code modified
- Depends on: `internal/core/`, `cuelang.org/go/cue`
- Will be consumed by `internal/transformer/` and `internal/pipeline/` in later changes
- SemVer: **MINOR** — new internal package, no CLI behavior changes
