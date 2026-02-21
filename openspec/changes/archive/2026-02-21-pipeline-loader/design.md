## Context

The PREPARATION phase of the render pipeline is currently implemented in `internal/build/module/loader.go`. It works correctly but is co-located with the legacy pipeline orchestration, making it hard to use independently or test in isolation.

The `pipeline-loader` change extracts this logic into a new `internal/loader/` package with a clean, documented contract. The implementation is mostly a lift-and-shift — the existing logic is already correct. The key decisions are about what the new package's boundaries look like and how it interacts with `core.Module`.

**Current state of `core.Module`**: The `value` field is private, accessed via `CUEValue()` / `SetCUEValue()`. The `pipeline-core-raw` change (a peer change) will rename this to a public `Raw cue.Value` field and remove the accessor methods. The loader design assumes `pipeline-core-raw` lands first — it writes `mod.Raw` directly rather than calling `SetCUEValue()`.

**`Validate()` constraint**: `core.Module.Validate()` currently checks `m.CUEValue().Exists()`. After `pipeline-core-raw`, this check will reference `m.Raw`. The loader must ensure `Raw` is set before returning so `Validate()` passes.

## Goals / Non-Goals

**Goals:**
- Single entry point: `Load(ctx *cue.Context, modulePath, registry string) (*core.Module, error)`
- All PREPARATION steps in one place: path resolution → CUE loading → evaluation → field extraction → `Raw` population
- Returned `*core.Module` always passes `mod.Validate()`
- No dependency on `internal/build/` or `internal/legacy/`

**Non-Goals:**
- Not responsible for constructing `#ModuleRelease` or injecting values (BUILD phase, `internal/builder/`)
- Not responsible for provider or transformer loading (separate packages)
- No caching — callers create a fresh CUE context per command per the project standard

## Decisions

### Decision 1: Lift-and-shift the existing loader logic

**Choice**: Copy `internal/build/module/loader.go` into `internal/loader/loader.go` with minimal changes (update imports, use `mod.Raw` instead of `SetCUEValue()`).

**Rationale**: The existing implementation is already correct and tested. The goal of this change is structural — establishing the new package boundary — not behavioral. Rewriting the logic introduces risk without benefit.

**Alternative considered**: Refactor the extraction logic (e.g., use a struct-based builder pattern). Rejected — YAGNI. The function-based approach is simple and sufficient.

---

### Decision 2: Keep `CUE_REGISTRY` via `os.Setenv` with `defer Unsetenv`

**Choice**: Set `CUE_REGISTRY` in the environment for the duration of `load.Instances`, then unset it via `defer`.

**Rationale**: `cuelang.org/go/cue/load` reads `CUE_REGISTRY` from the environment. There is no API to pass registry configuration directly to `load.Config`. This matches the existing approach in `internal/build/module/loader.go` and the experiment in `experiments/module-release-cue-eval/`.

**Risk**: Not goroutine-safe — concurrent calls to `Load` with different registries would race on the env var. Accepted: OPM commands are single-threaded; `Load` is called once per command.

---

### Decision 3: Keep `output.Debug` logging in the loader

**Choice**: Retain the `output.Debug(...)` call at the end of a successful load, logging path, name, fqn, version, defaultNamespace, and component count.

**Rationale**: This observability is useful during development and debugging. The `internal/output` package is a lightweight dependency with no circular issues.

**Alternative considered**: Remove the logging and let the caller log. Rejected — the caller doesn't have visibility into internal extraction details.

---

### Decision 4: `extractModuleMetadata` stays as a package-private helper

**Choice**: Keep `extractModuleMetadata` as an unexported function within `internal/loader/`.

**Rationale**: It is an implementation detail of `Load`. No other package needs to call it directly. If future changes need it, it can be promoted then (YAGNI).

---

### Decision 5: No `pkgName` extraction in the new loader

**Choice**: The new loader will NOT call `mod.SetPkgName()` / populate `mod.pkgName`.

**Rationale**: `pkgName` is populated in the legacy loader and passed to the release builder's CUE overlay. With Approach C (inject module via `FillPath`), the BUILD phase does not use `pkgName`. Once the legacy pipeline is removed, this field becomes dead. Populating it here would preserve a dependency on `SetPkgName()` that `pipeline-core-raw` should be free to remove.

**Risk**: If `pkgName` turns out to be needed by `internal/builder/`, this decision needs revisiting. Tracked as an open question.

## Risks / Trade-offs

**Race on `CUE_REGISTRY` env var** → Acceptable for single-threaded CLI use. If parallel loading is ever needed, switch to a subprocess or load.Config-level mechanism.

**Depends on `pipeline-core-raw` landing first** → If `pipeline-core-raw` is delayed, the loader can temporarily use `mod.SetCUEValue(baseValue)` and update to `mod.Raw` when the field is promoted. This is a one-line change.

**`mod.Validate()` reference to `CUEValue()`** → After `pipeline-core-raw`, `Validate()` must be updated to check `m.Raw.Exists()`. If it isn't, `Validate()` will always fail for modules loaded via the new loader. This must be coordinated in the `pipeline-core-raw` tasks.

## Open Questions

1. **Is `pkgName` needed by `internal/builder/`?** If Approach C via `FillPath` requires knowing the CUE package name for the overlay, the loader will need to populate it. Needs verification when implementing `pipeline-builder`.

2. **Should the loader accept a `*cue.Context` or create its own?** Current design accepts a caller-provided context (matching the legacy loader). This lets the orchestrator control context lifetime. If the loader creates its own, it would be simpler to call but less flexible. Decision: accept caller-provided context, consistent with existing usage.
