## Context

The CLI has no CI/CD infrastructure. Testing is manual via `task test`, and releases are ad hoc. The Taskfile already defines all test targets and cross-compilation, which workflows delegate to where practical.

Three distinct test tiers exist, each with different infrastructure requirements:

```text
┌─────────────────────────────────────────────────────────────┐
│ Test Tier       │ Command                        │ Requires │
├─────────────────┼────────────────────────────────┼──────────┤
│ unit            │ go test ./internal/...         │ Go       │
│ registry        │ go test ./internal/builder/... │ Go, CUE  │
│                 │                                │ registry │
│ integration     │ go run tests/integration/      │ Go, kind │
│                 │   */main.go   (5 scripts)      │ cluster  │
│ e2e             │ go test ./tests/e2e/... -v     │ Go       │
└─────────────────┴────────────────────────────────┴──────────┘
```

Lint tooling: `golangci-lint run ./...` (config in `.golangci.yml`).

A self-hosted runner will be provided later. All workflows are pre-wired but disabled until then.

## Goals / Non-Goals

**Goals:**

- Automated lint + unit tests on every push
- Full test suite gating PRs (lint, unit, registry, integration, e2e)
- Goreleaser-based release pipeline on `v*` tags producing raw binaries + checksums + changelog
- All workflows pre-wired but initially disabled (self-hosted runner not yet available)

**Non-Goals:**

- Container image publishing
- Homebrew tap (can be added to goreleaser config later)
- GitHub Pages or docs publishing
- Go module caching (nice-to-have, not v1)
- Windows arm64 build target (not in existing `build:all`)

## Workflow Structure

### File Layout

```text
.github/workflows/
  ci.yml           # push to any branch → lint + unit
  pr.yml           # pull request to main → lint + unit + registry + integration + e2e
  release.yml      # v* tag push → full tests → goreleaser release
.goreleaser.yml    # goreleaser configuration
```

### ci.yml — Commit Checks

```text
Trigger: push to any branch (disabled: workflow_dispatch only)

┌──────────────────────────────────────────────┐
│ ci.yml                                       │
│                                              │
│   ┌────────────┐     ┌────────────┐          │
│   │    lint    │     │    unit    │          │
│   │            │     │            │          │
│   │ checkout   │     │ checkout   │          │
│   │ setup-go   │     │ setup-go   │          │
│   │ golangci-  │     │ go test    │          │
│   │   lint     │     │ ./internal │          │
│   └────────────┘     └────────────┘          │
│       (parallel, no dependencies)            │
└──────────────────────────────────────────────┘
```

**lint job steps:**

1. `actions/checkout@v4`
2. `actions/setup-go@v5` with `go-version: '1.25.0'`
3. `golangci-lint run ./...`

**unit job steps:**

1. `actions/checkout@v4`
2. `actions/setup-go@v5` with `go-version: '1.25.0'`
3. `go test ./internal/...`

### pr.yml — Pull Request Checks

```text
Trigger: pull_request targeting main (disabled: workflow_dispatch only)

┌──────────────────────────────────────────────────────────────────┐
│ pr.yml                                                           │
│                                                                  │
│  ┌──────┐  ┌──────┐  ┌──────────┐  ┌─────────────┐  ┌─────┐      │
│  │ lint │  │ unit │  │ registry │  │ integration │  │ e2e │      │
│  └──────┘  └──────┘  └──────────┘  └─────────────┘  └─────┘      │
│      (all 5 jobs parallel, no inter-job dependencies)            │
└──────────────────────────────────────────────────────────────────┘
```

**lint job:** Same as ci.yml.

**unit job:** Same as ci.yml.

**registry job steps:**

1. `actions/checkout@v4`
2. `actions/setup-go@v5` with `go-version: '1.25.0'`
3. `go test ./internal/builder/... -v`
   - env: `OPM_REGISTRY: 'opmodel.dev=localhost:5000+insecure,registry.cue.works'`

The `OPM_REGISTRY` value matches the Taskfile default. In CI, `localhost:5000` will be unreachable and CUE falls through to `registry.cue.works`.

**integration job steps:**

1. `actions/checkout@v4`
2. `actions/setup-go@v5` with `go-version: '1.25.0'`
3. Install kind (download binary or use a setup action)
4. `kind create cluster --name opm-dev --config hack/kind-config.yaml --image kindest/node:v1.34.0`
5. Run integration tests sequentially:

   ```text
   go run tests/integration/deploy/main.go
   go run tests/integration/inventory-apply/main.go
   go run tests/integration/inventory-ops/main.go
   go run tests/integration/mod-list/main.go
   go run tests/integration/values-flow/main.go
   ```

   - env: `OPM_REGISTRY: 'opmodel.dev=localhost:5000+insecure,registry.cue.works'`
6. Cleanup: `kind delete cluster --name opm-dev` — runs via `if: always()` so the cluster is deleted even when tests fail

The integration tests use `go run` (not `go test`) with `//go:build ignore` tags. Each is a standalone program that exits non-zero on failure. They must run sequentially because they share the kind cluster.

**e2e job steps:**

1. `actions/checkout@v4`
2. `actions/setup-go@v5` with `go-version: '1.25.0'`
3. `go test ./tests/e2e/... -v`

E2e tests build the `opm` binary internally via `TestMain` and shell out to it. No cluster or registry needed.

### release.yml — Tag Release

```text
Trigger: push tags v* (disabled: workflow_dispatch only)

┌───────────────────────────────────────────────────────────┐
│ release.yml                                               │
│                                                           │
│   ┌────────────────────────────────────────────────────┐  │
│   │                    test                            │  │
│   │                                                    │  │
│   │  lint → unit → registry → integration → e2e        │  │
│   │  (sequential steps within one job)                 │  │
│   └────────────────────────┬───────────────────────────┘  │
│                            │                              │
│                      needs: [test]                        │
│                            │                              │
│                            ▼                              │
│   ┌────────────────────────────────────────────────────┐  │
│   │                  release                           │  │
│   │                                                    │  │
│   │  checkout (fetch-depth: 0)                         │  │
│   │  setup-go                                          │  │
│   │  goreleaser release                                │  │
│   └────────────────────────────────────────────────────┘  │
│                                                           │
└───────────────────────────────────────────────────────────┘
```

The `test` job in release.yml combines all test tiers into a single job with sequential steps (rather than 5 parallel jobs). This is intentional: for releases, we want a clean gate — all tests pass, then release. Parallelism is less important than simplicity in the release path.

**test job steps:**

1. `actions/checkout@v4`
2. `actions/setup-go@v5` with `go-version: '1.25.0'`
3. `golangci-lint run ./...`
4. `go test ./internal/...`
5. `go test ./internal/builder/... -v` (env: `OPM_REGISTRY`)
6. Install kind
7. `kind create cluster --name opm-dev --config hack/kind-config.yaml --image kindest/node:v1.34.0`
8. Run all 5 integration test scripts (env: `OPM_REGISTRY`)
9. `kind delete cluster --name opm-dev` (if: always)
10. `go test ./tests/e2e/... -v`

**release job steps:**

1. `actions/checkout@v4` with `fetch-depth: 0` (full history for changelog)
2. `actions/setup-go@v5` with `go-version: '1.25.0'`
3. `goreleaser/goreleaser-action@v6` with `args: release --clean`
   - env: `GITHUB_TOKEN: ${{ secrets.GITHUB_TOKEN }}`

`fetch-depth: 0` is critical — goreleaser needs the full git history to compare tags and generate the changelog.

## Goreleaser Configuration

```yaml
# .goreleaser.yml
version: 2
project_name: opm

builds:
  - main: ./cmd/opm
    binary: opm
    ldflags:
      - -X github.com/opmodel/cli/internal/version.Version={{ .Version }}
      - -X github.com/opmodel/cli/internal/version.GitCommit={{ .Commit }}
      - -X github.com/opmodel/cli/internal/version.BuildDate={{ .Date }}
    goos: [linux, darwin, windows]
    goarch: [amd64, arm64]
    ignore:
      - goos: windows
        goarch: arm64

archives:
  - format: binary
    name_template: >-
      {{ .ProjectName }}-{{ .Os }}-{{ .Arch }}

checksum:
  name_template: checksums.txt

changelog:
  sort: asc
  groups:
    - title: Features
      regexp: '^.*?feat(\(.+\))?!?:.+$'
    - title: Bug Fixes
      regexp: '^.*?fix(\(.+\))?!?:.+$'
    - title: Performance
      regexp: '^.*?perf(\(.+\))?!?:.+$'
    - title: Refactoring
      regexp: '^.*?refactor(\(.+\))?!?:.+$'
    - title: Other
      order: 999
  filters:
    exclude:
      - '^docs:'
      - '^test:'
      - '^ci:'
      - '^chore:'
```

**Build matrix** (5 targets, matching existing `build:all`):

```text
┌─────────┬───────┬──────────────────────────────────┐
│ OS      │ Arch  │ Output filename                  │
├─────────┼───────┼──────────────────────────────────┤
│ linux   │ amd64 │ opm-linux-amd64                  │
│ linux   │ arm64 │ opm-linux-arm64                  │
│ darwin  │ amd64 │ opm-darwin-amd64                 │
│ darwin  │ arm64 │ opm-darwin-arm64                 │
│ windows │ amd64 │ opm-windows-amd64.exe            │
└─────────┴───────┴──────────────────────────────────┘
```

**ldflags mapping** (goreleaser template variables → Go variables):

```text
┌──────────────────┬──────────────────────────────────────────────────────────┐
│ Goreleaser var   │ Go variable (internal/version)                           │
├──────────────────┼──────────────────────────────────────────────────────────┤
│ {{ .Version }}   │ version.Version     (tag without 'v' prefix)             │
│ {{ .Commit }}    │ version.GitCommit   (full commit SHA)                    │
│ {{ .Date }}      │ version.BuildDate   (RFC 3339 timestamp)                 │
└──────────────────┴──────────────────────────────────────────────────────────┘
```

**Changelog groups** map directly to the project's conventional commit types defined in AGENTS.md. The `Other` catch-all captures anything that passes the exclude filter but doesn't match a named group.

## Disablement and Activation

All three workflows follow the same pattern:

```yaml
on:
  workflow_dispatch:  # Manual trigger for testing
  # Uncomment when self-hosted runner is available:
  # push:
  #   branches: ['**']
```

**To activate** (when the self-hosted runner is registered):

1. Uncomment the real trigger block in each file
2. Keep `workflow_dispatch` alongside for manual runs
3. No other changes needed — `runs-on: self-hosted` is already set

## Environment Variables and Secrets

```text
┌──────────────────┬─────────────┬───────────────────────────────────────────┐
│ Variable         │ Used in     │ Value                                     │
├──────────────────┼─────────────┼───────────────────────────────────────────┤
│ OPM_REGISTRY     │ pr, release │ opmodel.dev=localhost:5000+insecure,      │
│                  │             │ registry.cue.works                        │
│ CUE_REGISTRY     │ pr, release │ Same as OPM_REGISTRY (set both)           │
│ GITHUB_TOKEN     │ release     │ Automatic via secrets.GITHUB_TOKEN        │
└──────────────────┴─────────────┴───────────────────────────────────────────┘
```

`GITHUB_TOKEN` is provided automatically by GitHub Actions. No additional secrets need to be configured.

`OPM_REGISTRY` and `CUE_REGISTRY` should both be set — the CUE SDK reads `CUE_REGISTRY` while some project code reads `OPM_REGISTRY`.

## Self-Hosted Runner Requirements

The runner must have the following available (either pre-installed or via setup actions in workflows):

```text
┌──────────────────┬────────────┬─────────────────────────────────────────┐
│ Tool             │ Used by    │ Notes                                   │
├──────────────────┼────────────┼─────────────────────────────────────────┤
│ Go 1.25+         │ all        │ Or installed via actions/setup-go       │
│ golangci-lint    │ lint jobs  │ Must match .golangci.yml expectations   │
│ kind             │ integration│ For ephemeral cluster creation          │
│ kubectl          │ integration│ Usually bundled with kind               │
│ Docker           │ integration│ Required by kind for node containers    │
│ goreleaser       │ release    │ Or via goreleaser/goreleaser-action     │
│ git              │ all        │ Full clone support for changelog        │
└──────────────────┴────────────┴─────────────────────────────────────────┘
```

## Decisions

### Three workflow files over two

Split `ci.yml` (commits), `pr.yml` (PRs), `release.yml` (tags) into separate files rather than a single file with conditional job inclusion. Each file is a single readable concern. The slight duplication of job definitions is acceptable — the workflows serve different purposes and will diverge over time.

Alternative considered: One workflow with `if: github.event_name == 'pull_request'` guards. Rejected: harder to read, conditional chains become unwieldy.

### workflow_dispatch-only triggers while disabled

Real triggers are commented out. Only `workflow_dispatch` is active. This keeps the workflow file accurate (the real triggers are visible and documented) while preventing accidental runs. Uncomment triggers when the self-hosted runner is registered.

Alternative considered: `if: false` on all jobs. Rejected: requires editing every job to re-enable, more fragile.

### Goreleaser for release artifacts

Use goreleaser rather than scripting `go build` + `gh release create` manually. Goreleaser handles: cross-compilation, checksums, archive/binary naming, changelog from git log, GitHub Release creation — all declaratively.

Alternative considered: Manual `go build` + shell scripting. Rejected: more code to maintain, checksums and changelog require non-trivial scripting.

### Raw binaries, not tarballs

`archives.format: binary` in goreleaser. Users download the binary and run it. No `tar -xzf` step.

### Full test re-run on release

The release workflow re-runs all tests before goreleaser runs. A tag could be pushed from any state, so we never trust "tests passed on the PR" as sufficient.

### Release test job is sequential, not parallel

The release workflow runs all test tiers in a single job with sequential steps, unlike pr.yml which uses 5 parallel jobs. For releases, a clean sequential gate is simpler and the time cost is acceptable (releases are infrequent).

### Integration tests use ephemeral kind cluster

PR and release workflows spin up a kind cluster inline. Each workflow run gets its own clean cluster. The cluster is deleted in an `if: always()` step even on failure. Cluster name is `opm-dev` matching `hack/kind-config.yaml` and the Taskfile.

### Registry tests on PRs and releases, not commits

`test:registry` runs in `pr.yml` and `release.yml` but NOT `ci.yml`. Commit runs stay fast (lint + unit only). Registry access adds a network dependency that slows feedback.

### Go version pinned in workflows

Hardcode `go-version: '1.25.0'` matching `go.mod`. Avoids accidental version drift if `setup-go` picks a different minor version.

### goreleaser/goreleaser-action for the release step

Use the official `goreleaser/goreleaser-action@v6` rather than installing goreleaser manually. The action handles version pinning and binary caching.

## Risks / Trade-offs

- [Integration test duration] kind cluster creation adds ~60-90s minimum to every PR → Acceptable; integration tests must run somewhere and this is self-contained.
- [Self-hosted runner availability] Workflows are inert until the runner is registered → Intentional; workflows are pre-wired and ready to activate.
- [Goreleaser version drift] The goreleaser-action version should be pinned to prevent surprise breakage → Pin to `@v6`.
- [Changelog quality] Auto-generated changelog depends on conventional commit discipline → The project already mandates conventional commits (AGENTS.md). Poor commit messages produce poor changelogs.
- [Registry access in CI] `test:registry` and integration tests require CUE registry reachable from the runner → Document runner requirements; fail clearly if unreachable.
- [kind + Docker on self-hosted] Integration tests need Docker running on the self-hosted runner for kind → This is a runner setup prerequisite, not a workflow concern.
