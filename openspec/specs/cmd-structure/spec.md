## Purpose

Defines the structural conventions for CLI command packages in the OPM codebase. This covers how commands are organized into sub-packages, how configuration is injected, and how the cobra command tree maps to the package layout under `internal/cmd/`.

## Requirements

### Requirement: Commands receive configuration via explicit injection

Global CLI configuration (OPMConfig, resolved registry, verbose flag) SHALL be passed
explicitly to command constructors via a `GlobalConfig` struct rather than read from
package-level variables or accessor functions.

#### Scenario: Sub-command accesses OPMConfig

- **WHEN** a sub-command in `internal/cmd/mod/` or `internal/cmd/config/` needs the loaded OPMConfig
- **THEN** it reads it from the `*GlobalConfig` parameter passed to its constructor, not from a package-level accessor

#### Scenario: No package-level mutable state in sub-packages

- **WHEN** any file in `internal/cmd/mod/` or `internal/cmd/config/` is inspected
- **THEN** it contains no package-level `var` declarations for flags or configuration state

### Requirement: Command packages are organised by command group

The `internal/cmd/` package SHALL be split into sub-packages that mirror the cobra command tree.

#### Scenario: mod commands are in their own package

- **WHEN** the `internal/cmd/mod/` directory is inspected
- **THEN** it contains all `mod` sub-command implementations (`init`, `build`, `vet`, `apply`, `diff`, `delete`, `status`)

#### Scenario: config commands are in their own package

- **WHEN** the `internal/cmd/config/` directory is inspected
- **THEN** it contains all `config` sub-command implementations (`init`, `vet`)
