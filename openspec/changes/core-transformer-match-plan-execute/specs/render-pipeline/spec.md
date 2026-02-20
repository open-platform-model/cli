## MODIFIED Requirements

### Requirement: Pipeline output is identical before and after AST refactor
The render pipeline SHALL produce byte-identical `RenderResult` output after the executor-to-receiver-method refactor. No user-facing behavior, resource content, metadata values, labels, or ordering SHALL change.

#### Scenario: Existing module renders identically
- **WHEN** a module that rendered successfully before the refactor is rendered after
- **THEN** the `RenderResult.Resources` SHALL contain the same resources with identical content
- **AND** `RenderResult.Module` SHALL contain the same metadata values
- **AND** `RenderResult.Errors` and `RenderResult.Warnings` SHALL be identical

#### Scenario: Release identity is preserved
- **WHEN** a module is rendered with the same `--name` and `--namespace` flags
- **THEN** `RenderResult.Release.UUID` SHALL be the same UUID as before the refactor
- **AND** all `module-release.opmodel.dev/*` labels SHALL have the same values

#### Scenario: Release identity is preserved after refactor
- **WHEN** a module is rendered with the same `--name` and `--namespace` flags before and after this change
- **THEN** `RenderResult.Release.UUID` SHALL be the same value as before
- **AND** all `module-release.opmodel.dev/*` labels SHALL have the same values

## ADDED Requirements

### Requirement: Pipeline GENERATE phase delegates to TransformerMatchPlan
The `build/pipeline.go` GENERATE phase SHALL call `matchPlan.Execute(ctx, rel)` instead of constructing and invoking an `Executor` service. The `pipeline` struct SHALL NOT hold an executor field after this change.

#### Scenario: Pipeline renders without Executor field
- **WHEN** `pipeline.Render()` executes the GENERATE phase
- **THEN** it SHALL invoke `matchPlan.Execute(ctx, rel)` on the `*core.TransformerMatchPlan` returned by the MATCHING phase
- **AND** the `pipeline` struct SHALL NOT hold an `executor` field

#### Scenario: Context cancellation propagated through Execute
- **WHEN** the context passed to `pipeline.Render()` is cancelled during the GENERATE phase
- **THEN** `matchPlan.Execute()` SHALL honour the cancellation
- **AND** `pipeline.Render()` SHALL return a cancellation error (not in `RenderResult.Errors`)
