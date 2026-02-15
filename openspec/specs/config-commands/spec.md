## ADDED Requirements

### Requirement: Config init command creates configuration

The `opm config init` command SHALL create the default configuration files in `~/.opm/`.

The command creates:

- `~/.opm/config.cue` - Main configuration file with embedded template
- `~/.opm/cue.mod/module.cue` - CUE module metadata for import resolution

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

### Requirement: Config init template includes provider import

The default config.cue template SHALL include the provider import and kubernetes provider configuration.

Template structure:

```cue
import (
    prov "opmodel.dev/providers@v0"
)

config: {
    registry: "registry.opmodel.dev"
    providers: {
        kubernetes: prov.#Registry["kubernetes"]
    }
    kubernetes: {
        kubeconfig: "~/.kube/config"
        context?: string
        namespace: "default"
    }
}
```

#### Scenario: Generated config includes documentation comments

- **WHEN** `opm config init` creates config.cue
- **THEN** file includes header comment: "// OPM CLI Configuration"
- **THEN** each field includes descriptive comment explaining purpose
- **THEN** comments reference override methods (flags, env vars)

### Requirement: Config init module template enables imports

The default cue.mod/module.cue SHALL define the module and dependencies for CUE import resolution.

Template structure:

```cue
module: "opmodel.dev/config@v0"

language: {
    version: "v0.15.0"
}

deps: {
    "opmodel.dev/providers@v0": {
        v: "v0.1.0"
    }
}
```

#### Scenario: Module template enables provider imports

- **WHEN** `opm config init` creates cue.mod/module.cue
- **THEN** module is named "opmodel.dev/config@v0"
- **THEN** language version matches CUE SDK requirements
- **THEN** deps includes providers dependency

### Requirement: Config vet command validates configuration

The `opm config vet` command SHALL validate the configuration file using CUE evaluation.

Checks performed:

1. Config file exists at resolved path
2. cue.mod/module.cue exists
3. Config file is syntactically valid CUE
4. Config evaluates without errors (imports resolve, constraints pass)

Each check SHALL print a styled line to stdout using `FormatVetCheck` as it passes, giving the user real-time feedback. On failure, all previously-passing checks SHALL remain visible.

#### Scenario: Valid configuration passes validation

- **WHEN** `opm config vet` is run
- **WHEN** config.cue exists and is valid CUE
- **WHEN** cue.mod/module.cue exists
- **THEN** command succeeds
- **THEN** output SHALL contain a checkmark line for each passing check
- **THEN** first line SHALL be: `✔ Config file found` with the config file path right-aligned in dim style
- **THEN** second line SHALL be: `✔ Module metadata found` with the module.cue path right-aligned in dim style
- **THEN** third line SHALL be: `✔ CUE evaluation passed`

#### Scenario: Missing config file fails with actionable error

- **WHEN** `opm config vet` is run
- **WHEN** `~/.opm/config.cue` does not exist
- **THEN** command fails with not-found error
- **THEN** no checkmark lines SHALL be printed (first check failed)
- **THEN** hint: "Run 'opm config init' to create default configuration"

#### Scenario: Missing cue.mod fails after config check passes

- **WHEN** `opm config vet` is run
- **WHEN** config.cue exists but cue.mod/module.cue does not
- **THEN** the config file check SHALL print a passing checkmark line
- **THEN** command fails with not-found error
- **THEN** hint: "Run 'opm config init' to create configuration"

#### Scenario: Invalid CUE fails after file checks pass

- **WHEN** `opm config vet` is run
- **WHEN** config.cue exists and cue.mod/module.cue exists
- **WHEN** config.cue contains syntax or evaluation errors
- **THEN** both file existence checks SHALL print passing checkmark lines
- **THEN** command fails with CUE error message
- **THEN** error includes file location

### Requirement: Config vet respects path overrides

The `opm config vet` command SHALL respect `--config` flag and `OPM_CONFIG` environment variable for config path resolution.

#### Scenario: Validate custom config path via flag

- **WHEN** `opm config vet --config /custom/config.cue` is run
- **THEN** validation uses `/custom/config.cue` instead of default

#### Scenario: Validate custom config path via environment

- **WHEN** `OPM_CONFIG=/custom/config.cue` is set
- **WHEN** `opm config vet` is run
- **THEN** validation uses `/custom/config.cue` instead of default

### Requirement: Config vet uses registry for import resolution

The `opm config vet` command SHALL pass the resolved registry to CUE evaluation for import resolution.

#### Scenario: Validate config with provider imports

- **WHEN** `opm config vet` is run
- **WHEN** config.cue imports from registry (e.g., `opmodel.dev/providers@v0`)
- **WHEN** `--registry localhost:5001` is specified
- **THEN** CUE evaluation uses `localhost:5001` for import resolution
