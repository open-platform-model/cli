## MODIFIED Requirements

### Requirement: Command packages are organised by command group

The `internal/cmd/` package SHALL be split into sub-packages that mirror the cobra command tree.

#### Scenario: mod commands are in their own package

- **WHEN** the `internal/cmd/mod/` directory is inspected
- **THEN** it SHALL contain module authoring commands: `init`, `build`, `vet`
- **AND** `build` and `apply` SHALL be thin aliases that delegate to the release render pipeline

#### Scenario: config commands are in their own package

- **WHEN** the `internal/cmd/config/` directory is inspected
- **THEN** it SHALL contain all `config` sub-command implementations (`init`, `vet`)

#### Scenario: release commands are in their own package

- **WHEN** the `internal/cmd/release/` directory is inspected
- **THEN** it SHALL contain all `release` sub-command implementations: `vet`, `build`, `apply`, `diff`, `status`, `tree`, `events`, `delete`, `list`

## ADDED Requirements

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

### Requirement: mod build and mod apply alias the release pipeline

`opm mod build` and `opm mod apply` SHALL internally construct an ephemeral `#ModuleRelease` from their flags (`--values`, `--namespace`, `--release-name`) and execute it through the same release render pipeline used by `opm release`. They SHALL NOT print deprecation notices — they remain the canonical module-author workflow.

#### Scenario: opm mod build uses release pipeline

- **WHEN** `opm mod build . -f values.cue -n production --release-name my-app` is run
- **THEN** the CLI SHALL construct an ephemeral release from the module at `.` with the provided flags
- **AND** render it through the same pipeline as `opm release build`

#### Scenario: opm mod apply uses release pipeline

- **WHEN** `opm mod apply . -f values.cue -n production` is run
- **THEN** the CLI SHALL construct an ephemeral release and apply it via the release pipeline
