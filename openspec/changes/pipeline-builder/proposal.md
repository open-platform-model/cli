## Why

The current `internal/legacy/build/release/builder.go` constructs `*core.ModuleRelease` by reimplementing in Go what CUE can evaluate natively — UUID derivation, label computation, and metadata propagation. The new `internal/builder/` package uses Approach C: load `#ModuleRelease` from `opmodel.dev/core@v0` (resolved from the module's own pinned dependencies), inject the module and values via `FillPath`, and let CUE evaluate the full release. Go only reads back the resulting concrete values.

## What Changes

- Create `internal/builder/builder.go` with `Build(ctx *cue.Context, mod *core.Module, opts Options, valuesFiles []string) (*core.ModuleRelease, error)`
  - Loads `#ModuleRelease` schema from `opmodel.dev/core@v0` using the module's dependency cache
  - Injects module (`mod.Raw`), release name, namespace, and selected values via `FillPath`
  - Validates concreteness of the resulting `#ModuleRelease`
  - Reads back metadata, components, and values into `*core.ModuleRelease`
- Create `internal/builder/values.go` with value selection logic: load and unify `--values` files, or fall back to `mod.Values`
- Supersedes `internal/legacy/build/release/`

## Capabilities

### New Capabilities

- `release-building`: Building a concrete `*core.ModuleRelease` from a loaded module using CUE-native evaluation via `#ModuleRelease` injection (Approach C), where UUID, labels, and metadata are derived by CUE rather than Go

### Modified Capabilities

_None._

## Impact

- New package `internal/builder/` — no existing code modified
- Depends on: `internal/core/`, `internal/loader/`, `cuelang.org/go/cue/load`
- Requires `OPM_REGISTRY` / `CUE_REGISTRY` to be set for `opmodel.dev/core@v0` resolution
- Will be consumed by `internal/pipeline/` in a later change
- SemVer: **MINOR** — new internal package, no CLI behavior changes
