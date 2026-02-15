## ADDED Requirements

### Requirement: CLI compares SDK and binary versions

The CLI SHALL compare the CUE SDK version (embedded at build time) against the installed CUE binary version using major.minor comparison.

Version comparison MUST:
- Extract major and minor version components only (ignore patch)
- Treat `v0.15.3` and `v0.15.4` as compatible (same major.minor)
- Treat `v0.15.4` and `v0.16.0` as incompatible (different minor)
- Return compatibility status indicating match or mismatch

#### Scenario: SDK and binary versions match

- **WHEN** CLI is built with CUE SDK `v0.15.4`
- **WHEN** installed `cue` binary is version `v0.15.3`
- **WHEN** version comparison is performed
- **THEN** versions are reported as compatible
- **THEN** no warning is generated

#### Scenario: SDK and binary major.minor match with different patch

- **WHEN** CLI is built with CUE SDK `v0.15.4`
- **WHEN** installed `cue` binary is version `v0.15.9`
- **WHEN** version comparison is performed
- **THEN** versions are reported as compatible
- **THEN** no warning is generated

#### Scenario: SDK and binary minor versions differ

- **WHEN** CLI is built with CUE SDK `v0.15.4`
- **WHEN** installed `cue` binary is version `v0.16.2`
- **WHEN** version comparison is performed
- **THEN** versions are reported as incompatible
- **THEN** warning message is generated: "CUE SDK (v0.15.4) and binary (v0.16.2) versions differ. Unexpected behavior may occur."

#### Scenario: SDK and binary major versions differ

- **WHEN** CLI is built with CUE SDK `v0.15.4`
- **WHEN** installed `cue` binary is version `v1.0.0`
- **WHEN** version comparison is performed
- **THEN** versions are reported as incompatible
- **THEN** warning message is generated

#### Scenario: Pre-release version comparison

- **WHEN** CLI is built with CUE SDK `v0.15.4`
- **WHEN** installed `cue` binary is version `v0.16.0-alpha.1`
- **WHEN** version comparison is performed
- **THEN** major.minor are extracted from pre-release version
- **THEN** versions are reported as incompatible (0.15 vs 0.16)

### Requirement: CLI warns on version mismatch but does not block execution

The CLI SHALL emit a warning when SDK and binary versions differ, but SHALL NOT prevent command execution.

Warning behavior MUST:
- Print warning to stderr using yellow color
- Display warning only once per command invocation
- Allow command to proceed after warning
- Include both versions in warning message

#### Scenario: Warning is emitted but command proceeds

- **WHEN** version mismatch is detected (SDK v0.15.4, binary v0.16.2)
- **WHEN** `opm config init` is executed
- **THEN** warning is printed to stderr
- **THEN** command execution continues
- **THEN** command completes successfully if no other errors occur

#### Scenario: Warning uses styled output

- **WHEN** version mismatch is detected
- **WHEN** warning is emitted
- **THEN** warning uses `output.Warn()` for structured logging
- **THEN** warning appears in yellow on terminals with color support
- **THEN** warning format: "CUE SDK (<sdk-version>) and binary (<bin-version>) versions differ. Unexpected behavior may occur."

#### Scenario: No warning when versions are compatible

- **WHEN** SDK and binary major.minor versions match
- **WHEN** command is executed
- **THEN** no version warning is emitted
- **THEN** stderr does not contain version mismatch messages

### Requirement: opm version command displays binary version

The `opm version` command SHALL display the CUE binary version alongside the SDK version.

Output MUST include:
- CUE SDK version (embedded at build time)
- CUE binary version (detected from installed binary)
- Compatibility status (compatible, version mismatch, or not found)
- Binary path (if found)

#### Scenario: Display version with compatible binary

- **WHEN** `opm version` is executed
- **WHEN** CUE binary is found on PATH at `/usr/local/bin/cue`
- **WHEN** SDK version is `v0.15.4`
- **WHEN** binary version is `v0.15.3`
- **THEN** output includes: `CUE SDK:   v0.15.4`
- **THEN** output includes: `CUE binary: v0.15.3 (compatible)`

#### Scenario: Display version with incompatible binary

- **WHEN** `opm version` is executed
- **WHEN** CUE binary is found on PATH
- **WHEN** SDK version is `v0.15.4`
- **WHEN** binary version is `v0.16.2`
- **THEN** output includes: `CUE SDK:   v0.15.4`
- **THEN** output includes: `CUE binary: v0.16.2 (version mismatch - unexpected behavior may occur)`

#### Scenario: Display version when binary not found

- **WHEN** `opm version` is executed
- **WHEN** CUE binary is not found on PATH
- **THEN** output includes: `CUE SDK:   v0.15.4`
- **THEN** output includes: `CUE binary: not found on PATH`

#### Scenario: Display version when binary version unknown

- **WHEN** `opm version` is executed
- **WHEN** CUE binary is found but version cannot be detected
- **THEN** output includes: `CUE SDK:   v0.15.4`
- **THEN** output includes: `CUE binary: /path/to/cue (version unknown)`

### Requirement: Version check is performed before binary invocation

The CLI SHALL perform version compatibility check before invoking the CUE binary for operations.

Check timing MUST:
- Occur after binary discovery succeeds
- Occur before executing CUE commands
- Emit warning if versions differ
- Allow execution to proceed regardless of compatibility status

#### Scenario: Version check before cue mod tidy

- **WHEN** `opm config init` attempts to run `cue mod tidy`
- **WHEN** CUE binary is found
- **WHEN** binary version is detected
- **THEN** version compatibility check is performed
- **THEN** warning is emitted if versions differ
- **THEN** `cue mod tidy` is invoked after check

#### Scenario: Version check skipped when binary not found

- **WHEN** `opm config init` is executed
- **WHEN** CUE binary is not found on PATH
- **THEN** version compatibility check is skipped
- **THEN** no version warning is emitted
- **THEN** command continues with binary-not-found handling
