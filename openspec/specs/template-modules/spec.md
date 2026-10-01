# Capability: template-modules

## Purpose

Templates are real published CUE modules, not embedded text (enhancement 0011 D20/D25, change cli-template-modules): the official set lives in the cli repo and ships to the reserved `opmodel.dev/templates` segment by the cli's own release pipeline through `opm module publish`, passing the same gates as any module. `opm mod init` consumes them by fetch and wholesale re-identification — the scaffold carries the new module's identity and nothing of the template's — with a syntactic shortcut grammar mapping bare template names into the reserved segment.

## Requirements

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

### Requirement: Shortcut expansion is syntactic and safe

A template reference that is a bare word (letters, digits, underscores) before an optional `@` suffix SHALL expand to `opmodel.dev/templates/<word>`; a reference containing `/` or `.` SHALL be treated as a literal module path and never expanded. An `@vN` suffix selects the newest release within that major (stable preferred, prerelease fallback); a full semver selects the exact tag; no suffix selects the CLI's default major. `--from` accepts the same forms explicitly and any published module as a clone source; `-t/--template` is an alias. A template-only invocation SHALL prompt for the new module path interactively and refuse non-interactively.

#### Scenario: Bare word expands, path never does

- **WHEN** `opm mod init example.com/modules/app@v0 standard@v1` runs
- **THEN** `standard@v1` resolves inside the reserved segment while the first argument is taken literally

#### Scenario: Typo fails inside the segment

- **WHEN** the bare word names no published template
- **THEN** init refuses naming the expanded path — never falling back elsewhere

#### Scenario: Interactive form asks for the path

- **WHEN** `opm mod init standard@v1` runs on a terminal
- **THEN** init prompts for the new module path before writing anything

### Requirement: template list

`opm module template list` SHALL print the official template set — name, description, default major — from the same table that drives shortcut expansion, offline.

#### Scenario: List matches expansion

- **WHEN** a name printed by `template list` is used as a shortcut
- **THEN** it expands and resolves

### Requirement: Module tree layout

A module tree SHALL contain `cue.mod/module.cue` (the CUE module file) and `identity/identity.cue` (the identity package declaring `ModulePath` and `Version`). The module body SHALL be a single CUE package at the tree root; its file names are free. The official templates use `module.cue` and `components.cue` by convention, and every root `.cue` file carries the same `package` clause. There SHALL be no `values.cue`: default values for validation and debugging live in the module's `debugValues` field, which `opm module vet` requires.

#### Scenario: Scaffold carries the mandatory files

- **WHEN** `opm mod init example.com/modules/my_app@v0 standard` completes
- **THEN** the scaffold SHALL contain `cue.mod/module.cue` and `identity/identity.cue`
- **AND** every root `.cue` file SHALL declare `package my_app`

#### Scenario: No values.cue is required

- **WHEN** `opm module vet` runs on a tree that has no `values.cue`
- **THEN** it SHALL validate the module with `debugValues` and SHALL NOT report a missing file
