# OPM CLI repository guide

## Purpose

OPM CLI — command-line interface for Open Platform Model workflows. Build, validate, render, deploy, inspect portable app releases defined with CUE. Focus: type safety, clear command behavior, Kubernetes-oriented workflows.

Stack: `cobra`, CUE Go SDK, Kubernetes `client-go`, `charmbracelet/log`, `testify`.

For coding agents working in `cli/`.

## Repository Rules

- Small, independently verifiable changes; `CONSTITUTION.md` prefers tiny batches.
- Update existing packages over new abstractions unless duplication/coupling justifies it.

## Entrypoint

Read when entering `cli/`:

- `CONSTITUTION.md` - repo principles + change-shaping constraints.
- `CLAUDE.md` - implementation guidance, commands, package map.
- `README.md` - product purpose, command groups, user workflows.
- `docs/STYLE.md` - doc prose style rules.

## Repository Layout

- `adr/` - Architecture Decision Records
- `cmd/opm/` - CLI entrypoint + root command wiring.
- `internal/cmd/` - Cobra command implementations.
- `internal/cmdutil/` - shared flags, annotations, command-facing helpers.
- `internal/config/` - config resolution, schema validation, defaults.
- `internal/kubernetes/` - cluster ops, status, apply, delete, events.
- `internal/output/` - terminal formatting, log output, tables, manifests.
- `internal/releasefile/` - release file detection + loading.
- `internal/workflow/` - shared render/apply/query orchestration.
- `pkg/loader/` - CUE loading for modules, providers, releases.
- `pkg/render/` - render pipeline logic.
- `pkg/errors/` - shared structured errors; alias as `oerrors`.
- `tests/integration/` - integration programs via `go run`.
- `tests/e2e/` - end-to-end Go tests.

## Environment Notes

- Go version in `go.mod`: `1.25.0`.
- Integration + CUE workflows need registry config.
- Local dev defaults:

```bash
export OPM_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
export CUE_REGISTRY='opmodel.dev=localhost:5000+insecure,registry.cue.works'
```

## Build And Dev Commands

### Core commands

- `task build` - build `./bin/opm` from `./cmd/opm` with version ldflags.
- `task build:all` - cross-compile for Linux, macOS, Windows.
- `task install` - install CLI with version ldflags into `$GOPATH/bin`.
- `task clean` - remove `bin/`, `coverage.out`, `coverage.html`.
- `task generate` - run `go generate ./...`.

### Formatting and static analysis

- `task fmt` - run `go fmt ./...` and `goimports -w .`.
- `task vet` - run `go vet ./...`.
- `task lint` - run `golangci-lint run ./...`.
- `task lint:fix` - run `golangci-lint run --fix ./...`.
- `task tidy` - run `go mod tidy`.
- `task check` - run `fmt`, `vet`, `lint`, all tests.

### Tests

- `task test` - run unit, integration, e2e suites.
- `task test:unit` - run `go test ./internal/...` and `go test ./pkg/...`.
- `task test:integration` - run integration programs; needs live kind cluster.
- `task test:e2e` - run `go test ./tests/e2e/... -v`.
- `task test:verbose` - run `go test -v ./...`.
- `task test:coverage` - run `go test -coverprofile=coverage.out ./...` then generate `coverage.html`.

### Running one test

- Preferred: `task test:run TEST=TestName`.
- Direct: `go test -v ./... -run "TestName"`.
- Narrow to one package for speed:
  - `go test ./internal/config -run TestLoad -v`
  - `go test ./pkg/render -run TestFinalize -v`
- Single subtest: use full regexp name via `go test -run`.

### Integration cluster helpers

- `task cluster:create` - create local `kind` cluster `opm-dev`.
- `task cluster:status` - check cluster running.
- `task cluster:delete` - remove local cluster.
- `task cluster:recreate` - recreate cluster from scratch.
- `task test:integration` checks for context `kind-opm-dev` before running.

## Coding Standards

### General

- Follow `gofmt` and `goimports`; no hand-formatting imports.
- Keep command packages thin; orchestration in `internal/workflow` or focused internal/pkg packages.
- Explicit behavior over magic inference; `CONSTITUTION.md` favors clear inputs + early validation.
- Preserve cross-platform behavior; no hardcoded Unix-only paths or shell assumptions.

### Imports

- Standard Go order: stdlib, third-party, internal project imports.
- Blank lines between groups as `goimports` produces.
- Alias `github.com/opmodel/cli/pkg/errors` as `oerrors`.
- No unnecessary aliases unless collision or strong clarity reason.

### Types and APIs

- Concrete structs as return values; interfaces at boundaries for testability.
- No `interface{}` / `any` unless API genuinely needs open-ended data.
- Config, flags, render inputs: strongly typed.
- Propagate `context.Context` through I/O, Kubernetes calls, longer workflows.
- Fresh CUE contexts per command/workflow, not one global mutable context.

### Naming

- Exported: PascalCase; unexported: camelCase.
- Descriptive domain names: `ReleaseSelectorFlags`, `ResolveModulePath`, `BootstrapRegistry`.
- Booleans read naturally: `HasWarnings`, `configHasProviders`.
- Error sentinels follow Go conventions; linter enables `errname` + revive naming rules.
- Package names: short, lowercase, responsibility-focused.

### Error handling

- Validate early, fail before execution on invalid flags/config/inputs.
- Wrap errors with context via `%w`: `fmt.Errorf("loading module: %w", err)`.
- Actionable user-facing errors with hints over raw internal failures.
- Reuse `pkg/errors` types, especially `DetailError` + validation helpers.
- Commands use `RunE`, return errors — no print-and-exit inline.
- Preserve sentinel errors via wrapping for `errors.Is` / `errors.As`.

### Control flow and package boundaries

- Commands parse flags + delegate; no core business logic.
- `internal/` depends on `pkg/`; `pkg/` stays reusable + command-agnostic.
- Output formatting separate from data generation.
- Small focused functions over large multipurpose helpers.

### Tests

- Table-driven tests for multiple scenarios.
- `require` for setup/fatal preconditions; `assert` for non-fatal expectations.
- `t.Helper()` in test helpers.
- `t.TempDir()` over manual fixture dirs when practical.
- Name `TestXxx` with behavior suffixes, e.g. `TestRenderFromReleaseFile_NilConfig`.

## Lint Configuration Highlights

- `golangci-lint` runs in readonly module download mode.
- Key linters: `errorlint`, `errname`, `gocritic`, `gocyclo`, `gosec`, `revive`, `staticcheck`, `tparallel`.
- `nolint` comments must be specific with explanation.
- `gocyclo` threshold: 15; refactor before complexity grows.
- Tests relax `dupl`, `errcheck`, `goconst`, `gosec`.
- `examples/`, `experiments/`, `third_party/`, `builtin/` excluded from lint/format.

## Documentation And Output Conventions

- ASCII-safe output in docs, examples, terminal text.
- Box-drawing: `[x]` / `[ ]` not Unicode checkmarks.
- CLI docs: emphasize what happened + how to fix failures.
- Follow SemVer + Conventional Commits for user-visible changes.

## Agent Checklist

- Read touched package + nearby tests before editing.
- Run targeted tests first, broader checks if warranted.
- Changed formatting files → run `task fmt`.
- Changed behavior → run smallest relevant `go test` + affected task.
- Before finishing substantial work → `task lint` + relevant test suite.
