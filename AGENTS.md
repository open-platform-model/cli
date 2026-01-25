# AGENTS.md - CLI Repository (Go Implementation)

## Overview

Go CLI for OPM module operations. Uses cobra, CUE SDK, and zap logging.

## Build/Test Commands

- Build: `task build` (output: ./bin/opm)
- Test all: `task test`
- Test verbose: `task test:verbose`
- Single test: `go test ./pkg/loader -v -run TestName`
- Coverage: `task test:coverage`
- All checks: `task check` (fmt + vet + test)
- Run: `task run -- mod vet ./testdata/test-module`

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
│   │   ├── config/    # config init, config vet commands
│   │   └── mod/       # mod apply, build, delete, init, status, template, tidy, vet
│   ├── config/        # Config loading, validation, paths
│   ├── cue/           # CUE loader, renderer, manifest, values
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
