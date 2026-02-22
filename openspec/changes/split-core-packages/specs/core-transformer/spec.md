## ADDED Requirements

### Requirement: Transformer types live in a dedicated subpackage
`Transformer`, `TransformerMetadata`, `TransformerRequirements`, `TransformerContext`, `TransformerComponentMetadata`, `TransformerMatchPlan`, `TransformerMatch`, and `TransformerMatchDetail` SHALL be defined in `internal/core/transformer` (package `transformer`), mirroring `transformer.cue` in the CUE catalog. The package SHALL only import `internal/core`, `internal/core/component`, `internal/core/modulerelease`, CUE SDK, K8s apimachinery, and stdlib.

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/core/transformer`
- **THEN** all transformer types and functions are accessible

#### Scenario: No circular imports
- **WHEN** `internal/core/transformer` is loaded
- **THEN** it SHALL NOT import `internal/core/provider`

### Requirement: `CollectWarnings` is defined in the transformer package
`CollectWarnings`, previously in `internal/transformer`, SHALL be defined in `internal/core/transformer` alongside `TransformerMatchPlan` which it operates on. The `internal/transformer` package SHALL be deleted.

#### Scenario: CollectWarnings is accessible from core/transformer
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/core/transformer`
- **THEN** `CollectWarnings` is accessible and produces identical output to the previous implementation

#### Scenario: internal/transformer package no longer exists
- **WHEN** the codebase is built after this change
- **THEN** no file imports `github.com/opmodel/cli/internal/transformer`

### Requirement: Transformer execution behavior is preserved
`TransformerMatchPlan.Execute()` SHALL produce identical resources and errors for identical inputs after the move.

#### Scenario: Matched transformers produce same resources
- **WHEN** `Execute()` is called with a valid match plan and `ModuleRelease`
- **THEN** the returned `[]*core.Resource` slice is identical to what the previous implementation produced

#### Scenario: Context cancellation is respected
- **WHEN** the context passed to `Execute()` is cancelled between matches
- **THEN** execution stops and `ctx.Err()` is returned as an error

### Requirement: Legacy MatchPlan types are preserved unchanged
`MatchPlan`, `TransformerMatchOld`, and `ToLegacyMatchPlan()` SHALL be carried forward into the `transformer` package without modification. They remain as-is pending a separate cleanup change.

#### Scenario: ToLegacyMatchPlan produces same output
- **WHEN** `ToLegacyMatchPlan()` is called on a `TransformerMatchPlan`
- **THEN** the resulting `MatchPlan` is identical to what the previous implementation produced
