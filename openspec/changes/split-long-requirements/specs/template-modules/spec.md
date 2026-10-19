## REMOVED Requirements

### Requirement: Templates are published modules in a reserved segment
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The official templates are real CUE modules published to the reserved `opmodel.dev/templates/<name>` segment through `opm module publish`", "A template passes `opm module vet` before it is published and a template that fails a gate or the vet fails the release", "The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it", "Publish never skips an already-published tag and the release job invokes it only for the versions its gates found unpublished", "A template's tree is exactly the module that publishes", "The artifact published at a template's version holds exactly that template's tree and any change to the template bumps its identity `Version`", "A template's `Version` is a stable SemVer and its module path is `opmodel.dev/templates/<name>@v<major>`", "The cli's PR check and release job refuse a template that breaks the tree, bump, `Version` or module path rules", "A template's identity package declares `Version` as a concrete string literal and the template tree loads through the kernel's module loader unmodified".

### Requirement: init scaffolds by fetch and re-identification
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched", "The scaffold carries nothing of the template's identity and a scaffolded module passes `opm module vet`, loads through the kernel's module loader and passes the `publish --dry-run` gates", "Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states", "Init requires a reachable registry for an uncached template, refuses offline naming the expansion and the registry tried, and embeds no fallback template".

### Requirement: Shortcut expansion is syntactic and safe
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A bare word expands into the templates segment and a reference with `/` or `.` is a literal path", "An `@vN` suffix floats within a major and a full semver pins a tag", "No suffix selects a default major for an official template only", "`--from` accepts the same forms explicitly and any published module as a clone source, and `-t/--template` is an alias", "A template-only invocation prompts for the new module path interactively and refuses non-interactively".

### Requirement: Module tree layout
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause", "A module tree has no `values.cue` and its default values for validation and debugging live in the `debugValues` field".

## ADDED Requirements

### Requirement: The official templates are real CUE modules published to the reserved `opmodel.dev/templates/<name>` segment through `opm module publish`

The official templates, `minimal`, `standard`, `advanced`, SHALL be real CUE modules hosted in the cli repo and published to the reserved `opmodel.dev/templates/<name>` segment by the cli's release pipeline through `opm module publish`, passing every publish gate.

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: Official templates are published through the release pipeline

- **WHEN** the cli release pipeline publishes the official templates
- **THEN** `minimal`, `standard` and `advanced` are published to the reserved `opmodel.dev/templates/<name>` segment through `opm module publish`, passing every publish gate

### Requirement: A template passes `opm module vet` before it is published and a template that fails a gate or the vet fails the release

A template SHALL also pass `opm module vet` with its own `debugValues`, rendered by the `opm` the release builds, before it is published. A template that fails a gate or the vet SHALL fail the release before any template is published, and the cli release SHALL stay a draft.

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: Templates are gated artifacts

- **WHEN** a template tree violates any publish gate
- **THEN** the cli release fails — a rotten template cannot ship

#### Scenario: Template renders its own debugValues

- **WHEN** `opm module vet` runs on any official template tree with no values flag
- **THEN** it renders every component and exits zero

### Requirement: The `templates` segment is module-kind and cli-CI-published only and no gate reserves a name within it

The segment SHALL be module-kind and cli-CI-published only. No gate SHALL reserve a name within it: the publish gates admit any single-segment name made of lowercase letters, digits, dots, underscores and hyphens, `index` included. (Requirement: The official templates are real CUE modules published to the reserved `opmodel.dev/templates/<name>` segment through `opm module publish`)

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: No name in the segment is reserved

- **WHEN** `opm module publish` checks a module whose path is `opmodel.dev/templates/index@v1`
- **THEN** the namespace gate and the kind gate admit the path, as they do for `opmodel.dev/templates/standard@v1`

### Requirement: Publish never skips an already-published tag and the release job invokes it only for the versions its gates found unpublished

Publish itself never skips an already-published tag; the release job SHALL invoke it only for the versions its gates found unpublished.

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: Release job is idempotent by filtering

- **WHEN** a release runs with no template changed and no template version bumped
- **THEN** no publish is invoked and the job succeeds

### Requirement: A template's tree is exactly the module that publishes

A template's tree SHALL be exactly the module that publishes: it SHALL hold no symbolic link, special file, nested module or other file its module zip leaves out.

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: A template tree holds only what its module zip keeps

- **WHEN** a template's tree is read
- **THEN** it holds no symbolic link, special file, nested module or other file its module zip leaves out

### Requirement: The artifact published at a template's version holds exactly that template's tree and any change to the template bumps its identity `Version`

Because published versions are immutable, the artifact published at a template's version SHALL hold exactly that template's tree. Any change to a template's files, comments and `cue.mod` pins included, SHALL bump its identity `Version` before the next release.

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: A change to a template bumps its identity Version

- **WHEN** a template's files, comments or `cue.mod` pins change
- **THEN** its identity `Version` is bumped before the next release, because the artifact published at a template's version holds exactly that template's tree

### Requirement: A template's `Version` is a stable SemVer and its module path is `opmodel.dev/templates/<name>@v<major>`

A template's `Version` SHALL be a stable SemVer, and SHALL be the highest stable version published for its major (the version `opm module init` resolves) or an unpublished version above it; its module path SHALL be `opmodel.dev/templates/<name>@v<major>`.

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: Template Version and module path

- **WHEN** a template's `Version` and module path are checked
- **THEN** the `Version` is a stable SemVer, the highest stable version published for its major or an unpublished version above it, and the module path is `opmodel.dev/templates/<name>@v<major>`

### Requirement: The cli's PR check and release job refuse a template that breaks the tree, bump, `Version` or module path rules

The cli's PR check and release job SHALL refuse a template that breaks any of these rules. (Requirements: "A template's tree is exactly the module that publishes", "The artifact published at a template's version holds exactly that template's tree and any change to the template bumps its identity `Version`", "A template's `Version` is a stable SemVer and its module path is `opmodel.dev/templates/<name>@v<major>`")

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: Template tree is exactly its published module

- **WHEN** a template tree holds a file its module zip leaves out, such as a symbolic link to a file elsewhere in the repository
- **THEN** the cli's PR check and release job refuse it, whatever its version

#### Scenario: Template version is the one init resolves

- **WHEN** the version on `main` is a prerelease, or an older version than the highest stable version published for its major
- **THEN** the cli's PR check and release job refuse it, so `opm module init` never resolves a version other than the tree on `main`

### Requirement: A template's identity package declares `Version` as a concrete string literal and the template tree loads through the kernel's module loader unmodified

A template's identity package SHALL declare `Version` as a concrete string literal and SHALL NOT declare a local `#VersionType`; its `metadata.version` SHALL be a plain reference to that literal. A template tree SHALL load through the kernel's module loader unmodified.

**Source**: enhancement 0011 ("Official templates are published modules in a reserved `templates` segment").

#### Scenario: Template identity is a literal the kernel loads

- **WHEN** any official template's `identity/identity.cue` is read
- **THEN** `Version` is a quoted SemVer literal with no disjunction and no `#VersionType` declaration in the file
- **AND** `opm module build` on the template tree passes the loader shape gate

### Requirement: `opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched

`opm mod init <new-module-path> [template]` SHALL fetch the template module, copy its source tree, and re-identify it to the new path: the `cue.mod` `module:` line, the identity package's `ModulePath` (with `Version` set to the initial version as a plain string literal), every literal self-import, and every package clause (renamed to the new snake leaf). Metadata SHALL be untouched, it derives from the identity package.

#### Scenario: Scaffold identity is a literal

- **WHEN** a module is scaffolded by `opm mod init`, or repaired so that init creates its `identity/identity.cue`
- **THEN** its `identity/identity.cue` declares `Version` as a plain literal with no default arm and no `#VersionType`
- **AND** a fresh scaffold declares `Version: "0.1.0"`
- **AND** `opm module build` on the result passes the loader shape gate

### Requirement: The scaffold carries nothing of the template's identity and a scaffolded module passes `opm module vet`, loads through the kernel's module loader and passes the `publish --dry-run` gates

The template's own identity SHALL NOT appear anywhere in the scaffold. A scaffolded module SHALL pass `opm module vet`, load through the kernel's module loader, and pass every `publish --dry-run` gate except already-published, unmodified. (Requirement: `opm mod init <new-module-path> [template]` fetches the template module, copies its source tree and re-identifies it to the new path, leaving metadata untouched)

#### Scenario: Scaffold is publishable and leak-free

- **WHEN** a module is scaffolded from any official template
- **THEN** `publish --dry-run` reports GO and no file carries the template's identity or path

### Requirement: Init with no template on a directory that already holds a module tree repairs that tree and adopts the version the tree states

Init with no template, on a directory that already holds a module tree, SHALL repair that tree instead of scaffolding. When repair has to create a missing identity package, `Version` SHALL be the tree's own `metadata.version`; init SHALL NOT choose a version, and SHALL refuse when the tree does not load or states no SemVer version. Repair SHALL leave the `Version` of an identity file that exists untouched.

#### Scenario: Repair adopts the version the tree states

- **WHEN** `opm mod init` repairs a tree that has no `identity/identity.cue` and whose `metadata.version` is `1.2.0`
- **THEN** the identity file it creates declares `Version: "1.2.0"`, not `0.1.0`

#### Scenario: Repair refuses a tree that states no version

- **WHEN** `opm mod init` repairs a tree that has no `identity/identity.cue` and states no `metadata.version`
- **THEN** it refuses naming `metadata.version`, and writes no identity file

### Requirement: Init requires a reachable registry for an uncached template, refuses offline naming the expansion and the registry tried, and embeds no fallback template

Init requires a reachable registry for an uncached template and SHALL refuse offline naming the expansion and registry tried; no fallback template is embedded.

#### Scenario: Offline refuses honestly

- **WHEN** init runs with the registry unreachable and the template uncached
- **THEN** it refuses, naming the expanded template path and the registry

### Requirement: A bare word expands into the templates segment and a reference with `/` or `.` is a literal path

A template reference that is a bare word (letters, digits, underscores) before an optional `@` suffix SHALL expand to `opmodel.dev/templates/<word>`; a reference containing `/` or `.` SHALL be treated as a literal module path and never expanded.

#### Scenario: Bare word expands, path never does

- **WHEN** `opm mod init example.com/modules/app@v0 standard@v1` runs
- **THEN** `standard@v1` resolves inside the reserved segment while the first argument is taken literally

#### Scenario: Typo fails inside the segment

- **WHEN** the bare word names no published template
- **THEN** init refuses naming the expanded path — never falling back elsewhere

### Requirement: An `@vN` suffix floats within a major and a full semver pins a tag

An `@vN` suffix SHALL select the newest release within that major (stable preferred, prerelease fallback); a full semver selects the exact tag.

#### Scenario: A suffix selects the release

- **WHEN** a template reference carries an `@vN` suffix, or a full semver
- **THEN** the `@vN` suffix selects the newest release within that major (stable preferred, prerelease fallback), and the full semver selects the exact tag

### Requirement: No suffix selects a default major for an official template only

No suffix SHALL select the default major of the official template the bare word names, and no other reference has a default major: an unknown bare word and a literal module path with no suffix select the newest release of that path. (Requirement: A bare word expands into the templates segment and a reference with `/` or `.` is a literal path)

#### Scenario: Only official templates get a default major

- **WHEN** `opm mod init example.com/modules/app@v0 standard` runs, and again with `example.com/modules/donor` as the template, both with no suffix
- **THEN** `standard` resolves within the major its table row names
- **AND** `example.com/modules/donor` resolves to its newest release with no major constraint

### Requirement: `--from` accepts the same forms explicitly and any published module as a clone source, and `-t/--template` is an alias

`--from` SHALL accept the same forms explicitly and any published module as a clone source; `-t/--template` SHALL be an alias. (Requirements: "A bare word expands into the templates segment and a reference with `/` or `.` is a literal path", "An `@vN` suffix floats within a major and a full semver pins a tag", "No suffix selects a default major for an official template only")

#### Scenario: --from takes the same forms and -t is an alias

- **WHEN** `opm mod init` is given `--from` with a template reference or any published module
- **THEN** it accepts the same forms explicitly and the published module as a clone source, and `-t/--template` is an alias

### Requirement: A template-only invocation prompts for the new module path interactively and refuses non-interactively

A template-only invocation SHALL prompt for the new module path interactively and refuse non-interactively.

#### Scenario: Interactive form asks for the path

- **WHEN** `opm mod init standard@v1` runs on a terminal
- **THEN** init prompts for the new module path before writing anything

### Requirement: A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause

A module tree SHALL contain `cue.mod/module.cue` (the CUE module file) and `identity/identity.cue` (the identity package declaring `ModulePath` and `Version`). The module body SHALL be a single CUE package at the tree root; its file names are free. The official templates SHALL use `module.cue` and `components.cue` by convention, and every root `.cue` file SHALL carry the same `package` clause.

#### Scenario: Scaffold carries the mandatory files

- **WHEN** `opm mod init example.com/modules/my_app@v0 standard` completes
- **THEN** the scaffold SHALL contain `cue.mod/module.cue` and `identity/identity.cue`
- **AND** every root `.cue` file SHALL declare `package my_app`

### Requirement: A module tree has no `values.cue` and its default values for validation and debugging live in the `debugValues` field

There SHALL be no `values.cue`: default values for validation and debugging live in the module's `debugValues` field, which `opm module vet` requires. (Requirement: A module tree contains `cue.mod/module.cue` and `identity/identity.cue`, the module body is a single CUE package at the tree root and every root `.cue` file carries the same `package` clause)

#### Scenario: No values.cue is required

- **WHEN** `opm module vet` runs on a tree that has no `values.cue`
- **THEN** it SHALL validate the module with `debugValues` and SHALL NOT report a missing file
