## Context

The GENERATE phase of `pipeline.Render()` currently works through three collaborating types in `internal/build/transform/`:

- `MatchResult` — output of `Matcher.Match()`: contains `ByTransformer map[string][]*component.Component`, `Unmatched`, and `Details`
- `Executor` — stateless service; takes a `MatchResult` + `*release.BuiltRelease` + transformer map, runs jobs sequentially
- `TransformerContext` — data type built inside each job execution; holds name/namespace/metadata for CUE injection

The executor needs a `*cue.Context` for `cueCtx.Encode()` calls. Currently it obtains this by calling `job.Transformer.Value.Context()` on the transformer's own CUE value — i.e., it reaches into the transformer to recover the context rather than receiving it explicitly.

The goal is to move execution onto `core.TransformerMatchPlan.Execute()`, eliminating the `Executor` service struct and making `TransformerContext` a `core` type so the receiver method can use it without an import cycle.

## Goals / Non-Goals

**Goals:**

- `core.TransformerMatchPlan` owns execution via `Execute(ctx, rel)` receiver method
- `TransformerContext` and `TransformerComponentMetadata` move to `internal/core/`
- `Executor` struct and `transform/executor.go` removed
- `transform/context.go` removed
- `pipeline.Render()` GENERATE phase becomes a single `matchPlan.Execute(ctx, rel)` call
- No change to observable output (byte-identical resources, errors, warnings)

**Non-Goals:**

- Changing the matching algorithm (stays in `Matcher` / future `Provider.Match()`)
- Eliminating `build/component.Component` in favour of `core.Component` (separate change)
- Parallelising execution (CUE context is not goroutine-safe)

## Decisions

### Decision 1: CUE context source in Execute()

**Current approach**: `executor.go` calls `job.Transformer.Value.Context()` to recover the CUE context from the transformer's own `cue.Value`. This works but is implicit — the context is extracted from a value rather than threaded through explicitly.

**New approach**: `TransformerMatchPlan` stores an unexported `cueCtx *cue.Context` field, set by `Provider.Match()` (or by `transform.LoadProvider()` in the interim). `Execute()` uses `m.cueCtx.Encode()` directly.

**Why**: Makes the dependency explicit. The plan knows from construction which CUE runtime it belongs to. This is consistent with `GlobalConfig.CueContext` being the single source of truth — it flows into the pipeline constructor, into `LoadProvider()`, into `Provider`, into `Match()`, and finally into the plan.

**Alternative considered**: Keep using `Value.Context()` to recover it. This avoids storing state on the plan. Rejected because it is invisible to callers and relies on a CUE SDK implementation detail.

### Decision 2: TransformerMatchPlan carries execution data, not MatchResult

`core.TransformerMatchPlan` already has `Matches []TransformerMatch` and `Unmatched []string`. For `Execute()` to run jobs it also needs:

- The matched `*Transformer` CUE values (to `FillPath`)
- The matched `*Component` values (to inject `#component`)

**Approach**: `TransformerMatch` gains unexported fields `transformer *Transformer` and `component *Component` populated at match time. `Execute()` iterates `m.Matches` and uses these directly.

**Why**: Avoids duplicating the job-building step that `executor.go` currently performs by re-walking `ByTransformer` + a transformer map lookup. The match plan is constructed once and already has all the pairs.

**Alternative considered**: Pass the transformer map separately to `Execute()`. Rejected — forces the caller to maintain parallel data structures that are already encoded in the plan.

### Decision 3: TransformerContext moves to internal/core

`TransformerContext` and `TransformerComponentMetadata` currently live in `internal/build/transform/context.go` and depend on `build/release.BuiltRelease` and `build/component.Component`. After this change, `Execute()` lives in `internal/core/` and needs to construct a `TransformerContext`.

**Approach**: Move both types to `internal/core/transformer_context.go`. Update `NewTransformerContext` to accept `*ModuleRelease` and `*Component` (both core types) instead of `*release.BuiltRelease` and `*component.Component`. The field mapping is identical; only the parameter types change.

**Why**: Avoids an import cycle (`core` cannot import `build/release` or `build/component`). The types themselves have no dependency on the build packages — they only hold metadata structs that are already in `core`.

**Alternative considered**: Keep `TransformerContext` in `build/transform` and pass a pre-built context into `Execute()`. Rejected — forces the caller to know about context construction, which is an internal detail of execution.

### Decision 4: Release type passed to Execute()

`Execute()` signature: `Execute(ctx context.Context, rel *ModuleRelease) ([]*Resource, []error)`

`ModuleRelease` carries `Metadata *ReleaseMetadata` and `Module Module` (which has `Metadata *ModuleMetadata`). `NewTransformerContext` inside `Execute()` reads from these directly — same fields as it currently reads from `BuiltRelease.ReleaseMetadata` and `BuiltRelease.ModuleMetadata`.

**Why `*ModuleRelease` rather than the two metadata structs separately**: The release may carry additional context in future. Passing the full release is consistent with how the pipeline already passes `rel` across phases.

### Decision 5: collectWarnings stays in pipeline.go

The warning collection logic (counting which traits are truly unhandled across all matched transformers) reads `MatchResult.Details`, an internal `build/transform` type. After this change, pipeline uses `TransformerMatchPlan` for execution, but the `MatchResult` (with its `Details` slice) is still returned by `Matcher.Match()` for this purpose.

**Approach**: Pipeline continues to hold both the `TransformerMatchPlan` (for execute) and `MatchResult` (for warnings and the public `MatchPlan` output) until the `core-provider-match` change absorbs matching entirely.

**Why**: Keeps this change scoped. Pulling warning logic into `core` can happen as part of the matching change.

## Risks / Trade-offs

**Risk: import cycle if TransformerContext is not cleanly separated**
`internal/core` must not import `internal/build/*`. The move is safe because `TransformerContext` only holds `*ModuleMetadata`, `*ReleaseMetadata`, and `*Component` — all already in `core`.
→ Mitigation: Verify with `go build ./internal/core/...` after the move. CI will catch any cycle.

**Risk: tests in transform/ that test context construction break**
`context_annotations_test.go` and `executor_test.go` import `build/release` types. These tests must be updated to use `core.ModuleRelease` and `core.Component` instead.
→ Mitigation: Update test files as part of the same task. The test logic is unchanged; only the types differ.

**Risk: subtle behavioural change from cueCtx source**
Currently `executeJob` calls `job.Transformer.Value.Context()` to get the CUE context. If the stored `cueCtx` on the plan is a different instance (even though logically the same), `FillPath` across values from different instances would panic.
→ Mitigation: The plan's `cueCtx` is always `GlobalConfig.CueContext`, which is also the context used to compile provider values. They are the same instance. Add an assertion in tests.

## Migration Plan

This is a pure internal refactor. No migration for end-users is required.

Order of changes within the task:

1. Add `TransformerContext` and `TransformerComponentMetadata` to `internal/core/transformer_context.go`
2. Add `Execute()` to `core.TransformerMatchPlan` (in `internal/core/provider.go` or a new `internal/core/match.go`)
3. Update `build/pipeline.go` GENERATE phase to call `matchPlan.Execute(ctx, rel)`
4. Remove `transform/executor.go` and `transform/context.go`
5. Update `transform/types.go` to remove `Job`, `JobResult`, `ExecuteResult` (no longer needed externally)
6. Update affected tests

All steps are in a single incremental change. The build must compile and all tests pass after step 6.

## Open Questions

None. Decisions are fully resolved for this scoped change.
