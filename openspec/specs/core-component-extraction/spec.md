# Core Component Extraction

## Purpose

Specifies how `extractComponent()` in `internal/core/component.go` populates `Component.Spec` and `Component.Blueprints` from the CUE-evaluated component value, and how UUID extraction is covered by `TestLoad_ValidModule`.

---

## Requirements

### Requirement: extractComponent populates Component.Spec

`extractComponent()` SHALL look up the `spec` field of the component CUE value and assign it to `comp.Spec` if it exists. `Component.Spec` represents the component's OpenAPI/merged spec and is used by transformers.

#### Scenario: Extracted component has Spec value

- **WHEN** `core.ExtractComponents()` extracts a component that defines a `spec` field
- **THEN** `Component.Spec` SHALL be the `cue.Value` of the component's `spec` field
- **AND** `Component.Spec.Exists()` SHALL return `true`

#### Scenario: Component without spec field has zero Spec value

- **WHEN** `core.ExtractComponents()` extracts a component that does not define a `spec` field
- **THEN** `Component.Spec.Exists()` SHALL return `false`
- **AND** no error SHALL be raised

### Requirement: extractComponent initializes and populates Component.Blueprints

`extractComponent()` SHALL always initialize `Component.Blueprints` to a non-nil map, consistent with how `Component.Traits` is initialized. If the component CUE value contains a `#blueprints` field, each entry SHALL be added to the map keyed by its field name.

#### Scenario: Blueprints map is always non-nil

- **WHEN** `core.ExtractComponents()` extracts any component
- **THEN** `Component.Blueprints` SHALL be a non-nil map
- **AND** ranging over `Component.Blueprints` SHALL NOT panic

#### Scenario: Component with blueprints has entries in map

- **WHEN** `core.ExtractComponents()` extracts a component that defines `#blueprints`
- **THEN** `Component.Blueprints` SHALL contain one entry per blueprint FQN
- **AND** each value SHALL be the corresponding blueprint `cue.Value`

#### Scenario: Component without blueprints has empty map

- **WHEN** `core.ExtractComponents()` extracts a component that does not define `#blueprints`
- **THEN** `Component.Blueprints` SHALL be an initialized empty map and no error SHALL be raised

### Requirement: TestLoad_ValidModule asserts UUID extraction

The `TestLoad_ValidModule` test SHALL assert that `mod.Metadata.UUID` is non-empty after `module.Load()`. The `test-module` fixture SHALL define a `metadata.identity` field so that UUID extraction is exercised end-to-end.

#### Scenario: UUID populated from metadata.identity

- **WHEN** a module defines `metadata: identity: "some-uuid-string"`
- **THEN** `mod.Metadata.UUID` SHALL equal `"some-uuid-string"` after `module.Load()`
- **AND** `TestLoad_ValidModule` SHALL assert this value is non-empty
