## Context

`internal/cmdutil/render.go` is the single integration point between all `opm mod` commands (apply, build, diff, vet) and the render pipeline. It currently creates a `build.NewPipeline()` from `internal/legacy/` and calls `pipeline.Render()`. All four commands go through `RenderRelease()` in cmdutil — they never touch the pipeline directly.

The `Pipeline` interface (`Render(ctx, RenderOptions) → *RenderResult`) is defined in `internal/legacy/types.go` alongside `RenderOptions` and `RenderResult`. These types flow through `cmdutil` and into command output formatting in `cmdutil/output.go`.

This change is the last in a series of seven, and also depends on `core-transformer-match-plan-execute` being merged. By the time it executes, the following packages and core changes exist and are tested:
- `internal/loader/` — PREPARATION
- `internal/builder/` — BUILD (Approach C)
- `internal/provider/` — provider + transformer definitions
- `internal/transformer/` — MATCHING (match only; execution lives on the plan)
- `internal/legacy/` — old pipeline, still operational
- `core.TransformerMatchPlan.Execute()` — GENERATE method on the match plan (from `core-transformer-match-plan-execute`)

## Goals / Non-Goals

**Goals:**
- Create `internal/pipeline/` with the `Pipeline` interface and a concrete implementation that sequences all phase packages
- Move `RenderOptions` and `RenderResult` types to `internal/pipeline/types.go`
- Update `cmdutil` to import `internal/pipeline` — the only consumer of `internal/legacy/` remaining
- Delete `internal/legacy/` once `cmdutil` is updated and all tests pass

**Non-Goals:**
- Changing the `Pipeline` interface shape — `Render(ctx, RenderOptions) → *RenderResult` is unchanged
- Changing `RenderOptions` or `RenderResult` fields — commands must continue to work without modification
- Changing any command behavior (apply, build, diff, vet)
- Changing `cmdutil.RenderRelease()` beyond the import swap and constructor call

## Decisions

### 1. Pipeline interface is unchanged

The `Pipeline` interface (`Render(ctx context.Context, opts RenderOptions) (*RenderResult, error)`) is moved verbatim from `internal/legacy/types.go` to `internal/pipeline/types.go`. No fields added or removed.

**Why**: `cmdutil` and four commands depend on this interface. Changing the shape here would break them all and expand this change's scope unnecessarily.

### 2. `RenderOptions` and `RenderResult` move as-is

Both types are copied to `internal/pipeline/types.go` with no field changes. `RenderResult` helper methods (`HasErrors()`, `HasWarnings()`, `ResourceCount()`) move with them.

**Why**: These types are the contract between the pipeline and cmdutil. Stability here keeps the cmdutil update to a single import swap.

### 3. `internal/pipeline/pipeline.go` orchestrates phases in sequence

The new implementation calls phase packages in order:

```
loader.Load()               → *core.Module
provider.Load()             → *LoadedProvider
builder.Build()             → *core.ModuleRelease
transformer.Match()         → *core.TransformerMatchPlan
matchPlan.Execute(ctx, rel) → []*core.Resource + []error
```

GENERATE is a receiver method on `core.TransformerMatchPlan` — not a function in `internal/transformer/`. The `pipeline` struct holds no executor field. This is a direct consequence of `core-transformer-match-plan-execute`, which moved execution onto the plan.

Resource sorting (weight → group → kind → namespace → name) moves from `internal/legacy/pipeline.go` into `internal/pipeline/pipeline.go`. Warning collection also stays at the pipeline level, reading unhandled trait information from `core.TransformerMatchPlan.Matches` rather than a separate `MatchResult.Details` slice (which was an internal type of the legacy `transform/` package).

**Why**: Sorting and warning collection are orchestration concerns — they belong at the pipeline level, not in individual phase packages. Using `matchPlan.Execute()` directly is consistent with `core-transformer-match-plan-execute`'s design requirement that the pipeline struct SHALL NOT hold an executor field.

### 4. `cmdutil` update is a single import swap

`internal/cmdutil/render.go` changes two things:
- `import "internal/build"` → `import "internal/pipeline"`
- `build.NewPipeline(...)` → `pipeline.NewPipeline(...)`

No logic changes. `cmdutil/output.go` and `cmd/mod/verbose_output_test.go` get the same import swap.

**Why**: Keeping the update surgical minimizes the blast radius and makes the diff easy to review.

### 5. Legacy deletion is gated on green tests

`internal/legacy/` is deleted only after:
1. `internal/pipeline/` compiles and its tests pass
2. `cmdutil` imports updated
3. `task test` passes across all packages

The deletion is part of this same change — not deferred to a follow-up — because leaving legacy around after the orchestrator is wired creates confusion about which pipeline is active.

**Why**: A follow-up "cleanup" change rarely happens and leaves stale code in the repo longer than needed.

## Risks / Trade-offs

- **`cmdutil` is used by four commands** → Mitigation: interface and types are unchanged; the update is purely additive at the import level. All four commands are covered by existing tests.
- **Legacy deletion is irreversible** → Mitigation: legacy code is preserved in git history. Gate deletion on `task test` passing. The six prior changes each have their own test coverage.
- **`internal/pipeline/` depends on all prior changes being merged** → Mitigation: this change is the seventh in a sequenced series. It must not be started until changes 1-6 are merged and green.

## Migration Plan

1. Confirm changes 1-6 are all merged, `core-transformer-match-plan-execute` is merged, and `task test` passes
2. Create `internal/pipeline/types.go` — copy `Pipeline`, `RenderOptions`, `RenderResult` from legacy
3. Create `internal/pipeline/pipeline.go` — new orchestrator calling phase packages
4. Update `internal/cmdutil/render.go` — import swap + constructor swap
5. Update `internal/cmdutil/output.go` — import swap
6. Update `internal/cmd/mod/verbose_output_test.go` — import swap
7. Run `task test` — all tests must pass
8. Delete `internal/legacy/`
9. Run `task test` again — confirm nothing broken by deletion

Rollback: revert the import swaps in cmdutil; restore legacy from git if needed.
