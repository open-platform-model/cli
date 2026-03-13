## Purpose

Defines the parse-stage and process-stage contract for `ModuleRelease` handling across internal release parsing and public release-processing APIs.

## Requirements

### Requirement: Internal release parsing returns a barebones ModuleRelease without validation
The system SHALL provide an internal `GetReleaseFile` release-parsing API that accepts an absolute path to a release file, detects when the file contains a `ModuleRelease`, and returns a barebones `render.ModuleRelease` without validating values or requiring a filled `#module` reference. The `ModuleRelease` and `ModuleReleaseMetadata` types SHALL reside in `pkg/render`.

#### Scenario: Parse module release file with unresolved module reference
- **WHEN** `GetReleaseFile` is called with an absolute `release.cue` path containing a `ModuleRelease`
- **AND** the release expects later `#module` injection
- **THEN** it SHALL return a barebones `render.ModuleRelease`
- **AND** the returned release SHALL contain `RawCUE`
- **AND** the returned release SHALL contain concrete decoded `Metadata` of type `*render.ModuleReleaseMetadata`
- **AND** the returned release SHALL contain `Config` extracted from the release's `#module.#config` view when available
- **AND** it SHALL NOT validate values

#### Scenario: Parse module release file with filled module reference
- **WHEN** `GetReleaseFile` is called with a `ModuleRelease` file whose `#module` reference is already concrete
- **THEN** it SHALL return a barebones `render.ModuleRelease` with the same guarantees as above

### Requirement: Internal release parsing returns a barebones BundleRelease without validation
The system SHALL provide an internal `GetReleaseFile` release-parsing API that accepts an absolute path to a release file, detects when the file contains a `BundleRelease`, and returns a barebones `render.BundleRelease`. The `BundleRelease` and `BundleReleaseMetadata` types SHALL reside in `pkg/render`.

#### Scenario: Parse bundle release file
- **WHEN** `GetReleaseFile` is called with a release file containing a `BundleRelease`
- **THEN** it SHALL return a barebones `render.BundleRelease`
- **AND** the returned release SHALL contain `RawCUE`
- **AND** the returned release SHALL contain concrete decoded `Metadata` of type `*render.BundleReleaseMetadata`

### Requirement: ModuleRelease exposes processing-stage fields
The `render.ModuleRelease` type SHALL expose the fields needed for explicit processing stages: `Metadata`, `Module`, `RawCUE`, `DataComponents`, `Config`, and `Values`.

#### Scenario: Barebones release contains only parse-stage fields
- **WHEN** a `render.ModuleRelease` is returned from `GetReleaseFile`
- **THEN** `Metadata` SHALL be concrete and decoded
- **AND** `Module`, `RawCUE`, and `Config` SHALL be set when decodable from the parse-stage release value
- **AND** `Values` SHALL be empty
- **AND** `DataComponents` SHALL be empty

#### Scenario: Processed release contains validated values and finalized components
- **WHEN** `ProcessModuleRelease` succeeds for a module release
- **THEN** `Values` SHALL contain the merged validated values supplied by the caller
- **AND** `RawCUE` SHALL represent the concrete release value after values have been filled
- **AND** `DataComponents` SHALL contain the finalized data-only `components` field used for transformer execution

### Requirement: Public config validation
The `pkg/render` package SHALL export `ValidateConfig` that validates user-supplied values against a module's `#config` schema.

#### Scenario: Valid config values
- **WHEN** `render.ValidateConfig(schema, values, context, name)` is called with valid values
- **THEN** it SHALL return the merged CUE value and nil error

#### Scenario: Invalid config values
- **WHEN** `render.ValidateConfig(schema, values, context, name)` is called with values that violate the schema
- **THEN** it SHALL return a `*errors.ConfigError` with grouped diagnostics

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

### Requirement: Public release processing orchestrates the full render pipeline
The `pkg/render` package SHALL export `ProcessModuleRelease` and `ProcessBundleRelease` functions (previously in `internal/releaseprocess`) that orchestrate config validation, CUE finalization, matching, and engine invocation. These are the top-level entry points for rendering.

#### Scenario: Process a module release end-to-end
- **WHEN** `render.ProcessModuleRelease(ctx, release, values, provider)` is called
- **THEN** it SHALL validate config, finalize CUE values, compute a match plan, invoke the renderer, and return a `*render.ModuleResult`

#### Scenario: Process a bundle release end-to-end
- **WHEN** `render.ProcessBundleRelease(ctx, release, values, provider)` is called
- **THEN** it SHALL process each child module release and return a `*render.BundleResult`

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
