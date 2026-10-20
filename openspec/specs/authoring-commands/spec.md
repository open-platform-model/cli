# Capability: authoring-commands

## Purpose

The commands that write artifact source deliberately and never invent identity (enhancement 0011, slice cli-authoring-commands, first of two changes): `opm module version set` and `opm catalog version set` put a commit between deciding a version and pushing an artifact. Writes go through `internal/cueedit`'s surgical byte-splice — idempotent, offline, preserving every byte the author committed except the version value itself. The `mod init` scaffold-and-repair rewrite composes the same writers in the sibling change `cli-template-modules`.

## Requirements

### Requirement: version set rewrites the identity Version in place

`opm module version set <semver> [path]` and `opm catalog version set <semver> [path]` SHALL write the artifact's `identity/identity.cue` `Version` in place via the schema-fixed-path surgical rewrite, preserving comments, field order, any type assertion on the field, and the release-automation marker line shape.

#### Scenario: Plain literal rewritten in place

- **WHEN** the field is declared `"1.2.0"` and `version set 1.3.0` runs
- **THEN** the result is `"1.3.0"` with every other byte of the file unchanged

#### Scenario: Assertion survives the write

- **WHEN** the field is declared `#VersionType & "1.2.0"` and `version set 1.3.0` runs
- **THEN** the result is `#VersionType & "1.3.0"` — the assertion is preserved

### Requirement: Setting the already-declared version writes nothing

Setting the already-declared version SHALL write nothing — no file modification, no mtime change — and report the no-op as success.

#### Scenario: Idempotent no-op

- **WHEN** `version set 1.3.0` runs against an identity already declaring `1.3.0`
- **THEN** the file is not written and the command reports "already 1.3.0" with exit 0

### Requirement: version set performs no registry I/O and refuses an identity file of the wrong shape

`opm module version set <semver> [path]` and `opm catalog version set <semver> [path]` SHALL perform no registry I/O; an identity file that does not structurally match the schema-fixed shape SHALL be refused with the path it failed to find and a pointer to the kind's own full-conformance check — `opm module vet` for modules, `opm catalog publish --dry-run` for catalogs (no catalog vet exists, per `cli-catalog-gates`' recorded scope call).

#### Scenario: Structural refusal

- **WHEN** the identity file lacks a `Version` field at the schema-fixed path
- **THEN** the command refuses naming the path it failed to find and pointing at the kind's conformance check

#### Scenario: Catalog refusal points at a command that exists

- **WHEN** `opm catalog version set` refuses a structurally non-conformant identity file
- **THEN** the pointer names `opm catalog publish --dry-run`, never `opm module vet`

#### Scenario: No registry access
- **WHEN** `opm module version set` or `opm catalog version set` runs
- **THEN** the command SHALL perform no registry I/O

### Requirement: The writer tolerates every authored form of the Version field and preserves the form it finds

The writer SHALL tolerate every authored form (plain literal, `#VersionType & "x.y.z"` assertion chain, defaulted disjunction) and preserve the form it finds.

#### Scenario: Defaulted declaration stays defaulted

- **WHEN** the field is declared as a defaulted disjunction carrying a release-automation marker
- **THEN** the write replaces the default value and the field remains a defaulted disjunction with the marker intact

### Requirement: Every identity file the CLI emits uses the plain-literal form

Every identity file the CLI itself emits (scaffold, repair, official templates) SHALL use the plain-literal form; no CLI code path SHALL introduce a default arm into an identity `Version`.

#### Scenario: Emitted identity files
- **WHEN** the CLI emits an identity file (scaffold, repair, official templates)
- **THEN** it SHALL use the plain-literal form, and no CLI code path SHALL introduce a default arm into an identity `Version`
