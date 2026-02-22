# Core Component

## Purpose

Defines the `core.Component` type and related helpers in `internal/core/`. These replace the former `internal/build/component/` package and provide a CUE-schema-aligned Go representation of a module component, including structural validation and concreteness detection.

---

## Requirements

### Requirement: core.Component is structurally aligned with the #Component CUE schema

`core.Component` SHALL be extended so its Go fields mirror the `#Component` CUE schema. The type SHALL have the following fields:

```go
type Component struct {
    ApiVersion string
    Kind       string
    Metadata   *ComponentMetadata
    Resources  map[string]cue.Value  // keyed by resource FQN
    Traits     map[string]cue.Value  // keyed by trait FQN (may be empty)
    Blueprints map[string]cue.Value  // keyed by blueprint FQN (may be empty)
    Spec       cue.Value             // merged component spec
    Value      cue.Value             // full component cue.Value
}
```

The existing flat `Name`, `Labels`, and `Annotations` fields on `Component` SHALL be moved into `ComponentMetadata`.

#### Scenario: Extracted component has ComponentMetadata sub-type
- **WHEN** `core.ExtractComponents()` extracts a component
- **THEN** the returned `*core.Component` SHALL have a non-nil `Metadata` field
- **AND** `Metadata.Name` SHALL match the component's `metadata.name` CUE field
- **AND** `Metadata.Labels` SHALL be populated from the component's `metadata.labels` map

#### Scenario: Extracted component has Resources map
- **WHEN** `core.ExtractComponents()` extracts a component that defines `#resources`
- **THEN** `Component.Resources` SHALL contain one entry per resource FQN
- **AND** each value SHALL be the corresponding resource `cue.Value`

#### Scenario: Extracted component has Traits map (optional)
- **WHEN** `core.ExtractComponents()` extracts a component that defines `#traits`
- **THEN** `Component.Traits` SHALL contain one entry per trait FQN
- **WHEN** a component does not define `#traits`
- **THEN** `Component.Traits` SHALL be an initialized empty map and no error SHALL be raised

#### Scenario: Extracted component has Spec value
- **WHEN** `core.ExtractComponents()` extracts a component that defines a `spec` field
- **THEN** `Component.Spec` SHALL be the `cue.Value` of the component's `spec` field

---

### Requirement: ComponentMetadata sub-type carries component identity fields

`core.ComponentMetadata` SHALL be introduced as a distinct struct type with fields `Name string`, `Labels map[string]string`, and `Annotations map[string]string`.

#### Scenario: ComponentMetadata maps are initialized
- **WHEN** a component is extracted by `core.ExtractComponents()`
- **THEN** `ComponentMetadata.Labels` SHALL be a non-nil map
- **AND** `ComponentMetadata.Annotations` SHALL be a non-nil map

---

### Requirement: Component.Validate() performs structural validation

`*core.Component` SHALL expose a `Validate() error` receiver method that checks structural correctness without requiring CUE concreteness. The method SHALL verify:

- `Metadata != nil`
- `Metadata.Name != ""`
- `len(Resources) > 0`
- `Value.Exists()`

#### Scenario: Structurally complete component passes Validate()
- **WHEN** `Validate()` is called on a component with non-nil Metadata, non-empty Name, at least one Resource, and an existing Value
- **THEN** `Validate()` SHALL return `nil`

#### Scenario: Component with no Resources fails Validate()
- **WHEN** `Validate()` is called on a component with an empty or nil Resources map
- **THEN** `Validate()` SHALL return a non-nil error

#### Scenario: Component with nil Metadata fails Validate()
- **WHEN** `Validate()` is called on a component with a nil Metadata field
- **THEN** `Validate()` SHALL return a non-nil error

#### Scenario: Non-concrete spec does not fail Validate()
- **WHEN** `Validate()` is called on a schema-level component whose `Spec` is not concrete
- **THEN** `Validate()` SHALL return `nil` — structural validation does not require concreteness

---

### Requirement: Component.IsConcrete() reports CUE concreteness of the component value

`*core.Component` SHALL expose an `IsConcrete() bool` receiver method that returns `true` when `Value.Validate(cue.Concrete(true))` returns `nil`. This method distinguishes schema-level components (extracted during `Load()`) from build-phase components (extracted after `FillPath()` during `Build()`).

#### Scenario: Schema-level component is not concrete
- **WHEN** `IsConcrete()` is called on a component extracted from `#components` before user values are applied
- **THEN** `IsConcrete()` SHALL return `false` (spec fields reference `#config` type constraints, not resolved values)

#### Scenario: Build-phase component is concrete
- **WHEN** `IsConcrete()` is called on a component extracted after `FillPath("#config", userValues)`
- **THEN** `IsConcrete()` SHALL return `true`

#### Scenario: Build() gates concrete-only operations on IsConcrete()
- **WHEN** `Build()` extracts components from the filled value
- **THEN** `comp.IsConcrete()` SHALL return `true` for every extracted component before they are passed to the executor
- **AND** `Build()` SHALL return a non-nil error if any component is not concrete after `FillPath`

---

### Requirement: core.ExtractComponents() extracts components from a CUE components value

`core.ExtractComponents(v cue.Value) (map[string]*Component, error)` SHALL be a package-level function in `internal/core/`. It SHALL iterate the hidden fields of the given CUE value (representing `#components`), construct a `*Component` for each, and call `comp.Validate()` before including it in the result.

#### Scenario: Components extracted from #components definition
- **WHEN** `core.ExtractComponents()` is called with the `#components` value from a loaded module
- **THEN** the returned map SHALL contain one entry per component defined in `#components`
- **AND** each entry's key SHALL match the component's struct field name (e.g., `"web"`, `"worker"`)

#### Scenario: Empty or missing components value returns empty map
- **WHEN** `core.ExtractComponents()` is called with a value that has no iterable hidden fields
- **THEN** the function SHALL return an empty (non-nil) map and a nil error

#### Scenario: Invalid component causes error
- **WHEN** `core.ExtractComponents()` encounters a component that fails `Validate()`
- **THEN** the function SHALL return a non-nil error identifying the component name and validation failure

---

### Requirement: internal/build/component/ package is removed

The `internal/build/component/` package SHALL be deleted. All import sites (10 files) SHALL be updated to use `core.Component` from `internal/core/`.

#### Scenario: No import of internal/build/component remains
- **WHEN** the codebase is compiled after this change
- **THEN** no Go file SHALL import `github.com/opmodel/cli/internal/build/component`
- **AND** all references to `component.Component` SHALL be replaced with `core.Component`

---

### Requirement: Component types live in a dedicated subpackage

`Component`, `ComponentMetadata`, and `ExtractComponents` SHALL be defined in `internal/core/component` (package `component`), mirroring `component.cue` in the CUE catalog. The package SHALL only import `internal/core` (for base types), CUE SDK, and stdlib.

#### Scenario: Package compiles with correct import path
- **WHEN** a consumer imports `github.com/opmodel/cli/internal/core/component`
- **THEN** all three exported symbols (`Component`, `ComponentMetadata`, `ExtractComponents`) are accessible and the package compiles without referencing `internal/core` for these types

#### Scenario: No circular imports
- **WHEN** `internal/core/component` is loaded
- **THEN** it SHALL NOT import any package that is higher in the chain (`module`, `modulerelease`, `transformer`, `provider`)

### Requirement: Behavioral equivalence after move

`ExtractComponents` SHALL produce identical output to the implementation it replaces in `internal/core/component.go`.

#### Scenario: Component extraction returns same results
- **WHEN** `ExtractComponents` is called with a valid CUE value
- **THEN** the returned `map[string]*Component` is identical in structure and content to what the previous implementation produced

#### Scenario: Validation errors are unchanged
- **WHEN** `ExtractComponents` is called with a CUE value containing a component that fails `Validate()`
- **THEN** the same error message and type are returned as before
