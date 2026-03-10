# AGENTS.md - CLI Repository (Go Implementation)

> **⚠️ UNDER HEAVY DEVELOPMENT** - This project is actively being developed and APIs may change frequently.

## Overview

Go CLI for OPM module operations. Uses cobra, CUE SDK, and charmbracelet/log.

## Related folders

- catalog - catalog of definitions: `../catalog/`
- v1alpha1 definitions: `../catalog/v1alpha1/`
- v1alpha1 definition INDEX.md - Useful to get a quick overview: `../catalog/v1alpha1/INDEX.md`
- opm - main repo and docs: `../opm/`

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

## Commands

### Build

- `task build` — build binary (output: `./bin/opm`)
- `task build:all` — cross-compile for all platforms (linux, darwin, windows)
- `task install` — install to `$GOPATH/bin`
- `task clean` — remove build artifacts and coverage files

### Testing

- `task test` — run all tests (unit + integration + e2e)
- `task test:unit` — unit tests only (`./internal/...`)
- `task test:integration` — integration tests (requires `kind-opm-dev` cluster; run `task cluster:create` first)
- `task test:e2e` — end-to-end tests
- `task test:run TEST=TestName` — run a single test by name across all packages
- `task test:verbose` — all tests with verbose output
- `task test:coverage` — generate HTML coverage report

### Code Quality

- `task fmt` — format code (`go fmt` + `goimports`)
- `task vet` — run `go vet`
- `task lint` — run `golangci-lint`
- `task lint:fix` — run linter with auto-fix
- `task tidy` — run `go mod tidy`
- `task check` — run all checks (fmt + vet + lint + test)

### Cluster (for integration tests)

- `task cluster:create` — create local `kind` cluster (`kind-opm-dev`)
- `task cluster:delete` — delete local `kind` cluster
- `task cluster:status` — show cluster status
- `task cluster:recreate` — delete and recreate cluster

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

### Diff & Comparison

- homeport/dyff: Human-readable semantic diffs

### Testing

- stretchr/testify: Assertions and mocking

## Code Style

- **Go**: gofmt, golangci-lint compliant
- **Imports**: stdlib first, then external, then internal
- **Errors**: Wrap with context (`fmt.Errorf("loading module: %w", err)`)
- **Interfaces**: Accept interfaces, return concrete structs
- **Context**: Propagate context.Context in all APIs
- **Tests**: Table-driven with testify assertions

## Project Structure

```text
├── cmd/opm/               # CLI entry point (main.go, root.go, version.go)
├── docs/                  # Design docs, RFCs, comparisons, vision
├── examples/              # Example OPM modules (jellyfin, minecraft, webapp, etc.)
├── experiments/           # Experimental pipeline explorations (not production)
├── internal/
│   ├── cmd/               # Command implementations
│   │   ├── config/        # config init, vet commands
│   │   ├── module/        # module/apply, build, delete, init, list, status, tree, vet
│   │   └── release/       # release/apply, build, delete, diff, events, list, status, tree, vet
│   ├── cmdutil/           # Small CLI helpers (flag groups, annotations, release targeting)
│   ├── config/            # Config loading and validation
│   ├── inventory/         # Release inventory: secret CRUD, digest, stale detection
│   ├── kubernetes/        # K8s client, apply, delete, discovery, health, status
│   ├── output/            # Terminal output: spinner, table, log, styles, format
│   ├── releasefile/       # Release file parsing and kind detection
│   ├── templates/         # Module templates (simple, standard, advanced)
│   ├── version/           # Version info
│   └── workflow/          # Shared command workflows (render, apply, query)
├── pkg/
│   ├── bundle/            # Bundle domain types
│   ├── bundlerelease/     # BundleRelease domain types
│   ├── core/              # Rendered resource primitives and shared helpers
│   ├── engine/            # Render execution and bundle/module renderers
│   ├── errors/            # Shared structured errors and exit wrappers
│   ├── loader/            # CUE loading for modules, providers, and releases
│   ├── match/             # Component-to-transformer matching
│   ├── module/            # Module domain model
│   ├── modulerelease/     # ModuleRelease domain model
│   ├── provider/          # Provider domain model
│   └── releaseprocess/    # Validation, synthesis, and render pipeline orchestration
├── tests/
│   ├── e2e/               # End-to-end tests
│   ├── fixtures/
│   │   ├── valid/         # Valid test module fixtures
│   │   └── invalid/       # Invalid test module fixtures
│   └── integration/
│       ├── deploy/
│       ├── inventory-apply/
│       ├── inventory-ops/
│       └── values-flow/
├── Taskfile.yml           # Build automation
└── go.mod
```

## Maintenance Notes

- **Project Structure Tree**: Update the tree above when adding new packages or directories.

## Key Packages

- `cmd/opm/` - Entry point
- `internal/cmd/` - Cobra command implementations (`config/`, `module/`, `release/`)
- `internal/cmdutil/` - Small CLI helpers shared across commands
- `internal/workflow/` - Shared application workflows used by multiple commands
- `internal/config/` - Config loading, precedence resolution, and validation
- `internal/inventory/` - Inventory persistence, change tracking, stale detection
- `internal/kubernetes/` - Kubernetes client and cluster operations
- `internal/output/` - Terminal formatting and log styling
- `pkg/loader/` - CUE loading for modules, providers, and release files
- `pkg/releaseprocess/` - Value validation, synthesis, and render pipeline orchestration
- `pkg/match/` - Component-to-transformer matching
- `pkg/engine/` - Module and bundle render execution
- `pkg/core/` - Shared rendered resource primitives
- `pkg/module/` - Module domain model
- `pkg/modulerelease/` - ModuleRelease domain model
- `pkg/provider/` - Provider domain model
- `internal/version/` - Version info

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
