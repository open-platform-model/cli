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
- Single test: `go test ./internal/build -v -run TestName`
- Coverage: `task test:coverage`
- All checks: `task check` (fmt + vet + test)
- Run: `task run -- mod init ./my-module`

## OPM and CUE

Use these environment variables during development and validation. Commands like "cue mod tidy" or "cue vet ./..."

```bash
export OPM_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
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
│   ├── build/         # Render pipeline: public API + subpackages
│   │   ├── module/    # Module inspection: path resolution, AST metadata extraction
│   │   ├── release/   # Release building: CUE overlay, values, metadata, validation
│   │   └── transform/ # Transformer logic: provider loading, matching, execution
│   ├── cmd/           # Command implementations (root, version, exit)
│   │   ├── config/    # config init, vet commands
│   │   └── mod/       # mod apply, build, delete, diff, init, status, vet commands
│   ├── cmdutil/       # Shared command utilities (flag groups, render pipeline, K8s client, output)
│   ├── config/        # Config loading, validation, paths
│   ├── core/          # Shared domain types: Resource, labels, metadata, errors, weights
│   ├── errors/        # Error types and handling
│   ├── inventory/     # Release inventory: secret CRUD, digest, stale detection
│   ├── kubernetes/    # K8s client, apply, delete, discovery, health, status
│   ├── output/        # Spinner, table, log, styles, format
│   ├── templates/     # Module templates (simple, standard, advanced)
│   └── version/       # Version handling
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
- `internal/build/` - CUE module loading and render pipeline
- `internal/version/` - Version info
- `internal/cmd/` - Command implementations (root, mod/, config/)
- `internal/cmdutil/` - Shared command utilities (flag groups, render pipeline, K8s client, output)

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

## Documentation Style

### Box-Drawing Diagrams and ASCII Art

**Symbols for Yes/No in Tables and Diagrams**

When creating box-drawing tables or ASCII art diagrams in markdown code blocks, use **monospace-safe** symbols that render consistently across all terminals, editors, and GitHub.

**DO NOT USE** Unicode checkmarks (`✓` U+2713, `✗` U+2717) — these are ambiguous-width characters that break alignment in monospace fonts.

**Recommended Replacements:**

| Context | Yes | No | Example |
|---------|-----|-----|---------|
| **Box-drawing table cells** | `[x]` | `[ ]` | `│ No CRDs req. │  [x]   │  [ ]   │` |
| **Bullet-style property lists** | `[x]` | `[ ]` | `│    [x] Same resources → same digest` |
| **Inline after text** | `OK` | `FAIL` | `Apply: SS/jellyfin-media OK, Svc/jellyfin-media FAIL` |
| **Section headings** | `[x]` | `[ ]` | `### Scenario A: Normal Rename [x]` |
| **Parenthetical notes** | `ok` | `fail` | `Label check: "opm" (3 ok), name (≤63 ok)` |

**Rationale:**

1. **`[x]` / `[ ]`** - Checkbox-style brackets are exactly 3 ASCII characters wide, easy to align in tables
2. **`OK` / `FAIL`** - More readable mid-sentence than brackets
3. **`ok` / `fail`** - Lowercase variant for lightweight inline use

**Table Alignment Example:**

```text
┌──────────────┬────────┬────────┬────────┐
│ Feature      │ Secret │ CRD    │ DB     │
├──────────────┼────────┼────────┼────────┤
│ No CRDs req. │  [x]   │  [ ]   │  [x]   │  ← 3 chars each, properly aligned
│ Inventory    │  [x]   │  [x]   │  [x]   │
└──────────────┴────────┴────────┴────────┘
```

**Why This Matters:**

- Unicode `✓` renders as 1 cell in some fonts, 2 cells in others (especially CJK locales)
- Broken alignment makes diagrams unreadable in terminals
- GitHub code blocks don't always match terminal rendering
- ASCII/bracket combinations are universally safe
