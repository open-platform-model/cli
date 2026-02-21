## Why

`internal/transformer/` was originally planned to house MATCHING and GENERATE phases. However, `core-transformer-match-plan-execute` has moved both into `internal/core/`: matching lives on `core.Provider.Match()` and generation on `core.TransformerMatchPlan.Execute()`. The remaining concern not yet in a dedicated package is warning collection — `collectWarnings()` currently inlined in `internal/legacy/build/pipeline.go` — which belongs with transformer-matching concerns rather than the orchestrator.

## What Changes

- Create `internal/transformer/` package
- Create `internal/transformer/warnings.go` with `CollectWarnings(plan *core.TransformerMatchPlan) []string` — extracted directly from `collectWarnings()` in `internal/legacy/build/pipeline.go`; a trait is considered unhandled only if no matched transformer handles it across all component-transformer pairs
- **Not created**: `matcher.go` (matching is `core.Provider.Match()`), `generator.go` (generation is `core.TransformerMatchPlan.Execute()`), `context.go` (context injection is `internal/core/transformer_context.go`)

## Capabilities

### New Capabilities

- `transformer-warnings`: Collecting unhandled-trait warnings from a `*core.TransformerMatchPlan` after matching — identifies traits present on a component that no matched transformer declares as required or optional

### Modified Capabilities

_None._

## Impact

- New package `internal/transformer/` — single file, thin concern
- `internal/legacy/build/pipeline.go` — `collectWarnings()` superseded (removed in `pipeline-orchestrator`)
- Depends on: `internal/core/`
- Consumed by `internal/pipeline/` in `pipeline-orchestrator`
- SemVer: **MINOR** — new internal package, no CLI behavior changes
