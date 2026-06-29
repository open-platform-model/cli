## ADDED Requirements

<!-- Renamed from `release-identity-labeling` (enhancement 0002 D4/D10). Spec dir is git mv'd at archive. Only the requirements whose normative text changes under the rename are restated here; unchanged requirements (OPMNamespace constant, ComputeReleaseUUID determinism, identity-label preservation) ride the archive spec-sync. -->

### Requirement: Identity fields on build metadata

The build pipeline SHALL extract `metadata.uuid` from both the `#Module` and `#ModuleInstance` CUE evaluation output and populate them on the Go-side metadata structs (`ModuleMetadata` and `InstanceMetadata`). <!-- Was: #ModuleRelease / ReleaseMetadata (0002 D8/D9) -->

#### Scenario: Module with uuid field

- **WHEN** a module is built whose CUE schema includes `metadata.uuid`
- **THEN** the `ModuleMetadata.UUID` field SHALL contain the UUID string from the CUE evaluation

#### Scenario: Instance UUID computed in Go, not from CUE overlay

- **WHEN** an instance is built
- **THEN** `InstanceMetadata.UUID` SHALL be computed by `core.ComputeReleaseUUID(fqn, name, namespace)`
- **AND** no CUE overlay SHALL be generated or applied to compute this value

### Requirement: Instance-id label injection

The `mod apply` command SHALL inject a `module-instance.opmodel.dev/uuid` label on every applied resource, set to the instance identity UUID from `InstanceMetadata.UUID`. <!-- Was: module-release.opmodel.dev/uuid / ReleaseMetadata.UUID (0002 D4) -->

#### Scenario: Apply with instance identity available

- **WHEN** `mod apply` is run and the instance metadata contains a non-empty UUID
- **THEN** every applied resource SHALL have the label `module-instance.opmodel.dev/uuid` set to the instance identity UUID

#### Scenario: Apply without instance identity (backwards compatibility)

- **WHEN** `mod apply` is run and the instance metadata UUID is empty
- **THEN** the `module-instance.opmodel.dev/uuid` label SHALL NOT be set on resources
- **AND** all existing labels (managed-by, name, namespace, version, component) SHALL still be set

### Requirement: Label constants

The label keys for identity labels SHALL be defined as constants alongside existing OPM label constants.

#### Scenario: Label key values

- **WHEN** the identity labeling system is used
- **THEN** the instance identity label key SHALL be `module-instance.opmodel.dev/uuid` <!-- Was: module-release.opmodel.dev/uuid -->
- **AND** the module identity label key SHALL be `module.opmodel.dev/uuid`

### Requirement: Runtime identity injected via catalog mandatory field

The CLI's `mod apply` (and any other render entrypoint that produces Kubernetes resources) MUST fill the catalog's `#TransformerContext.#runtimeName` field with `core.LabelManagedByValue` (`"opm-cli"`). The catalog declares `#runtimeName` as a mandatory field; CUE evaluation MUST fail if the CLI omits it. The `#runtimeName` value drives the `app.kubernetes.io/managed-by` label on every rendered resource.

#### Scenario: CLI-applied resources carry runtime identity

- **WHEN** `opm mod apply` renders a `#ModuleInstance` and applies the resulting resources <!-- Was: #ModuleRelease -->
- **THEN** every applied resource has `metadata.labels["app.kubernetes.io/managed-by"]` set to `"opm-cli"`
- **AND** no applied resource carries the legacy literal `"open-platform-model"` for that label key

#### Scenario: Runtime identity stays in sync with Go constant

- **GIVEN** the CLI render pipeline executed against a minimal valid `#ModuleInstance`
- **WHEN** the rendered resources are inspected
- **THEN** the value of `metadata.labels["app.kubernetes.io/managed-by"]` exactly equals `core.LabelManagedByValue`
- **AND** the value of `metadata.labels["module-instance.opmodel.dev/uuid"]` is non-empty <!-- Was: module-release.opmodel.dev/uuid -->
