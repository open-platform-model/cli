## ADDED Requirements

### Requirement: Pipeline output is identical before and after AST refactor

The render pipeline SHALL produce byte-identical `RenderResult` output after the AST-based refactor. No user-facing behavior, resource content, metadata values, labels, or ordering SHALL change.

#### Scenario: Existing module renders identically

- **WHEN** a module that rendered successfully before the refactor is rendered after
- **THEN** the `RenderResult.Resources` SHALL contain the same resources with identical content
- **AND** `RenderResult.Module` SHALL contain the same metadata values
- **AND** `RenderResult.Errors` and `RenderResult.Warnings` SHALL be identical

#### Scenario: Release identity is preserved

- **WHEN** a module is rendered with the same `--name` and `--namespace` flags
- **THEN** `RenderResult.Module.ReleaseIdentity` SHALL be the same UUID as before the refactor
- **AND** all `module-release.opmodel.dev/*` labels SHALL have the same values
