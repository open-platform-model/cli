# Module Release Type

## Purpose

Defines the `pkg/module.Release` type as the fully prepared module release used by the render pipeline.

## Requirements

### Requirement: module.Release represents a fully prepared module release
The `pkg/module` package SHALL export a `Release` struct containing four fields: `Metadata *ReleaseMetadata` (decoded release identity), `Module Module` (the original module definition), `Spec cue.Value` (the concrete, values-filled, complete `#ModuleRelease` CUE value), and `Values cue.Value` (the concrete, merged values applied to the release).

The `Release` struct SHALL NOT contain `RawCUE`, `DataComponents`, or `Config` fields.

#### Scenario: Release invariant — Spec is concrete and complete
- **WHEN** a `*module.Release` exists
- **THEN** `Release.Spec` SHALL be a fully concrete CUE value (passes `cue.Concrete(true)` validation)
- **AND** `Release.Spec` SHALL contain the complete `#ModuleRelease` definition with `#module` filled and `values` filled

#### Scenario: Release invariant — Spec is NOT finalized
- **WHEN** a `*module.Release` exists
- **THEN** `Release.Spec` SHALL preserve CUE definition fields (`#resources`, `#traits`, `#blueprints`) within its `components` subtree
- **AND** `Release.Spec` SHALL NOT have been processed through `cue.Final()` or `finalizeValue`
- **AND** code that needs constraint-free component data for transformer execution SHALL derive it transiently via `finalizeValue` during rendering, not from `Spec`

#### Scenario: Release invariant — Values is concrete and merged
- **WHEN** a `*module.Release` exists
- **THEN** `Release.Values` SHALL be a concrete CUE value representing the merged result of all input values
- **AND** `Release.Values` SHALL have passed validation against `Module.Config`

#### Scenario: Release invariant — Metadata is decoded
- **WHEN** a `*module.Release` exists
- **THEN** `Release.Metadata` SHALL be a non-nil `*ReleaseMetadata` decoded from the concrete `Spec`

### Requirement: Config accessed via Module.Config
The `Release` struct SHALL NOT have a `Config` field. Code that needs the config schema SHALL access it via `Release.Module.Config`.

#### Scenario: Config schema access
- **WHEN** code needs the `#config` schema for a release
- **THEN** it SHALL read `release.Module.Config`
- **AND** there SHALL be no `Config` field on `Release`

### Requirement: Release exposes MatchComponents accessor
The `Release` type SHALL expose a `MatchComponents()` method that returns the schema-preserving components CUE value from `Spec`, suitable for matching. This method SHALL preserve definition fields (`#resources`, `#traits`, `#blueprints`).

#### Scenario: MatchComponents returns schema-preserving components
- **WHEN** `release.MatchComponents()` is called
- **THEN** it SHALL return `release.Spec.LookupPath(cue.ParsePath("components"))`
- **AND** the returned value SHALL preserve CUE definition fields needed for matching

### Requirement: No ExecuteComponents method
The `Release` type SHALL NOT expose an `ExecuteComponents()` method. Finalized, constraint-free components are transient local variables in the rendering pipeline, not stored on the release.

#### Scenario: No stored data components
- **WHEN** code needs finalized components for transformer execution
- **THEN** it SHALL derive them transiently via `finalizeValue` during processing
- **AND** there SHALL be no `DataComponents` field or `ExecuteComponents()` method on `Release`

### Requirement: Old constructor removed
The `pkg/module` package SHALL NOT export the `NewRelease` constructor. Release construction SHALL go through `ParseModuleRelease`.

#### Scenario: No NewRelease function
- **WHEN** code attempts to call `module.NewRelease`
- **THEN** compilation SHALL fail — the function does not exist
