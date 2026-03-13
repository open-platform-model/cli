## MODIFIED Requirements

### Requirement: Internal release parsing returns a barebones ModuleRelease without validation
The system SHALL provide an internal `GetReleaseFile` release-parsing API that accepts an absolute path to a release file, detects when the file contains a `ModuleRelease`, and returns a barebones `render.ModuleRelease` (previously `modulerelease.ModuleRelease`) without validating values or requiring a filled `#module` reference. The `ModuleRelease` and `ModuleReleaseMetadata` (previously `ReleaseMetadata`) types SHALL reside in `pkg/render`.

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
