# instance-building — Delta

## MODIFIED Requirements

### Requirement: Instance metadata and labels are derived by CUE evaluation

The builder SHALL load `#ModuleInstance` from `opmodel.dev/core@v2` (resolved from the module's own dependency cache) and inject the module, instance name, namespace, and values via `FillPath`. UUID, labels, and derived metadata fields SHALL be computed by CUE evaluation, not by Go code.

#### Scenario: UUID is deterministic

- **WHEN** the same module, instance name, and namespace are provided
- **THEN** the resulting `ModuleInstance.Metadata.UUID` SHALL be identical across builds

#### Scenario: Labels are populated from CUE evaluation

- **WHEN** the instance is built successfully
- **THEN** `ModuleInstance.Metadata.Labels` SHALL contain all expected OPM labels as evaluated by `#ModuleInstance`

#### Scenario: Core v2 schema loaded

- **WHEN** the builder loads the core schema
- **THEN** it SHALL load `opmodel.dev/core@v2` (not `opmodel.dev/core@v1`)
- **THEN** error messages SHALL reference `opmodel.dev/core@v2`
