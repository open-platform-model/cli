## MODIFIED Requirements

### Requirement: RenderResult carries both module and release metadata

`RenderResult` SHALL carry two distinct metadata fields: `Module ModuleMetadata`
for canonical module-level identity (name, FQN, version, module UUID) and
`Release ReleaseMetadata` for release-level identity (release name, namespace,
release UUID, labels). These correspond to FR-RP-021 in the base spec.

The `Module` field MUST reflect the canonical module name from
`#Module.metadata.name`, which may differ from the release name when `--name`
overrides the default. The `Release` field MUST reflect the deployed release
identity (name, namespace, computed UUID).

#### Scenario: Module and release names differ when --name is overridden

- **WHEN** a module with `metadata.name: "my-app"` is rendered with `--name my-app-staging`
- **THEN** `RenderResult.Module.Name` SHALL equal `"my-app"` (canonical module name)
- **AND** `RenderResult.Release.Name` SHALL equal `"my-app-staging"` (release name)

#### Scenario: Module UUID and release UUID are distinct

- **WHEN** any module is rendered
- **THEN** `RenderResult.Module.UUID` SHALL be the module identity UUID (from `metadata.identity`)
- **AND** `RenderResult.Release.UUID` SHALL be the release UUID (deterministic UUID5 from FQN+name+namespace)
- **AND** the two UUID values SHALL be different values

#### Scenario: Release identity is preserved after refactor

- **WHEN** a module is rendered with the same `--name` and `--namespace` flags before and after this change
- **THEN** `RenderResult.Release.UUID` SHALL be the same value as before
- **AND** all `module-release.opmodel.dev/*` labels SHALL have the same values

### Requirement: BuiltRelease carries typed metadata directly

The internal `BuiltRelease` type (output of `Builder.Build()`) SHALL carry
`ReleaseMetadata` and `ModuleMetadata` as direct struct fields, populated by the
builder from the fully evaluated CUE value. No intermediate `Metadata` grab-bag
struct SHALL exist. The builder is responsible for extracting both metadata types
from the CUE value before returning `BuiltRelease`.

`TransformerContext` SHALL read module name from `BuiltRelease.ModuleMetadata.Name`
(the canonical module name) rather than from the release name field.

#### Scenario: Builder populates ModuleMetadata with FQN and version

- **WHEN** `Builder.Build()` is called on a module that defines `metadata.fqn` and `metadata.version`
- **THEN** the returned `BuiltRelease.ModuleMetadata.FQN` SHALL equal the module's `metadata.fqn` value
- **AND** `BuiltRelease.ModuleMetadata.Version` SHALL equal the module's `metadata.version` value
- **AND** `BuiltRelease.ModuleMetadata.DefaultNamespace` SHALL equal the module's `metadata.defaultNamespace` value

#### Scenario: Builder populates ReleaseMetadata with release-level fields

- **WHEN** `Builder.Build()` is called with `Name: "my-release"` and `Namespace: "production"`
- **THEN** `BuiltRelease.ReleaseMetadata.Name` SHALL equal `"my-release"`
- **AND** `BuiltRelease.ReleaseMetadata.Namespace` SHALL equal `"production"`
- **AND** `BuiltRelease.ReleaseMetadata.UUID` SHALL be the computed release UUID

#### Scenario: TransformerContext uses canonical module name

- **WHEN** a module with `metadata.name: "my-app"` is rendered with `--name my-app-staging`
- **THEN** `TransformerContext.ModuleMetadata.Name` SHALL equal `"my-app"` (canonical module name)
- **AND** `TransformerContext.ReleaseMetadata.Name` SHALL equal `"my-app-staging"` (release name)

## REMOVED Requirements

### Requirement: Internal Metadata grab-bag struct

**Reason**: Replaced by `ReleaseMetadata` and `ModuleMetadata` fields on
`BuiltRelease`. The `release.Metadata` struct mixed module-level and
release-level fields, making it impossible to distinguish which identity UUID
belonged to the module vs. the release without reading field names.

**Migration**: Code that accessed `BuiltRelease.Metadata.Identity` SHALL use
`BuiltRelease.ModuleMetadata.UUID`. Code that accessed
`BuiltRelease.Metadata.ReleaseIdentity` SHALL use
`BuiltRelease.ReleaseMetadata.UUID`. Code that accessed
`BuiltRelease.Metadata.Name` (release name) SHALL use
`BuiltRelease.ReleaseMetadata.Name`. Code that accessed
`BuiltRelease.Metadata.FQN` or `BuiltRelease.Metadata.Version` SHALL use
`BuiltRelease.ModuleMetadata.FQN` or `BuiltRelease.ModuleMetadata.Version`.
