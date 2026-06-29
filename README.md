# OPM CLI

> **WARNING: UNDER HEAVY DEVELOPMENT** - This project is actively being developed and APIs may change frequently.

Command-line interface for the Open Platform Model (OPM). Build, validate, deploy, and inspect portable application releases defined with CUE.

## Quick Start

```bash
# Build the CLI
task build

# Initialize a new module
./bin/opm module init ./my-module

# Validate a module
./bin/opm module vet ./my-module

# Validate an instance file
./bin/opm instance vet ./instance.cue

# Render an instance file
./bin/opm instance build ./instance.cue

# Apply an instance file
./bin/opm instance apply ./instance.cue
```

## Features

- **Type-safe definitions** using CUE
- **Kubernetes-native** resource management
- **Portable blueprints** across providers
- **OCI-based distribution** for modules and definitions
- **Interactive CLI** with rich terminal output

## Commands

### Module Operations (`opm module`)

`opm mod` remains available as a compatibility alias.

Use `opm module` when you are starting from module source. For rendering, deploying, or inspecting instances, use `opm instance`.

| Command | Description |
|---------|-------------|
| `module init` | Create a new module from a template |
| `module vet` | Validate a module without rendering manifests |

### Instance Operations (`opm instance`)

<!-- Renamed from `opm release` / `opm rel` (enhancement 0002 D6). The old `release`/`rel` verb is removed — no back-compat alias (D8). -->

`opm inst` is the short alias.

Use `opm instance` when you are starting from an instance file or when you want to inspect, list, or delete deployed instances.

| Command | Description |
|---------|-------------|
| `instance vet` | Validate an instance file without generating manifests |
| `instance build` | Render an instance file to manifests |
| `instance apply` | Deploy an instance file to a cluster |
| `instance diff` | Compare an instance file with live cluster state |
| `instance status` | Show resource status for a deployed instance |
| `instance tree` | Show instance resource hierarchy |
| `instance delete` | Delete instance resources from a cluster |
| `instance list` | List deployed instances |
| `instance events` | Show events for an instance |

### Configuration (`opm config`)

| Command | Description |
|---------|-------------|
| `config init` | Initialize OPM configuration |
| `config vet` | Validate configuration |

## Example Instance Workflow

```bash
# Validate an instance file
opm instance vet ./releases/jellyfin/release.cue

# Render manifests from an instance file
opm instance build ./releases/jellyfin/release.cue

# Apply an instance file to the cluster
opm instance apply ./releases/jellyfin/release.cue

# Inspect deployed state by file, name, or UUID
opm instance status ./releases/jellyfin/release.cue
opm instance status jellyfin -n media
```

## Documentation

For development guidelines, architecture details, and agent instructions, see `AGENTS.md`.

## Build And Test

```bash
# Run all checks (format, vet, lint, test)
task check

# Build binary
task build

# Install binary
task install

# Run tests
task test

# Run tests with coverage
task test:coverage
```

## Requirements

- Go 1.25+
- Kubernetes cluster for deployment and integration-test workflows

## License

This project is licensed under the Apache License 2.0. See `LICENSE`.
