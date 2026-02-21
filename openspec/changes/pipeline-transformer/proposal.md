## Why

MATCHING and GENERATE are two distinct phases that together turn a concrete `*core.ModuleRelease` into Kubernetes resources, but they currently live intermingled in `internal/legacy/build/transform/`. A dedicated `internal/transformer/` package separates the concerns cleanly — matching maps release components to transformer definitions, generation executes those transformers — while keeping them co-located since the match result feeds directly into generation.

## What Changes

- Create `internal/transformer/matcher.go` with `Match(release *core.ModuleRelease, provider *provider.LoadedProvider) MatchPlan`
- Create `internal/transformer/generator.go` with `Generate(ctx context.Context, plan MatchPlan, release *core.ModuleRelease) ([]*core.Resource, []error)`
- Create `internal/transformer/context.go` with CUE context injection helpers (release metadata, component metadata injected into transformer CUE values)
- Create `internal/transformer/types.go` with `MatchPlan`, `Job`, `JobResult`
- Supersedes `internal/legacy/build/transform/matcher.go`, `executor.go`, `context.go`, `types.go`

## Capabilities

### New Capabilities

- `component-matching`: Matching concrete release components against transformer definitions by evaluating required labels, resources, and traits — producing a `MatchPlan`
- `resource-generation`: Executing matched transformers sequentially, injecting release and component context into each CUE transformer, and collecting the resulting Kubernetes resources

### Modified Capabilities

_None._

## Impact

- New package `internal/transformer/` — no existing code modified
- Depends on: `internal/core/`, `internal/provider/`, `cuelang.org/go/cue`
- Will be consumed by `internal/pipeline/` in a later change
- SemVer: **MINOR** — new internal package, no CLI behavior changes
