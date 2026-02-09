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
