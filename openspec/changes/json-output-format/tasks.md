## 1. Config Layer — Format Field and Resolution

- [ ] 1.1 Add `Format string` field to `Config` struct in `internal/config/config.go`
- [ ] 1.2 Add `format?: "text" | "json"` to CUE schema in `internal/config/schema/config.cue`
- [ ] 1.3 Add Format row to config precedence table comment in `resolver.go`
- [ ] 1.4 Add `Format` field to `ResolvedBaseConfig` struct in `resolver.go`
- [ ] 1.5 Implement format resolution in `ResolveBase()` using `resolveStringField()` with `OPM_FORMAT` env var and default `"text"`
- [ ] 1.6 Write tests for format resolution precedence (flag > env > config > default)

## 2. Root Command — Global Flag and Wiring

- [ ] 2.1 Add `formatFlag string` package-level var to `internal/cmd/root.go`
- [ ] 2.2 Register `--format` persistent flag with validation (accepts `text` or `json`)
- [ ] 2.3 Wire formatFlag into `initializeGlobals()` → pass to `output.SetupLogging()`
- [ ] 2.4 Update debug log line in `initializeGlobals()` to include resolved format value

## 3. Output Package — Format State and Primitives

- [ ] 3.1 Add `Format string` field to `LogConfig` struct in `internal/output/log.go`
- [ ] 3.2 Add package-level `currentFormat Format` var to store resolved format
- [ ] 3.3 Implement `IsJSON() bool` function returning `currentFormat == FormatJSON`
- [ ] 3.4 Update `SetupLogging()` to store format and call `logger.SetFormatter(log.JSONFormatter)` when format is `json`
- [ ] 3.5 Modify `Println()` and `Print()` to become no-ops when `IsJSON()` returns true
- [ ] 3.6 Write tests for format state and IsJSON() function

## 4. Output Package — JSON Envelope Types

- [ ] 4.1 Create `CommandResult` struct with `Command`, `Success`, `Result`, `Warnings`, `Errors` fields in `internal/output/result.go`
- [ ] 4.2 Implement `WriteResult(CommandResult) error` function that marshals and writes JSON to stdout when `IsJSON()` is true
- [ ] 4.3 Make `WriteResult()` a no-op when format is `text`
- [ ] 4.4 Write tests for WriteResult JSON marshaling and text mode no-op

## 5. Error Package — JSON Serialization

- [ ] 5.1 Implement `MarshalJSON() ([]byte, error)` method on `DetailError` in `internal/errors/errors.go`
- [ ] 5.2 Ensure all DetailError fields (Type, Message, Location, Field, Context, Hint) are included in JSON output
- [ ] 5.3 Write tests for DetailError JSON serialization

## 6. Main Entry Point — JSON Error Path

- [ ] 6.1 Update `cmd/opm/main.go` error handling to check `output.IsJSON()`
- [ ] 6.2 When JSON mode, emit error as JSON on stderr instead of plain text
- [ ] 6.3 Preserve `ExitError.Printed` flag behavior in JSON mode
- [ ] 6.4 Write test for JSON error output path

## 7. Command: version — JSON Envelope

- [ ] 7.1 Define `versionResult` struct with version, commit, built, go, cueSdk fields
- [ ] 7.2 Add `if output.IsJSON()` branch that builds versionResult and calls `WriteResult()`
- [ ] 7.3 Preserve existing text output path
- [ ] 7.4 Write test for version JSON output

## 8. Command: mod vet — JSON Envelope

- [ ] 8.1 Define `vetResult` struct with checks array and passed boolean
- [ ] 8.2 Collect vet check results into result struct instead of printing inline
- [ ] 8.3 Add `if output.IsJSON()` branch that calls `WriteResult()`
- [ ] 8.4 Preserve existing text output path
- [ ] 8.5 Write test for mod vet JSON output

## 9. Command: config vet — JSON Envelope

- [ ] 9.1 Define `configVetResult` struct (same shape as mod vet)
- [ ] 9.2 Collect config vet check results into result struct
- [ ] 9.3 Add `if output.IsJSON()` branch that calls `WriteResult()`
- [ ] 9.4 Preserve existing text output path
- [ ] 9.5 Write test for config vet JSON output

## 10. Command: mod delete — JSON Envelope

- [ ] 10.1 Define `deleteResult` struct with deleted boolean and release name
- [ ] 10.2 Add `if output.IsJSON()` branch that calls `WriteResult()`
- [ ] 10.3 Preserve existing text output path
- [ ] 10.4 Write test for mod delete JSON output

## 11. Command: mod apply — JSON Envelope

- [ ] 11.1 Define `applyResult` struct with resources array, upToDate boolean
- [ ] 11.2 Collect resource apply statuses into result struct
- [ ] 11.3 Add `if output.IsJSON()` branch that calls `WriteResult()`
- [ ] 11.4 Preserve existing text output path
- [ ] 11.5 Write test for mod apply JSON output

## 12. Command: mod diff — JSON Envelope

- [ ] 12.1 Define `diffResult` struct with modified, new, orphaned arrays
- [ ] 12.2 Structure diff output data into result struct
- [ ] 12.3 Add `if output.IsJSON()` branch that calls `WriteResult()`
- [ ] 12.4 Preserve existing text output path
- [ ] 12.5 Write test for mod diff JSON output

## 13. Command: mod init — Flag-Only Mode

- [ ] 13.1 Add check for `output.IsJSON()` at start of command
- [ ] 13.2 When JSON mode, validate all required flags are provided (--template, --path)
- [ ] 13.3 Return error if required flags missing: "interactive prompts not supported in JSON mode; provide required flags: --template, --path"
- [ ] 13.4 Define `initResult` struct with path, template, files array
- [ ] 13.5 Add JSON output branch that calls `WriteResult()`
- [ ] 13.6 Write test for mod init JSON output and flag requirement

## 14. Command: config init — Flag-Only Mode

- [ ] 14.1 Add check for `output.IsJSON()` at start of command
- [ ] 14.2 When JSON mode, validate all required flags are provided
- [ ] 14.3 Return error if required flags missing with list of required flags
- [ ] 14.4 Define `configInitResult` struct with path and files array
- [ ] 14.5 Add JSON output branch that calls `WriteResult()`
- [ ] 14.6 Write test for config init JSON output and flag requirement

## 15. Commands: mod build, mod status — Verify No Envelope

- [ ] 15.1 Verify mod build with -o flag does NOT emit envelope in JSON mode (only logs switch to JSON)
- [ ] 15.2 Verify mod status with -o flag does NOT emit envelope in JSON mode
- [ ] 15.3 Write integration test: `--format json -o yaml` produces raw YAML on stdout, JSON logs on stderr
- [ ] 15.4 Write integration test: `--format text -o json` produces raw JSON on stdout, styled logs on stderr

## 16. Integration Tests — End-to-End Format Switching

- [ ] 16.1 Write integration test: `OPM_FORMAT=json` env var sets format globally
- [ ] 16.2 Write integration test: `--format` flag overrides `OPM_FORMAT` env
- [ ] 16.3 Write integration test: config.cue `format: "json"` sets format
- [ ] 16.4 Write integration test: all commands emit valid JSON in JSON mode
- [ ] 16.5 Write integration test: stderr/stdout separation preserved in JSON mode

## 17. Validation Gates

- [ ] 17.1 Run `task fmt` — all Go files formatted
- [ ] 17.2 Run `task lint` — golangci-lint passes with no new warnings
- [ ] 17.3 Run `task test` — all unit and integration tests pass
- [ ] 17.4 Run `task test:coverage` — ensure coverage doesn't drop

## 18. Documentation

- [ ] 18.1 Update CLI help text for `--format` flag
- [ ] 18.2 Add `OPM_FORMAT` to environment variables documentation
- [ ] 18.3 Add example CI/CD usage to README or docs
- [ ] 18.4 Document JSON envelope structure and per-command result schemas
