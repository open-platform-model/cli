## Purpose

Defines the structural conventions for CLI command packages in the OPM codebase. This covers how commands are organized into sub-packages, how configuration is injected, and how the cobra command tree maps to the package layout under `internal/cmd/`.

## Requirements

### Requirement: Commands receive configuration via explicit injection

Global CLI configuration (OPMConfig, resolved registry, verbose flag) SHALL be passed
explicitly to command constructors via a `GlobalConfig` struct rather than read from
package-level variables or accessor functions.

#### Scenario: Sub-command accesses OPMConfig

- **WHEN** a sub-command in `internal/cmd/mod/` or `internal/cmd/config/` needs the loaded OPMConfig
- **THEN** it reads it from the `*GlobalConfig` parameter passed to its constructor, not from a package-level accessor

#### Scenario: No package-level mutable state in sub-packages

- **WHEN** any file in `internal/cmd/mod/` or `internal/cmd/config/` is inspected
- **THEN** it contains no package-level `var` declarations for flags or configuration state

### Requirement: Command packages are organised by command group

The `internal/cmd/` package SHALL be split into sub-packages that mirror the cobra command tree.

#### Scenario: module commands are in their own package

- **WHEN** the `internal/cmd/module/` directory is inspected
- **THEN** it SHALL contain module authoring commands: `init`, `vet`, `build`
- **AND** the `build` subcommand SHALL synthesize a `#ModuleRelease` from a module-package directory and render it through the shared release-render pipeline
- **AND** the `build` subcommand SHALL reject a single-file argument with an error directing the user to `opm release build <file>` for release files

#### Scenario: config commands are in their own package

- **WHEN** the `internal/cmd/config/` directory is inspected
- **THEN** it contains all `config` sub-command implementations (`init`, `vet`)

#### Scenario: release commands are in their own package

- **WHEN** the `internal/cmd/release/` directory is inspected
- **THEN** it SHALL contain all `release` sub-command implementations: `vet`, `build`, `apply`, `diff`, `status`, `tree`, `events`, `delete`, `list`
- **AND** the `release build` subcommand SHALL accept either a release `.cue` file or a module-package directory as its positional argument
- **AND** when the argument is a directory the subcommand SHALL delegate to the same module-synthesis path used by `opm module build`

### Requirement: Release command group registered at root level

The root command SHALL register `opm release` (alias: `rel`) as a top-level command group via `cmdrelease.NewReleaseCmd(&cfg)`. This SHALL follow the same dependency injection pattern as `mod` and `config` groups.

#### Scenario: Root command registers release group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdrelease.NewReleaseCmd(&cfg))`

### Requirement: Cluster-query commands migrate from mod to release

The cluster-query commands (`status`, `tree`, `events`, `delete`, `list`) SHALL be implemented in `internal/cmd/release/`. The `internal/cmd/mod/` package SHALL retain alias versions that delegate to `opm release` equivalents and print a deprecation notice.

#### Scenario: opm mod status delegates to release status

- **WHEN** `opm mod status --release-name jellyfin` is run
- **THEN** the CLI SHALL print a deprecation notice suggesting `opm release status jellyfin`
- **AND** execute the same logic as `opm release status jellyfin`

#### Scenario: opm mod delete delegates to release delete

- **WHEN** `opm mod delete --release-name jellyfin` is run
- **THEN** the CLI SHALL print a deprecation notice suggesting `opm release delete jellyfin`
- **AND** execute the same logic as `opm release delete jellyfin`

### Requirement: `opm release build` branches on argument type

The `opm release build` subcommand SHALL stat its positional argument and choose between the release-file rendering path and the module-synthesis rendering path based on whether the path resolves to a regular file or a directory.

#### Scenario: Argument is a release file

- **WHEN** the user runs `opm release build ./jellyfin_release.cue` and the path resolves to a regular file
- **THEN** the subcommand SHALL load the file via the existing release-file loader and render it (existing behaviour)

#### Scenario: Argument is a module directory

- **WHEN** the user runs `opm release build ./my-module` and the path resolves to a directory
- **THEN** the subcommand SHALL invoke the module-synthesis pipeline, using `-f`/`--values` (or the module's `debugValues`) for values and `--name`/`--namespace` (or defaults) for synthetic metadata

#### Scenario: Argument does not exist

- **WHEN** the positional argument cannot be `os.Stat`'ed
- **THEN** the subcommand SHALL return a clear error naming the missing path

### Requirement: `opm module build` (alias `opm mod build`) accepts only module directories

The `module` command group SHALL register a `build` subcommand that accepts an optional positional argument defaulting to `"."`. The subcommand SHALL accept only directory inputs.

#### Scenario: Default to current directory

- **WHEN** the user runs `opm module build` with no positional argument from inside a module package directory
- **THEN** the subcommand SHALL synthesize and render that directory

#### Scenario: Explicit module directory

- **WHEN** the user runs `opm module build ./my-module`
- **THEN** the subcommand SHALL synthesize and render that directory

#### Scenario: File argument rejected

- **WHEN** the user runs `opm module build ./my-module/module.cue`
- **THEN** the subcommand SHALL return an error stating that module build expects a directory and pointing the user to `opm release build <file>` for release files

### Requirement: `--name` flag for synthetic-release builds

The `opm release build` subcommand (when used with a directory argument) and the `opm module build` subcommand SHALL accept a `--name <string>` flag that overrides the synthetic `metadata.name`. Defaults are described in the `module-synthetic-release` capability spec.

#### Scenario: Flag overrides the default name

- **WHEN** the user passes `--name foo`
- **THEN** the synthetic `metadata.name` SHALL be `foo`

#### Scenario: Flag is ignored for release-file builds

- **WHEN** the user runs `opm release build ./real-release.cue --name foo`
- **THEN** the CLI SHALL warn that `--name` is only meaningful for module-directory builds and SHALL render the release file's declared `metadata.name`
