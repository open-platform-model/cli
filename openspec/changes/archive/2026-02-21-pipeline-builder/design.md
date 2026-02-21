## Context

`internal/legacy/build/release/builder.go` constructs `*core.ModuleRelease` entirely in Go: it fills `#config` with user values, extracts `#components` via Go iteration, and calls `core.ComputeReleaseUUID()` to derive a UUID. Labels and metadata are assembled by Go logic. This duplicates CUE semantics in Go and drifts whenever `#ModuleRelease` in `opmodel.dev/core@v0` evolves.

The experiment `experiments/module-release-cue-eval` (Strategy B / Approach C) proves a superior mechanic: load `#ModuleRelease` directly from `opmodel.dev/core@v0` using the module's own pinned dependency cache, inject the module and user values via `FillPath`, and let CUE evaluate UUID, labels, components, and metadata natively. Go only reads back the resulting concrete values.

`internal/builder/` replaces `internal/legacy/build/release/` with this mechanic. No existing commands are modified — the new package is consumed by `internal/pipeline/` in a later change.

## Goals / Non-Goals

**Goals:**
- Implement Approach C: load `#ModuleRelease` from `opmodel.dev/core@v0` via the module dir, not a separate catalog load
- Expose a single `Build(ctx, mod, opts, valuesFiles)` function that returns `*core.ModuleRelease`
- Move value selection (file loading, unification, fallback) into a focused `values.go` file
- Validate values against `#config` before injection; validate concreteness of the result

**Non-Goals:**
- Replacing any command implementations — `internal/pipeline/` wires this in later
- Changing `*core.ModuleRelease` or `*core.Module` struct shapes (handled by `pipeline-core-raw`)
- Supporting modules that do not import `opmodel.dev/core@v0`

## Decisions

### Decision 1: Load `#ModuleRelease` from module dir, not a separate catalog

**Chosen:** Load `opmodel.dev/core@v0` using `load.Config{Dir: mod.ModulePath}`. CUE resolves the package from the module's pinned deps — no separate registry call, no dual-context problem.

**Alternative considered:** Load a local catalog copy alongside the module. Rejected: requires maintaining a catalog path, breaks portability, and was the approach Strategy A (dual-load) explored and abandoned.

**Rationale:** The module already declares its `opmodel.dev/core@v0` version in `cue.mod/module.cue`. Loading from within its directory is the canonical CUE idiom for accessing a dependency. The experiment confirms this works end-to-end.

### Decision 2: One `*cue.Context` passed in, used for both loads

**Chosen:** Accept `ctx *cue.Context` as a parameter to `Build`. Both the core load and the module load use the same context. `FillPath` requires this — values from different contexts cannot be unified.

**Alternative considered:** Create a fresh context inside `Build`. Rejected: the module's `cue.Value` (in `mod.Raw`) was built in the caller's context. A new internal context would make `FillPath` fail silently or panic.

**Rationale:** The experiment helper `buildRealModuleWithSchema` demonstrates this constraint explicitly — it receives `ctx` and uses it for both `BuildInstance` calls.

### Decision 3: `builder.go` + `values.go` split

**Chosen:** Two files in `internal/builder/`:
- `builder.go` — Approach C injection sequence (core load → schema extract → FillPath chain → concreteness check → read-back into `*core.ModuleRelease`)
- `values.go` — value selection: load files from disk, unify multiple files, fall back to `mod.Values`

**Alternative considered:** Single file. Rejected: file I/O for values loading is a distinct concern from pure CUE evaluation. The split mirrors the proposal and makes each file independently testable.

**Rationale:** The split reflects the two concerns cleanly. `builder.go` never touches the filesystem; `values.go` never knows about `#ModuleRelease`.

### Decision 4: FillPath order follows experiment

**Chosen:** Inject in this order:
1. `FillPath(cue.MakePath(cue.Def("module")), moduleVal)` — inject module as `#module`
2. `FillPath(cue.ParsePath("metadata.name"), ...)` — release name
3. `FillPath(cue.ParsePath("metadata.namespace"), ...)` — namespace
4. `FillPath(cue.ParsePath("values"), selectedValues)` — user values

**Alternative considered:** Injecting values before module. Rejected: `_#module: #module & {#config: values}` in the schema depends on `#module` being present first. The experiment's `fillRelease` helper documents this dependency explicitly.

**Rationale:** Matches the proven sequence from `helpers_test.go:fillRelease`. Order is observable and tested.

### Decision 5: Validate concreteness via `cue.Validate(cue.Concrete(true))` after FillPath

**Chosen:** After the full FillPath chain, call `result.Validate(cue.Concrete(true))`. Return a `*core.ValidationError` wrapping any CUE error if non-concrete.

**Alternative considered:** Check each component individually (as the legacy builder does). Rejected: the new builder delegates component extraction to CUE — checking top-level concreteness is sufficient and simpler.

**Rationale:** Spec requirement: "The builder SHALL validate that the `#ModuleRelease` value is fully concrete after injection." Single top-level check satisfies this with minimal code.

### Decision 6: Read-back uses `core.ExtractComponents` + direct field lookups

**Chosen:** After concreteness is confirmed, read back:
- `metadata.uuid`, `metadata.version`, `metadata.labels` — via `LookupPath` + typed decode
- `components` — via `core.ExtractComponents(result.LookupPath(cue.ParsePath("components")))`

**Alternative considered:** Decode the entire `#ModuleRelease` into a Go struct via `result.Decode(&rel)`. Rejected: requires the Go struct to match the CUE schema exactly; any schema evolution breaks decoding. Field-by-field read-back is more resilient.

**Rationale:** Minimises coupling between the Go struct and the CUE schema. Only the fields we actually need are extracted.

## Risks / Trade-offs

**`OPM_REGISTRY` / `CUE_REGISTRY` must be set at runtime**
→ The core load (`load.Instances([]string{"opmodel.dev/core@v0"}, ...)`) resolves against a registry. If neither env var is set, the load fails. Mitigation: document the requirement in the function godoc. The existing `registry` field in the legacy builder is already threaded through from commands — the caller sets it before invoking.

**Registry round-trip adds latency on first build**
→ CUE caches resolved module deps in the module cache. Subsequent calls within a session are served from cache. Mitigation: acceptable for CLI usage; document that cold starts require network access.

**FillPath injection is order-sensitive**
→ Injecting values before `#module` causes silent failure. Mitigation: fixed order is encoded in `builder.go` with a comment referencing Decision 4. The experiment test suite guards this.

**`mod.Raw` depends on `pipeline-core-raw`**
→ This change uses `mod.Raw` (the renamed public field). Until `pipeline-core-raw` lands, it must use `mod.CUEValue()`. Mitigation: apply `pipeline-core-raw` first, or use a build tag to bridge. Dependency order in the change list already sequences `pipeline-core-raw` before `pipeline-builder`.

## Migration Plan

1. Apply `pipeline-core-raw` first (renames `value` → `Raw`, removes accessors).
2. Create `internal/builder/values.go` — no external dependencies.
3. Create `internal/builder/builder.go` — depends on `internal/core/` and `values.go`.
4. `internal/pipeline/` (later change) replaces its call to `internal/legacy/build/release/` with `internal/builder/`.
5. No command changes; no rollback needed — the legacy path remains until `pipeline-orchestrator` wires everything together.

## Open Questions

- **Schema lookup path**: The experiment uses `coreVal.LookupPath(cue.ParsePath("#ModuleRelease"))`. Confirm the path is stable in `opmodel.dev/core@v0` before finalising.
- **Read-back field paths**: The exact CUE paths for `metadata.uuid`, `metadata.labels`, and `components` inside `#ModuleRelease` should be verified against the current `opmodel.dev/core@v0` schema before implementation.
