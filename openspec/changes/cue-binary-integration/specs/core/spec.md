## ADDED Requirements

### Requirement: Version command displays CUE SDK and binary information

The `opm version` command SHALL display comprehensive version information including both the embedded CUE SDK version and the installed CUE binary version.

Output MUST include:
- OPM CLI version (set via ldflags at build time)
- Git commit hash
- Build timestamp
- Go version used to build
- CUE SDK version (embedded at build time)
- CUE binary version (detected from PATH) with compatibility status

The CUE binary version display SHALL indicate:
- Full semantic version if binary is found and version is detected
- Compatibility status: "compatible" or "version mismatch - unexpected behavior may occur"
- "not found on PATH" if binary is not available
- "(version unknown)" if binary is found but version cannot be determined

#### Scenario: Display version with all information available

- **WHEN** `opm version` is executed
- **WHEN** OPM CLI version is `0.2.0`
- **WHEN** CUE SDK version is `v0.15.4`
- **WHEN** CUE binary is found with version `v0.15.3`
- **THEN** output includes:
  ```
  opm version 0.2.0
    Commit:      abc1234
    Built:       2026-02-15T10:30:00Z
    Go:          go1.23.5
    CUE SDK:     v0.15.4
    CUE binary:  v0.15.3 (compatible)
  ```

#### Scenario: Display version with binary version mismatch

- **WHEN** `opm version` is executed
- **WHEN** CUE SDK version is `v0.15.4`
- **WHEN** CUE binary version is `v0.16.2`
- **THEN** CUE binary line shows: `CUE binary:  v0.16.2 (version mismatch - unexpected behavior may occur)`

#### Scenario: Display version when binary not found

- **WHEN** `opm version` is executed
- **WHEN** `cue` binary does not exist on PATH
- **THEN** CUE binary line shows: `CUE binary:  not found on PATH`

#### Scenario: Display version when binary version unknown

- **WHEN** `opm version` is executed
- **WHEN** CUE binary is found at `/usr/local/bin/cue`
- **WHEN** version detection fails (unparseable output)
- **THEN** CUE binary line shows: `CUE binary:  /usr/local/bin/cue (version unknown)`

#### Scenario: Version output is parseable by scripts

- **WHEN** `opm version` is executed
- **THEN** output format is consistent and line-based
- **THEN** each component is on its own line
- **THEN** labels are consistent (e.g., "CUE SDK:", "CUE binary:")
- **THEN** output can be parsed by grep/awk for automation
