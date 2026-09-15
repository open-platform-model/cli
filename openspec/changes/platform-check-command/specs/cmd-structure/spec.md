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

#### Scenario: platform commands are in their own package

- **WHEN** the `internal/cmd/platform/` directory is inspected
- **THEN** it SHALL contain the `platform` sub-command implementations: `check`
- **AND** the commands SHALL be thin cobra wiring that delegates the report to `internal/platform/`

## ADDED Requirements

### Requirement: Platform command group registered at root level

The root command SHALL register `opm platform` as a top-level command group via `cmdplatform.NewPlatformCmd(&cfg)`, following the same `GlobalConfig` dependency-injection pattern as the `module`, `instance`, `config` and `operator` groups. The group is noun-first: there is no `opm check` verb group at root level.

#### Scenario: Root command registers platform group

- **WHEN** `internal/cmd/root.go` is inspected
- **THEN** it SHALL contain `rootCmd.AddCommand(cmdplatform.NewPlatformCmd(&cfg))`

#### Scenario: The group lists its subcommands

- **WHEN** `opm platform --help` is run
- **THEN** it SHALL list `check`
