# OPM CLI

> **⚠️ UNDER HEAVY DEVELOPMENT** - This project is actively being developed and APIs may change frequently.

Command-line interface for the Open Platform Model (OPM). Build, deploy, and manage portable application modules with CUE-based definitions.

## Quick Start

```bash
# Build the CLI
task build

# Initialize a new module
./bin/opm module init ./my-module

# Apply a module
./bin/opm module apply ./my-module

# Validate a release file
./bin/opm release vet ./release.cue

# Apply a release file
./bin/opm release apply ./release.cue
```

## Features

- **Type-safe module definitions** using CUE
- **Kubernetes-native** resource management
- **Portable blueprints** across providers
- **OCI-based distribution** for modules
- **Interactive CLI** with rich terminal output

## Commands

### Module Operations (`opm module`)

`opm mod` remains available as a compatibility alias.

Use `opm module` when you are starting from module source. For inspecting or operating on deployed releases, use `opm release`.

| Command | Description |
|---------|-------------|
| `module init` | Create a new module from template |
| `module build` | Render module to manifests |
| `module vet` | Validate module without generating manifests |
| `module apply` | Deploy module to cluster |

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
| `release tree` | Show release resource hierarchy (component → resource → K8s children) |
| `release delete` | Delete release resources from cluster |
| `release list` | List deployed releases |
| `release events` | Show events for a release |

### Configuration (`opm config`)

| Command | Description |
|---------|-------------|
| `config init` | Initialize OPM configuration |
| `config vet` | Validate configuration |

### `opm release tree`

Show the component and resource hierarchy of a deployed release.

```bash
# Full tree (default depth=2: components → resources → K8s children)
opm release tree my-app -n production

# Component summary only (depth=0)
opm release tree my-app -n production --depth 0

# Resources without K8s children (depth=1)
opm release tree my-app -n production --depth 1

# JSON output for scripting
opm release tree my-app -n production -o json
```

Depth levels:

- **0** — Component summary (resource counts and aggregate status)
- **1** — OPM-managed resources grouped by component
- **2** — Full tree with Kubernetes-owned children (Deployment→ReplicaSet→Pod, StatefulSet→Pod)

### Release Workflow

Use `opm release` when you are starting from a release definition instead of module source.

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

Legacy compatibility note: `opm module status`, `opm module tree`, `opm module delete`, `opm module list`, and `opm module events` still exist, but `opm release` is the preferred interface for deployed release operations.

## Documentation

For development guidelines, architecture details, and agent instructions, see [AGENTS.md](./AGENTS.md).

## Build & Test

```bash
# Run all checks (format, vet, test)
task check

# Build binary
task build

# Install binary
task install

# Run tests
task test

# Run with coverage
task test:coverage
```

## Requirements

- Go 1.21+
- CUE tools (integrated via cuelang.org/go)
- Kubernetes cluster (for deployment operations)

## License

See LICENSE file for details.
