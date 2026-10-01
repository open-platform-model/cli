## MODIFIED Requirements

### Requirement: Templates are published modules in a reserved segment

The official templates — `minimal`, `standard`, `advanced` — SHALL be real CUE modules hosted in the cli repo and published to the reserved `opmodel.dev/templates/<name>` segment by the cli's release pipeline through `opm module publish`, passing every publish gate. A template SHALL also pass `opm module vet` with its own `debugValues`, rendered by the `opm` the release builds, before it is published. A template that fails a gate or the vet SHALL fail the release. The segment is module-kind, cli-CI-published only, and the name `index` within it is reserved. Publish itself never skips an already-published tag; the release job SHALL filter to unpublished versions before invoking it. Because published versions are immutable, a template's tree on main SHALL equal its published content at its declared version: any change to a template's files SHALL bump its identity `Version`. Source: 0011:D25.

A template's identity package SHALL declare `Version` as a concrete string literal and SHALL NOT declare a local `#VersionType`; its `metadata.version` SHALL be a plain reference to that literal. A template tree SHALL load through the kernel's module loader unmodified.

#### Scenario: Templates are gated artifacts

- **WHEN** a template tree violates any publish gate
- **THEN** the cli release fails — a rotten template cannot ship

#### Scenario: Release job is idempotent by filtering

- **WHEN** a release runs with no template version bumped
- **THEN** no publish is invoked and the job succeeds

#### Scenario: Template identity is a literal the kernel loads

- **WHEN** any official template's `identity/identity.cue` is read
- **THEN** `Version` is a quoted SemVer literal with no disjunction and no `#VersionType` declaration in the file
- **AND** `opm module build` on the template tree passes the loader shape gate

#### Scenario: Template renders its own debugValues

- **WHEN** `opm module vet` runs on any official template tree with no values flag
- **THEN** it renders every component and exits zero
