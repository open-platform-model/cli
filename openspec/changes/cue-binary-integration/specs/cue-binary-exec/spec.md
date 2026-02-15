## ADDED Requirements

### Requirement: CLI can locate CUE binary on PATH

The CLI SHALL locate the `cue` binary on the system PATH using cross-platform binary discovery.

The discovery mechanism MUST use `exec.LookPath("cue")` to find the binary in a platform-agnostic way (respects `%PATH%` on Windows, `$PATH` on Unix).

#### Scenario: CUE binary found on PATH

- **WHEN** `cue` binary exists in a PATH directory
- **WHEN** binary discovery is invoked
- **THEN** absolute path to the `cue` binary is returned
- **THEN** no error is returned

#### Scenario: CUE binary not found on PATH

- **WHEN** `cue` binary does not exist in any PATH directory
- **WHEN** binary discovery is invoked
- **THEN** error is returned with message: "cue binary not found on PATH"
- **THEN** error includes hint: "Install from https://cuelang.org/docs/install/"

#### Scenario: Multiple CUE binaries on PATH

- **WHEN** multiple `cue` binaries exist in different PATH directories
- **WHEN** binary discovery is invoked
- **THEN** first binary found according to PATH order is returned
- **THEN** PATH precedence is respected per operating system conventions

### Requirement: CLI can execute CUE commands

The CLI SHALL execute arbitrary `cue` subcommands with arguments, capturing stdout, stderr, and exit code.

Execution MUST:
- Run in a specified working directory
- Inherit the current process environment (including `$CUE_REGISTRY`, `$HOME`, etc.)
- Support context-based cancellation
- Capture all output without truncation

#### Scenario: Execute cue mod tidy successfully

- **WHEN** `cue mod tidy` is executed in a valid module directory
- **WHEN** `cue` binary is available
- **WHEN** module has resolvable dependencies
- **THEN** command completes with exit code 0
- **THEN** stdout contains tidy operation output
- **THEN** stderr is empty or contains informational messages
- **THEN** `cue.mod/module.cue` is updated with resolved dependencies

#### Scenario: Execute cue command with error

- **WHEN** `cue vet invalid.cue` is executed
- **WHEN** `invalid.cue` contains CUE syntax errors
- **THEN** command completes with non-zero exit code
- **THEN** stderr contains CUE error messages
- **THEN** error message includes file location and line number

#### Scenario: Execute cue command with custom working directory

- **WHEN** `cue mod tidy` is executed with working directory set to `~/.opm/`
- **WHEN** `cue` binary is available
- **THEN** command runs in the specified directory
- **THEN** relative paths in command output are relative to working directory

#### Scenario: Execute cue command respects inherited environment

- **WHEN** `CUE_REGISTRY=localhost:5000+insecure` environment variable is set
- **WHEN** `cue mod tidy` is executed
- **THEN** `cue` subprocess receives `CUE_REGISTRY` environment variable
- **THEN** registry resolution uses `localhost:5000+insecure`

#### Scenario: Execute cue command with timeout

- **WHEN** `cue export` is executed with a context timeout of 5 seconds
- **WHEN** command takes longer than 5 seconds
- **THEN** command is terminated
- **THEN** error indicates context deadline exceeded

### Requirement: CLI provides version detection for CUE binary

The CLI SHALL detect the installed CUE binary version by invoking `cue version` and parsing the output.

Version detection MUST:
- Parse semantic version string in format `vX.Y.Z` (e.g., `v0.15.4`)
- Support pre-release versions (e.g., `v0.16.0-alpha.1`)
- Handle variations in `cue version` output format
- Return error if version cannot be parsed

#### Scenario: Detect stable CUE version

- **WHEN** `cue version` outputs `cue version v0.15.4`
- **WHEN** version detection is invoked
- **THEN** version string `v0.15.4` is returned
- **THEN** no error is returned

#### Scenario: Detect pre-release CUE version

- **WHEN** `cue version` outputs `cue version v0.16.0-alpha.1`
- **WHEN** version detection is invoked
- **THEN** version string `v0.16.0-alpha.1` is returned
- **THEN** no error is returned

#### Scenario: Detect development CUE version

- **WHEN** `cue version` outputs `cue version v0.0.0-20240215-abc1234`
- **WHEN** version detection is invoked
- **THEN** version string is extracted using regex pattern
- **THEN** version is returned if pattern matches

#### Scenario: Parse failure returns unknown version

- **WHEN** `cue version` outputs unexpected format
- **WHEN** version string cannot be extracted via regex
- **THEN** error is returned indicating parse failure
- **THEN** error message includes raw output for debugging

#### Scenario: Binary execution fails during version detection

- **WHEN** `cue version` command fails to execute
- **WHEN** binary permissions prevent execution
- **THEN** error is returned with execution failure details
- **THEN** error includes stderr output if available

### Requirement: CLI handles binary execution errors gracefully

The CLI SHALL provide actionable error messages when CUE binary operations fail.

Error handling MUST:
- Distinguish between "binary not found" and "execution failed"
- Include stderr output in error messages for debugging
- Preserve exit codes from CUE binary
- Provide installation guidance when binary is missing

#### Scenario: Binary not found error includes installation hint

- **WHEN** CUE binary is not found on PATH
- **WHEN** command requires CUE binary
- **THEN** error message states: "cue binary not found on PATH"
- **THEN** error includes hint: "Install from https://cuelang.org/docs/install/"

#### Scenario: Binary execution failure includes stderr

- **WHEN** `cue mod tidy` fails with exit code 1
- **WHEN** stderr contains: "module path invalid"
- **THEN** error message includes stderr output
- **THEN** error indicates command failed with exit code 1

#### Scenario: Binary permission denied error is clear

- **WHEN** `cue` binary exists but is not executable
- **WHEN** execution is attempted
- **THEN** error message indicates permission denied
- **THEN** error suggests checking file permissions
