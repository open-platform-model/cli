## Context

The pipeline currently uses three separate service objects for the matching phase: a `ProviderLoader` that loads transformers from CUE values, a stateless `Matcher` that evaluates components against transformers, and an internal `MatchResult` type that bridges the two. The pipeline orchestrates these manually at `pipeline.go:153-170`.

The `Matcher` struct has no state — it is a bundle of functions attached to a zero-value struct solely to group related logic. The `MatchResult` internal type is an intermediate representation that exists only to bridge `Matcher.Match()` output to `Executor.ExecuteWithTransformers()` input. Neither type carries domain meaning.

`core.Provider` already exists as a domain type in `internal/core/provider.go` but is unused by the pipeline today — the pipeline works entirely with `transform.LoadedProvider` and `transform.LoadedTransformer`. The goal is to collapse the matching responsibility into `core.Provider`, making the provider the domain owner of "which of my transformers handle this component?"

The `GlobalConfig.CueContext` is the single CUE runtime for the entire process. All CUE values (`provider.Transformers[i].Value`, `component.Value`) are compiled against this context. The pipeline stores `p.cueCtx` and passes it explicitly to `Execute()` (added in `core-transformer-match-plan-execute`) for `cueCtx.Encode()` during transformer injection — neither `Provider` nor `TransformerMatchPlan` carry the context (see Decision 2).

## Goals / Non-Goals

**Goals:**

- Move matching logic from `transform/matcher.go` into `core.Provider.Match()`
- Return `*core.TransformerMatchPlan` from `Match()` instead of `*transform.MatchResult`
- Update `transform.LoadProvider()` (replacing `ProviderLoader.Load()`) to return `*core.Provider` with `Transformers` populated — no `cueCtx` parameter (see Decision 2)
- Update `pipeline.Render()` MATCHING phase to call `provider.Match(rel.Components)` directly
- Delete `transform/matcher.go` and the `Matcher` struct

**Non-Goals:**

- Migrating execution logic (`Executor`) — that is `core-transformer-match-plan-execute`
- Consolidating `build/component.Component` → `core.Component` — deferred change
- Changing the matching algorithm — same label/resource/trait checks, same O(n×m) loop
- Changing `RenderResult` shape or any user-facing output

## Decisions

### Decision 1: `Match()` takes `map[string]*Component`, not a slice

The current `Matcher.Match()` takes `[]*component.Component`. The pipeline converts the map to a slice at `pipeline.go:159`. With `Match()` on `core.Provider`, it is cleaner to accept `map[string]*Component` directly, matching the field type on `core.ModuleRelease.Components`. The method iterates the map values internally.

**Alternative considered**: Keep slice signature to minimize churn. Rejected — the map→slice conversion in the pipeline is boilerplate that only exists because the matcher predates the map type. Removing it is a net simplification.

### Decision 2: CUE context is not stored on any core domain type

`core.Provider` and `core.TransformerMatchPlan` are pure data types. Neither carries a `*cue.Context`. The pipeline (`build/pipeline.go`) stores `cueCtx` as a private field (set in `NewPipeline()` from `GlobalConfig.CueContext`) and passes it explicitly where needed.

For `core-transformer-match-plan-execute`: `TransformerMatchPlan.Execute()` will take `cueCtx *cue.Context` as a parameter. The pipeline calls `matchPlan.Execute(ctx, p.cueCtx, rel)`.

The one operation requiring an external context is `cueCtx.Encode(contextMap)` in the executor — converting the `TransformerContext` Go map into a `cue.Value` for `#context` injection. All other CUE operations (`FillPath`, `LookupPath`, `Decode`) operate on existing `cue.Value` fields that carry their runtime internally.

**Alternative considered**: Store `CueCtx` on `core.Provider` as a transit field to the match plan. Rejected — the pipeline already owns `p.cueCtx`; a second copy on `Provider` has no purpose and misleads readers about what `Provider` is responsible for.

### Decision 3: `transform.LoadProvider()` takes no `cueCtx` parameter

The loader's job is CUE field extraction from an existing `cue.Value` (the provider value from `GlobalConfig.Providers`). These are all category-A operations — `LookupPath`, `Fields`, `String` — that operate on an existing value and require no external context. The `cueCtx` argument is removed from `LoadProvider()`'s signature.

**Alternative considered**: Pass `cueCtx` to `LoadProvider()` to set it on `Provider`. Rejected per Decision 2 — `Provider` does not carry the context.

### Decision 4: `transform.MatchResult` and `transform.MatchDetail` are replaced by `core.TransformerMatch` and `core.TransformerMatchDetail`

The detail records (per-component, per-transformer matching decisions) currently live in `transform.MatchDetail`. These are replaced by two new types in `core`:

- `TransformerMatch` — one entry per (component × transformer) evaluation, covering both hits (`Matched: true`) and misses (`Matched: false`). Carries `*TransformerMatchDetail` for diagnostic data.
- `TransformerMatchDetail` — diagnostic fields: `ComponentName`, `TransformerFQN`, `Reason`, `MissingLabels`, `MissingResources`, `MissingTraits`, `UnhandledResources`, `UnhandledTraits`.

`TransformerMatchPlan.Matches` is the complete evaluation log. Filtering by `Matched == true` gives hits; scanning all entries for `UnhandledTraits` drives `collectWarnings()`. `TransformerMatchPlan.Unmatched` lists component names that have no `Matched: true` entry.

`transform.MatchResult`, `transform.MatchDetail`, and the old `core.TransformerMatchOld` / `core.MatchPlan` legacy types are deleted after migration.

### Decision 5: `transform.LoadedProvider` and `transform.LoadedTransformer` are retained for now

`transform.LoadedProvider` and `transform.LoadedTransformer` are the intermediate types produced by the CUE extraction step. Collapsing these into `core.Provider` and `core.Transformer` is a larger migration involving the CUE extraction logic. This change only adds `Match()` on `core.Provider` — the `core.Transformer` type in `provider.go` is not yet wired into the pipeline. That consolidation is deferred.

The matching method on `core.Provider` will operate on the `core.Transformer` type's fields (labels, resources, traits), which means `transform.LoadProvider()` must populate `core.Provider.Transformers` in addition to returning `transform.LoadedProvider`'s transformer list. Concretely, `transform.LoadProvider()` returns a `*core.Provider` with `Transformers` populated from the extracted `LoadedTransformer` data.

## Risks / Trade-offs

- **`MatchResult` internal type disappears** → The `collectWarnings()` function in `pipeline.go` currently reads `transform.MatchResult.Details`. After this change it reads from `core.TransformerMatchPlan`. The warning logic must be migrated carefully — same semantics, different type. Test coverage on `collectWarnings` guards against regression.

- **Map iteration order is non-deterministic** → `Match()` iterates `map[string]*Component`. The order components are evaluated does not affect correctness (each component is evaluated against all transformers independently), but the order of entries in `TransformerMatchPlan.Matches` may vary. Since the executor processes jobs sequentially by match plan order (next change), non-deterministic ordering could affect resource output ordering. Mitigation: sort component names before iterating, as the current `componentsToSlice` helper does not guarantee order either.

- **Import cycle risk** → `core` already imports `cuelang.org/go/cue` (for `cue.Value` fields on `Transformer`). No new imports are needed; `transform` already imports `core`. No cycle is introduced.

## Migration Plan

1. ~~Add `CueCtx *cue.Context` field to `core.Provider`~~ — not needed; `core.Provider` is a pure data type (see Decision 2)
2. Implement `core.Provider.Match(components map[string]*Component) *TransformerMatchPlan` — port matching algorithm from `transform/matcher.go`
3. Update `transform.LoadProvider()` to return `*core.Provider` with `Transformers` populated; no `cueCtx` parameter needed
4. ~~Pipeline stores `p.cueCtx`~~ — already present in `pipeline.go:26`; no struct change needed
5. Update `pipeline.Render()` MATCHING phase: replace `p.matcher.Match(...)` with `provider.Match(rel.Components)`
6. Migrate `collectWarnings()` to use `core.TransformerMatchPlan` fields
7. Delete `transform/matcher.go`
8. Run `task check` — all tests must pass

Rollback: The change is a pure internal refactor with no external API or behavior changes. If tests fail, revert to previous `Matcher.Match()` call in `pipeline.go`. No data migrations or config changes required.

## Open Questions

- Should component iteration in `Match()` be sorted by name for deterministic plan output, or is determinism only required at the resource sort in Phase 6? Current code does not sort components before matching, so matching order is already non-deterministic. Recommend: sort by component name in `Match()` to make the plan deterministic and simplify future testing.
