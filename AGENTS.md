# OPM CLI repository guide

The `opm` command-line tool of Open Platform Model. It builds, validates, renders, deploys and
inspects portable app releases defined in CUE, with a focus on type safety, clear command behavior
and Kubernetes workflows. Stack: `cobra`, the CUE Go SDK, Kubernetes `client-go`,
`charmbracelet/log`, `testify`. Go version: see `go.mod`. `CONSTITUTION.md` holds the repo
principles; `docs/STYLE.md` extends the workspace prose style.

## Branch model

| Branch | What lands |
| --- | --- |
| `main` | All work, by PR. Releases are beta pre-releases (`prerelease-type: beta` in `release-please-config.json`; see `references/`). |

The remote has no `v1` or `release/*` branch (checked 2026-10-14).

## Repository rules

- The cli relies fully on the library. A frontend calls the kernel and nothing else: a gap is a
  library defect to fix upstream, never a CLI-side fork. Reason: structure belongs to the library,
  presentation to the frontend. The cli holds no built platform value and no matching or transformer
  execution; one `Kernel.Render` call renders.
- No local Kubernetes object, label, order, inventory or health package. These are the library's
  `opm/k8s/object`, `opm/k8s/labels` (import as `opmlabels`), `opm/k8s/inventory` (import as
  `k8sinventory`) and `opm/k8s/health` (import as `health`). A `depguard` rule in `.golangci.yml`
  refuses the retired `pkg/core`, `pkg/resourceorder` and `pkg/inventory` paths, and
  a refusal test in `internal/kubernetes/` fails on a local readiness evaluator.
- Read no error text. Read a registry fetch or dependency answer from the library's typed errors
  (`liberrors.Classify`, `*FetchError`, `*ResolutionError`), and a CUE failure from its error path
  (the error-path helper in `internal/config/`). A refusal test in `pkg/errors/` refuses a new text match in non-test code and lists the
  four that stay, each with the typed cause that is missing. It reads syntax, not types: it misses a
  message built by formatting the error (`fmt.Sprint(err)`) or passed on as a string. Pass the error,
  never its text.
- One deletion function in `internal/kubernetes/` deletes the objects of an instance, following the
  library's deletion plan. A test in that folder lists the only functions that may send a DELETE.
- `internal/cuemod` is the only site of `os.Chdir` and `os.Setenv`. `cmd/cue` reads the module root
  from the working directory and the registry from `CUE_REGISTRY`, so `Tidy` sets both under a
  package mutex and restores them on every return path.
- The operator pin file in `internal/operator/` is written only by `hack/operator-pin` (`task
  operator:pin VERSION=<module version>`). Do not hand-edit it.
- Ship changes as sections that each merge green with their own commit (`CONSTITUTION.md` has the
  principle). Prefer updating an existing package over a new abstraction, unless duplication or
  coupling justifies it.
- A comment names an enhancement only where the rationale lives there. In CUE the reference goes in a
  `// WHY` block, one blank line below the doc comment, because `cue lsp` hover, the doc-comment
  accessor of the CUE API and `cue def` replay a doc comment verbatim. Never cite an enhancement in scaffold templates,
  generated files, fixtures a user copies or CLI output strings: their readers have no access to the
  enhancements store. `task cite-lint` checks comments.

## Layout

```text
adr/                       Architecture Decision Records
cmd/opm/                   entrypoint and root command wiring
internal/cmd/              Cobra commands: module, catalog, instance, config, operator, registry, platform;
                           `module`/`catalog` carry `version set`; `module` carries `template list` and `init`;
                           `registry` carries `login`; `instance` carries `init`
internal/cmdutil/          shared flags, annotations, argument resolution, command-facing helpers
internal/compat/           compatibility comparison of contract surface (levels, provenance denylist)
internal/config/           config resolution, schema validation, defaults
internal/cueedit/          surgical authoring-file rewrites (Version, ModulePath, module path, re-identification)
internal/cuemod/           in-process `cue mod tidy [--check]`; `cuemodtest/` is a hermetic registry harness
internal/dockercfg/        read-modify-write of the docker credential file (`registry login`)
internal/exit/             exit codes: 0 ok, 1 general, 2 validation, 3 connectivity, 4 permission, 5 not found
internal/instinit/         the `opm instance init` package generator
internal/inventory/        instance inventory records, ownership, discovery and prune gates
internal/kubernetes/       cluster ops: status, apply, delete, events
internal/modref/           published-module resolution: path grammar, `--version` selector, core-compatible walk;
                           apart from `scaffold` because their template grammar and selector differ on purpose
internal/operator/         `opm operator install` and `uninstall`; its pin file is written by `hack/operator-pin`
internal/output/           terminal formatting, log output, tables, manifests
internal/platform/         platform-source resolution and the Platform seed for `operator install`
internal/publish/          identity-driven publish pipeline: gates, refusals, plan, registry push
internal/scaffold/         fetch-based `module init` engine: template refs, re-identification, repair
internal/version/          build version values
internal/workflow/         render, apply and query orchestration; `render` makes the one `Kernel.Render` call
pkg/errors/                shared structured errors; import as `oerrors`
pkg/loader/                local-replacement provenance (module root, local-module replacements)
templates/                 official template modules (`minimal`, `standard`, `advanced`)
hack/docskit-dump/         program docs-kit runs for the command reference; not linked into `opm`
hack/operator-pin/         writes the operator pin file; not linked into `opm`
hack/platform/             maintained platform module for offline tests
tests/fixtures/            fixture modules on `testing.opmodel.dev`; `tests/e2e/` end-to-end Go tests
tests/integration/         integration programs run with `go run`
docs/site/                 site pages opmodel.dev builds; `docs-kit.cue` is the docs bundle
openspec/                  specs, active changes and archive
.tasks/, .github/scripts/  cascade and CI shell tooling
```

## Commands

| Command | What it does | When |
| --- | --- | --- |
| `task build` / `task build:all` / `task install` | Build `bin/opm` / cross-compile / install with version ldflags | Run the tool |
| `task fmt` / `task vet` / `task tidy` / `task generate` | `go fmt` and `goimports` / `go vet` / `go mod tidy` / `go generate` | After a Go edit |
| `task lint` | `task lint:config`, then `golangci-lint run ./...` | Before you finish |
| `task lint:config` | Verify `.golangci.yml` against the committed schema, offline | After a lint config or version change |
| `task lint:fix` | `golangci-lint run --fix` | Lint findings that auto-fix |
| `task cite-lint` | Refuse a decision number or lone enhancement id in a comment | After a comment edit |
| `task openspec:check` | `openspec validate --all` | After a spec or change edit; `--strict` is off until the long-requirement rewrite lands |
| `task test` / `task test:unit` | All suites / `internal` and `pkg` | Default checks |
| `task test:run TEST=Name` | One test by name | While you work |
| `task test:e2e` / `task test:integration` | E2E / integration programs (need `kind-opm-dev`) | Cluster behavior |
| `task test:fixtures` | Reproduce the PR fixtures job locally | After a fixture change |
| `task cluster:create` / `cluster:operator` / `cluster:delete` | Make, prepare, remove the `opm-dev` kind cluster | Cluster tests |
| `task docs:bundle` / `docs:bundle:check` | Build / check the docs bundle | After a help or docs page change |
| `task cascade:wiring:check` | Offline cascade wiring check | After a workflow change |
| `task operator:pin VERSION=...` | Rewrite the operator pin file | Move the operator pin |
| `task -x deps:cascade` (`:title`, `:body`, `:test`) | Move upstream pins in the working tree; `-x` is required | Cascade work |
| `task check` | `fmt`, `vet`, `lint`, `cite-lint`, `openspec:check`, `docs:bundle:check`, all tests | Before you finish |

## Style

- Output and docs are ASCII-safe: write `[x]` and `[ ]`, not Unicode checkmarks. Say what happened and
  how to fix a failure.
- Alias `github.com/open-platform-model/cli/pkg/errors` as `oerrors`. Add no other alias without a
  collision or a clear gain.
- Keep command packages thin: parse flags, delegate to `internal/workflow` or a focused package.
  `internal/` depends on `pkg/`; `pkg/` stays command-agnostic.
- Use a fresh CUE context per command, never one global.
- Reuse `pkg/errors` types (`DetailError`) and keep sentinels matchable with `errors.Is` and
  `errors.As`. Commands use `RunE` and return errors; no print-and-exit.
- A `nolint` comment names the linter and gives the reason.

@CONTEXT.md
