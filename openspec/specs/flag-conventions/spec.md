# flag-conventions Specification

## Purpose
Fixes the meaning of the flags and exit codes that several `opm` commands share, so that a script written against one command reads the same on every other: what `--force` and `--yes` do, how a deprecated flag behaves, which exit code a usage error has, and which commands print machine-readable output with `-o`/`--output`.

## Requirements

### Requirement: Force overrides a protective refusal and never answers a prompt

On every command that offers `--force` without deprecation, the flag SHALL mean one thing: carry out what the command otherwise refuses in order to protect existing state. `opm instance apply` and `opm module apply` SHALL prune every tracked resource on an empty render only with `--force`; `opm platform pull` SHALL replace the contents of a non-empty directory only with `--force`; `opm config init` SHALL overwrite an existing configuration only with `--force`. `--force` SHALL default to false. No command SHALL offer a non-deprecated `--force` whose effect is to skip a confirmation prompt.

#### Scenario: Force is not the prompt flag of instance delete

- **WHEN** the help of `opm instance delete` is printed
- **THEN** it SHALL list `--yes` and SHALL NOT list `--force`

#### Scenario: Force keeps its meaning on the commands that refuse

- **WHEN** the flags of `opm instance apply`, `opm module apply`, `opm platform pull` and `opm config init` are read
- **THEN** each SHALL offer `--force`, default false, not deprecated

### Requirement: Every confirming command accepts yes

Every command that asks the user to confirm an action SHALL accept `--yes` with the shorthand `-y`, default false, and with it SHALL NOT prompt. Today these are `opm instance delete` and `opm module init`. `--yes` SHALL NOT bypass any refusal: the operator guards of `opm instance delete` hold with it.

#### Scenario: Instance delete with yes does not prompt

- **WHEN** `opm instance delete jellyfin -n media --yes` is run
- **THEN** the command SHALL NOT print a confirmation prompt and SHALL NOT read standard input

#### Scenario: Short form

- **WHEN** `opm instance delete jellyfin -n media -y` is run
- **THEN** the command SHALL behave as with `--yes`

### Requirement: Instance delete keeps force as a deprecated alias of yes

`opm instance delete --force` SHALL keep skipping the confirmation prompt, exactly as `--yes` does. The flag SHALL NOT appear in the command's help, and its use SHALL print one line on standard error that says `--force` is deprecated and names `--yes`. The exit code and the standard output of the command SHALL be those of the same run with `--yes`.

#### Scenario: Old spelling still skips the prompt

- **WHEN** `opm instance delete jellyfin -n media --force` is run
- **THEN** the command SHALL NOT prompt
- **AND** standard error SHALL carry one line saying that `--force` is deprecated and naming `--yes`

### Requirement: A usage error exits 1 and the root help documents the exit codes

A usage error SHALL exit with code 1 and print the error on standard error: an unknown command, an unknown flag, a flag value of the wrong type, flags that exclude each other, and a wrong number of positional arguments. `opm --help` SHALL print the exit code table of the cli: 0 success; 1 general error, usage errors included; 2 validation error or refusal; 3 connectivity error; 4 permission denied; 5 not found. The table SHALL say that a command's own help states where it differs.

#### Scenario: Unknown flag

- **WHEN** `opm version --bogus` is run
- **THEN** the process SHALL exit 1 and standard error SHALL name the unknown flag

#### Scenario: Unknown command

- **WHEN** `opm nosuchcommand` is run
- **THEN** the process SHALL exit 1

#### Scenario: Missing argument

- **WHEN** `opm instance build` is run without its positional argument
- **THEN** the process SHALL exit 1 and nothing SHALL be rendered

#### Scenario: Root help carries the table

- **WHEN** `opm --help` is run
- **THEN** the output SHALL contain a section named `Exit codes` that lists the codes 0 to 5 with their meanings

### Requirement: Version and template list print machine-readable output

`opm version` SHALL accept `-o`/`--output` with exactly the values `text` (default), `json` and `yaml`. `text` SHALL print what the command printed before the flag existed. `json` and `yaml` SHALL print one object on standard output with the fields `version`, `gitCommit`, `buildDate`, `goVersion` and `cueSDKVersion`.

`opm module template list` SHALL accept `-o`/`--output` with exactly the values `table` (default), `json` and `yaml`. `json` and `yaml` SHALL print a list with one object per official template, in the table's order, with the fields `name`, `description` and `defaultMajor`.

On both commands any other value SHALL exit 1 with the message `invalid output format "<value>" (valid: <the command's values>)` and print nothing on standard output.

#### Scenario: Version as JSON

- **WHEN** `opm version -o json` is run
- **THEN** standard output SHALL be one JSON object whose `version` field is the cli version

#### Scenario: Version default is unchanged

- **WHEN** `opm version` is run
- **THEN** standard output SHALL be the text the command printed before `--output` existed

#### Scenario: Template list as JSON

- **WHEN** `opm module template list -o json` is run
- **THEN** standard output SHALL be a JSON list with one entry per official template, each with `name`, `description` and `defaultMajor`

#### Scenario: Invalid format

- **WHEN** `opm version -o toml` is run
- **THEN** the command SHALL exit 1 and print `invalid output format "toml" (valid: text, json, yaml)`

### Requirement: Help text carries no tab-indented lines

No line of the long description of any `opm` command SHALL begin with a tab character. Indentation inside help text SHALL be spaces, so a paragraph prints flush left and an example prints at the indentation its author wrote.

#### Scenario: Group help prints flush left

- **WHEN** `opm catalog --help` and `opm registry --help` are run
- **THEN** every paragraph of the description SHALL start in the first column

#### Scenario: The whole command tree

- **WHEN** the long description of every command in the tree is read
- **THEN** no line SHALL begin with a tab character
