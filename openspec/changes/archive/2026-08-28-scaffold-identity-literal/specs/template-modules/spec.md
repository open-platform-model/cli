## MODIFIED Requirements

### Requirement: Templates are published modules in a reserved segment

The official templates — `minimal`, `standard`, `advanced` — SHALL be real CUE modules hosted in the cli repo and published to the reserved `opmodel.dev/templates/<name>` segment by the cli's release pipeline through `opm module publish`, passing every publish gate. A template that fails a gate SHALL fail the release. The segment is module-kind, cli-CI-published only, and the name `index` within it is reserved. Publish itself never skips an already-published tag; the release job SHALL filter to unpublished versions before invoking it.

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

### Requirement: init scaffolds by fetch and re-identification

`opm mod init <new-module-path> [template]` SHALL fetch the template module, copy its source tree, and re-identify it to the new path: the `cue.mod` `module:` line, the identity package's `ModulePath` (with `Version` set to the initial version as a plain string literal), every literal self-import, and every package clause (renamed to the new snake leaf). Metadata SHALL be untouched — it derives from the identity package. The template's own identity SHALL NOT appear anywhere in the scaffold. A scaffolded module SHALL pass `opm module vet`, load through the kernel's module loader, and pass every `publish --dry-run` gate except already-published, unmodified. Init requires a reachable registry for an uncached template and SHALL refuse offline naming the expansion and registry tried; no fallback template is embedded.

#### Scenario: Scaffold is publishable and leak-free

- **WHEN** a module is scaffolded from any official template
- **THEN** `publish --dry-run` reports GO and no file carries the template's identity or path

#### Scenario: Scaffold identity is a literal

- **WHEN** a module is scaffolded or repaired by `opm mod init`
- **THEN** its `identity/identity.cue` declares `Version: "0.1.0"` as a plain literal with no default arm and no `#VersionType`
- **AND** `opm module build` on the scaffold passes the loader shape gate

#### Scenario: Offline refuses honestly

- **WHEN** init runs with the registry unreachable and the template uncached
- **THEN** it refuses, naming the expanded template path and the registry
