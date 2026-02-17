# CLI Output Format

## Purpose

Enables the CLI to switch between human-readable text output and machine-readable JSON output for CI/CD integration. Controls format resolution, log formatter selection, command result envelopes, and error serialization.

## Requirements

### Requirement: Global --format flag

The CLI SHALL provide a global `--format` persistent flag accepting `text` or `json`. The default value SHALL be `text`.

#### Scenario: Format flag accepts text

- **WHEN** user runs `opm mod apply --format text`
- **THEN** the command SHALL emit human-readable styled output

#### Scenario: Format flag accepts json

- **WHEN** user runs `opm mod apply --format json`
- **THEN** the command SHALL emit JSON-structured output

#### Scenario: Default format is text

- **WHEN** user runs `opm mod apply` without `--format` flag
- **THEN** the format SHALL be `text`

#### Scenario: Invalid format value is rejected

- **WHEN** user runs `opm --format xml version`
- **THEN** the CLI SHALL reject the value with error message indicating valid values are `text` or `json`

### Requirement: OPM_FORMAT environment variable

The CLI SHALL recognize an `OPM_FORMAT` environment variable for setting the output format globally without repeating the flag on each command invocation.

#### Scenario: Environment variable sets format

- **WHEN** `OPM_FORMAT=json` is set in the environment
- **WHEN** user runs `opm version`
- **THEN** the command SHALL emit JSON output

#### Scenario: Invalid environment value is rejected

- **WHEN** `OPM_FORMAT=yaml` is set in the environment
- **WHEN** user runs any command
- **THEN** the CLI SHALL reject the value with an error message

### Requirement: Format field in config.cue

The CLI configuration schema SHALL include an optional `format` field accepting `"text"` or `"json"`. The CUE schema SHALL enforce this constraint.

#### Scenario: Config file sets format

- **WHEN** config.cue contains `format: "json"`
- **WHEN** no `--format` flag or `OPM_FORMAT` env var is set
- **THEN** the format SHALL resolve to `json`

#### Scenario: CUE schema rejects invalid format value

- **WHEN** config.cue contains `format: "xml"`
- **THEN** config loading SHALL fail with a CUE validation error

### Requirement: Format resolution precedence

The CLI SHALL resolve the format using precedence: `--format` flag > `OPM_FORMAT` env > `config.format` > default `"text"`.

#### Scenario: Flag overrides environment

- **WHEN** `OPM_FORMAT=text` is set
- **WHEN** user runs `opm version --format json`
- **THEN** the format SHALL be `json`

#### Scenario: Environment overrides config

- **WHEN** config.cue contains `format: "text"`
- **WHEN** `OPM_FORMAT=json` is set
- **THEN** the format SHALL be `json`

#### Scenario: Config overrides default

- **WHEN** config.cue contains `format: "json"`
- **WHEN** no flag or env var is set
- **THEN** the format SHALL be `json` (not `text`)

### Requirement: JSON formatter for logs in JSON mode

When the format is `json`, the `charmbracelet/log` logger SHALL use `JSONFormatter` instead of `TextFormatter`. All log messages on stderr SHALL be emitted as structured JSON.

#### Scenario: JSON mode switches log formatter

- **WHEN** format is `json`
- **WHEN** a command logs `output.Info("hello", "key", "value")`
- **THEN** stderr SHALL contain a JSON log line: `{"time":"...","level":"info","msg":"hello","key":"value"}`

#### Scenario: Text mode uses text formatter

- **WHEN** format is `text`
- **WHEN** a command logs `output.Info("hello")`
- **THEN** stderr SHALL contain a human-readable styled log line

### Requirement: Shared JSON envelope for commands without -o flags

Commands that do not have a per-command `-o` flag SHALL emit a JSON envelope on stdout when format is `json`. The envelope structure SHALL be:

```json
{
  "command": "<command-name>",
  "success": true|false,
  "result": { ... },
  "warnings": ["..."],
  "errors": ["..."]
}
```

The `result` field shape is command-specific. `warnings` and `errors` arrays are optional (omitted if empty).

#### Scenario: mod apply emits JSON envelope

- **WHEN** format is `json`
- **WHEN** user runs `opm mod apply`
- **THEN** stdout SHALL contain a JSON envelope with `"command": "mod apply"`
- **THEN** the envelope SHALL include `"success": true` on success

#### Scenario: mod delete emits JSON envelope

- **WHEN** format is `json`
- **WHEN** user runs `opm mod delete --release-name foo`
- **THEN** stdout SHALL contain a JSON envelope with `"command": "mod delete"`
- **THEN** the `result` field SHALL contain `{"deleted": true, "release": "foo"}`

#### Scenario: version command emits JSON envelope

- **WHEN** format is `json`
- **WHEN** user runs `opm version`
- **THEN** stdout SHALL contain a JSON envelope with `"result"` containing version fields

### Requirement: Commands with -o flags skip the envelope

Commands that have a per-command `-o` flag (`mod build`, `mod status`) SHALL NOT emit the JSON envelope. They SHALL emit raw data on stdout per their `-o` setting. The `--format` flag SHALL only affect their log output on stderr.

#### Scenario: mod build with -o yaml emits raw YAML

- **WHEN** format is `json`
- **WHEN** user runs `opm mod build -o yaml`
- **THEN** stdout SHALL contain raw YAML manifests (no envelope)
- **THEN** stderr SHALL contain JSON-formatted logs

#### Scenario: mod status with -o json emits raw JSON

- **WHEN** format is `json`
- **WHEN** user runs `opm mod status -o json`
- **THEN** stdout SHALL contain raw JSON status data (no envelope)
- **THEN** stderr SHALL contain JSON-formatted logs

#### Scenario: mod build in text mode with -o json works

- **WHEN** format is `text`
- **WHEN** user runs `opm mod build -o json`
- **THEN** stdout SHALL contain raw JSON manifests
- **THEN** stderr SHALL contain human-readable styled logs

### Requirement: Println and Print suppression in JSON mode

The `output.Println()` and `output.Print()` functions SHALL become no-ops when format is `json`. All stdout data in JSON mode MUST go through `output.WriteResult()`.

#### Scenario: Println is suppressed in JSON mode

- **WHEN** format is `json`
- **WHEN** a command calls `output.Println("some text")`
- **THEN** stdout SHALL NOT contain "some text"

#### Scenario: Print is suppressed in JSON mode

- **WHEN** format is `json`
- **WHEN** a command calls `output.Print("data")`
- **THEN** stdout SHALL NOT contain "data"

#### Scenario: Println works in text mode

- **WHEN** format is `text`
- **WHEN** a command calls `output.Println("✔ Done")`
- **THEN** stdout SHALL contain "✔ Done"

### Requirement: Structured JSON error output

When format is `json`, errors SHALL be emitted as structured JSON on stderr. The `DetailError` type SHALL serialize to JSON with all its fields.

#### Scenario: DetailError serializes to JSON

- **WHEN** format is `json`
- **WHEN** a command returns a `DetailError` with type, message, location, field, and hint
- **THEN** stderr SHALL contain a JSON object with all DetailError fields

#### Scenario: Error included in envelope on failure

- **WHEN** format is `json`
- **WHEN** a command fails with an error
- **THEN** the JSON envelope SHALL include `"success": false`
- **THEN** the `errors` array SHALL contain the error message

### Requirement: Interactive commands require flags in JSON mode

Commands that use interactive prompts (`charmbracelet/huh`) SHALL refuse to run in JSON mode unless all required inputs are provided via flags.

#### Scenario: mod init requires template flag in JSON mode

- **WHEN** format is `json`
- **WHEN** user runs `opm mod init` without `--template` flag
- **THEN** the command SHALL fail with error: "interactive prompts not supported in JSON mode; provide required flags: --template, --path"

#### Scenario: mod init works with flags in JSON mode

- **WHEN** format is `json`
- **WHEN** user runs `opm mod init --template standard --path ./my-module`
- **THEN** the command SHALL succeed and emit a JSON envelope

#### Scenario: config init requires all flags in JSON mode

- **WHEN** format is `json`
- **WHEN** user runs `opm config init` without flags
- **THEN** the command SHALL fail with error listing required flags

### Requirement: IsJSON function for mode detection

The `output` package SHALL provide an `IsJSON() bool` function that returns true when the current format is `json`.

#### Scenario: IsJSON returns true in JSON mode

- **WHEN** format is `json`
- **THEN** `output.IsJSON()` SHALL return `true`

#### Scenario: IsJSON returns false in text mode

- **WHEN** format is `text`
- **THEN** `output.IsJSON()` SHALL return `false`

### Requirement: WriteResult function for envelope emission

The `output` package SHALL provide a `WriteResult(CommandResult)` function that marshals and emits the JSON envelope to stdout.

#### Scenario: WriteResult emits valid JSON

- **WHEN** format is `json`
- **WHEN** a command calls `WriteResult(CommandResult{Command: "test", Success: true, Result: map[string]string{"foo": "bar"}})`
- **THEN** stdout SHALL contain valid JSON matching the envelope structure

#### Scenario: WriteResult is no-op in text mode

- **WHEN** format is `text`
- **WHEN** a command calls `WriteResult(...)`
- **THEN** stdout SHALL NOT be written to (commands use Println in text mode)
