## MODIFIED Requirements

### Requirement: Pipeline output is identical before and after AST refactor

The render pipeline SHALL produce byte-identical `RenderResult` output after the receiver-method refactor. No user-facing behavior, resource content, metadata values, labels, or ordering SHALL change.

#### Scenario: Existing module renders identically
- **WHEN** a module that rendered successfully before the refactor is rendered after
- **THEN** the `RenderResult.Resources` SHALL contain the same resources with identical content
- **AND** `RenderResult.Module` SHALL contain the same metadata values
- **AND** `RenderResult.Errors` and `RenderResult.Warnings` SHALL be identical

#### Scenario: Release identity is preserved
- **WHEN** a module is rendered with the same `--name` and `--namespace` flags before and after this change
- **THEN** `RenderResult.Release.UUID` SHALL be the same UUID as before the refactor
- **AND** all `module-release.opmodel.dev/*` labels SHALL have the same values

#### Scenario: Path resolution error is a fatal error
- **WHEN** `Pipeline.Render()` is called with a `ModulePath` that does not exist or is not a CUE module
- **THEN** `Render()` SHALL return a non-nil `error` (fatal error, not a render error)
- **AND** `RenderResult` SHALL be `nil`

#### Scenario: Module structural validation error is a fatal error
- **WHEN** the loaded `core.Module` fails `Validate()` (e.g., missing `Metadata.Name`)
- **THEN** `Pipeline.Render()` SHALL return a non-nil `error` (fatal error)
- **AND** `RenderResult` SHALL be `nil`
