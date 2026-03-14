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

# Validate a release file
./bin/opm release vet ./release.cue

# Render a release file
./bin/opm release build ./release.cue

# Apply a release file
./bin/opm release apply ./release.cue
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

Use `opm module` when you are starting from module source. For rendering, deploying, or inspecting releases, use `opm release`.

| Command | Description |
|---------|-------------|
| `module init` | Create a new module from a template |
| `module vet` | Validate a module without rendering manifests |

### Release Operations (`opm release`)

`opm rel` remains available as a compatibility alias.

Use `opm release` when you are starting from a release file or when you want to inspect, list, or delete deployed releases.

| Command | Description |
|---------|-------------|
| `release vet` | Validate a release file without generating manifests |
| `release build` | Render a release file to manifests |
| `release apply` | Deploy a release file to a cluster |
| `release diff` | Compare a release file with live cluster state |
| `release status` | Show resource status for a deployed release |
| `release tree` | Show release resource hierarchy |
| `release delete` | Delete release resources from a cluster |
| `release list` | List deployed releases |
| `release events` | Show events for a release |

### Configuration (`opm config`)

| Command | Description |
|---------|-------------|
| `config init` | Initialize OPM configuration |
| `config vet` | Validate configuration |

## Example Release Workflow

```bash
# Validate a release file
opm release vet ./releases/jellyfin/release.cue

# Render manifests from a release file
opm release build ./releases/jellyfin/release.cue

# Apply a release file to the cluster
opm release apply ./releases/jellyfin/release.cue

# Inspect deployed state by file, name, or UUID
opm release status ./releases/jellyfin/release.cue
opm release status jellyfin -n media
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
