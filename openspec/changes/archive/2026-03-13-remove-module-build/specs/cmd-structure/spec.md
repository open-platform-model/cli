## MODIFIED Requirements

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

## REMOVED Requirements

### Requirement: mod build and mod apply alias the release pipeline

**Reason**: The `opm module build` command is removed entirely. Module-source rendering is no longer supported as a CLI command. Users SHALL use `opm release build -r <release-file>` instead.

**Migration**: Create a `release.cue` file for the module and use `opm release build -r release.cue` to render manifests. The `opm module vet` command remains available for validating module config without rendering.
