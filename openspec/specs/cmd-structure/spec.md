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

#### Scenario: mod commands are in their own package

- **WHEN** the `internal/cmd/mod/` directory is inspected
- **THEN** it SHALL contain module authoring commands: `init`, `vet`
- **AND** it SHALL NOT contain a `build` command

#### Scenario: config commands are in their own package

- **WHEN** the `internal/cmd/config/` directory is inspected
- **THEN** it contains all `config` sub-command implementations (`init`, `vet`)

#### Scenario: release commands are in their own package

- **WHEN** the `internal/cmd/release/` directory is inspected
- **THEN** it SHALL contain all `release` sub-command implementations: `vet`, `build`, `apply`, `diff`, `status`, `tree`, `events`, `delete`, `list`

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


