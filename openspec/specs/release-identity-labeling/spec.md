## Purpose

Defines how the OPM CLI extracts identity UUIDs from CUE evaluation output and injects them as labels on Kubernetes resources during `mod apply`. This enables deterministic resource discovery by release identity.

## Requirements

### Requirement: Identity fields on build metadata

The build pipeline SHALL extract `metadata.uuid` from both the `#Module` and `#ModuleRelease` CUE evaluation output and populate them on the Go-side metadata structs (`ModuleMetadata` and `ReleaseMetadata`).

> **Change from prior spec**: The field name changes from `metadata.identity` to `metadata.uuid`, matching the catalog schema. The Go struct fields are renamed accordingly: `ModuleMetadata.Identity` → `ModuleMetadata.UUID`, `ReleaseMetadata.Identity` → `ReleaseMetadata.UUID`.

#### Scenario: Module with uuid field

- **WHEN** a module is built whose CUE schema includes `metadata.uuid`
- **THEN** the `ModuleMetadata.UUID` field SHALL contain the UUID string from the CUE evaluation

#### Scenario: Module without uuid field (pre-catalog-upgrade)

- **WHEN** a module is built whose CUE schema does NOT include `metadata.uuid` (older catalog version)
- **THEN** the `ModuleMetadata.UUID` field SHALL be empty string
- **AND** no error SHALL be raised

#### Scenario: Release UUID computed in Go, not from CUE overlay

- **WHEN** a release is built
- **THEN** `ReleaseMetadata.UUID` SHALL be computed by `core.ComputeReleaseUUID(fqn, name, namespace)`
- **AND** the value SHALL equal `uuid.NewSHA1(OPMNamespace, []byte(fqn+":"+name+":"+namespace))`
- **AND** no CUE overlay SHALL be generated or applied to compute this value

---

### Requirement: Release-id label injection

The `mod apply` command SHALL inject a `module-release.opmodel.dev/uuid` label on every applied resource, set to the release identity UUID from `ReleaseMetadata.UUID`.

#### Scenario: Apply with release identity available

- **WHEN** `mod apply` is run and the release metadata contains a non-empty UUID
- **THEN** every applied resource SHALL have the label `module-release.opmodel.dev/uuid` set to the release identity UUID

#### Scenario: Apply without release identity (backwards compatibility)

- **WHEN** `mod apply` is run and the release metadata UUID is empty
- **THEN** the `module-release.opmodel.dev/uuid` label SHALL NOT be set on resources
- **AND** all existing labels (managed-by, name, namespace, version, component) SHALL still be set

---

### Requirement: Module-id label injection

The `mod apply` command SHALL inject a `module.opmodel.dev/uuid` label on every applied resource, set to the module identity UUID from `ModuleMetadata.UUID`.

#### Scenario: Apply with module identity available

- **WHEN** `mod apply` is run and the module metadata contains a non-empty UUID
- **THEN** every applied resource SHALL have the label `module.opmodel.dev/uuid` set to the module identity UUID

#### Scenario: Apply without module identity (backwards compatibility)

- **WHEN** `mod apply` is run and the module metadata UUID is empty
- **THEN** the `module.opmodel.dev/uuid` label SHALL NOT be set on resources

---

### Requirement: Identity labels preserve existing labels

Identity label injection SHALL NOT remove or overwrite any user-defined labels or existing OPM labels on resources.

#### Scenario: Resource with user-defined labels

- **WHEN** a resource has user-defined labels in its CUE definition
- **AND** `mod apply` injects identity labels
- **THEN** all user-defined labels SHALL be preserved alongside the identity labels

---

### Requirement: Label constants

The label keys for identity labels SHALL be defined as constants alongside existing OPM label constants.

#### Scenario: Label key values

- **WHEN** the identity labeling system is used
- **THEN** the release identity label key SHALL be `module-release.opmodel.dev/uuid`
- **AND** the module identity label key SHALL be `module.opmodel.dev/uuid`

---

### Requirement: OPMNamespace constant is correct and canonical

`internal/core` SHALL define `OPMNamespace = "11bc6112-a6e8-4021-bec9-b3ad246f9466"` as a Go constant. This value SHALL match `OPMNamespace` in `catalog/v0/core/common.cue` exactly. It is the root namespace for all OPM SHA1 UUID derivations.

#### Scenario: OPMNamespace matches catalog value

- **WHEN** `core.OPMNamespace` is used to compute a UUID
- **THEN** the result SHALL be identical to the UUID that `uid.SHA1(OPMNamespace, input)` would produce in CUE with the same input string

#### Scenario: Old namespace constant is removed

- **WHEN** the codebase is compiled after this change
- **THEN** the constant previously holding `"c1cbe76d-5687-5a47-bfe6-83b081b15413"` SHALL no longer exist
- **AND** all UUID computation SHALL use `core.OPMNamespace`

---

### Requirement: ComputeReleaseUUID() produces deterministic release identity

`core.ComputeReleaseUUID(fqn, name, namespace string) string` SHALL be a package-level function in `internal/core/` that computes a release UUID using `uuid.NewSHA1(uuid.MustParse(OPMNamespace), []byte(fqn+":"+name+":"+namespace))`. The formula SHALL match the CUE expression `uid.SHA1(OPMNamespace, "\(fqn):\(name):\(namespace)")` in the catalog.

#### Scenario: Same inputs always produce the same UUID

- **WHEN** `ComputeReleaseUUID()` is called twice with identical `fqn`, `name`, and `namespace`
- **THEN** both calls SHALL return the same UUID string

#### Scenario: Different releases produce different UUIDs

- **WHEN** `ComputeReleaseUUID()` is called with different `name` or `namespace` values for the same `fqn`
- **THEN** the returned UUIDs SHALL differ

#### Scenario: Release UUID is version 5 (SHA1-based)

- **WHEN** `ComputeReleaseUUID()` returns a UUID string
- **THEN** parsing it SHALL yield a UUID with version 5

#### Scenario: Release UUID does not collide with module UUID

- **WHEN** `ComputeReleaseUUID(fqn, name, namespace)` and the module UUID formula `uuid.NewSHA1(OPMNamespace, fqn+":"+version)` are called with overlapping inputs
- **THEN** their results SHALL differ (different input encodings prevent collision)

---

### Requirement: Runtime identity injected via catalog mandatory field

The CLI's `mod apply` (and any other render entrypoint that produces Kubernetes resources) MUST fill the catalog's `#TransformerContext.#runtimeName` field with `core.LabelManagedByValue` (`"opm-cli"`). The catalog declares `#runtimeName` as a mandatory field; CUE evaluation MUST fail if the CLI omits it. The `#runtimeName` value drives the `app.kubernetes.io/managed-by` label on every rendered resource.

#### Scenario: CLI-applied resources carry runtime identity

- **WHEN** `opm mod apply` renders a `#ModuleRelease` and applies the resulting resources
- **THEN** every applied resource has `metadata.labels["app.kubernetes.io/managed-by"]` set to `"opm-cli"`
- **AND** no applied resource carries the legacy literal `"open-platform-model"` for that label key

#### Scenario: Render fails fast when runtime identity is omitted

- **WHEN** a code path inside the CLI render pipeline constructs a CUE evaluation that includes `#TransformerContext` without filling `#runtimeName`
- **THEN** CUE evaluation returns an error mentioning the missing required field
- **AND** no resources are produced

#### Scenario: Runtime identity stays in sync with Go constant

- **GIVEN** the CLI render pipeline executed against a minimal valid `#ModuleRelease`
- **WHEN** the rendered resources are inspected
- **THEN** the value of `metadata.labels["app.kubernetes.io/managed-by"]` exactly equals `core.LabelManagedByValue`
- **AND** the value of `metadata.labels["module-release.opmodel.dev/uuid"]` is non-empty (sanity check that the catalog ownership labels continue to flow)
