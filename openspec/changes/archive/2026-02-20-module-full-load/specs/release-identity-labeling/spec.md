## MODIFIED Requirements

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

### Requirement: Label constants

The label keys for identity labels SHALL be defined as constants alongside existing OPM label constants.

#### Scenario: Label key values
- **WHEN** the identity labeling system is used
- **THEN** the release identity label key SHALL be `module-release.opmodel.dev/uuid`
- **AND** the module identity label key SHALL be `module.opmodel.dev/uuid`

## ADDED Requirements

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

## REMOVED Requirements

### Requirement: Release UUID extracted from CUE overlay evaluation

**Reason**: The CUE overlay (`generateOverlayAST`, `overlay.go`) injected `#opmReleaseMeta` into the module's CUE namespace so the CUE `uid.SHA1` function could compute the release UUID. With UUID computation moved to Go via `core.ComputeReleaseUUID()`, the overlay has no remaining function.

**Migration**: `release.Builder.Build()` now calls `core.ComputeReleaseUUID(mod.Metadata.FQN, opts.Name, opts.Namespace)` directly. `overlay.go`, `generateOverlayAST()`, and `formatNode()` are deleted. `Options.PkgName` is removed from `release.Options` (the package name is no longer needed).
