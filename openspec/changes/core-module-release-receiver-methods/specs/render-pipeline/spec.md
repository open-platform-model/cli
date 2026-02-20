## MODIFIED Requirements

### Requirement: BuiltRelease carries typed metadata directly

The internal `BuiltRelease` type (output of `Builder.Build()`) SHALL carry
`ReleaseMetadata` and `ModuleMetadata` as direct struct fields, populated by the
builder from the fully evaluated CUE value. No intermediate `Metadata` grab-bag
struct SHALL exist. The builder is responsible for extracting both metadata types
from the CUE value before returning `BuiltRelease`.

`TransformerContext` SHALL read module name from `BuiltRelease.ModuleMetadata.Name`
(the canonical module name) rather than from the release name field.

`Builder.Build()` SHALL return `*core.ModuleRelease` instead of the build-internal
`BuiltRelease` type. The `core.ModuleRelease` type SHALL carry the same typed
metadata fields (`Module ModuleMetadata`, `Metadata *ReleaseMetadata`,
`Components map[string]*Component`, `Values cue.Value`). The build-internal
`BuiltRelease` type SHALL be removed.

#### Scenario: Builder populates ModuleMetadata with FQN and version

- **WHEN** `release.Build()` is called on a module that defines `metadata.fqn` and `metadata.version`
- **THEN** the returned `core.ModuleRelease.Module.FQN` SHALL equal the module's `metadata.fqn` value
- **AND** `core.ModuleRelease.Module.Version` SHALL equal the module's `metadata.version` value
- **AND** `core.ModuleRelease.Module.DefaultNamespace` SHALL equal the module's `metadata.defaultNamespace` value

#### Scenario: Builder populates ReleaseMetadata with release-level fields

- **WHEN** `release.Build()` is called with `Name: "my-release"` and `Namespace: "production"`
- **THEN** `core.ModuleRelease.Metadata.Name` SHALL equal `"my-release"`
- **AND** `core.ModuleRelease.Metadata.Namespace` SHALL equal `"production"`
- **AND** `core.ModuleRelease.Metadata.UUID` SHALL be the computed release UUID

#### Scenario: TransformerContext uses canonical module name

- **WHEN** a module with `metadata.name: "my-app"` is rendered with `--name my-app-staging`
- **THEN** `TransformerContext.ModuleMetadata.Name` SHALL equal `"my-app"` (canonical module name)
- **AND** `TransformerContext.ReleaseMetadata.Name` SHALL equal `"my-app-staging"` (release name)

## ADDED Requirements

### Requirement: Pipeline BUILD phase delegates validation to ModuleRelease receiver methods

The `pipeline.Render()` BUILD phase SHALL call `rel.ValidateValues()` and then
`rel.Validate()` on the returned `*core.ModuleRelease` rather than calling
standalone validation functions directly. The pipeline SHALL NOT call validation
functions that bypass these receiver methods.

#### Scenario: BUILD phase calls ValidateValues before Validate

- **WHEN** `pipeline.Render()` executes the BUILD phase
- **THEN** `rel.ValidateValues()` SHALL be called immediately after `release.Build()` returns
- **AND** `rel.Validate()` SHALL be called immediately after `rel.ValidateValues()` returns `nil`

#### Scenario: Validation errors surfaced identically to previous behavior

- **WHEN** user-supplied values fail schema validation after this change
- **THEN** the error returned from `pipeline.Render()` SHALL be the same type and contain the same message as before this change

#### Scenario: Release output is identical before and after this change

- **WHEN** a module that rendered successfully before this change is rendered after
- **THEN** `RenderResult.Resources` SHALL contain the same resources with identical content
- **AND** `RenderResult.Module` and `RenderResult.Release` SHALL contain the same metadata values
- **AND** `RenderResult.Errors` and `RenderResult.Warnings` SHALL be identical
