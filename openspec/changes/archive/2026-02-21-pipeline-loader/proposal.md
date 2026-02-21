## Why

The PREPARATION phase — loading a CUE module from disk into a fully-populated `*core.Module` — is currently buried inside `internal/legacy/build/module/`, entangled with pipeline orchestration. A dedicated `internal/loader/` package gives this phase a clean, focused contract: given a path and registry, resolve the module, evaluate its CUE, extract all fields, and return a ready `*core.Module`.

## What Changes

- Create `internal/loader/` package with a single entry point: `Load(ctx *cue.Context, modulePath, registry string) (*core.Module, error)`
- Implements all PREPARATION steps: path resolution, CUE instance loading, full evaluation, metadata extraction, `#config` extraction, `values` extraction, `#components` extraction, `Raw` field population
- Supersedes `internal/legacy/build/module/loader.go`

## Capabilities

### New Capabilities

- `module-loading`: Loading a CUE module from disk into a fully-populated `*core.Module` with all fields set — metadata, config schema, default values, components, and raw CUE value

### Modified Capabilities

_None._

## Impact

- New package `internal/loader/` — no existing code modified
- Depends on: `internal/core/`, `cuelang.org/go/cue/load`
- Will be consumed by `internal/pipeline/` in a later change
- SemVer: **MINOR** — new internal package, no CLI behavior changes
