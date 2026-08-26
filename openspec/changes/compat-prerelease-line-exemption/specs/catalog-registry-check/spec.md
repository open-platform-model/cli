# catalog-registry-check — Delta

## ADDED Requirements

### Requirement: `--compat` follows the publish gate's prerelease-line rule

`opm catalog registry check <path@version> --compat` SHALL exempt beta and GA members of a fetched build whose version is a release prerelease (`-alpha.N`, `-beta.N`, `-rc.N`) exactly as `opm catalog publish` does, reporting them under `prerelease-exempt` on the `compat` row, and SHALL compare them for a stable fetched version.

#### Scenario: Prerelease build reports exempt members

- **WHEN** `--compat` runs against a published `-alpha.N` build whose beta member narrowed a field relative to the previous alpha
- **THEN** the report's `compat` row shows the member under `prerelease-exempt`, no violation is listed, and the exit code is 0 unless another finding exists

#### Scenario: Stable build is compared

- **WHEN** `--compat` runs against a published stable build with the same narrowing relative to the last alpha
- **THEN** the report lists the violation and exits 2
