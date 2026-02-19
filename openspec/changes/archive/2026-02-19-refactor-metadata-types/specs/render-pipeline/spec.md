## ADDED Requirements

### Requirement: RenderResult carries separate module and release metadata

The `RenderResult` struct SHALL carry both a `Module` field of type `ModuleMetadata` and a `Release` field of type `ReleaseMetadata`, providing clean separation between module-level and release-level metadata.

#### Scenario: Module metadata populated on RenderResult

- **WHEN** the pipeline produces a `RenderResult`
- **THEN** `RenderResult.Module` SHALL contain the module's canonical name, FQN, version, UUID (module identity), labels, and components

#### Scenario: Release metadata populated on RenderResult

- **WHEN** the pipeline produces a `RenderResult`
- **THEN** `RenderResult.Release` SHALL contain the release name, namespace, UUID (release identity), labels, and components

#### Scenario: Module-level fields not duplicated on release metadata

- **WHEN** a consumer accesses `RenderResult.Release`
- **THEN** the `ReleaseMetadata` type SHALL NOT contain module-level fields (Version, FQN, ModuleName, module UUID)

### Requirement: ModuleMetadata type definition

The `module.ModuleMetadata` struct SHALL contain the following fields, each with a corresponding `json:"..."` struct tag:
- `Name string` — canonical module name from `module.metadata.name`
- `DefaultNamespace string` — default namespace from the module
- `FQN string` — fully qualified module name
- `Version string` — module version (semver)
- `UUID string` — module identity UUID from `#Module.metadata.identity`
- `Labels map[string]string` — module labels
- `Annotations map[string]string` — module annotations (may be empty)
- `Components []string` — component names in the module

#### Scenario: ModuleMetadata populated from CUE evaluation

- **WHEN** a release is built from a module
- **THEN** `ModuleMetadata.Name` SHALL be the canonical module name (from `module.metadata.name`)
- **AND** `ModuleMetadata.FQN` SHALL be the fully qualified module name
- **AND** `ModuleMetadata.Version` SHALL be the module semver version
- **AND** `ModuleMetadata.UUID` SHALL be the module identity UUID
- **AND** `ModuleMetadata.Labels` SHALL contain the module labels
- **AND** `ModuleMetadata.Components` SHALL list the component names

#### Scenario: ModuleMetadata JSON serialization

- **WHEN** `ModuleMetadata` is marshaled to JSON
- **THEN** all fields SHALL use their defined `json:"..."` tag names

### Requirement: ReleaseMetadata type definition

The `release.ReleaseMetadata` struct SHALL contain the following fields, each with a corresponding `json:"..."` struct tag:
- `Name string` — release name (from `--name` flag or module name)
- `Namespace string` — target namespace
- `UUID string` — release identity UUID (deterministic UUID5 computed from fqn+name+namespace)
- `Labels map[string]string` — release labels
- `Annotations map[string]string` — release annotations (may be empty)
- `Components []string` — component names rendered in this release

#### Scenario: ReleaseMetadata populated from CUE evaluation

- **WHEN** a release is built from a module
- **THEN** `ReleaseMetadata.Name` SHALL be the release name (from `--name` or module name)
- **AND** `ReleaseMetadata.Namespace` SHALL be the target namespace
- **AND** `ReleaseMetadata.UUID` SHALL be the deterministic release identity UUID
- **AND** `ReleaseMetadata.Labels` SHALL contain the release labels
- **AND** `ReleaseMetadata.Components` SHALL list the component names

#### Scenario: ReleaseMetadata JSON serialization

- **WHEN** `ReleaseMetadata` is marshaled to JSON
- **THEN** all fields SHALL use their defined `json:"..."` tag names

### Requirement: TransformerContext uses ModuleMetadata and ReleaseMetadata

The `TransformerContext` SHALL hold references to both `ModuleMetadata` and `ReleaseMetadata` instead of the former `TransformerModuleReleaseMetadata` type.

#### Scenario: TransformerContext populated from both metadata types

- **WHEN** `NewTransformerContext()` is called with a `BuiltRelease` and `LoadedComponent`
- **THEN** the resulting `TransformerContext` SHALL have `ModuleMetadata` populated with module-level fields (Name, FQN, Version, UUID, Labels)
- **AND** `ReleaseMetadata` populated with release-level fields (Name, Namespace, UUID, Labels)

#### Scenario: CUE output map unchanged

- **WHEN** `TransformerContext.ToMap()` is called
- **THEN** the resulting map SHALL contain a `#moduleReleaseMetadata` key with the same field names as before: `name`, `namespace`, `fqn`, `version`, `identity`, and optionally `labels`
- **AND** the `identity` value SHALL be the release UUID (from `ReleaseMetadata.UUID`)
- **AND** the `name` value SHALL be the release name (from `ReleaseMetadata.Name`)
- **AND** the `fqn` value SHALL be the module FQN (from `ModuleMetadata.FQN`)
- **AND** the `version` value SHALL be the module version (from `ModuleMetadata.Version`)

## REMOVED Requirements

### Requirement: TransformerMetadata type

**Reason**: The `release.TransformerMetadata` struct is obsoleted by `ModuleMetadata` + `ReleaseMetadata` collectively containing all required fields. The `ReleaseIdentity` → `Identity` rename that `TransformerMetadata` performed is no longer needed because each type has its own unambiguous `UUID` field.

**Migration**: `TransformerContext` now holds both `ModuleMetadata` and `ReleaseMetadata` directly. `ToMap()` composes the CUE output from both types, producing identical output.

## MODIFIED Requirements

### Requirement: RenderResult.Module MUST contain source module metadata

RenderResult SHALL carry module metadata in a `Module` field of type `ModuleMetadata`. The `ModuleMetadata` type SHALL include `Name`, `DefaultNamespace`, `FQN`, `Version`, `UUID`, `Labels`, `Annotations`, and `Components` fields. This replaces the previous `ModuleMetadata` definition which had only `Name`, `Namespace`, `Version`, `Labels`, and `Components`.

#### Scenario: Module metadata available to all consumers

- **WHEN** a consumer receives a `RenderResult`
- **THEN** `result.Module.Name` SHALL be the canonical module name
- **AND** `result.Module.Version` SHALL be the module semver version
- **AND** `result.Module.FQN` SHALL be the fully qualified module name
- **AND** `result.Module.UUID` SHALL be the module identity UUID
- **AND** `result.Module.Labels` SHALL contain the module labels
- **AND** `result.Module.Components` SHALL list the component names
