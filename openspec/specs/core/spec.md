# CLI Core

## Purpose

Defines the core CLI functionality for OPM: configuration management (`config init`, `config vet`), module scaffolding (`mod init`), and module management (`mod tidy`, `mod vet`).

## User Stories

### New user creating first module

As a Module Author new to OPM, I need to quickly scaffold a valid module structure so that I can start defining my application without learning the file layout manually.

### Platform operator initializing configuration

As a Platform Operator, I need to initialize CLI configuration with sensible defaults so that I can start using OPM commands without manual setup.

### Developer validating configuration

As a Module Author, I need to validate my CLI configuration before running module commands so that config errors don't cause cryptic failures during build or deploy.

### Developer resolving dependencies

As a Module Author, I need to resolve and fetch CUE module dependencies so that my imports work correctly.

## Requirements

### Module Commands

- **FR-001**: The CLI MUST provide `mod init` to create a new module from a template. Support `--template` flag accepting `simple`, `standard` (default), or `advanced`.
- **FR-002**: The CLI MUST provide `mod vet` for module validation and `mod tidy` for dependency management.

### Configuration

- **FR-008**: The CLI MUST use a CUE module at `~/.opm/` as the configuration directory, containing `config.cue` and `cue.mod/module.cue`. The `config init` command MUST include the kubernetes provider and set secure file permissions (0700 dir, 0600 files).
- **FR-009**: The CLI MUST resolve the registry URL using precedence: (1) `--registry` flag, (2) `OPM_REGISTRY` env, (3) `config.registry`.
- **FR-010**: When the registry is unreachable, commands MUST fail fast with a clear error message.
- **FR-011**: The config MUST support optional fields: `registry`, `kubeconfig`, `context`, `namespace`.
- **FR-012**: The `config.providers` field MUST be a map of provider aliases to provider definitions loaded via CUE imports.
- **FR-013**: The CLI MUST implement two-phase config loading: (1) Extract `config.registry` via simple parsing; (2) Load full config with imports using resolved registry.
- **FR-014**: When providers are configured but no registry is resolvable, the CLI MUST fail fast.

### CLI Behavior

- **FR-015**: All commands MUST be non-interactive.
- **FR-016**: The CLI MUST provide structured logging to `stderr` with colors. `--verbose` increases detail.
- **FR-017**: The CLI MUST provide a global `--output-format` flag (`-o`) supporting `text` (default), `yaml`, and `json`.
- **FR-018**: Configuration precedence: (1) Flags, (2) Environment variables, (3) Config file, (4) Built-in defaults.
- **FR-019**: The CLI SHOULD provide shell completion via a `completion` subcommand.

## Design Rationale

### Why CUE for configuration

CUE enables type-safe provider references via imports. YAML would require separate schema validation and couldn't express provider module references cleanly.

### Why two-phase config loading

Providers are loaded from a CUE registry, but we need the registry URL from the config file. Two-phase loading resolves this chicken-and-egg: first extract registry URL with simple parsing, then load full config with registry available.

### Why secure file permissions

Config files may reference sensitive provider settings. Setting 0700/0600 permissions during `config init` prevents accidental exposure.
