## Purpose

Defines the structural conventions for CLI command packages in the OPM codebase. This covers how commands are organized into sub-packages, how configuration is injected, and how the cobra command tree maps to the package layout under `internal/cmd/`.

## Requirements

### Requirement: Commands receive configuration via explicit injection

Global CLI configuration (OPMConfig, resolved registry, verbose flag) SHALL be passed
explicitly to command constructors via a `GlobalConfig` struct rather than read from
package-level variables or accessor functions.

#### Scenario: Sub-command accesses OPMConfig

- **WHEN** a sub-command in `internal/cmd/module/`, `internal/cmd/instance/` or `internal/cmd/config/` needs the loaded OPMConfig
- **THEN** it reads it from the `*GlobalConfig` parameter passed to its constructor, not from a package-level accessor

#### Scenario: No package-level mutable state in sub-packages

- **WHEN** any file in `internal/cmd/module/`, `internal/cmd/instance/` or `internal/cmd/config/` is inspected
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

#### Scenario: operator commands are in their own package

- **WHEN** the `internal/cmd/operator/` directory is inspected
- **THEN** it SHALL contain the `operator` sub-command implementations: `install`, `uninstall`
- **AND** the commands SHALL be thin cobra wiring that delegates all behavior to `internal/operator/`

### Requirement: Instance command group registered at root level

The root command SHALL register `opm instance` (alias: `inst`) as a top-level command group via `cmdinstance.NewInstanceCmd(&cfg)`. This SHALL follow the same dependency injection pattern as `mod` and `config` groups. The former `cmdrelease.NewReleaseCmd` registration is removed (no back-compat alias — enhancement 0002 D8).

#### Scenario: Root command registers instance group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdinstance.NewInstanceCmd(&cfg))`
- **AND** it SHALL NOT contain `cmdrelease.NewReleaseCmd`

### Requirement: Operator command group registered at root level

The root command SHALL register `opm operator` as a top-level command group via `cmdoperator.NewOperatorCmd(&cfg)`, following the same `GlobalConfig` dependency-injection pattern as the `module`, `instance`, and `config` groups. The group is noun-first (enhancement 0006 D32): there are no `opm install` or `opm uninstall` verb groups at root level.

#### Scenario: Root command registers operator group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdoperator.NewOperatorCmd(&cfg))`
- **AND** the root command SHALL NOT register `install` or `uninstall` as top-level commands

### Requirement: Cluster-query commands live only under instance

The cluster-query commands (`status`, `tree`, `events`, `delete`, `list`) SHALL be implemented in `internal/cmd/instance/` and registered only under the `instance` command group. The `module` group (`internal/cmd/module/`) SHALL NOT carry them or aliases for them: it registers `init`, `template`, `vet`, `build`, `apply`, `publish` and `version` only.

#### Scenario: opm mod status is not a command

- **WHEN** `opm mod status jellyfin` is run
- **THEN** the CLI SHALL fail with an unknown-command error

#### Scenario: opm instance status is the command

- **WHEN** `opm instance status jellyfin` is run
- **THEN** the CLI SHALL execute the status command for the instance `jellyfin`

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

### Requirement: `opm module build` output format, split files and ordering

The `opm module build` subcommand SHALL accept `--output`/`-o` with exactly the values `yaml` (default) and `json`; any other value SHALL exit with code 1 and the message `invalid output format "<value>" (valid: yaml, json)`. With `--split`, it SHALL write one file per resource into `--out-dir` (default `./manifests`) named `<lowercase-kind>-<name>.<yaml|json>`; the second resource that resolves to the same base name SHALL receive the suffix `-2`, the third `-3`, and so on. Resources SHALL be emitted in a deterministic order, by apply weight (`pkg/resourceorder.GetWeight`), then namespace, then name, so identical input always yields identical output.

#### Scenario: Unsupported output format

- **WHEN** the user runs `opm module build -o toml`
- **THEN** the subcommand SHALL exit with code 1 and print `invalid output format "toml" (valid: yaml, json)`

#### Scenario: Split output names files by kind and name

- **WHEN** the user runs `opm module build --split --out-dir ./manifests` on a module rendering a Deployment `web` and a Service `web`
- **THEN** `./manifests` SHALL contain `deployment-web.yaml` and `service-web.yaml`

#### Scenario: Split output disambiguates colliding names

- **WHEN** two rendered resources share kind and name
- **THEN** the files SHALL be `<kind>-<name>.yaml` and `<kind>-<name>-2.yaml`

#### Scenario: Output order is deterministic

- **WHEN** the same module is built twice
- **THEN** both outputs SHALL list resources in identical order: ascending weight, then namespace, then name

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
