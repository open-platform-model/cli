## MODIFIED Requirements

### Requirement: Internal release parsing returns a barebones BundleRelease without validation
The system SHALL provide an internal `GetReleaseFile` release-parsing API that accepts an absolute path to a release file, detects when the file contains a `BundleRelease`, and returns a barebones `render.BundleRelease` (previously `bundlerelease.BundleRelease`). The `BundleRelease` and `BundleReleaseMetadata` types SHALL reside in `pkg/render`.

#### Scenario: Parse bundle release file
- **WHEN** `GetReleaseFile` is called with a release file containing a `BundleRelease`
- **THEN** it SHALL return a barebones `render.BundleRelease`
- **AND** the returned release SHALL contain `RawCUE`
- **AND** the returned release SHALL contain concrete decoded `Metadata` of type `*render.BundleReleaseMetadata`
