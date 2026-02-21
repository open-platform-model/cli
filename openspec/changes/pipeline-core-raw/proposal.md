## Why

`core.Module` stores its evaluated CUE value behind a private field accessed through `CUEValue()`/`SetCUEValue()` methods. The new pipeline design treats this value as a first-class field — `Raw` — passed directly between phases. Making it an explicit public field removes indirection, clarifies intent, and prepares `core.Module` for use in the new phase-structured pipeline.

## What Changes

- Rename private `value cue.Value` field → `Raw cue.Value` (public field, direct access)
- **BREAKING** (internal): Remove `CUEValue()` and `SetCUEValue()` accessor methods
- Update all callers in `internal/legacy/` and `internal/core/` to use `mod.Raw` directly

## Capabilities

### New Capabilities

_None. This is a foundational type change, not a new capability._

### Modified Capabilities

_None. No spec-level behavior changes — the same CUE value is stored, just named and accessed differently._

## Impact

- `internal/core/module.go` — field rename + method removal
- `internal/legacy/module/loader.go` — `mod.SetCUEValue(v)` → `mod.Raw = v`
- `internal/legacy/release/builder.go` — `mod.CUEValue()` → `mod.Raw`
- `internal/legacy/transform/executor.go` — any direct CUE value access
- Related tests in `internal/core/` and `internal/legacy/`
- SemVer: **PATCH** — internal type change, no CLI behavior changes
