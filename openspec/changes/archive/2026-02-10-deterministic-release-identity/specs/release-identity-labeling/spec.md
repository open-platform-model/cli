## ADDED Requirements

### Requirement: Identity fields on build metadata

The build pipeline SHALL extract `metadata.identity` from both the `#Module` and `#ModuleRelease` CUE evaluation output and populate them on the Go-side metadata structs (`ModuleMetadata` and `ReleaseMetadata`).

#### Scenario: Module with identity field

- **WHEN** a module is built whose CUE schema includes `metadata.identity`
- **THEN** the `ModuleMetadata.Identity` field SHALL contain the UUID string from the CUE evaluation

#### Scenario: Module without identity field (pre-catalog-upgrade)

- **WHEN** a module is built whose CUE schema does NOT include `metadata.identity` (older catalog version)
- **THEN** the `ModuleMetadata.Identity` field SHALL be empty string
- **AND** no error SHALL be raised

#### Scenario: Release with identity field

- **WHEN** a release is built whose CUE schema includes `metadata.identity`
- **THEN** the `ReleaseMetadata.Identity` field SHALL contain the UUID string from the CUE evaluation

### Requirement: Release-id label injection

The `mod apply` command SHALL inject a `module-release.opmodel.dev/uuid` label on every applied resource, set to the release identity UUID from `ReleaseMetadata.Identity`.

#### Scenario: Apply with release identity available

- **WHEN** `mod apply` is run and the release metadata contains a non-empty identity
- **THEN** every applied resource SHALL have the label `module-release.opmodel.dev/uuid` set to the release identity UUID

#### Scenario: Apply without release identity (backwards compatibility)

- **WHEN** `mod apply` is run and the release metadata identity is empty
- **THEN** the `module-release.opmodel.dev/uuid` label SHALL NOT be set on resources
- **AND** all existing labels (managed-by, name, namespace, version, component) SHALL still be set

### Requirement: Module-id label injection

The `mod apply` command SHALL inject a `module.opmodel.dev/uuid` label on every applied resource, set to the module identity UUID from `ModuleMetadata.Identity`.

#### Scenario: Apply with module identity available

- **WHEN** `mod apply` is run and the module metadata contains a non-empty identity
- **THEN** every applied resource SHALL have the label `module.opmodel.dev/uuid` set to the module identity UUID

#### Scenario: Apply without module identity (backwards compatibility)

- **WHEN** `mod apply` is run and the module metadata identity is empty
- **THEN** the `module.opmodel.dev/uuid` label SHALL NOT be set on resources

### Requirement: Identity labels preserve existing labels

Identity label injection SHALL NOT remove or overwrite any user-defined labels or existing OPM labels on resources.

#### Scenario: Resource with user-defined labels

- **WHEN** a resource has user-defined labels in its CUE definition
- **AND** `mod apply` injects identity labels
- **THEN** all user-defined labels SHALL be preserved alongside the identity labels

### Requirement: Label constants

The label keys for identity labels SHALL be defined as constants alongside existing OPM label constants.

#### Scenario: Label key values

- **WHEN** the identity labeling system is used
- **THEN** the release identity label key SHALL be `module-release.opmodel.dev/uuid`
- **AND** the module identity label key SHALL be `module.opmodel.dev/uuid`
