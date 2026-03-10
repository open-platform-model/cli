## Purpose

Defines the parse-stage and process-stage contract for `ModuleRelease` handling across internal release parsing and public release-processing APIs.

## Requirements

### Requirement: Internal release parsing returns a barebones ModuleRelease without validation
The system SHALL provide an internal `GetReleaseFile` release-parsing API that accepts an absolute path to a release file, detects when the file contains a `ModuleRelease`, and returns a barebones `modulerelease.ModuleRelease` without validating values or requiring a filled `#module` reference.

#### Scenario: Parse module release file with unresolved module reference
- **WHEN** `GetReleaseFile` is called with an absolute `release.cue` path containing a `ModuleRelease`
- **AND** the release expects later `#module` injection
- **THEN** it SHALL return a barebones `modulerelease.ModuleRelease`
- **AND** the returned release SHALL contain `RawCUE`
- **AND** the returned release SHALL contain concrete decoded `Metadata`
- **AND** the returned release SHALL contain `Config` extracted from the release's `#module.#config` view when available
- **AND** it SHALL NOT validate values

#### Scenario: Parse module release file with filled module reference
- **WHEN** `GetReleaseFile` is called with a `ModuleRelease` file whose `#module` reference is already concrete
- **THEN** it SHALL return a barebones `modulerelease.ModuleRelease`
- **AND** the returned `Module` field SHALL contain module metadata and raw module data when those values are decodable
- **AND** it SHALL NOT populate `Values`
- **AND** it SHALL NOT populate `DataComponents`

### Requirement: ModuleRelease exposes processing-stage fields
The `modulerelease.ModuleRelease` type SHALL expose the fields needed for explicit processing stages: `Metadata`, `Module`, `RawCUE`, `DataComponents`, `Config`, and `Values`.

#### Scenario: Barebones release contains only parse-stage fields
- **WHEN** a `modulerelease.ModuleRelease` is returned from `GetReleaseFile`
- **THEN** `Metadata` SHALL be concrete and decoded
- **AND** `Module`, `RawCUE`, and `Config` SHALL be set when decodable from the parse-stage release value
- **AND** `Values` SHALL be empty
- **AND** `DataComponents` SHALL be empty

#### Scenario: Processed release contains validated values and finalized components
- **WHEN** `ProcessModuleRelease` succeeds for a module release
- **THEN** `Values` SHALL contain the merged validated values supplied by the caller
- **AND** `RawCUE` SHALL represent the concrete release value after values have been filled
- **AND** `DataComponents` SHALL contain the finalized data-only `components` field used for transformer execution

### Requirement: ValidateConfig returns merged concrete values or combined diagnostics
The public release-processing API SHALL provide `ValidateConfig(schema cue.Value, values []cue.Value)` that validates each supplied values input against the schema, checks merge conflicts across all inputs, and returns structured config errors when validation fails.

#### Scenario: Multiple value files are unified before validation
- **WHEN** `ValidateConfig` is called with two or more `cue.Value` inputs
- **THEN** it SHALL return the unified concrete value when validation succeeds
- **AND** the returned merged value SHALL be the concrete value used by later processing stages

#### Scenario: Conflicting value files are reported alongside schema diagnostics
- **WHEN** two input value files contain conflicting assignments
- **THEN** `ValidateConfig` SHALL return a structured config error describing the value conflict
- **AND** it SHALL still include any per-file schema violations found in the supplied values inputs
- **AND** it SHALL return an empty `cue.Value`

#### Scenario: Schema mismatch returns structured diagnostics
- **WHEN** one or more supplied values inputs do not satisfy the supplied schema
- **THEN** `ValidateConfig` SHALL return a structured config error suitable for grouped CLI display

#### Scenario: Validation failure does not return partial merged values
- **WHEN** `ValidateConfig` detects any schema error or merge conflict
- **THEN** it SHALL return an empty `cue.Value`
- **AND** the caller SHALL rely on the returned `ConfigError` for all user-facing diagnostics

### Requirement: ProcessModuleRelease owns the module-release processing flow
The public release-processing API SHALL provide `ProcessModuleRelease` that validates values, fills them into the release, derives concrete and finalized component views, computes a match plan, renders the release with the given provider, and returns the module render result.

#### Scenario: Processing succeeds with valid values and matching transformers
- **WHEN** `ProcessModuleRelease` is called with a barebones module release, concrete values, and a provider
- **THEN** it SHALL validate the values against `ModuleRelease.Config`
- **AND** it SHALL store the merged validated value in `ModuleRelease.Values`
- **AND** it SHALL fill the merged values into `ModuleRelease.RawCUE`
- **AND** it SHALL derive `DataComponents` from the concrete release's `components` field
- **AND** it SHALL compute a match plan for the concrete component view
- **AND** it SHALL render the release using the supplied provider
- **AND** it SHALL return the module render result

#### Scenario: Validation failure stops processing before matching
- **WHEN** `ProcessModuleRelease` is called with values that do not satisfy `ModuleRelease.Config`
- **THEN** it SHALL return a config validation error
- **AND** it SHALL NOT compute a match plan
- **AND** it SHALL NOT execute any transformers

#### Scenario: Matching uses concrete non-finalized components
- **WHEN** `ProcessModuleRelease` computes the match plan
- **THEN** it SHALL use the concrete `components` value derived from `RawCUE`
- **AND** it SHALL NOT use finalized `DataComponents` for matching

#### Scenario: Transform execution uses finalized data components
- **WHEN** `ProcessModuleRelease` calls the engine renderer
- **THEN** the renderer SHALL execute transforms against `ModuleRelease.DataComponents`
- **AND** it SHALL NOT inject the non-finalized concrete component view into `#transform`
