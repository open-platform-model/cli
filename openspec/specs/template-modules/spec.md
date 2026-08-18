# Capability: template-modules

## Purpose

Templates are real published CUE modules, not embedded text (enhancement 0011 D20/D25, change cli-template-modules): the official set lives in the cli repo and ships to the reserved `opmodel.dev/templates` segment by the cli's own release pipeline through `opm module publish`, passing the same gates as any module. `opm mod init` consumes them by fetch and wholesale re-identification — the scaffold carries the new module's identity and nothing of the template's — with a syntactic shortcut grammar mapping bare template names into the reserved segment.

## Requirements

### Requirement: Templates are published modules in a reserved segment

The official templates — `minimal`, `standard`, `advanced` — SHALL be real CUE modules hosted in the cli repo and published to the reserved `opmodel.dev/templates/<name>` segment by the cli's release pipeline through `opm module publish`, passing every publish gate. A template that fails a gate SHALL fail the release. The segment is module-kind, cli-CI-published only, and the name `index` within it is reserved. Publish itself never skips an already-published tag; the release job SHALL filter to unpublished versions before invoking it.

#### Scenario: Templates are gated artifacts

- **WHEN** a template tree violates any publish gate
- **THEN** the cli release fails — a rotten template cannot ship

#### Scenario: Release job is idempotent by filtering

- **WHEN** a release runs with no template version bumped
- **THEN** no publish is invoked and the job succeeds

### Requirement: init scaffolds by fetch and re-identification

`opm mod init <new-module-path> [template]` SHALL fetch the template module, copy its source tree, and re-identify it to the new path: the `cue.mod` `module:` line, the identity package's `ModulePath` (with `Version` reset to the defaulted initial form), every literal self-import, and every package clause (renamed to the new snake leaf). Metadata SHALL be untouched — it derives from the identity package. The template's own identity SHALL NOT appear anywhere in the scaffold. A scaffolded module SHALL pass `opm module vet` and every `publish --dry-run` gate except already-published, unmodified. Init requires a reachable registry for an uncached template and SHALL refuse offline naming the expansion and registry tried; no fallback template is embedded.

#### Scenario: Scaffold is publishable and leak-free

- **WHEN** a module is scaffolded from any official template
- **THEN** `publish --dry-run` reports GO and no file carries the template's identity or path

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
