## Context

The current PREPARATION phase of `pipeline.Render()` uses two standalone functions from `internal/build/module/`:

- `module.ResolvePath(modulePath)` — validates directory and `cue.mod/` presence, returns abs path
- `module.InspectModule(modulePath, registry)` + `module.ExtractMetadata(cueCtx, modulePath, registry)` — AST inspection with a CUE fallback for when `metadata.name` was not a string literal

The pipeline then manually assembles a `module.MetadataPreview` from inspection results, resolves the release name and namespace from it, and passes the now-resolved path and package name into `release.Builder.Build()`.

`core.Module` exists as a data type but has no behavior and is not returned by the module loader — `module.InspectModule` returns `*module.Inspection` and `module.ExtractMetadata` returns `*module.MetadataPreview`, both build-internal types that the pipeline must bridge manually.

This change adds `ResolvePath()` and `Validate()` receiver methods to `core.Module` and introduces a `module.Load()` free-standing constructor function that returns a fully populated `*core.Module`, replacing the fragmented inspection path.

## Goals / Non-Goals

**Goals:**

- `core.Module` owns its own path resolution (`ResolvePath()`) and structural validation (`Validate()`)
- `internal/build/module/` exposes a single `Load()` function returning `*core.Module`
- `pipeline.Render()` PREPARATION phase reads as a clean sequence: load → validate
- Remove the `ExtractMetadata` CUE fallback and `MetadataPreview` type — both are unnecessary given `metadata.name!` is a mandatory CUE field
- No observable behavior change — same errors, same resolved paths, same metadata values

**Non-Goals:**

- Consolidating `build/component.Component` with `core.Component` (separate change)
- Any changes to Phase 2 (release building) or later phases
- Namespace resolution precedence changes (handled in `namespace-resolution-precedence` change)

## Decisions

### Decision 1: ResolvePath mutates ModulePath in-place

`ResolvePath()` converts `ModulePath` to an absolute path and updates `Module.ModulePath` directly rather than returning the resolved path as a string.

**Why**: The module knows its own path. Returning a new string would require the caller to re-assign it and maintain the invariant manually. In-place mutation keeps the module self-consistent and makes downstream code (`module.Load()`, `release.Build()`) simply read `mod.ModulePath` without needing a separate variable.

**Alternative considered**: Return `(string, error)` like the old `module.ResolvePath()`. Rejected — callers would need to track the resolved path separately from the module, recreating the exact coupling we're removing.

### Decision 2: module.Load() calls ResolvePath() internally

`module.Load(cueCtx, modulePath, registry)` constructs `Module{ModulePath: modulePath}`, calls `mod.ResolvePath()`, and returns early on error. The caller (`pipeline.Render()`) never calls `ResolvePath()` directly.

**Why**: `Load()` is the primary construction path. Having it call `ResolvePath()` means a returned `*core.Module` always has a validated, absolute `ModulePath` — it is an invariant of the type after construction via `Load()`. `ResolvePath()` is still public so it can be called independently when a `Module` is constructed without `Load()` (e.g., test fixtures, future commands that don't go through the pipeline).

**Alternative considered**: Require callers to call `ResolvePath()` before `Load()`. Rejected — it fragments the construction protocol and puts the burden on every caller to know the correct sequence.

### Decision 3: Validate() is structural only, no CUE evaluation

`Validate()` checks `ModulePath != ""`, `Metadata != nil`, `Metadata.Name != ""`, `Metadata.FQN != ""`. It does not call `cue.Value.Validate()` on `Config` or `Values`.

**Why**: At the end of the PREPARATION phase, `Config` and `Values` may not yet be concrete — they are populated during the BUILD phase (overlay + user values). Enforcing CUE concreteness here would be expensive and out of place. The structural check is sufficient to guard against a malformed `core.Module` reaching `release.Build()`.

### Decision 4: ExtractMetadata CUE fallback is removed entirely

The `module.ExtractMetadata()` function and the `module.MetadataPreview` type are deleted. `module.Load()` uses only AST inspection to populate `Metadata.Name` and `Metadata.DefaultNamespace`.

**Why**: `metadata.name!` is a mandatory required field in the CUE `#Module` schema (enforced by the `!` suffix). A module without a name cannot pass CUE validation and will never reach a valid state. If AST inspection returns an empty name (because the value is a computed expression rather than a string literal), that is an edge case the module author should avoid — and the full CUE build in Phase 2 will correctly populate all metadata from the evaluated value anyway. The PREPARATION phase only needs name to derive the release name before Phase 2; if AST inspection misses a computed name, Phase 2 will catch it. Adding a full CUE eval fallback in PREPARATION for this edge case is not worth the complexity.

```text
module.Load(cueCtx, modulePath, registry):
  1. mod.ResolvePath()
  2. AST inspection → populate mod.Metadata.Name, DefaultNamespace, PkgName
  3. Return populated *core.Module
```

**Alternative considered**: Keep the CUE fallback for computed `metadata.name` values. Rejected — the CUE schema guarantees name is present and valid; a module with a computed name that can't be read by AST is an unusual authoring pattern that Phase 2 handles correctly regardless.

### Decision 5: module.Load() is a free-standing constructor, not a receiver method

`module.Load()` lives in `internal/build/module/` as a package-level function, not as a method on `core.Module`.

**Why**: `Load()` requires `cuelang.org/go/cue/load` for `load.Instances()` and performs `os.Setenv("CUE_REGISTRY", ...)` — environment mutation that is side-effectful and belongs in a loader layer, not a domain type. Keeping `Load()` in `build/module/` preserves `core` as a package with only lightweight stdlib dependencies (`os`, `filepath` for `ResolvePath()`; no `cue/load` or `cue/ast`). The function is still freely callable by any future command that needs module metadata without running the full pipeline.

**Alternative considered**: `mod.Load(cueCtx, registry)` as a receiver method on `core.Module`. Rejected — would pull `cuelang.org/go/cue/load`, `cuelang.org/go/cue/ast`, and environment mutation into `core`, significantly changing the character of the package.

### Decision 6: Standalone module.ResolvePath() function is removed

The existing `func ResolvePath(modulePath string) (string, error)` in `internal/build/module/loader.go` is deleted. All callers are updated to use `module.Load()` or construct a `Module` and call `mod.ResolvePath()` directly.

**Why**: Keeping both would leave an inconsistent API — two ways to resolve a path, one of which doesn't return a `*core.Module`. The receiver method supersedes the standalone function for all existing use cases.

### Decision 7: pkgName stays as an internal load detail

The `Inspection.PkgName` field (the CUE package name) is currently passed from the inspection step into `release.Build()` via `release.Options.PkgName`. `module.Load()` will store it in an unexported field on `core.Module` accessible to `release.Build()` via a package-level accessor. The exact mechanism is an implementation detail resolved during tasks.

## Risks / Trade-offs

**`ResolvePath()` mutates the receiver** — mutation on a pointer receiver is idiomatic Go but requires callers to understand the side effect. Mitigated by clear documentation on the method and the fact that `Load()` is the primary caller.

**`core` package now has filesystem dependencies** — `ResolvePath()` imports `os` and `filepath`. `core` is currently a pure data package. This is an intentional trade-off: `Module` gains meaningful behavior at the cost of stdlib dependencies. `os` and `filepath` are stdlib, not external, so this does not introduce new dependency risk.

**AST inspection may miss computed `metadata.name`** — if a module uses a computed expression for `name`, AST inspection returns `""` and the returned `core.Module` will have an empty `Metadata.Name`. `Validate()` will catch this and return a fatal error before Phase 2. Module authors should use string literals for `metadata.name`. This is the intended behavior — the CUE fallback that previously papered over this is intentionally removed.

## Migration Plan

1. Add `ResolvePath()` and `Validate()` to `internal/core/module.go`
2. Add `module.Load()` to `internal/build/module/loader.go`; have it call `mod.ResolvePath()` then AST inspection only (no CUE fallback)
3. Delete `module.ExtractMetadata()` and `module.MetadataPreview` from `internal/build/module/`
4. Update `pipeline.Render()` PREPARATION phase to call `module.Load()` → `mod.Validate()` and remove direct calls to the old standalone functions
5. Delete standalone `module.ResolvePath()` function; update any remaining callers
6. Verify `task test` passes — no behavior change expected for modules with string-literal `metadata.name`

Rollback: the change is self-contained to `core/module.go`, `build/module/loader.go`, and `build/pipeline.go`. Reverting these three files restores the previous behavior completely.

## Open Questions

- **pkgName propagation**: Should `core.Module` carry an unexported `pkgName` field that `module.Load()` sets and `release.Build()` reads via an accessor, or should `Load()` return `(mod *core.Module, pkgName string, err error)` temporarily? The cleaner path is an unexported field with a package-level accessor, but worth confirming before implementation.

## Related Changes

- **`namespace-resolution-precedence`**: Depends on `module.Load()` introduced here to surface `module.metadata.defaultNamespace` from AST inspection. Also removes the `ExtractMetadata` fallback independently (but that removal is handled in this change). The namespace precedence logic itself (inserting `defaultNamespace` as step 3) is scoped to that change.
