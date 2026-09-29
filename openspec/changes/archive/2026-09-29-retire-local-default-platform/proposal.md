## Why

The local default platform (`~/.opm/platform/`) is a platform no cluster runs. Every render that falls back to it answers "what would an imaginary platform do": its catalog pins are whatever `opm config init` shipped, and they drift from both the module's own pins and the cluster's Platform. Since `module build` and `module vet` already render against the module's own deps (cli issue 229), the only remaining users of the local default are the instance commands, `module apply`, and the apply path that seeds a cluster Platform from it. This change points every one of them at a real source instead: the cluster's Platform when one exists, otherwise the render's own dependency pins.

Decided with the user on 2026-09-29, without a new enhancement entry. It reverses delivered 0006 behaviour: the local-default step of the precedence (0006:D11), the offline-commands-never-read-the-cluster rule for `instance build` and `instance vet` (as specified in `platform-resolution`), and the solo-cluster Platform seed on apply (0006:D12/D22). The accessibility intent of 0006:D17 holds: a user who cannot read the cluster Platform still renders, against the deps.

## What Changes

- **BREAKING** `opm instance build`, `instance vet`, `instance diff`, `instance apply` and `module apply` resolve their platform as `--platform <dir>`, else the cluster `Platform` named `cluster` when it can be read, else a platform generated from the render's own dependency pins (the instance package's `cue.mod/module.cue`, or the module's for `module apply`), the same generator `module build` uses.
- **BREAKING** `opm instance build` and `opm instance vet` now read the cluster when a kubeconfig context exists. They gain `--kubeconfig`, `--context` and `--offline` (never contact the cluster). A missing context, an absent or unreadable Platform, or an unreachable API server falls back to the deps with a warning; build and vet never fail because a cluster is unreachable.
- **BREAKING** `~/.opm/platform/` is no longer written or read. `opm config init` writes only `config.cue`; `opm config vet` stops building a platform and warns when a leftover `~/.opm/platform/` exists. The seeded-platform pins `DefaultCorePin` and `DefaultCatalogPins` are removed.
- **BREAKING** `opm instance apply` and `opm module apply` no longer create a cluster Platform. `opm operator install` still seeds its own (unchanged).
- `opm platform check` resolves `[dir]`, else `--platform`, else the cluster Platform, and refuses when none is available. It gains `--kubeconfig` and `--context`.
- The `--platform` help text, the provenance line and the fallback warnings name the deps instead of the local default.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-resolution`: the precedence loses the local default and gains the deps fallback for every instance command and `module apply`; `instance build` and `instance vet` read the cluster; the local-default module requirement is removed; the solo-cluster seed on apply is removed, the operator-install seed keeps its write contract; the module-deps requirement extends to instance packages.
- `config-commands`: `config init` no longer writes a platform module; `config vet` no longer builds one and warns about a leftover directory.
- `inst-commands`: `build` and `vet` accept `--kubeconfig`, `--context` and `--offline`; the `--platform` requirement stops naming the local default.
- `kernel-render`: the local-replacement scenarios stop naming the local default platform as a source.

## Impact

- Code: `internal/platform` (`Resolve`, `ResolveOptions`, `SourceLocalDefault` removed, `EnsureClusterPlatform` and `SpecFromPlatform` removed, `EnsureClusterPlatformForCatalog` kept), `internal/workflow/render` (`env.go`, `render.go`, `validation.go` hints, `types.go` drops `PlatformSpec`), `internal/workflow/apply` (seed removed), `internal/config` (`platform.go`, `paths.go`, `templates.go` platform templates and pins), `internal/cmd/instance` (`build.go`, `vet.go`, `diff.go`, `apply.go`), `internal/cmd/module/apply.go`, `internal/cmd/platform/check.go`, `internal/cmd/config` (`init.go`, `vet.go`), `internal/cmdutil/flags.go`.
- Tests: `tests/e2e/instance_build_test.go`, `tests/e2e/mod_build_test.go`, `tests/integration/render-parity`, `tests/integration/platform-build`, `tests/integration/module-apply`, and the unit tests beside each package. Tests that need a platform module directory use the repo's maintained `hack/platform/`.
- Docs in this repo: `QUICKSTART.md`, `AGENTS.md` (Environment Notes, package map), `docs/site/`, command help. Docs in other repos and the workspace root deps task are follow-ups listed in `orchestration.md`.
- Users: anyone relying on an edited `~/.opm/platform/` passes it with `--platform ~/.opm/platform` instead; the directory is left on disk untouched. Scripts running `instance build` with a kubeconfig context pointing at a cluster with a Platform now get that Platform's catalogs; `--offline` restores a cluster-free render.
- Release class: MAJOR by the cli constitution (changed defaults and a removed seeded file), landing as `feat!` on the `1.0.0-alpha` line.
- Other changes in the set (see `orchestration.md`): `add-skip-unprovided-flag` builds on this precedence and starts after this change merges.
