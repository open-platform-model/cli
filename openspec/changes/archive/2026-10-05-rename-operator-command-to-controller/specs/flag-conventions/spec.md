## MODIFIED Requirements

### Requirement: Every confirming command accepts yes

Every command that asks the user to confirm an action SHALL accept `--yes` with the shorthand `-y`, default false, and with it SHALL NOT prompt. Today these are `opm instance delete` and `opm module init`. `--yes` SHALL NOT bypass any refusal: the controller guards of `opm instance delete` hold with it.

#### Scenario: Instance delete with yes does not prompt

- **WHEN** `opm instance delete jellyfin -n media --yes` is run
- **THEN** the command SHALL NOT print a confirmation prompt and SHALL NOT read standard input

#### Scenario: Short form

- **WHEN** `opm instance delete jellyfin -n media -y` is run
- **THEN** the command SHALL behave as with `--yes`
