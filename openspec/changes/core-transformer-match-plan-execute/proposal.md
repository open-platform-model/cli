## Why

The GENERATE phase of the pipeline — running each matched transformer against its component to produce `core.Resource` objects — is currently owned by `internal/build/transform/executor.go`. The `Executor` struct is stateless and exists purely to run the jobs described by a `MatchResult`. Since the match plan already represents "what work needs to be done", it is the natural owner of the execution logic. Moving it to `TransformerMatchPlan.Execute()` completes the interface-driven pipeline architecture where each phase's output drives the next.

## What Changes

- Add `Execute(ctx context.Context, rel *ModuleRelease) ([]*Resource, []error)` receiver method to `core.TransformerMatchPlan`
- `TransformerMatchPlan` stores `*cue.Context` (received from `Provider.Match()` in the prior change) and uses it for `cueCtx.Encode()` when injecting `#context` into transformer CUE values
- The execution algorithm (sequential job processing, `FillPath(#component)`, `FillPath(#context)`, output extraction, resource decoding) moves from `transform/executor.go` into this method
- `internal/build/transform/executor.go` and `Executor` struct removed (logic migrated to receiver method)
- `internal/build/transform/context.go` (`TransformerContext`, `NewTransformerContext`, `ToMap`) moves to `internal/core/` as it is now called from the core receiver method
- `build/pipeline.go` GENERATE phase updated to call `matchPlan.Execute(ctx, rel)`

## Capabilities

### New Capabilities

- `transformer-match-plan-execute`: `Execute()` receiver method on `core.TransformerMatchPlan` encapsulating sequential transformer execution and resource generation

### Modified Capabilities

- `render-pipeline`: The GENERATE phase of `pipeline.Render()` now invokes `matchPlan.Execute()` instead of constructing and calling a separate `Executor` service; `pipeline.Render()` becomes a thin orchestrator across all four phases

## Impact

- `internal/core/provider.go` — `TransformerMatchPlan` gains unexported `cueCtx *cue.Context` field and `Execute()` method
- `internal/core/` — `TransformerContext` type and constructor migrated from `internal/build/transform/context.go`
- `internal/build/transform/executor.go` — removed; all logic migrated to `core.TransformerMatchPlan.Execute()`
- `internal/build/transform/context.go` — removed; type moved to `internal/core/`
- `internal/build/pipeline.go` — GENERATE phase simplified to a single method call; `pipeline` struct reduced
- SemVer: **PATCH** — internal refactor, no change to CLI behavior or public-facing pipeline interface
