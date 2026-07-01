## MODIFIED Requirements

### Requirement: Transformer types live in a dedicated subpackage

`Transformer`, `TransformerMetadata`, `TransformerRequirements`, `TransformerContext`, `TransformerComponentMetadata`, `TransformerMatchPlan`, `TransformerMatch`, and `TransformerMatchDetail` SHALL be defined in `internal/core/transformer` (package `transformer`), mirroring `transformer.cue` in the CUE catalog. The package SHALL only import `internal/core`, `internal/core/component`, `internal/core/module`, `internal/core/modulerelease`, CUE SDK, K8s apimachinery, and stdlib. (`internal/core/module` is needed because `TransformerContext` holds a `*module.ModuleMetadata` reference.)

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/open-platform-model/cli/internal/core/transformer`
- **THEN** all transformer types and functions are accessible

### Requirement: `CollectWarnings` is defined in the transformer package

`CollectWarnings`, previously in `internal/transformer`, SHALL be defined in `internal/core/transformer` alongside `TransformerMatchPlan` which it operates on. The `internal/transformer` package SHALL be deleted.

#### Scenario: CollectWarnings is accessible from core/transformer
- **WHEN** a consumer imports `github.com/open-platform-model/cli/internal/core/transformer`
- **THEN** `CollectWarnings` is accessible and produces identical output to the previous implementation

#### Scenario: internal/transformer package no longer exists
- **WHEN** the codebase is built after this change
- **THEN** no file imports `github.com/open-platform-model/cli/internal/transformer`
