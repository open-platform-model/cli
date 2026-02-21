## Context

`internal/legacy/build/pipeline.go` contains a `collectWarnings()` function inlined in the pipeline orchestrator. Its sole job is to inspect a `*core.TransformerMatchPlan` after matching and produce a list of human-readable warning strings for traits that no matched transformer handles. This logic belongs with transformer-matching concerns, not the orchestrator.

`core-transformer-match-plan-execute` already moved matching (`core.Provider.Match()`) and generation (`core.TransformerMatchPlan.Execute()`) into `internal/core/`. The matching spec (`specs/component-matching/spec.md`) covers both matching semantics and the unhandled-trait tracking that `CollectWarnings` depends on — these are already implemented in `core.Provider.evaluateMatch()`, which populates `TransformerMatchDetail.UnhandledTraits`. `CollectWarnings` is purely a read-and-aggregate step over the completed plan.

This change is therefore a targeted extraction: move `collectWarnings()` out of the pipeline, give it a stable public home in `internal/transformer/`, and make it callable by `internal/pipeline/` in `pipeline-orchestrator`.

## Goals / Non-Goals

**Goals:**
- Create `internal/transformer/` as a thin package with a single exported function
- Extract `collectWarnings()` → `CollectWarnings(plan *core.TransformerMatchPlan) []string`, preserving its exact semantics
- Provide a stable import target for `pipeline-orchestrator`

**Non-Goals:**
- Implementing matching logic (lives in `core.Provider.Match()`)
- Implementing generation logic (lives in `core.TransformerMatchPlan.Execute()`)
- Changing warning semantics or output format
- Removing `collectWarnings()` from `internal/legacy/build/pipeline.go` — that is deferred to `pipeline-orchestrator`

## Decisions

### Decision: Single file, single exported symbol

The package's entire scope is one function. A single `warnings.go` file avoids over-engineering a package that has one job. No interfaces, no types, no configuration — just `CollectWarnings(plan) []string`.

Alternative considered: a `Matcher` type wrapping `core.Provider.Match()` with warnings collected as a side-effect. Rejected because matching is already in `core` and adding a wrapper type would duplicate the API surface without adding value.

### Decision: Preserve exact `collectWarnings()` semantics verbatim

The algorithm in `internal/legacy/build/pipeline.go:220–257` is:
1. Count how many transformers matched each component (`componentMatchCount`).
2. Count how many of those matched transformers mark each trait as unhandled (`traitUnhandledCount`).
3. Emit a warning only when `unhandledCount == matchCount` — i.e., every matched transformer fails to handle the trait.

This is the correct semantics per the spec: a trait is unhandled only if no matched transformer handles it. The implementation is copied directly; no logic changes are made in this change.

Alternative considered: restructuring the algorithm for clarity (single-pass). Deferred — correctness and traceability of the extraction are the goals here; refactoring is a separate concern.

### Decision: Package accepts `*core.TransformerMatchPlan` directly

`CollectWarnings` takes the plan produced by `core.Provider.Match()`. No intermediate type or adapter is needed. The `internal/transformer` package imports `internal/core` and nothing else from the pipeline.

## Risks / Trade-offs

- **Risk: Divergence between legacy and new implementation** — if `collectWarnings()` in `internal/legacy/build/pipeline.go` is changed before `pipeline-orchestrator` removes it, the two implementations could drift. Mitigation: the legacy copy is explicitly superseded by this package in the proposal; `pipeline-orchestrator` removes it in the same sweep that wires `internal/transformer`.

- **Trade-off: Tiny package** — `internal/transformer/` contains one file and one function. This may feel over-structured. Justified because: (a) it gives `pipeline-orchestrator` a clean import boundary, (b) it matches the architectural decomposition described in the 7-change plan, and (c) future warning types (e.g., unhandled resources) can be added here without touching the orchestrator.

## Migration Plan

1. Create `internal/transformer/warnings.go` with `CollectWarnings()` copied from `internal/legacy/build/pipeline.go:collectWarnings()`.
2. Add a unit test in `internal/transformer/warnings_test.go` covering the scenarios in the spec.
3. `internal/legacy/build/pipeline.go:collectWarnings()` is left unchanged — removal is deferred to `pipeline-orchestrator`.
4. No command behavior changes; no flag changes; no user-visible output changes.

Rollback: delete `internal/transformer/` — nothing yet imports it.

## Open Questions

_None. The scope is fully bounded by the existing `collectWarnings()` implementation and the component-matching spec._
