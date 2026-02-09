# AGENTS.md - CLI Repository (Go Implementation)

> **⚠️ UNDER HEAVY DEVELOPMENT** - This project is actively being developed and APIs may change frequently.

## Overview

Go CLI for OPM module operations. Uses cobra, CUE SDK, and zap logging.

## Constitution

This project follows the **Open Platform Model Constitution**.
All agents MUST read and adhere to `openspec/config.yaml`.

**Core Principles:**

1. **Type Safety First**: All definitions in CUE. Validation at definition time.
2. **Separation of Concerns**: Module (Dev) -> ModuleRelease (Consumer). Clear ownership boundaries.
3. **Composability**: Definitions compose without implicit coupling. Resources, Traits, Blueprints are independent.
4. **Declarative Intent**: Express WHAT, not HOW. Provider-specific steps in ProviderDefinitions.
5. **Portability by Design**: Definitions must be runtime-agnostic.
6. **Semantic Versioning**: SemVer v2.0.0 and Conventional Commits v1 required.
7. **Simplicity & YAGNI**: Justify complexity. Prefer explicit over implicit.

**Governance**: The constitution supersedes this file in case of conflict.

## Build/Test Commands

- Build: `task build` (output: ./bin/opm)
- Test all: `task test`
- Test verbose: `task test:verbose`
- Single test: `go test ./pkg/loader -v -run TestName`
- Coverage: `task test:coverage`
- All checks: `task check` (fmt + vet + test)
- Run: `task run -- mod init ./my-module`

## OPM and CUE

Use these environment variables during development and validation. Commands like "cue mod tidy" or "cue vet ./..."

```bash
export CUE_REGISTRY=localhost:5000
export OPM_REGISTRY=localhost:5000
export CUE_CACHE_DIR=/var/home/emil/Dev/open-platform-model/.cue-cache
```

## Technology Standards

### CLI Framework & UX

- spf13/cobra: Commands, auto-generated help, shell completion
- charmbracelet/lipgloss: Terminal styling
- charmbracelet/log: Structured logging with key-value output
- charmbracelet/glamour: Markdown rendering in terminal
- charmbracelet/huh: Interactive prompts for init commands

### Configuration

- CUE-native config (~/.opm/config.cue) - NOT viper/yaml
- Aligns with Principle I: config validated by CUE schema at load time

### CUE Integration

- cuelang.org/go: Native CUE evaluation (no external cue binary)
- Fresh CUE context per command (avoid memory bloat)
- Directory-based module loading (not file paths)

### Kubernetes

- k8s.io/client-go: Server-side apply, resource discovery
- k8s.io/apimachinery: Unstructured types, version parsing

### Distribution

- oras.land/oras-go/v2: OCI push/pull for module distribution

### Diff & Comparison

- homeport/dyff: Human-readable semantic diffs

### Testing

- stretchr/testify: Assertions and mocking
- sigs.k8s.io/controller-runtime/pkg/envtest: K8s integration tests

## Code Style

- **Go**: gofmt, golangci-lint compliant
- **Imports**: stdlib first, then external, then internal
- **Errors**: Wrap with context (`fmt.Errorf("loading module: %w", err)`)
- **Interfaces**: Accept interfaces, return concrete structs
- **Context**: Propagate context.Context in all APIs
- **Tests**: Table-driven with testify assertions

## Project Structure

```text
├── cmd/opm/           # CLI entry point (main.go, root.go, version.go)
├── internal/
│   ├── cmd/           # Command implementations
│   │   ├── config/    # config init command
│   │   └── mod/       # mod apply, build, delete, init, status, template
│   ├── config/        # Config loading, validation, paths
│   ├── errors/        # Error types and handling
│   ├── kubernetes/    # K8s client, apply, delete, discovery, health, status
│   ├── output/        # Spinner, table, log, styles, format
│   ├── templates/     # Module templates (simple, standard, advanced)
│   └── version/       # Version handling
├── pkg/weights/       # Resource weights
├── tests/
│   ├── fixtures/      # Test module fixtures
│   └── integration/   # Integration tests
├── Taskfile.yml       # Build automation
└── go.mod
```

## Maintenance Notes

- **Project Structure Tree**: Update the tree above when adding new packages or directories.

## Key Packages

- `cmd/opm/` - Entry point
- `pkg/loader/` - CUE module loading
- `pkg/version/` - Version info
- `internal/commands/` - Command implementations (root, mod/, config/)

## Patterns

- Fresh CUE context per command (avoid memory bloat)
- Directory-based module loading (not file paths)
- Commands use `RunE` for error handling

## Glossary

See [full glossary](../opm/docs/glossary.md) for detailed definitions.

### Personas

- **Infrastructure Operator** - Operates underlying infrastructure (clusters, cloud, networking)
- **Module Author** - Develops and maintains ModuleDefinitions with sane defaults
- **Platform Operator** - Curates module catalog, bridges infrastructure and end-users
- **End-user** - Consumes modules via ModuleRelease with concrete values
