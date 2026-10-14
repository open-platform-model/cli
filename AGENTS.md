# OPM CLI repository guide

The `opm` command-line tool of Open Platform Model. It builds, validates, renders, deploys and
inspects portable app releases defined in CUE, with a focus on type safety, clear command behavior
and Kubernetes workflows. Stack: `cobra`, the CUE Go SDK, Kubernetes `client-go`,
`charmbracelet/log`, `testify`. Go version: see `go.mod`.

`CONSTITUTION.md` holds the repo principles and the change-shaping constraints. `README.md` has the
product purpose and the command groups. `docs/STYLE.md` extends the workspace prose style.

## Branch model

| Branch | What lands |
| --- | --- |
| `main` | All work, by PR. Releases are `1.0.0-beta.N` (`prerelease-type: beta` in `release-please-config.json`). |

No `v1` or `release/*` branch exists on the remote today. For the release line, the beta promise
and how a break is typed, read `references/release-line.md`.

## Repository rules

- The cli relies fully on the library. A frontend calls the kernel and nothing else: a gap is a
  library defect to fix upstream, never a CLI-side fork. Reason: structure belongs to the library,
  presentation to the frontend. The cli holds no built platform value and no matching or transformer
  execution; one `Kernel.Render` call renders.
- No local Kubernetes object, label, order, inventory or health package. These are the library's
  `opm/k8s/object`, `opm/k8s/labels` (import as `opmlabels`), `opm/k8s/inventory` (import as
  `k8sinventory`) and `opm/k8s/health` (import as `health`). A `depguard` rule in `.golangci.yml`
  refuses the retired `pkg/core`, `pkg/resourceorder` and `pkg/inventory` paths, and
  `TestNoLocalHealthEvaluator` (`internal/kubernetes/health_refusal_test.go`) refuses a local
  readiness evaluator.
- Read no error text. Read a registry fetch or dependency answer from the library's typed errors
  (`liberrors.Classify`, `*FetchError`, `*ResolutionError`), and a CUE failure from its error path
  (`internal/config` `cueErrorUnder`). `TestNoErrorTextMatch` (`pkg/errors/textmatch_refusal_test.go`)
  refuses a new text match in non-test code and lists the four that stay, each with the typed cause
  that is missing. It reads syntax, not types: it misses a message built by formatting the error
  (`fmt.Sprint(err)`) or passed on as a string. Pass the error, never its text.
- Only `internal/kubernetes` `RunDeletion` deletes an object of an instance. `TestDeleteCallSites`
  (`internal/kubernetes/deletesites_test.go`) lists the only functions that may send a DELETE.
- `internal/cuemod` is the only site of `os.Chdir` and `os.Setenv`. `cmd/cue` reads the module root
  from the working directory and the registry from `CUE_REGISTRY`, so `Tidy` sets both under a
  package mutex and restores them on every return path.
- Ship changes as mergeable sections, each ending green with its own commit (`CONSTITUTION.md`
  section VIII). Prefer updating an existing package over a new abstraction, unless duplication or
  coupling justifies it.
- A broken cli release is fixed by the next release, with a Go `retract` when the broken version must
  not be selected. A published GitHub Release is never deleted. A draft release is finished with the
  release workflow's manual run. Only the owner deletes a duplicate or tagless draft (runbook in
  `.github/workflows/release.yml`).
- Any change under `templates/<t>/` bumps that template with `opm module version set` in the same
  change. Reason: published versions are immutable. Detail in `references/fixtures.md`.
- Never give a test fixture an `opmodel.dev/*` path. Fixtures live on `testing.opmodel.dev/*`. Detail
  in `references/fixtures.md`.
- Run `task -x deps:cascade`, never plain `task deps:cascade`: go-task turns its exit 3 (nothing to
  do) into 201. Do not run the workspace root `deps:update`, `deps:update:templates` or `deps:pins:*`
  against the cli: they move core past the catalog's pin and ignore `.cascade-frozen`. Detail in
  `references/release-cascade.md`.
- Never edit `.tasks/cascade/wiring-check.sh`: it is a byte-identical copy of the `.github` file at
  the pinned SHA. Detail in `references/cascade-wiring.md`.
- `hack/fixtures.sh` and `tests/fixtures/fixtures.go` are byte-identical copies of opm-operator's.
  Edit both, then run the workspace root `task fixtures:lint`.
- The only writer of `internal/operator/pin.go` is `hack/operator-pin` (`task operator:pin
  VERSION=<module version>`). Do not hand-edit the file.
- A comment cites an enhancement as the id beside the reason in words, once at the symbol:
  `// Version writes are offline by design (enhancement 0011).` Never an id alone, a decision number
  or a section, slice or task number. Never in scaffold templates, generated files, fixtures a user
  copies or CLI output strings. In CUE the reference goes in a `// WHY` block, one blank line below
  the doc comment. `task cite-lint` checks it. The workspace `references/enhancements.md` holds the
  general rule.
- A change that delivers a line of an enhancement's `tasks.md` has `## Enhancement` in its
  `proposal.md` with `Delivers NNNN, task X.Y.`, otherwise `None.`.

## Layout

```text
adr/                       Architecture Decision Records
cmd/opm/                   entrypoint and root command wiring
internal/cmd/              Cobra commands: module, catalog, instance, config, operator, registry, platform
internal/cmdutil/          shared flags, annotations, argument resolution, command-facing helpers
internal/compat/           compatibility comparison of contract surface (levels, provenance denylist)
internal/config/           config resolution, schema validation, defaults
internal/cueedit/          surgical authoring-file rewrites (Version, ModulePath, module path, re-identification)
internal/cuemod/           in-process `cue mod tidy [--check]`; `cuemodtest/` is a hermetic registry harness
internal/dockercfg/        read-modify-write of the docker credential file (`registry login`)
internal/exit/             exit codes: 0 ok, 1 general, 2 validation, 3 connectivity, 4 permission, 5 not found
internal/instinit/         the `opm instance init` package generator
internal/inventory/        instance inventory records, ownership, discovery and prune gates
internal/kubernetes/       cluster ops: status, apply, delete (`RunDeletion`), events
internal/modref/           published-module resolution: path grammar, `--version` selector, core-compatible walk
internal/operator/         `opm operator install` and `uninstall`; `pin.go` is generated by `hack/operator-pin`
internal/output/           terminal formatting, log output, tables, manifests
internal/platform/         platform-source resolution and the Platform seed for `operator install`
internal/publish/          identity-driven publish pipeline: gates, refusals, plan, registry push
internal/scaffold/         fetch-based `module init` engine: template refs, re-identification, repair
internal/version/          build version values
internal/workflow/         render, apply and query orchestration; `render` makes the one `Kernel.Render` call
pkg/errors/                shared structured errors; import as `oerrors`
pkg/loader/                local-replacement provenance (module root, `cue.mod/local-module.cue`)
templates/                 official template modules (`minimal`, `standard`, `advanced`)
hack/docskit-dump/         program docs-kit runs for the command reference; not linked into `opm`
hack/operator-pin/         the only writer of `internal/operator/pin.go`; not linked into `opm`
hack/platform/             maintained platform module for offline tests
tests/fixtures/            fixture modules on `testing.opmodel.dev`; `tests/e2e/` end-to-end Go tests
tests/integration/         integration programs run with `go run`
docs/site/                 site pages opmodel.dev builds; `docs-kit.cue` is the docs bundle
openspec/                  specs, active changes and archive
.tasks/, .github/scripts/  cascade and CI shell tooling
```

Layout caveats:

- `module`/`catalog` each carry a `version set` subgroup. `module` also carries `template list` and an
  `init`. `registry` carries `login`. `instance` carries `init`.
- The CUE fixtures lag the shipped pins until the next `task -x deps:cascade` run. Read a fixture's
  own `cue.mod` for its versions.
- There is no local default platform. `opm config init` writes `~/.opm/config.cue` only.
- Platform resolution, `--skip-unprovided` and skew policy: `references/render-path.md`.

## Commands

| Command | What it does | When |
| --- | --- | --- |
| `task build` / `task build:all` / `task install` | Build `bin/opm` / cross-compile / install with version ldflags | Run the tool |
| `task fmt` / `task vet` / `task tidy` / `task generate` | `go fmt` and `goimports` / `go vet` / `go mod tidy` / `go generate` | After a Go edit |
| `task lint` | `task lint:config`, then `golangci-lint run ./...` | Before you finish |
| `task lint:config` | Verify `.golangci.yml` against the committed schema, offline | After a lint config or version change |
| `task lint:fix` | `golangci-lint run --fix` | Lint findings that auto-fix |
| `task cite-lint` | Refuse a decision number or lone enhancement id in a comment | After a comment edit |
| `task openspec:check` | `openspec validate --all --strict` | After a spec or change edit |
| `task test` / `task test:unit` | All suites / `internal` and `pkg` | Default checks |
| `task test:run TEST=Name` | One test by name | While you work |
| `task test:e2e` / `task test:integration` | E2E / integration programs (need `kind-opm-dev`) | Cluster behavior |
| `task test:fixtures` | Reproduce the PR fixtures job locally | After a fixture change |
| `task cluster:create` / `cluster:operator` / `cluster:delete` | Make, prepare, remove the `opm-dev` kind cluster | Cluster tests; `references/e2e-cluster.md` |
| `task docs:bundle` / `docs:bundle:check` | Build / check the docs bundle | After a help or docs page change |
| `task cascade:wiring:check` | Offline cascade wiring check | After a workflow change |
| `task operator:pin VERSION=...` | Rewrite `internal/operator/pin.go` | Move the operator pin |
| `task -x deps:cascade` (`:title`, `:body`, `:test`) | Move upstream pins in the working tree | Cascade work |
| `task check` | `fmt`, `vet`, `lint`, `cite-lint`, `openspec:check`, `docs:bundle:check`, all tests | Before you finish |

Registry mapping for integration and CUE work: the workspace `references/registry.md`. Both
`opmodel.dev/*` and `testing.opmodel.dev/*` resolve from GHCR. No local registry is required in this
repo; `KIND_CUE_REGISTRY` is the opt-in for iterating on a locally published module.

## Style

- Doc and prose style: `docs/STYLE.md`. Output and docs are ASCII-safe: write `[x]` and `[ ]`, not
  Unicode checkmarks. Say what happened and how to fix a failure.
- Alias `github.com/open-platform-model/cli/pkg/errors` as `oerrors`. Add no other alias without a
  collision or a clear gain.
- Keep command packages thin: parse flags, delegate to `internal/workflow` or a focused package. No
  business logic in a command. `internal/` depends on `pkg/`; `pkg/` stays command-agnostic.
- Return concrete structs and take interfaces at boundaries. No `any` unless the API is open-ended.
  Type config, flags and render inputs. Pass `context.Context` through I/O. Use a fresh CUE context
  per command, never one global.
- Validate early. Wrap errors with `%w` and context. Commands use `RunE` and return errors; no
  print-and-exit. Reuse `pkg/errors` types (`DetailError`) and keep sentinels matchable with
  `errors.Is` and `errors.As`.
- Tests: table-driven, `require` for preconditions, `assert` for expectations, `t.Helper()` in
  helpers, `t.TempDir()` over manual dirs. Name `TestXxx_Behavior`.
- Lint: `references/lint-config.md`. `nolint` comments name the linter and give the reason.
- Conventional Commits decide the release: `feat`, `fix`, `perf`, `revert`, `deps` and `refactor`
  release; `chore`, `test`, `ci`, `build` and `docs` do not. Edits under `templates/` are never
  `docs`. Pins in `templates/*` are `fix(deps)`; bumps of `examples/*`, `tests/fixtures/*` and
  `hack/platform/` are `test(fixtures)`.

@CONTEXT.md
