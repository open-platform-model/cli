# AGENTS.md - OPM CLI repository guide

## Purpose

This repository contains the OPM CLI, the command-line interface for Open
Platform Model workflows. Its purpose is to let users build, validate,
render, deploy, and inspect portable application releases defined with CUE,
with a strong focus on type safety, clear command behavior, and Kubernetes-
oriented workflows.

Primary stack: `cobra`, CUE Go SDK, Kubernetes `client-go`, `charmbracelet/log`, `testify`.

This file is for coding agents working in `cli/`.

## Repository Rules

- Keep changes small and independently verifiable; `CONSTITUTION.md` explicitly prefers tiny batches.
- Prefer updating existing packages over introducing new abstractions unless duplication or coupling justifies it.

## Entrypoint

Read these documents when entering `cli/`:

- `CONSTITUTION.md` - repository principles and change-shaping constraints. Read `CONSTITUTION.md` for the full list of design principles.
- `AGENTS.md` - repository-specific implementation guidance, commands, and package map.
- `README.md` - product purpose, command groups, and user-facing workflows.
- `docs/STYLE.md` - documentation prose style rules for this repo.

## Repository Layout

- `adr/` - Architecture Decision Records
- `cmd/opm/` - CLI entrypoint and root command wiring.
- `internal/cmd/` - Cobra command implementations.
- `internal/cmdutil/` - shared flags, annotations, and command-facing helpers.
- `internal/config/` - config resolution, schema validation, defaults.
- `internal/kubernetes/` - cluster operations, status, apply, delete, events.
- `internal/output/` - terminal formatting, log output, tables, manifests.
- `internal/releasefile/` - release file detection and loading.
- `internal/workflow/` - shared render/apply/query orchestration.
- `pkg/loader/` - CUE loading for modules, providers, and releases.
- `pkg/render/` - render pipeline logic.
- `pkg/errors/` - shared structured errors; alias this import as `oerrors`.
- `tests/integration/` - integration programs run with `go run`.
- `tests/e2e/` - end-to-end Go tests.

## Architecture Decision Records

ADRs capture significant technical decisions with their context and consequences.

- Location: `adr/`
- Template: `adr/TEMPLATE.md`
- Naming: `NNN-kebab-case-title.md` (three-digit, zero-padded)

### Creating a new ADR

1. Copy `adr/TEMPLATE.md` to `adr/NNN-title.md` using the next available number.
2. Set status to `Proposed`.
3. Fill in Context, Decision, and Consequences.
4. Update status to `Accepted` once the decision is agreed on.

### Updating an ADR

- Never delete an ADR — update its status instead.
- To retire a decision: set status to `Deprecated`.
- To replace a decision: set status to `Superseded by ADR-NNN` and create the new ADR.
- One decision per ADR.

## Environment Notes

- Go version in `go.mod`: `1.25.0`.
- Integration and some CUE workflows rely on registry configuration.
- Useful defaults during local development:

```bash
export OPM_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
```

## Build And Dev Commands

### Core commands

- `task build` - build `./bin/opm` from `./cmd/opm` with version ldflags.
- `task build:all` - cross-compile for Linux, macOS, and Windows.
- `task install` - install the CLI with version ldflags into `$GOPATH/bin`.
- `task clean` - remove `bin/`, `coverage.out`, and `coverage.html`.
- `task generate` - run `go generate ./...`.

### Formatting and static analysis

- `task fmt` - run `go fmt ./...` and `goimports -w .`.
- `task vet` - run `go vet ./...`.
- `task lint` - run `golangci-lint run ./...`.
- `task lint:fix` - run `golangci-lint run --fix ./...`.
- `task tidy` - run `go mod tidy`.
- `task check` - run `fmt`, `vet`, `lint`, and all tests.

### Tests

- `task test` - run unit, integration, and e2e suites.
- `task test:unit` - run `go test ./internal/...` and `go test ./pkg/...`.
- `task test:integration` - run integration programs; requires a live kind cluster.
- `task test:e2e` - run `go test ./tests/e2e/... -v`.
- `task test:verbose` - run `go test -v ./...`.
- `task test:coverage` - run `go test -coverprofile=coverage.out ./...` then generate `coverage.html`.

### Running one test

- Preferred repo task: `task test:run TEST=TestName`.
- Exact implementation: `go test -v ./... -run "TestName"`.
- Narrow to one package when possible for speed, for example:
  - `go test ./internal/config -run TestLoad -v`
  - `go test ./pkg/render -run TestFinalize -v`
- For a single subtest, use the full regexp name accepted by `go test -run`.

### Integration cluster helpers

- `task cluster:create` - create local `kind` cluster `opm-dev`.
- `task cluster:status` - check whether the cluster is running.
- `task cluster:delete` - remove the local cluster.
- `task cluster:recreate` - recreate the cluster from scratch.
- `task test:integration` checks for context `kind-opm-dev` before it runs.

## Coding Standards

### General

- Follow `gofmt` and `goimports`; do not hand-format imports.
- Keep command packages thin; orchestration belongs in `internal/workflow` or focused internal/pkg packages.
- Prefer explicit behavior over magic inference; `CONSTITUTION.md` favors clear inputs and early validation.
- Preserve cross-platform behavior; avoid hardcoded Unix-only paths or shell assumptions.

### Imports

- Group imports in standard Go order: stdlib, third-party, internal project imports.
- Use blank lines between groups exactly as `goimports` produces them.
- Alias `github.com/opmodel/cli/pkg/errors` as `oerrors` when imported.
- Avoid unnecessary aliases for other packages unless there is a collision or a strong clarity reason.

### Types and APIs

- Prefer concrete structs as return values; accept interfaces at boundaries when they improve testability.
- Avoid `interface{}` / `any` unless the API genuinely requires open-ended data.
- Keep config, flags, and render inputs strongly typed.
- Propagate `context.Context` through APIs that perform I/O, Kubernetes calls, or longer workflows.
- This repo uses fresh CUE contexts per command/workflow rather than sharing one global mutable context.

### Naming

- Exported identifiers use Go's standard PascalCase; unexported identifiers use camelCase.
- Use descriptive names tied to the domain: `ReleaseSelectorFlags`, `ResolveModulePath`, `BootstrapRegistry`.
- Boolean helpers should read naturally, for example `HasWarnings` or `configHasProviders`.
- Error sentinel names should follow Go conventions; the linter enables `errname` and revive naming rules.
- Keep package names short, lowercase, and responsibility-focused.

### Error handling

- Validate early and fail before execution when flags, config, or inputs are invalid.
- Wrap errors with context using `%w`, for example `fmt.Errorf("loading module: %w", err)`.
- Prefer actionable user-facing errors with hints over raw internal failures.
- Reuse structured error types in `pkg/errors` where appropriate, especially `DetailError` and validation helpers.
- Commands should use `RunE` and return errors rather than printing-and-exiting inline.
- Preserve sentinel errors with wrapping so callers can use `errors.Is` / `errors.As`.

### Control flow and package boundaries

- Commands parse flags and delegate; they should not hold core business logic.
- `internal/` packages may depend on `pkg/`; `pkg/` must stay reusable and command-agnostic.
- Keep output formatting separate from data generation.
- Prefer small functions with clear responsibilities over large multipurpose helpers.

### Tests

- Prefer table-driven tests when covering multiple scenarios.
- Use `require` for setup and fatal preconditions; use `assert` for non-fatal expectations.
- Use `t.Helper()` in test helpers.
- Prefer `t.TempDir()` over manual fixture directories when practical.
- Name tests `TestXxx` with behavior-oriented suffixes, matching existing patterns like `TestRenderFromReleaseFile_NilConfig`.

## Lint Configuration Highlights

- `golangci-lint` runs in readonly module download mode.
- Important enabled linters include `errorlint`, `errname`, `gocritic`, `gocyclo`, `gosec`, `revive`, `staticcheck`, and `tparallel`.
- `nolint` comments must be specific and include an explanation.
- `gocyclo` threshold is 15; refactor large branches before complexity grows.
- Tests have some relaxed exclusions for `dupl`, `errcheck`, `goconst`, and `gosec`.
- `examples/`, `experiments/`, `third_party/`, and `builtin/` are excluded from lint/format scopes.

## Documentation And Output Conventions

- Favor ASCII-safe output in docs, examples, and terminal-oriented text.
- For box-drawing tables or diagrams, use `[x]` / `[ ]` instead of Unicode checkmarks.
- When documenting CLI behavior, emphasize what happened and how to fix failures.
- Follow SemVer and Conventional Commits for user-visible changes and commit messages.

## Agent Checklist

- Read the touched package and nearby tests before editing.
- Run targeted tests first, then broader checks if the change warrants it.
- If you changed formatting-sensitive files, run `task fmt`.
- If you changed behavior, run the smallest relevant `go test` command plus any affected task.
- Before finishing substantial work, prefer `task lint` and the relevant test suite.
