## Purpose

Defines the parse-stage and process-stage contract for `BundleRelease` handling across internal release parsing and public release-processing APIs.

## Requirements

### Requirement: Internal release parsing returns a barebones BundleRelease without validation
The system SHALL provide an internal `GetReleaseFile` release-parsing API that accepts an absolute path to a release file, detects when the file contains a `BundleRelease`, and returns a barebones `bundlerelease.BundleRelease` without validating values or requiring a filled `#bundle` reference.

#### Scenario: Parse bundle release file with unresolved bundle reference
- **WHEN** `GetReleaseFile` is called with an absolute `release.cue` path containing a `BundleRelease`
- **AND** the release expects later `#bundle` resolution or injection
- **THEN** it SHALL return a barebones `bundlerelease.BundleRelease`
- **AND** the returned release SHALL contain `RawCUE`
- **AND** the returned release SHALL contain concrete decoded `Metadata`
- **AND** it SHALL NOT validate values

#### Scenario: Parse bundle release file with concrete bundle reference
- **WHEN** `GetReleaseFile` is called with a `BundleRelease` file whose `#bundle` reference is already concrete
- **THEN** the returned `Bundle` field SHALL contain bundle metadata and raw bundle data when those values are decodable
- **AND** `Values` SHALL remain empty
- **AND** `Releases` SHALL remain empty

### Requirement: BundleRelease exposes processing-stage fields
The `bundlerelease.BundleRelease` type SHALL expose the fields needed for explicit processing stages: `Metadata`, `Bundle`, `RawCUE`, `Releases`, `Config`, and `Values`.

#### Scenario: Barebones bundle release contains only parse-stage fields
- **WHEN** a `bundlerelease.BundleRelease` is returned from `GetReleaseFile`
- **THEN** `Metadata` SHALL be concrete and decoded
- **AND** `Bundle`, `RawCUE`, and `Config` SHALL be set when decodable from the parse-stage release value
- **AND** `Values` SHALL be empty
- **AND** `Releases` SHALL be empty

### Requirement: ProcessBundleRelease validates bundle values and establishes the public API shape
The public release-processing API SHALL provide `ProcessBundleRelease` that validates values against `BundleRelease.Config`, stores the merged validated values in `BundleRelease.Values`, and returns a not-yet-implemented error until full bundle rendering is introduced.

#### Scenario: Bundle validation succeeds before stub exit
- **WHEN** `ProcessBundleRelease` is called with values that satisfy `BundleRelease.Config`
- **THEN** it SHALL store the merged validated value in `BundleRelease.Values`
- **AND** it SHALL return a not-yet-implemented error after validation

#### Scenario: Bundle validation failure stops before any release expansion
- **WHEN** `ProcessBundleRelease` is called with values that do not satisfy `BundleRelease.Config`
- **THEN** it SHALL return a structured config validation error
- **AND** it SHALL NOT populate `BundleRelease.Releases`

#### Scenario: ProcessBundleRelease creates or uses a cue context for bundle processing
- **WHEN** `ProcessBundleRelease` begins processing
- **THEN** it SHALL create or obtain a CUE context suitable for later bundle concretization work
- **AND** this setup SHALL occur before any bundle-specific processing steps beyond validation
