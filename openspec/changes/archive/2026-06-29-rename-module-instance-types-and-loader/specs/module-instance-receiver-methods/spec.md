## ADDED Requirements

<!-- Renamed from `module-release-receiver-methods` (enhancement 0002 D10). Spec dir is git mv'd at archive. -->

### Requirement: ModuleInstance exposes ValidateValues method

`core.ModuleInstance` SHALL expose a `ValidateValues() error` receiver method that
validates the user-supplied `Values` field against the `Module.Config` CUE schema.
Validation SHALL use recursive CUE field walking on the already-populated
`cue.Value` fields; no external CUE context SHALL be required.

#### Scenario: Valid values pass validation

- **WHEN** `inst.ValidateValues()` is called on a `ModuleInstance` whose `Values` satisfy the `Module.Config` schema
- **THEN** the method SHALL return `nil`

#### Scenario: Values that violate schema return error

- **WHEN** `inst.ValidateValues()` is called on a `ModuleInstance` whose `Values` contain a field that fails `Module.Config` type or constraint checks
- **THEN** the method SHALL return a non-nil error describing which field(s) failed validation
- **AND** the error SHALL be or wrap a `*core.ValidationError`

#### Scenario: ValidateValues does not require concrete values

- **WHEN** `inst.ValidateValues()` is called on a `ModuleInstance` whose `Values` are structurally valid but not yet fully concrete
- **THEN** the method SHALL return `nil` (schema conformance only, not concreteness)

#### Scenario: ValidateValues called without user-supplied values

- **WHEN** a `ModuleInstance` is built with no additional values files (only module defaults)
- **THEN** `inst.ValidateValues()` SHALL return `nil`

### Requirement: ModuleInstance exposes Validate method

`core.ModuleInstance` SHALL expose a `Validate() error` receiver method that
checks that all components in `Components` are concrete CUE values and that the
instance is ready for transformer matching. This is a readiness gate, not a schema
check; it SHALL NOT re-run values-against-config validation.

#### Scenario: Fully concrete instance passes validation

- **WHEN** `inst.Validate()` is called on a `ModuleInstance` where all component `cue.Value` fields are concrete
- **THEN** the method SHALL return `nil`

#### Scenario: Non-concrete component causes validation failure

- **WHEN** `inst.Validate()` is called on a `ModuleInstance` where at least one component's `Value` field is not concrete (e.g., still has open constraints)
- **THEN** the method SHALL return a non-nil error identifying the non-concrete component(s)
- **AND** the error SHALL be or wrap a `*core.ValidationError`

#### Scenario: Empty components map passes validation

- **WHEN** `inst.Validate()` is called on a `ModuleInstance` with an empty `Components` map
- **THEN** the method SHALL return `nil` (a module with no components is valid — it produces no resources)

### Requirement: ValidateValues and Validate are called in sequence

The `build/pipeline.go` BUILD phase SHALL call `inst.ValidateValues()` before
`inst.Validate()`. If `ValidateValues()` returns an error, `Validate()` SHALL NOT
be called.

#### Scenario: Values failure short-circuits instance validation

- **WHEN** `inst.ValidateValues()` returns an error
- **THEN** `pipeline.Render()` SHALL return that error without calling `inst.Validate()`

#### Scenario: Both methods called on success

- **WHEN** `inst.ValidateValues()` returns `nil`
- **THEN** `pipeline.Render()` SHALL call `inst.Validate()` immediately after

### Requirement: ValidateValues and Validate carry no side effects

Both `ValidateValues()` and `Validate()` SHALL be pure read operations. They
SHALL NOT mutate any field on `ModuleInstance`.

#### Scenario: Repeated calls return identical results

- **WHEN** `ValidateValues()` or `Validate()` is called multiple times on the same `ModuleInstance`
- **THEN** each call SHALL return the same result

#### Scenario: ModuleInstance fields unchanged after validation

- **WHEN** either validation method is called
- **THEN** all fields on the `ModuleInstance` SHALL have the same values as before the call
