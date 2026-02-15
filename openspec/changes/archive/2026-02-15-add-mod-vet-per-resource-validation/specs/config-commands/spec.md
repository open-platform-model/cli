## MODIFIED Requirements

### Requirement: Config vet command validates configuration

The `opm config vet` command SHALL validate the configuration file using CUE evaluation.

Checks performed:

1. Config file exists at resolved path
2. cue.mod/module.cue exists
3. Config file is syntactically valid CUE
4. Config evaluates without errors (imports resolve, constraints pass)

Each check SHALL print a styled line to stdout using `FormatVetCheck` as it passes, giving the user real-time feedback. On failure, all previously-passing checks SHALL remain visible.

#### Scenario: Valid configuration passes validation

- **WHEN** `opm config vet` is run
- **WHEN** config.cue exists and is valid CUE
- **WHEN** cue.mod/module.cue exists
- **THEN** command succeeds
- **THEN** output SHALL contain a checkmark line for each passing check
- **THEN** first line SHALL be: `✔ Config file found` with the config file path right-aligned in dim style
- **THEN** second line SHALL be: `✔ Module metadata found` with the module.cue path right-aligned in dim style
- **THEN** third line SHALL be: `✔ CUE evaluation passed`

#### Scenario: Missing config file fails with actionable error

- **WHEN** `opm config vet` is run
- **WHEN** `~/.opm/config.cue` does not exist
- **THEN** command fails with not-found error
- **THEN** no checkmark lines SHALL be printed (first check failed)
- **THEN** hint: "Run 'opm config init' to create default configuration"

#### Scenario: Missing cue.mod fails after config check passes

- **WHEN** `opm config vet` is run
- **WHEN** config.cue exists but cue.mod/module.cue does not
- **THEN** the config file check SHALL print a passing checkmark line
- **THEN** command fails with not-found error
- **THEN** hint: "Run 'opm config init' to create configuration"

#### Scenario: Invalid CUE fails after file checks pass

- **WHEN** `opm config vet` is run
- **WHEN** config.cue exists and cue.mod/module.cue exists
- **WHEN** config.cue contains syntax or evaluation errors
- **THEN** both file existence checks SHALL print passing checkmark lines
- **THEN** command fails with CUE error message
- **THEN** error includes file location
