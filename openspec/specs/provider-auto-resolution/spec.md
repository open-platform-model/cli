## ADDED Requirements

### Requirement: Auto-select provider when single provider configured

The CLI SHALL automatically select the default provider when exactly one provider is defined in the configuration file and no `--provider` flag is specified.

The auto-resolved provider SHALL be recorded with source `"config-auto"` to distinguish it from explicit provider selection via flag or config.

#### Scenario: Single provider auto-selected

- **WHEN** config.cue defines exactly one provider (e.g., `providers: { kubernetes: ... }`)
- **WHEN** no `--provider` flag is specified
- **THEN** the CLI SHALL use that provider as the default
- **THEN** the resolution source SHALL be `"config-auto"`

#### Scenario: Multiple providers require explicit selection

- **WHEN** config.cue defines more than one provider
- **WHEN** no `--provider` flag is specified
- **THEN** the CLI SHALL NOT auto-select a provider
- **THEN** commands that require a provider SHALL fail with a clear error message

#### Scenario: No providers configured

- **WHEN** config.cue defines no providers
- **WHEN** no `--provider` flag is specified
- **THEN** the provider SHALL remain empty
- **THEN** commands that require a provider SHALL fail with a clear error message

#### Scenario: Flag overrides auto-selection

- **WHEN** config.cue defines exactly one provider named `kubernetes`
- **WHEN** `--provider nomad` flag is specified
- **THEN** the CLI SHALL use `nomad` as the provider
- **THEN** the resolution source SHALL be `"flag"`

### Requirement: Provider auto-resolution visible in verbose output

The CLI SHALL include the resolved provider and its source in the "initializing CLI" debug log when `--verbose` is specified.

#### Scenario: Auto-resolved provider shown in verbose log

- **WHEN** `--verbose` flag is specified
- **WHEN** provider is auto-resolved from a single configured provider
- **THEN** the "initializing CLI" debug log SHALL include `provider=kubernetes` (or the provider name)
- **THEN** the log SHALL indicate the source as `config-auto`
