## ADDED Requirements

### Requirement: Config init writes only the configuration file

The `opm config init` command SHALL create the default configuration in `~/.opm/`.

The command creates:

- `~/.opm/config.cue` — scalar-only configuration file (registry, kubernetes, log) with no CUE imports

The command SHALL NOT write a platform module or any platform file: no `~/.opm/platform/` directory and no data-only `~/.opm/platform.cue`. An existing `~/.opm/platform/` directory SHALL be left untouched, with `--force` or without. When a legacy data-only `~/.opm/platform.cue` exists, `--force` SHALL remove it with a printed note. The command SHALL remain normatively offline: it SHALL NOT run `cue mod tidy` or any registry operation.

#### Scenario: Initialize configuration for first time

- **WHEN** `opm config init` is run
- **WHEN** no configuration exists at `~/.opm/config.cue`
- **THEN** `~/.opm/` directory is created with 0700 permissions
- **THEN** `~/.opm/config.cue` is written with 0600 permissions
- **THEN** no `~/.opm/platform/` directory and no `~/.opm/platform.cue` file is written
- **THEN** success message lists created files
- **THEN** message suggests: "Validate with: opm config vet"

#### Scenario: An existing platform directory is left alone

- **WHEN** `opm config init --force` is run and `~/.opm/platform/` exists from an earlier release
- **THEN** the directory and its contents SHALL be unchanged afterwards

#### Scenario: Legacy data-only platform file is removed

- **WHEN** `opm config init --force` is run and a legacy data-only `~/.opm/platform.cue` exists
- **THEN** the legacy file is removed, and the output notes the removal

#### Scenario: Refuse to overwrite existing configuration

- **WHEN** `opm config init` is run
- **WHEN** `~/.opm/config.cue` already exists
- **THEN** command fails with validation error
- **THEN** error message: "configuration already exists"
- **THEN** hint: "Use --force to overwrite existing configuration."

#### Scenario: Force overwrite existing configuration

- **WHEN** `opm config init --force` is run
- **WHEN** `~/.opm/config.cue` already exists
- **THEN** existing files are overwritten
- **THEN** success message lists created files

### Requirement: Config vet checks only the configuration file

The `opm config vet` command SHALL validate the `~/.opm` configuration.

Checks performed:

1. Config file exists at resolved path
2. Config file is syntactically valid CUE and satisfies the embedded config schema (no imports, no removed fields)

Vet SHALL NOT build, load or require any platform module. When a `platform/` directory exists beside the config file, vet SHALL pass and SHALL print a warning that the directory is no longer read by any command, can be passed with `--platform <dir>`, and is safe to delete. A leftover legacy data-only `~/.opm/platform.cue` SHALL fail vet naming the file, with the hint to re-run `opm config init --force`. Each check SHALL print a styled line to stdout using `FormatVetCheck` as it passes; on failure, previously-passing checks SHALL remain visible.

#### Scenario: Valid configuration passes validation

- **WHEN** `opm config vet` is run
- **WHEN** config.cue is valid
- **THEN** command succeeds
- **THEN** output SHALL contain a checkmark line for each passing check and no platform check

#### Scenario: Missing config file fails with actionable error

- **WHEN** `opm config vet` is run
- **WHEN** `~/.opm/config.cue` does not exist
- **THEN** command fails with not-found error
- **THEN** hint: "Run 'opm config init' to create default configuration"

#### Scenario: A leftover platform directory warns

- **WHEN** `opm config vet` is run
- **WHEN** config.cue is valid and `~/.opm/platform/` exists
- **THEN** command succeeds
- **THEN** a warning SHALL say the directory is no longer read, can be passed with `--platform`, and is safe to delete
- **THEN** the directory SHALL NOT be built or loaded

#### Scenario: Legacy platform file fails with migration hint

- **WHEN** `opm config vet` is run and a data-only `~/.opm/platform.cue` exists
- **THEN** validation SHALL fail naming the legacy file
- **AND** the hint SHALL say to re-run `opm config init --force`

#### Scenario: Stale providers block fails with migration hint

- **WHEN** `opm config vet` is run against a pre-D39 config.cue containing `providers:` or a `~/.opm/cue.mod/`
- **THEN** validation SHALL fail naming the removed field
- **AND** the hint SHALL say to re-run `opm config init` (or remove the field and `cue.mod/`)

## REMOVED Requirements

### Requirement: Config init command creates configuration

**Reason**: Replaced by "Config init writes only the configuration file": `opm config init` no longer seeds a local default platform module, which no command reads any more.

**Migration**: Nothing to do for a fresh setup. An existing `~/.opm/platform/` is left on disk; pass it with `--platform <dir>` to keep rendering against it.

### Requirement: Config vet command validates configuration

**Reason**: Replaced by "Config vet checks only the configuration file": there is no local default platform module to build.

**Migration**: Check a platform module directory with `opm platform check <dir>`.
