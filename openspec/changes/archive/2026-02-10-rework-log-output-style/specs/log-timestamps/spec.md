## ADDED Requirements

### Requirement: Timestamps enabled by default

The CLI SHALL display timestamps on all human-readable log lines by default. Timestamps SHALL use the RFC 3339 time-only format: `15:04:05` (24-hour, no date, no timezone).

Timestamps SHALL be rendered in dim/faint style to reduce visual noise while remaining available for timing correlation.

#### Scenario: Default log output includes timestamps

- **WHEN** the CLI runs with no timestamp-related flags or config
- **THEN** every log line SHALL include a timestamp in `15:04:05` format
- **THEN** the timestamp SHALL appear as the first element on the line

#### Scenario: Timestamp reflects wall-clock time

- **WHEN** a log line is emitted during a module apply operation
- **THEN** the timestamp SHALL reflect the actual wall-clock time of the log event
- **THEN** consecutive lines MAY show different seconds as time progresses

### Requirement: Timestamps controllable via CLI flag

The CLI SHALL provide a `--timestamps` persistent flag that controls timestamp display.

The flag SHALL accept boolean values: `--timestamps=true` or `--timestamps=false`.

When the flag is not provided, its value SHALL be unset (nil), falling through to config or default.

#### Scenario: Disable timestamps via flag

- **WHEN** `opm mod apply --timestamps=false` is run
- **THEN** log lines SHALL NOT include timestamps

#### Scenario: Explicitly enable timestamps via flag

- **WHEN** `opm mod apply --timestamps=true` is run
- **THEN** log lines SHALL include timestamps regardless of config setting

#### Scenario: Flag overrides config

- **WHEN** config.cue contains `log: { timestamps: true }`
- **WHEN** `--timestamps=false` flag is provided
- **THEN** timestamps SHALL be disabled (flag takes precedence)

### Requirement: Timestamps controllable via config file

The CUE configuration file SHALL support a `log.timestamps` field that controls timestamp display.

The field SHALL be defined as: `log: { timestamps: bool | *true }` (defaults to `true`).

#### Scenario: Config disables timestamps

- **WHEN** config.cue contains `log: { timestamps: false }`
- **WHEN** no `--timestamps` flag is provided
- **THEN** log lines SHALL NOT include timestamps

#### Scenario: Config field missing uses default

- **WHEN** config.cue does not contain a `log` section
- **WHEN** no `--timestamps` flag is provided
- **THEN** timestamps SHALL be enabled (default: `true`)

### Requirement: Timestamp precedence chain

The CLI SHALL resolve the timestamp display setting using precedence: Flag > Config > Default.

| Source | Mechanism | Precedence |
|--------|-----------|------------|
| Flag | `--timestamps` | 1 (highest) |
| Config | `log.timestamps` | 2 |
| Default | `true` | 3 (lowest) |

#### Scenario: Full precedence chain

- **WHEN** `--timestamps=false` flag is provided
- **WHEN** config.cue contains `log: { timestamps: true }`
- **THEN** timestamps SHALL be disabled (flag wins)

#### Scenario: Config overrides default

- **WHEN** no `--timestamps` flag is provided
- **WHEN** config.cue contains `log: { timestamps: false }`
- **THEN** timestamps SHALL be disabled (config overrides default)

#### Scenario: Default when nothing is set

- **WHEN** no `--timestamps` flag is provided
- **WHEN** config.cue has no `log.timestamps` field
- **THEN** timestamps SHALL be enabled (default: `true`)

### Requirement: Verbose mode always shows timestamps

When `--verbose` is enabled, timestamps SHALL always be displayed regardless of the `--timestamps` flag or config setting.

#### Scenario: Verbose overrides timestamps-off

- **WHEN** `opm mod apply --verbose --timestamps=false` is run
- **THEN** timestamps SHALL be displayed (verbose requires timestamps for debugging)
