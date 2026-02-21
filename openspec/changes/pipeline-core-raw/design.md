## Context

`core.Module` holds the fully-evaluated CUE value of a loaded module in a private field `value cue.Value`, exposed via `CUEValue()` and `SetCUEValue()` accessor methods. This was originally encapsulated to control access, but in the new phase-structured pipeline the evaluated CUE value is a first-class data item — passed from the loader directly to the builder as `mod.Raw`. The indirection adds noise without providing safety.

This change runs after `pipeline-legacy-move`, so all callers are in `internal/legacy/` at the time of implementation.

Current callers of `CUEValue()` / `SetCUEValue()`:
- `internal/legacy/module/loader.go` — `mod.SetCUEValue(baseValue)`
- `internal/legacy/release/builder.go` — `base := mod.CUEValue()`
- `internal/core/module.go` — method definitions + any internal usages

Note: `internal/build/transform/executor.go` was removed by `core-transformer-match-plan-execute` (already applied) and was never a caller of these methods. It will not be present in `internal/legacy/`.

## Goals / Non-Goals

**Goals:**
- Replace `value cue.Value` (private) with `Raw cue.Value` (public) on `core.Module`
- Remove `CUEValue()` and `SetCUEValue()` methods entirely
- Update all callers to direct field access (`mod.Raw = v`, `mod.Raw`)
- All existing tests pass without behavior change

**Non-Goals:**
- No changes to CUE evaluation logic
- No changes to what value is stored — same `cue.Value`, different name and access pattern
- No changes to any other `core.Module` fields or methods

## Decisions

### Public field over accessor methods

The original accessor methods suggest the field needs controlled access — but `cue.Value` is itself immutable (all CUE operations return new values). There is no mutation risk to guard against. A public `Raw cue.Value` field is simpler, more idiomatic Go for a plain data-carrying struct, and directly matches how the new pipeline phases will use it (`mod.Raw` passed as an argument).

Alternatives considered:
- **Keep methods, just rename**: `CUEValue()` → `Raw()`. Rejected — methods imply behaviour; a plain field is more honest about what this is.
- **Keep private, add `Raw` as alias**: Redundant. One field, one name.

### Naming: `Raw`

`Raw` signals "the unprocessed CUE value as loaded from disk" — the full evaluated `#Module` value before Go has extracted anything from it. It distinguishes clearly from the extracted Go fields (`Metadata`, `Config`, `Values`, `Components`) which are derived from `Raw`.

Alternatives considered:
- `Value` — too generic, clashes with CUE's own `cue.Value` type name in context
- `CUEValue` — redundant given the field type is already `cue.Value`
- `Evaluated` — accurate but verbose

## Risks / Trade-offs

- **Compile-time safety**: Removing the methods means any missed caller is a compile error, not a silent nil. This is a feature — the build will fail loudly if any call site is missed. → No mitigation needed; let the compiler catch it.
- **`internal/legacy/` dependency**: This change assumes `pipeline-legacy-move` has already been applied. If applied before the move, callers are in `internal/build/` instead. → Enforce ordering: apply `pipeline-legacy-move` first.
