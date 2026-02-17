## MODIFIED Requirements

### Requirement: Human log output preserves two-channel architecture

All styled log messages SHALL be written to `os.Stderr`. Data output (manifests, tables, file trees, diffs, completion checkmarks) SHALL be written to `os.Stdout`.

When format is `json`, both channels SHALL emit JSON: stderr emits JSON-formatted logs via `charmbracelet/log`'s `JSONFormatter`, stdout emits JSON envelopes or raw data (depending on command and presence of `-o` flag).

When format is `text`, the existing behavior is preserved: stderr emits styled logs via `TextFormatter`, stdout emits styled data via `output.Println()` and formatting functions.

#### Scenario: Log messages go to stderr

- **WHEN** `modLog.Info(output.FormatResourceLine(...))` is called
- **THEN** the output SHALL be written to `os.Stderr`

#### Scenario: Completion line goes to stdout

- **WHEN** a command prints `✔ Module applied`
- **THEN** the output SHALL be written to `os.Stdout` via `output.Println`

#### Scenario: Piping stdout captures only data

- **WHEN** the user runs `opm mod apply > output.txt`
- **THEN** `output.txt` SHALL contain only data output (completion messages)
- **THEN** styled log lines SHALL appear in the terminal via stderr

#### Scenario: JSON mode preserves channel separation

- **WHEN** format is `json`
- **WHEN** a command logs `output.Info("processing")` and emits a result envelope
- **THEN** stderr SHALL contain JSON log: `{"time":"...","level":"info","msg":"processing"}`
- **THEN** stdout SHALL contain JSON envelope: `{"command":"...","success":true,...}`

## ADDED Requirements

### Requirement: Styled output functions are no-ops in JSON mode

When format is `json`, all styled output functions (`FormatResourceLine`, `FormatCheckmark`, `FormatNotice`, `FormatVetCheck`, etc.) SHALL either return empty strings or skip rendering. Commands SHALL not call these functions in JSON mode — they SHALL build structured result data instead.

The `output.Println()` and `output.Print()` functions SHALL become no-ops in JSON mode.

#### Scenario: Println is suppressed in JSON mode

- **WHEN** format is `json`
- **WHEN** a command calls `output.Println("some text")`
- **THEN** stdout SHALL NOT contain "some text"

#### Scenario: Format functions not called in JSON mode

- **WHEN** format is `json`
- **WHEN** a command checks `output.IsJSON()` and branches
- **THEN** the command SHALL call `output.WriteResult()` instead of `output.Println(output.FormatCheckmark(...))`

### Requirement: JSON level names use full lowercase words

When format is `json`, log level names SHALL be full lowercase words: `"debug"`, `"info"`, `"warn"`, `"error"`, `"fatal"`. This matches the `charmbracelet/log` `JSONFormatter` output.

When format is `text`, log level labels SHALL use 4-character uppercase abbreviations as rendered by `charmbracelet/log`'s default `MaxWidth(4)`: `DEBU`, `INFO`, `WARN`, `ERRO`, `FATA`.

#### Scenario: JSON output uses full level names

- **WHEN** format is `json`
- **WHEN** an error-level message is logged
- **THEN** the stderr JSON SHALL contain `"level": "error"` (full lowercase word)

#### Scenario: Human output shows abbreviated level labels

- **WHEN** format is `text`
- **WHEN** an info-level message is logged
- **THEN** the level label SHALL render as `INFO` (4 characters, bold, teal-green)
