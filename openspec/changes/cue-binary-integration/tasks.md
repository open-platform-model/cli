## 1. Core Package - internal/cue/

- [ ] 1.1 Create `internal/cue/` package directory and basic structure
- [ ] 1.2 Implement `FindBinary() (string, error)` using `exec.LookPath("cue")`
- [ ] 1.3 Implement `GetVersion(ctx, binPath) (string, error)` with regex parsing of `cue version` output
- [ ] 1.4 Define `CompatibilityStatus` struct with SDK/binary versions, compatible flag, and warning message
- [ ] 1.5 Implement `CheckCompatibility(sdkVersion, binaryVersion) CompatibilityStatus` with major.minor comparison
- [ ] 1.6 Define `RunResult` struct with Stdout, Stderr, and ExitCode fields
- [ ] 1.7 Implement `Run(ctx, binPath, workDir, args...) (*RunResult, error)` with environment inheritance
- [ ] 1.8 Add error types: `ErrBinaryNotFound`, `ErrVersionParseFailed`, `ErrExecutionFailed`
- [ ] 1.9 Add helper function `extractMajorMinor(version string) string` for version comparison

## 2. Version Package - internal/version/

- [ ] 2.1 Add `CUEBinaryVersion string` field to `Info` struct
- [ ] 2.2 Add `CUEBinaryPath string` field to `Info` struct
- [ ] 2.3 Add `CUECompatible bool` field to `Info` struct for compatibility status
- [ ] 2.4 Update `Get()` function to populate new fields using `internal/cue` package
- [ ] 2.5 Update `String()` method to include CUE binary version in formatted output

## 3. Command: opm version

- [ ] 3.1 Update `internal/cmd/version.go` to import `internal/cue` package
- [ ] 3.2 Modify `runVersion()` to call `cue.FindBinary()` after displaying SDK version
- [ ] 3.3 Add logic to display "not found on PATH" when binary is missing
- [ ] 3.4 Add logic to call `cue.GetVersion()` when binary is found
- [ ] 3.5 Add logic to call `cue.CheckCompatibility()` and format status message
- [ ] 3.6 Display binary version with compatibility status: "(compatible)" or "(version mismatch - unexpected behavior may occur)"
- [ ] 3.7 Handle version detection failures gracefully with "(version unknown)" message

## 4. Command: opm config init

- [ ] 4.1 Update `internal/cmd/config_init.go` to import `internal/cue` package
- [ ] 4.2 Add CUE binary discovery logic after writing config files (call `cue.FindBinary()`)
- [ ] 4.3 Add graceful fallback when binary not found: print yellow notice with installation link
- [ ] 4.4 Add version compatibility check before running `cue mod tidy`
- [ ] 4.5 Emit warning to stderr if SDK/binary versions differ using `output.Warn()`
- [ ] 4.6 Add `cue.Run()` invocation for `mod tidy` with `~/.opm/` as working directory
- [ ] 4.7 Handle `cue mod tidy` success: display green checkmark with "Dependencies resolved"
- [ ] 4.8 Handle `cue mod tidy` failure: warn and show yellow notice to run manually
- [ ] 4.9 Ensure command succeeds even if `cue mod tidy` fails (graceful degradation)

## 5. Unit Tests - internal/cue/

- [ ] 5.1 Create `internal/cue/cue_test.go` with table-driven tests
- [ ] 5.2 Test `FindBinary()` with mocked PATH (binary found, binary not found)
- [ ] 5.3 Test `GetVersion()` with various `cue version` output formats (stable, pre-release, dev)
- [ ] 5.4 Test `GetVersion()` parse failures return appropriate errors
- [ ] 5.5 Test `CheckCompatibility()` with matching major.minor (compatible = true)
- [ ] 5.6 Test `CheckCompatibility()` with different patch versions (compatible = true)
- [ ] 5.7 Test `CheckCompatibility()` with different minor versions (compatible = false, warning set)
- [ ] 5.8 Test `CheckCompatibility()` with different major versions (compatible = false)
- [ ] 5.9 Test `extractMajorMinor()` helper with various version strings
- [ ] 5.10 Test `Run()` with successful command execution (exit code 0, stdout captured)
- [ ] 5.11 Test `Run()` with failed command execution (non-zero exit code, stderr captured)
- [ ] 5.12 Test `Run()` with custom working directory
- [ ] 5.13 Test `Run()` with context cancellation/timeout

## 6. Unit Tests - internal/version/

- [ ] 6.1 Update `internal/version/version_test.go` to test new fields
- [ ] 6.2 Test `Get()` populates CUE binary fields when binary is available
- [ ] 6.3 Test `Get()` handles missing binary gracefully (empty path, unknown version)
- [ ] 6.4 Test `String()` includes CUE binary version in output

## 7. Unit Tests - Commands

- [ ] 7.1 Update `internal/cmd/version_test.go` to verify binary version display
- [ ] 7.2 Test `opm version` output when binary found and compatible
- [ ] 7.3 Test `opm version` output when binary found but incompatible
- [ ] 7.4 Test `opm version` output when binary not found
- [ ] 7.5 Update `internal/cmd/config_init_test.go` to test `cue mod tidy` integration
- [ ] 7.6 Test `config init` with binary available runs `cue mod tidy`
- [ ] 7.7 Test `config init` with binary not found shows yellow notice
- [ ] 7.8 Test `config init` with `cue mod tidy` failure shows warning and succeeds

## 8. Integration Tests

- [ ] 8.1 Create `tests/integration/cue_binary_test.go` for real binary tests
- [ ] 8.2 Add skip condition: `t.Skip("cue binary not available")` when binary missing
- [ ] 8.3 Test real `cue mod tidy` execution in temp directory
- [ ] 8.4 Test version detection with actual installed CUE binary
- [ ] 8.5 Test `config init` end-to-end with real `cue` binary (if available)

## 9. Documentation

- [ ] 9.1 Add godoc comments to all exported functions in `internal/cue/`
- [ ] 9.2 Add package-level documentation for `internal/cue/` explaining purpose
- [ ] 9.3 Update `internal/cmd/version.go` help text to mention CUE binary version display
- [ ] 9.4 Update `internal/cmd/config_init.go` help text to mention automatic dependency resolution

## 10. Validation

- [ ] 10.1 Run `task fmt` to format all Go files
- [ ] 10.2 Run `task lint` and fix any linting errors
- [ ] 10.3 Run `task test` and verify all tests pass
- [ ] 10.4 Run `task build` and verify binary builds successfully
- [ ] 10.5 Manual test: `opm version` shows CUE binary version (if installed)
- [ ] 10.6 Manual test: `opm config init --force` runs `cue mod tidy` automatically
- [ ] 10.7 Manual test: Temporarily remove `cue` from PATH and verify graceful degradation
- [ ] 10.8 Manual test: Test with mismatched CUE versions and verify warning appears
