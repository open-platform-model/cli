## ADDED Requirements

### Requirement: `opm module apply` (alias `opm mod apply`) accepts only module directories

The `module` command group SHALL register an `apply` subcommand that accepts an optional positional argument defaulting to `"."`. The subcommand SHALL accept only directory inputs.

The subcommand SHALL synthesize a `#ModuleRelease` from the directory (reusing the `module-synthetic-release` capability), render the result through the same pipeline as `opm release apply`, and apply the produced resources to a Kubernetes cluster with full inventory, prune, dry-run, and ownership semantics.

#### Scenario: Default to current directory

- **WHEN** the user runs `opm module apply` with no positional argument from inside a module package directory
- **THEN** the subcommand SHALL synthesize and apply that directory

#### Scenario: Explicit module directory

- **WHEN** the user runs `opm module apply ./my-module`
- **THEN** the subcommand SHALL synthesize and apply that directory

#### Scenario: File argument rejected

- **WHEN** the user runs `opm module apply ./my-module/module.cue`
- **THEN** the subcommand SHALL return an error stating that `module apply` expects a directory
- **AND** SHALL point the user to `opm release apply <file>` for release files

#### Scenario: Alias `mod apply` resolves to `module apply`

- **WHEN** the user runs `opm mod apply ./my-module`
- **THEN** the CLI SHALL execute the same subcommand as `opm module apply ./my-module`

## MODIFIED Requirements

### Requirement: `--name` flag for synthetic-release builds

The `opm release build` subcommand (when used with a directory argument), the `opm module build` subcommand, and the `opm module apply` subcommand SHALL accept a `--name <string>` flag that overrides the synthetic `metadata.name`. Defaults are described in the `module-synthetic-release` capability spec.

#### Scenario: Flag overrides the default name

- **WHEN** the user passes `--name foo`
- **THEN** the synthetic `metadata.name` SHALL be `foo`

#### Scenario: Flag is ignored for release-file builds

- **WHEN** the user runs `opm release build ./real-release.cue --name foo`
- **THEN** the CLI SHALL warn that `--name` is only meaningful for module-directory builds and SHALL render the release file's declared `metadata.name`

#### Scenario: Flag participates in synthetic release identity for `module apply`

- **WHEN** the user runs `opm module apply ./foo --name custom`
- **THEN** the synthetic `metadata.name` SHALL be `custom`
- **AND** the resolved release UUID SHALL be derived from `custom` (not the default `<module>-debug`)
- **AND** running the same command again with a different `--name` value SHALL produce a distinct release identity and a separate inventory record
