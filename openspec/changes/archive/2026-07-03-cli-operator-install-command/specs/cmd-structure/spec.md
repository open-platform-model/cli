# cmd-structure (delta)

Adds the `operator` command group to the command-package organization (slice B2 of enhancement 0006, D32).

## MODIFIED Requirements

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

## ADDED Requirements

### Requirement: Operator command group registered at root level

The root command SHALL register `opm operator` as a top-level command group via `cmdoperator.NewOperatorCmd(&cfg)`, following the same `GlobalConfig` dependency-injection pattern as the `module`, `instance`, and `config` groups. The group is noun-first (enhancement 0006 D32): there are no `opm install` or `opm uninstall` verb groups at root level.

#### Scenario: Root command registers operator group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdoperator.NewOperatorCmd(&cfg))`
- **AND** the root command SHALL NOT register `install` or `uninstall` as top-level commands
