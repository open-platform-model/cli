# catalog-registry-check — Delta

## ADDED Requirements

### Requirement: `--compat` follows the publish gate's dev-build rules

`opm catalog registry check <path@version> --compat` SHALL apply the same dev-build rules as `opm catalog publish`: a fetched build whose version is a dev prerelease SHALL report `dev-exempt` on the `compat` row instead of a comparison, and the predecessor window for any fetched build SHALL exclude dev tags. The two commands SHALL agree on every build.

#### Scenario: Dev-tagged build reports exempt

- **WHEN** `--compat` runs against a published `-0.dev.*` build
- **THEN** the report's `compat` row reads `dev-exempt` and the exit code is 0 unless another finding exists

#### Scenario: Release build ignores dev history

- **WHEN** `--compat` runs against a release build whose history holds a newer dev tag than the latest release
- **THEN** the comparison runs against the release tag and never names a dev coordinate
