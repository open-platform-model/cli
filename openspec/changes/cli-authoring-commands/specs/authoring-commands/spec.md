# authoring-commands — Delta

## ADDED Requirements

### Requirement: version set writes in place, idempotently, offline

`opm module version set <semver> [path]` and `opm catalog version set <semver> [path]` SHALL write the artifact's `identity/identity.cue` `Version` in place via the schema-fixed-path surgical rewrite, preserving comments, field order, any type assertion on the field, and the release-automation marker line shape. Setting the already-declared version SHALL write nothing — no file modification, no mtime change — and report the no-op as success. The command SHALL perform no registry I/O; an identity file that does not structurally match the schema-fixed shape SHALL be refused with the path it failed to find and a pointer to `opm module vet` for full conformance.

#### Scenario: Idempotent no-op

- **WHEN** `version set 1.3.0` runs against an identity already declaring `1.3.0`
- **THEN** the file is not written and the command reports "already 1.3.0" with exit 0

#### Scenario: Assertion survives the write

- **WHEN** the field is declared `#VersionType & "1.2.0"` and `version set 1.3.0` runs
- **THEN** the result is `#VersionType & "1.3.0"` — the assertion is preserved

#### Scenario: Defaulted declaration stays defaulted

- **WHEN** the field is declared as a defaulted disjunction carrying a release-automation marker
- **THEN** the write replaces the default value and the field remains a defaulted disjunction with the marker intact

#### Scenario: Structural refusal

- **WHEN** the identity file lacks a `Version` field at the schema-fixed path
- **THEN** the command refuses naming the path it failed to find and pointing at vet
