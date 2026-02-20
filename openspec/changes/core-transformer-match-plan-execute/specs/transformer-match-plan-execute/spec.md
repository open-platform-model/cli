## ADDED Requirements

### Requirement: TransformerMatchPlan owns resource generation
`core.TransformerMatchPlan` SHALL provide an `Execute(ctx context.Context, rel *ModuleRelease) ([]*Resource, []error)` receiver method that sequentially processes all transformer-component matches and returns the generated resources.

#### Scenario: Execute runs all matches and returns resources
- **WHEN** `Execute` is called on a `TransformerMatchPlan` with one or more matches
- **THEN** each match SHALL be processed in sequence (component × transformer)
- **AND** the returned `[]*Resource` SHALL contain all successfully generated resources
- **AND** any per-match errors SHALL be returned in the `[]error` slice

#### Scenario: Execute respects context cancellation
- **WHEN** the context passed to `Execute` is cancelled mid-execution
- **THEN** execution SHALL stop after the current match completes
- **AND** `Execute` SHALL return the resources generated so far and a cancellation error

#### Scenario: Execute with no matches returns empty resources
- **WHEN** `Execute` is called on a `TransformerMatchPlan` with no matches
- **THEN** the returned `[]*Resource` SHALL be an empty (non-nil) slice
- **AND** the returned `[]error` SHALL be empty

### Requirement: TransformerMatchPlan carries CUE context from construction
`core.TransformerMatchPlan` SHALL hold an unexported `*cue.Context` field set by `core.Provider.Match()` at construction time. This context SHALL be used internally by `Execute()` for CUE encoding operations.

#### Scenario: CUE context set when Match produces a plan
- **WHEN** `provider.Match(components)` is called
- **THEN** the returned `*TransformerMatchPlan` SHALL have its internal `cueCtx` set to the same `*cue.Context` that was stored on the `Provider`
- **AND** no external caller SHALL need to pass a `*cue.Context` to `Execute()`

### Requirement: TransformerContext resides in core package
The `TransformerContext` type, its constructor, and its `ToMap()` method SHALL reside in `internal/core/` so they are accessible from `TransformerMatchPlan.Execute()` without creating an import cycle.

#### Scenario: TransformerContext constructed from ModuleRelease and Component
- **WHEN** the transformer context is built for a matched pair inside `Execute()`
- **THEN** it SHALL be constructed using the `ModuleRelease.Metadata` and `ModuleRelease.Module.Metadata` fields
- **AND** component name, labels, and annotations SHALL be populated from the matched `*Component`

#### Scenario: No import cycle introduced
- **WHEN** `internal/core/` is compiled
- **THEN** it SHALL NOT import any package under `internal/build/`
- **AND** `internal/build/` packages MAY continue to import `internal/core/`

### Requirement: Executor service removed from build/transform
The `Executor` struct and `ExecuteWithTransformers()` function in `internal/build/transform/executor.go` SHALL be removed once `TransformerMatchPlan.Execute()` is implemented. No duplicate execution path SHALL exist.

#### Scenario: Pipeline uses Execute() exclusively
- **WHEN** `build/pipeline.go` processes the GENERATE phase
- **THEN** it SHALL call `matchPlan.Execute(ctx, rel)` directly
- **AND** it SHALL NOT construct or call an `Executor` service

#### Scenario: Removal does not change observable output
- **WHEN** a module that previously rendered successfully is rendered after the executor is removed
- **THEN** `RenderResult.Resources` SHALL contain byte-identical resources
- **AND** `RenderResult.Errors` SHALL be identical to before
