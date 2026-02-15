## MODIFIED Requirements

### Requirement: Config init command creates configuration

The `opm config init` command SHALL create the default configuration files in `~/.opm/` and attempt to resolve CUE module dependencies automatically.

The command creates:

- `~/.opm/config.cue` - Main configuration file with embedded template
- `~/.opm/cue.mod/module.cue` - CUE module metadata for import resolution

After creating files, the command SHALL attempt to run `cue mod tidy` to resolve dependencies automatically. If the `cue` binary is not found, the command SHALL succeed with a warning message directing users to install CUE and run `cue mod tidy` manually.

#### Scenario: Initialize configuration for first time

- **WHEN** `opm config init` is run
- **WHEN** no configuration exists at `~/.opm/config.cue`
- **THEN** `~/.opm/` directory is created with 0700 permissions
- **THEN** `~/.opm/cue.mod/` directory is created with 0700 permissions
- **THEN** `~/.opm/config.cue` is written with 0600 permissions
- **THEN** `~/.opm/cue.mod/module.cue` is written with 0600 permissions
- **THEN** success message lists created files
- **THEN** message suggests: "Validate with: opm config vet"

#### Scenario: Refuse to overwrite existing configuration

- **WHEN** `opm config init` is run
- **WHEN** `~/.opm/config.cue` already exists
- **THEN** command fails with validation error
- **THEN** error message: "configuration already exists"
- **THEN** hint: "Use --force to overwrite existing configuration."

#### Scenario: Force overwrite existing configuration

- **WHEN** `opm config init --force` is run
- **WHEN** `~/.opm/config.cue` already exists
- **THEN** existing files are overwritten
- **THEN** success message lists created files

#### Scenario: Auto-run cue mod tidy when binary is available

- **WHEN** `opm config init` is run
- **WHEN** `cue` binary is found on PATH
- **WHEN** CUE SDK and binary versions are compatible
- **THEN** `cue mod tidy` is executed in `~/.opm/` directory
- **THEN** dependencies in `cue.mod/module.cue` are resolved
- **THEN** success message includes: "Dependencies resolved with cue mod tidy"

#### Scenario: Auto-run cue mod tidy with version mismatch warning

- **WHEN** `opm config init` is run
- **WHEN** `cue` binary is found on PATH
- **WHEN** CUE SDK and binary versions differ (major.minor)
- **THEN** version compatibility warning is emitted to stderr
- **THEN** `cue mod tidy` is executed despite version mismatch
- **THEN** command succeeds if `cue mod tidy` completes successfully

#### Scenario: Binary not found - graceful fallback

- **WHEN** `opm config init` is run
- **WHEN** `cue` binary is not found on PATH
- **THEN** configuration files are created successfully
- **THEN** command succeeds (does not fail due to missing binary)
- **THEN** yellow notice is displayed: "CUE binary not found. Run 'cue mod tidy' manually to resolve dependencies"
- **THEN** notice includes installation link: "Install CUE from: https://cuelang.org/docs/install/"

#### Scenario: cue mod tidy fails - graceful fallback

- **WHEN** `opm config init` is run
- **WHEN** `cue` binary is found but `cue mod tidy` fails
- **WHEN** failure is due to network issues or invalid registry
- **THEN** configuration files are created successfully
- **THEN** warning is emitted: "cue mod tidy failed: <error message>"
- **THEN** yellow notice is displayed: "Run 'cue mod tidy' manually in ~/.opm/"
- **THEN** command succeeds (does not fail due to tidy failure)
