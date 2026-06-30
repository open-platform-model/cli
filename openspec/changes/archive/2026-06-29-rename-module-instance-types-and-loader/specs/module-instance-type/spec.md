## ADDED Requirements

<!-- Renamed from `module-release-type` (enhancement 0002 D10). Spec dir is git mv'd at archive. -->

### Requirement: module.Instance represents a fully prepared module instance
The `pkg/module` package SHALL export an `Instance` struct containing four fields: `Metadata *InstanceMetadata` (decoded instance identity), `Module Module` (the original module definition), `Spec cue.Value` (the concrete, values-filled, complete `#ModuleInstance` CUE value), and `Values cue.Value` (the concrete, merged values applied to the instance).

The `Instance` struct SHALL NOT contain `RawCUE`, `DataComponents`, or `Config` fields.

#### Scenario: Instance invariant — Spec is concrete and complete
- **WHEN** a `*module.Instance` exists
- **THEN** `Instance.Spec` SHALL be a fully concrete CUE value (passes `cue.Concrete(true)` validation)
- **AND** `Instance.Spec` SHALL contain the complete `#ModuleInstance` definition with `#module` filled and `values` filled

#### Scenario: Instance invariant — Spec is NOT finalized
- **WHEN** a `*module.Instance` exists
- **THEN** `Instance.Spec` SHALL preserve CUE definition fields (`#resources`, `#traits`, `#blueprints`) within its `components` subtree
- **AND** `Instance.Spec` SHALL NOT have been processed through `cue.Final()` or `finalizeValue`
- **AND** code that needs constraint-free component data for transformer execution SHALL derive it transiently via `finalizeValue` during rendering, not from `Spec`

#### Scenario: Instance invariant — Values is concrete and merged
- **WHEN** a `*module.Instance` exists
- **THEN** `Instance.Values` SHALL be a concrete CUE value representing the merged result of all input values
- **AND** `Instance.Values` SHALL have passed validation against `Module.Config`

#### Scenario: Instance invariant — Metadata is decoded
- **WHEN** a `*module.Instance` exists
- **THEN** `Instance.Metadata` SHALL be a non-nil `*InstanceMetadata` decoded from the concrete `Spec`

### Requirement: Config accessed via Module.Config
The `Instance` struct SHALL NOT have a `Config` field. Code that needs the config schema SHALL access it via `Instance.Module.Config`.

#### Scenario: Config schema access
- **WHEN** code needs the `#config` schema for an instance
- **THEN** it SHALL read `instance.Module.Config`
- **AND** there SHALL be no `Config` field on `Instance`

### Requirement: Instance exposes MatchComponents accessor
The `Instance` type SHALL expose a `MatchComponents()` method that returns the schema-preserving components CUE value from `Spec`, suitable for matching. This method SHALL preserve definition fields (`#resources`, `#traits`, `#blueprints`).

#### Scenario: MatchComponents returns schema-preserving components
- **WHEN** `instance.MatchComponents()` is called
- **THEN** it SHALL return `instance.Spec.LookupPath(cue.ParsePath("components"))`
- **AND** the returned value SHALL preserve CUE definition fields needed for matching

### Requirement: No ExecuteComponents method
The `Instance` type SHALL NOT expose an `ExecuteComponents()` method. Finalized, constraint-free components are transient local variables in the rendering pipeline, not stored on the instance.

#### Scenario: No stored data components
- **WHEN** code needs finalized components for transformer execution
- **THEN** it SHALL derive them transiently via `finalizeValue` during processing
- **AND** there SHALL be no `DataComponents` field or `ExecuteComponents()` method on `Instance`

### Requirement: Old constructor removed
The `pkg/module` package SHALL NOT export the `NewInstance` constructor. Instance construction SHALL go through `ParseModuleInstance`.

#### Scenario: No NewInstance function
- **WHEN** code attempts to call `module.NewInstance`
- **THEN** compilation SHALL fail — the function does not exist
