## MODIFIED Requirements

### Requirement: Maintained fixtures track the current schema line

Fixtures and examples consumed by tests or presented as current documentation SHALL import only the current published schema line (`opmodel.dev/core@v2` and the versioned `opmodel.dev/catalogs/opm@v2` packages). Artifacts kept for a retired line SHALL live under an explicitly marked legacy location with a note naming the line they document, and SHALL NOT be consumed by any test.

A maintained module fixture's identity package SHALL declare `Version` as a plain string literal with no default arm and no local `#VersionType`, and its `metadata.version` SHALL reference that literal directly. A fixture SHALL load through the kernel's module loader with no interpolation or other workaround in `metadata`.

#### Scenario: No retired-schema imports outside legacy

- **WHEN** the repo is grepped for retired-line imports (`opmodel.dev/core@v1`, `core/v1alpha1`, `modulerelease@v1`, `opm/v1alpha1`) outside the marked legacy location
- **THEN** there SHALL be no matches in fixtures, examples, or test inputs

#### Scenario: Vet fixtures exercise current schema

- **WHEN** the module vet tests run
- **THEN** their fixtures SHALL be `core@v2`-line modules exercising the same behaviors (valid module, secrets discovery, debug values) as before the port

#### Scenario: Fixture identity is a literal

- **WHEN** `tests/fixtures/modules/*/identity/identity.cue` is read
- **THEN** each declares `Version: "<semver>"` with no `|`, no `*` and no `#VersionType`
- **AND** the matching `module.cue` declares `version: id.Version`
- **AND** `opm module build` on the fixture passes the loader shape gate
