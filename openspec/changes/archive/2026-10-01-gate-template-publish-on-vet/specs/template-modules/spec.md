## MODIFIED Requirements

### Requirement: Templates are published modules in a reserved segment

The official templates — `minimal`, `standard`, `advanced` — SHALL be real CUE modules hosted in the cli repo and published to the reserved `opmodel.dev/templates/<name>` segment by the cli's release pipeline through `opm module publish`, passing every publish gate. A template SHALL also pass `opm module vet` with its own `debugValues`, rendered by the `opm` the release builds, before it is published. A template that fails a gate or the vet SHALL fail the release before any template is published, and the cli release SHALL stay a draft. The segment is module-kind, cli-CI-published only, and the name `index` within it is reserved. Publish itself never skips an already-published tag; the release job SHALL invoke it only for the versions its gates found unpublished. A template's tree SHALL be exactly the module that publishes: it SHALL hold no symbolic link, special file, nested module or other file its module zip leaves out. Because published versions are immutable, the artifact published at a template's version SHALL hold exactly that template's tree. Any change to a template's files, comments and `cue.mod` pins included, SHALL bump its identity `Version` before the next release. A template's `Version` SHALL be a stable SemVer, and SHALL be the highest stable version published for its major (the version `opm module init` resolves) or an unpublished version above it; its module path SHALL be `opmodel.dev/templates/<name>@v<major>`. The cli's PR check and release job refuse a template that breaks any of these rules. Source: 0011:D25.

A template's identity package SHALL declare `Version` as a concrete string literal and SHALL NOT declare a local `#VersionType`; its `metadata.version` SHALL be a plain reference to that literal. A template tree SHALL load through the kernel's module loader unmodified.

#### Scenario: Templates are gated artifacts

- **WHEN** a template tree violates any publish gate
- **THEN** the cli release fails — a rotten template cannot ship

#### Scenario: Release job is idempotent by filtering

- **WHEN** a release runs with no template changed and no template version bumped
- **THEN** no publish is invoked and the job succeeds

#### Scenario: Template identity is a literal the kernel loads

- **WHEN** any official template's `identity/identity.cue` is read
- **THEN** `Version` is a quoted SemVer literal with no disjunction and no `#VersionType` declaration in the file
- **AND** `opm module build` on the template tree passes the loader shape gate

#### Scenario: Template renders its own debugValues

- **WHEN** `opm module vet` runs on any official template tree with no values flag
- **THEN** it renders every component and exits zero

#### Scenario: Template tree is exactly its published module

- **WHEN** a template tree holds a file its module zip leaves out, such as a symbolic link to a file elsewhere in the repository
- **THEN** the cli's PR check and release job refuse it, whatever its version

#### Scenario: Template version is the one init resolves

- **WHEN** the version on `main` is a prerelease, or an older version than the highest stable version published for its major
- **THEN** the cli's PR check and release job refuse it, so `opm module init` never resolves a version other than the tree on `main`
