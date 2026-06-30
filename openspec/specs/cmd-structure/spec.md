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
- **AND** the `build` subcommand SHALL synthesize a `#ModuleInstance` from a module-package directory and render it through the shared instance-render pipeline
- **AND** the `build` subcommand SHALL reject a single-file argument with an error directing the user to `opm instance build <file>` for instance files

#### Scenario: config commands are in their own package

- **WHEN** the `internal/cmd/config/` directory is inspected
- **THEN** it contains all `config` sub-command implementations (`init`, `vet`)

#### Scenario: instance commands are in their own package

- **WHEN** the `internal/cmd/instance/` directory is inspected
- **THEN** it SHALL contain all `instance` sub-command implementations: `vet`, `build`, `apply`, `diff`, `status`, `tree`, `events`, `delete`, `list`
- **AND** the `instance build` subcommand SHALL accept either an instance `.cue` file or a module-package directory as its positional argument
- **AND** when the argument is a directory the subcommand SHALL delegate to the same module-synthesis path used by `opm module build`

### Requirement: Instance command group registered at root level

The root command SHALL register `opm instance` (alias: `inst`) as a top-level command group via `cmdinstance.NewInstanceCmd(&cfg)`. This SHALL follow the same dependency injection pattern as `mod` and `config` groups. The former `cmdrelease.NewReleaseCmd` registration is removed (no back-compat alias — enhancement 0002 D8).

#### Scenario: Root command registers instance group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdinstance.NewInstanceCmd(&cfg))`
- **AND** it SHALL NOT contain `cmdrelease.NewReleaseCmd`

### Requirement: Cluster-query commands migrate from mod to instance

The cluster-query commands (`status`, `tree`, `events`, `delete`, `list`) SHALL be implemented in `internal/cmd/instance/`. The `internal/cmd/mod/` package SHALL retain alias versions that delegate to `opm instance` equivalents and print a deprecation notice.

#### Scenario: opm mod status delegates to instance status

- **WHEN** `opm mod status --instance-name jellyfin` is run
- **THEN** the CLI SHALL print a deprecation notice suggesting `opm instance status jellyfin`
- **AND** execute the same logic as `opm instance status jellyfin`

#### Scenario: opm mod delete delegates to instance delete

- **WHEN** `opm mod delete --instance-name jellyfin` is run
- **THEN** the CLI SHALL print a deprecation notice suggesting `opm instance delete jellyfin`
- **AND** execute the same logic as `opm instance delete jellyfin`

### Requirement: `opm instance build` branches on argument type

The `opm instance build` subcommand SHALL stat its positional argument and choose between the instance-file rendering path and the module-synthesis rendering path based on whether the path resolves to a regular file or a directory.

#### Scenario: Argument is an instance file

- **WHEN** the user runs `opm instance build ./jellyfin_instance.cue` and the path resolves to a regular file
- **THEN** the subcommand SHALL load the file via the existing instance-file loader and render it (existing behaviour)

#### Scenario: Argument is a module directory

- **WHEN** the user runs `opm instance build ./my-module` and the path resolves to a directory
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
- **THEN** the subcommand SHALL return an error stating that module build expects a directory and pointing the user to `opm instance build <file>` for instance files

### Requirement: `--name` flag for synthetic-release builds

The `opm instance build` subcommand (when used with a directory argument), the `opm module build` subcommand, and the `opm module apply` subcommand SHALL accept a `--name <string>` flag that overrides the synthetic `metadata.name`. Defaults are described in the `module-synthetic-instance` capability spec.

#### Scenario: Flag overrides the default name

- **WHEN** the user passes `--name foo`
- **THEN** the synthetic `metadata.name` SHALL be `foo`

#### Scenario: Flag is ignored for instance-file builds

- **WHEN** the user runs `opm instance build ./real-instance.cue --name foo`
- **THEN** the CLI SHALL warn that `--name` is only meaningful for module-directory builds and SHALL render the instance file's declared `metadata.name`

#### Scenario: Flag participates in synthetic release identity for `module apply`

- **WHEN** the user runs `opm module apply ./foo --name custom`
- **THEN** the synthetic `metadata.name` SHALL be `custom`
- **AND** the resolved release UUID SHALL be derived from `custom` (not the default `<module>-debug`)
- **AND** running the same command again with a different `--name` value SHALL produce a distinct release identity and a separate inventory record

### Requirement: `opm module apply` (alias `opm mod apply`) accepts only module directories

The `module` command group SHALL register an `apply` subcommand that accepts an optional positional argument defaulting to `"."`. The subcommand SHALL accept only directory inputs.

The subcommand SHALL synthesize a `#ModuleInstance` from the directory (reusing the `module-synthetic-instance` capability), render the result through the same pipeline as `opm instance apply`, and apply the produced resources to a Kubernetes cluster with full inventory, prune, dry-run, and ownership semantics.

#### Scenario: Default to current directory

- **WHEN** the user runs `opm module apply` with no positional argument from inside a module package directory
- **THEN** the subcommand SHALL synthesize and apply that directory

#### Scenario: Explicit module directory

- **WHEN** the user runs `opm module apply ./my-module`
- **THEN** the subcommand SHALL synthesize and apply that directory

#### Scenario: File argument rejected

- **WHEN** the user runs `opm module apply ./my-module/module.cue`
- **THEN** the subcommand SHALL return an error stating that `module apply` expects a directory
- **AND** SHALL point the user to `opm instance apply <file>` for instance files

#### Scenario: Alias `mod apply` resolves to `module apply`

- **WHEN** the user runs `opm mod apply ./my-module`
- **THEN** the CLI SHALL execute the same subcommand as `opm module apply ./my-module`
