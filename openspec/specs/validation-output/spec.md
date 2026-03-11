# validation-output Specification

## Purpose
TBD - created by archiving change e2e-vet-errors. Update Purpose after archive.
## Requirements
### Requirement: E2E Tests for Validation Grouping
The CLI test suite SHALL verify that validation output during module/release vetting preserves grouped formatting rather than emitting flattened error lines.

#### Scenario: Vetting an invalid module release
- **WHEN** a user runs `opm rel vet` with conflicting or not-allowed values
- **THEN** the CLI output groups errors by error type and prints associated file line paths cleanly without duplicating `ERRO render failed` for every line.

#### Scenario: Vetting an invalid module directly
- **WHEN** a user runs `opm mod vet` with conflicting or not-allowed values
- **THEN** the CLI output groups errors cleanly without duplicating `ERRO values do not satisfy #config` for every line.

