# OPM CLI

> **⚠️ UNDER HEAVY DEVELOPMENT** - This project is actively being developed and APIs may change frequently.

Command-line interface for the Open Platform Model (OPM). Build, deploy, and manage portable application modules with CUE-based definitions.

## Quick Start

```bash
# Build the CLI
task build

# Initialize a new module
./bin/opm mod init ./my-module

# Apply a module
./bin/opm mod apply ./my-module
```

## Features

- **Type-safe module definitions** using CUE
- **Kubernetes-native** resource management
- **Portable blueprints** across providers
- **OCI-based distribution** for modules
- **Interactive CLI** with rich terminal output

## Commands

### Module Operations (`opm mod`)

| Command | Description |
|---------|-------------|
| `mod init` | Create a new module from template |
| `mod build` | Render module to manifests |
| `mod vet` | Validate module without generating manifests |
| `mod apply` | Deploy module to cluster |
| `mod status` | Show resource status |
| `mod tree` | Show module resource hierarchy (component → resource → K8s children) |

| `mod delete` | Delete release resources from cluster |
| `mod list` | List deployed module releases |
| `mod events` | Show events for a release |

### Configuration (`opm config`)

| Command | Description |
|---------|-------------|
| `config init` | Initialize OPM configuration |
| `config vet` | Validate configuration |

### `opm mod tree`

Show the component and resource hierarchy of a deployed release.

```bash
# Full tree (default depth=2: components → resources → K8s children)
opm mod tree --release-name my-app -n production

# Component summary only (depth=0)
opm mod tree --release-name my-app -n production --depth 0

# Resources without K8s children (depth=1)
opm mod tree --release-name my-app -n production --depth 1

# JSON output for scripting
opm mod tree --release-name my-app -n production -o json
```

Depth levels:
- **0** — Component summary (resource counts and aggregate status)
- **1** — OPM-managed resources grouped by component
- **2** — Full tree with Kubernetes-owned children (Deployment→ReplicaSet→Pod, StatefulSet→Pod)

## Documentation

For development guidelines, architecture details, and agent instructions, see [AGENTS.md](./AGENTS.md).

## Build & Test

```bash
# Run all checks (format, vet, test)
task check

# Build binary
task build

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
