## REMOVED Requirements

### Requirement: Force overrides a protective refusal and never answers a prompt
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Force carries out what a command otherwise refuses in order to protect existing state", "No command offers a non-deprecated `--force` whose effect is to skip a confirmation prompt".

### Requirement: Version and template list print machine-readable output
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`opm version` accepts `-o`/`--output` with exactly the values `text`, `json` and `yaml`", "`opm module template list` accepts `-o`/`--output` with exactly the values `table`, `json` and `yaml`", "Any other `--output` value exits 1 with an invalid output format message and prints nothing on standard output".

### Requirement: Delete-data is the only flag that allows deletion of PersistentVolumeClaims
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "`--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false", "The help of each command that accepts `--delete-data` says that PersistentVolumeClaims are kept by default".

## ADDED Requirements

### Requirement: Force carries out what a command otherwise refuses in order to protect existing state

On every command that offers `--force` without deprecation, the flag SHALL mean one thing: carry out what the command otherwise refuses in order to protect existing state. `opm instance apply` and `opm module apply` SHALL prune every tracked resource on an empty render only with `--force`; `opm platform pull` SHALL replace the contents of a non-empty directory only with `--force`; `opm config init` SHALL overwrite an existing configuration only with `--force`. `--force` SHALL default to false.

#### Scenario: Force keeps its meaning on the commands that refuse

- **WHEN** the flags of `opm instance apply`, `opm module apply`, `opm platform pull` and `opm config init` are read
- **THEN** each SHALL offer `--force`, default false, not deprecated

### Requirement: No command offers a non-deprecated `--force` whose effect is to skip a confirmation prompt

No command SHALL offer a non-deprecated `--force` whose effect is to skip a confirmation prompt.

#### Scenario: Force is not the prompt flag of instance delete

- **WHEN** the help of `opm instance delete` is printed
- **THEN** it SHALL list `--yes` and SHALL NOT list `--force`

### Requirement: `opm version` accepts `-o`/`--output` with exactly the values `text`, `json` and `yaml`

`opm version` SHALL accept `-o`/`--output` with exactly the values `text` (default), `json` and `yaml`. `text` SHALL print what the command printed before the flag existed. `json` and `yaml` SHALL print one object on standard output with the fields `version`, `gitCommit`, `buildDate`, `goVersion` and `cueSDKVersion`.

#### Scenario: Version as JSON

- **WHEN** `opm version -o json` is run
- **THEN** standard output SHALL be one JSON object whose `version` field is the cli version

#### Scenario: Version default is unchanged

- **WHEN** `opm version` is run
- **THEN** standard output SHALL be the text the command printed before `--output` existed

### Requirement: `opm module template list` accepts `-o`/`--output` with exactly the values `table`, `json` and `yaml`

`opm module template list` SHALL accept `-o`/`--output` with exactly the values `table` (default), `json` and `yaml`. `json` and `yaml` SHALL print a list with one object per official template, in the table's order, with the fields `name`, `description` and `defaultMajor`.

#### Scenario: Template list as JSON

- **WHEN** `opm module template list -o json` is run
- **THEN** standard output SHALL be a JSON list with one entry per official template, each with `name`, `description` and `defaultMajor`

### Requirement: Any other `--output` value exits 1 with an invalid output format message and prints nothing on standard output

On both commands any other value SHALL exit 1 with the message `invalid output format "<value>" (valid: <the command's values>)` and print nothing on standard output. (Requirement: `opm version` accepts `-o`/`--output` with exactly the values `text`, `json` and `yaml`) (Requirement: `opm module template list` accepts `-o`/`--output` with exactly the values `table`, `json` and `yaml`)

#### Scenario: Invalid format

- **WHEN** `opm version -o toml` is run
- **THEN** the command SHALL exit 1 and print `invalid output format "toml" (valid: text, json, yaml)`

### Requirement: `--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false

`opm instance delete`, `opm instance apply` and `opm module apply` SHALL each accept `--delete-data`, a boolean flag with default false and no shorthand, and with it SHALL delete tracked PersistentVolumeClaims that they keep without it. No other flag SHALL have that effect: `--yes` only skips the confirmation prompt, and `--force` only allows the prune of an empty render.

#### Scenario: The flag is offered with a safe default

- **WHEN** the flags of `opm instance delete`, `opm instance apply` and `opm module apply` are read
- **THEN** each SHALL offer `--delete-data`, default false, not deprecated, with no shorthand

#### Scenario: Yes alone keeps claims

- **WHEN** `opm instance delete jellyfin -n media --yes` is run for a CLI-owned instance that tracks a PersistentVolumeClaim
- **THEN** the claim SHALL be kept

#### Scenario: Force alone keeps claims

- **WHEN** `opm instance apply --force` prunes every tracked resource of an empty render
- **THEN** a tracked PersistentVolumeClaim SHALL be kept

### Requirement: The help of each command that accepts `--delete-data` says that PersistentVolumeClaims are kept by default

The help of each of the three commands SHALL say that PersistentVolumeClaims are kept by default and SHALL name `--delete-data`. (Requirement: `--delete-data` is the only flag that deletes tracked PersistentVolumeClaims and it defaults to false)

#### Scenario: The help names the flag that deletes claims

- **WHEN** the help of `opm instance delete`, `opm instance apply` or `opm module apply` is printed
- **THEN** it SHALL say that PersistentVolumeClaims are kept by default and SHALL name `--delete-data`
