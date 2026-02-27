## MODIFIED Requirements

### Requirement: Release metadata and labels are derived by CUE evaluation

The builder SHALL load `#ModuleRelease` from `opmodel.dev/core@v1` (resolved from the module's own dependency cache) and inject the module, release name, namespace, and values via `FillPath`. UUID, labels, and derived metadata fields SHALL be computed by CUE evaluation, not by Go code.

#### Scenario: UUID is deterministic

- **WHEN** the same module, release name, and namespace are provided
- **THEN** the resulting `ModuleRelease.Metadata.UUID` SHALL be identical across builds

#### Scenario: Labels are populated from CUE evaluation

- **WHEN** the release is built successfully
- **THEN** `ModuleRelease.Metadata.Labels` SHALL contain all expected OPM labels as evaluated by `#ModuleRelease`

#### Scenario: Core v1 schema loaded

- **WHEN** the builder loads the core schema
- **THEN** it SHALL load `opmodel.dev/core@v1` (not `opmodel.dev/core@v0`)
- **THEN** error messages SHALL reference `opmodel.dev/core@v1`
