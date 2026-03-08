## Context

`opm mod build` and `opm mod apply` call `cmdutil.RenderRelease()`, which always calls `loader.LoadReleasePackage()`. That function resolves a `release.cue` file from the given path. When none exists, it fails immediately. Module authors who only have `module.cue`, `components.cue`, and `debugValues` are blocked from using `build` or `apply` locally.

`opm mod vet` already handles this in two ways:
- **Module-only mode** (no `release.cue`): validates `#config` against `debugValues` or `-f`, no render pipeline
- **Release mode** (has `release.cue`): passes `DebugValues: len(rf.Values) == 0` to `RenderRelease`, which fills the release package with the module's `debugValues` when no `-f` is given

The key CUE insight driving the synthesis approach (from `catalog/v0/core/module_release.cue`):

```cue
_#module:   #module & {#config: values}   // fill #config with concrete values
components: _#module.#components          // components evaluate with concrete config
```

This means: filling `#config` with `debugValues` on the module value (`modVal.FillPath("#config", debugValues)`) produces a fully evaluable module whose `#components` field is concrete. We can build a valid `*modulerelease.ModuleRelease` from this without any `release.cue`.

## Goals / Non-Goals

**Goals:**
- `opm mod build .` and `opm mod apply .` work on a bare module directory using `debugValues` or `-f`
- `opm mod build/apply` default to `debugValues` when no `-f` flag is given, whether or not `release.cue` exists (consistent with how `vet` works in release mode)
- Clear error when no `release.cue`, no `debugValues`, and no `-f` flag
- No new flags; the existing `DebugValues` opt and `hasReleaseFile` detection are sufficient

**Non-Goals:**
- Changing `opm mod vet` behavior (unchanged)
- Changing any `opm release *` commands
- Computing UUIDs for synthesized releases (empty UUID; `apply` inventory degrades gracefully)
- Supporting `BundleRelease` synthesis

## Decisions

### 1. `SynthesizeModuleRelease` lives in `pkg/loader/`

The synthesis logic needs `validateConfig`, `finalizeValue`, and `extractModuleInfo` — all private functions in `pkg/loader/`. Placing the new function there avoids exporting private helpers or creating circular imports.

**Alternative considered**: A standalone `pkg/synthesizer/` package. Rejected — added a package boundary for code that tightly couples to loader internals.

### 2. Auto-detect via `hasReleaseFile()` — no new flag

`RenderRelease` inspects whether `release.cue` exists on disk. If absent, it takes the synthesis path. This is invisible to callers.

**Alternative considered**: A new `NoReleaseFile bool` field in `RenderReleaseOpts`. Rejected — every caller would need to add the check themselves, violating the "smart defaults" principle (VII).

### 3. `DebugValues: len(rf.Values) == 0` added to `build` and `apply`

When no `-f` is given, both commands now pass `DebugValues: true`. This is consistent with how `vet` already works in release mode. The `DebugValues` flag drives the extraction of `debugValues` from the loaded module value.

### 4. UUID left empty for synthesized releases

The release UUID is normally CUE-computed: `uid.SHA1(OPMNamespace, "\(fqn):\(name):\(namespace)")`. Replicating this in Go is possible but adds coupling to an implementation detail of the CUE catalog schema.

For `build`: UUID is only metadata in the manifest labels — not critical.
For `apply`: `apply.go:187` already guards on `releaseID != ""` before doing any inventory work. Empty UUID means inventory tracking is skipped gracefully.

**Alternative considered**: Compute UUID in Go. Deferred — can be added without breaking anything if inventory support for no-release-file modules becomes a requirement.

### 5. Namespace: module's `defaultNamespace` as synthesis fallback

For a synthesized release there is no `release.cue` to provide a namespace. The module's `metadata.defaultNamespace` is the right default. The existing post-synthesis `SourceFlag`/`SourceEnv` override is applied identically to how it works for normal releases — the release "owns" its namespace unless explicitly overridden.

### 6. `schemaComponents` sourced from `#components` (definition path)

`MatchComponents()` does `schema.LookupPath("components")` (regular field, no `#`). The module defines `#components` (a definition, `#` prefix). We wrap: `syntheticSchema.FillPath("components", filledMod.LookupPath("#components"))`. This preserves the full definition field structure (`#resources`, `#traits`, etc.) needed by the CUE match plan evaluator.

## Risks / Trade-offs

**[Risk] Synthesized releases produce no inventory UUID**
→ `apply` with a bare module directory will not track inventory state. Subsequent `apply` runs cannot prune stale resources. Module authors using this path for production deploys should create a proper `release.cue`.

**[Risk] `#components` lookup returns wrong value if module uses a non-standard field name**
→ Only modules following the standard `#Module` contract (defining `#components`) benefit from synthesis. Modules with non-standard layouts remain unaffected — they simply need a `release.cue`.

**[Risk] Module Gate validation path is slightly different**
→ In the normal release path, Module Gate uses `#module.#config` (reached through the release value). In synthesis, it uses `modVal.LookupPath("#config")` directly. These are equivalent — both point to the same `#config` definition — but any future divergence must be kept in sync.

## Migration Plan

No migration required. All changes are additive:
- Existing `release.cue`-based workflows are unchanged — `hasReleaseFile()` returns `true` and takes the existing path
- New module-only workflow activates only when `release.cue` is absent

No rollback strategy needed; the synthesis path is purely additive.

## Open Questions

None — design is fully resolved based on the existing codebase analysis.
