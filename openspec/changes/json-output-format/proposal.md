## Why

The CLI cannot be used in CI/CD pipelines. All command output is human-readable styled text — colored with lipgloss, formatted with Unicode symbols, and unstructured. Automated tooling cannot reliably parse results, detect errors, or extract resource statuses. A machine-readable JSON output format is required for pipeline integration.

## What Changes

- Add a global `--format` persistent flag accepting `text` (default) or `json`
- Add `OPM_FORMAT` environment variable for CI (set once, all commands emit JSON)
- Add `format` field to `config.cue` schema with CUE validation (`"text" | "json"`)
- Resolution follows existing precedence: `--format` flag > `OPM_FORMAT` env > `config.format` > default `"text"`
- When format is `json`:
  - `charmbracelet/log` switches from `TextFormatter` to `JSONFormatter` — all log messages on stderr become structured JSON automatically (no call-site changes needed)
  - Commands without `-o` flags emit a shared JSON envelope on stdout: `{"command","success","result","warnings","errors"}`
  - Commands with `-o` flags (`mod build`, `mod status`) skip the envelope — emit raw data per their `-o` setting. `--format` only affects their logs on stderr.
  - `output.Println()` / `output.Print()` become no-ops in JSON mode
  - Errors on stderr are emitted as structured JSON
  - Interactive prompts (`charmbracelet/huh` in `mod init`, `config init`) are blocked — all inputs must be provided via flags
- `--format` and per-command `-o` flags are independent:
  - `--format` controls CLI chrome (logs, errors, command results)
  - `-o` controls data serialization (manifests, status records)
  - When both are set, `-o` wins for data on stdout

This is a **MINOR** version change — new flag with a backward-compatible default (`text`).

## Capabilities

### New Capabilities

- `cli-output-format`: Global output format switching between human-readable text and machine-readable JSON, including the shared JSON envelope structure, format resolution via flag/env/config, per-command result schemas, suppression of styled output in JSON mode, and structured JSON error output

### Modified Capabilities

- `config`: Adding `format` to the configuration precedence chain (new row in the resolution table, new CUE schema field, new env var `OPM_FORMAT`)
- `log-output-style`: Adding format mode awareness — styled output functions become no-ops in JSON mode, two-channel architecture (stderr/stdout) preserved but both channels switch to JSON

## Impact

- **Packages**: `internal/output` (format state, `WriteResult`, `IsJSON`, `Println`/`Print` suppression), `internal/config` (new `Format` field, resolver, CUE schema), `internal/cmd` (root flag, per-command result structs), `internal/errors` (`DetailError.MarshalJSON`), `cmd/opm` (JSON error path in `main.go`)
- **All commands without `-o`**: Each needs a JSON result struct and `if output.IsJSON()` branch (`mod apply`, `mod delete`, `mod diff`, `mod vet`, `config vet`, `config init`, `mod init`, `version`)
- **Commands with `-o`**: `mod build` and `mod status` — no envelope changes, only logs affected
- **Interactive commands**: `mod init` and `config init` need flag-only mode when format is `json`
- **No breaking changes**: Default is `text`, all existing behavior unchanged
- **No new dependencies**: `charmbracelet/log` v0.4.2 already supports `JSONFormatter`
