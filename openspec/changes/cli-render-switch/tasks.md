# Tasks: cli-render-switch

Gates: a published `library` release carrying `opm/helper/platformmodule` and the `AcquireInstanceFromDir` values option (`library-platform-module-generator`); `cli-config-platform-module` merged (supplies `config.PlatformDir` and the local module).

## 1. Dependencies and config

- [x] 1.1 Bump `go.mod` to the library release; `go mod tidy`; confirm the only compile failures are the twelve files this change rewrites.
- [x] 1.2 `internal/config/schema/config.cue`: optional `skewPolicy?: "warn" | "refuse"`; `GlobalConfig` field + resolution (absent = warn); `templates.go` config comment; loader test for the invalid value.

## 2. internal/platform

- [x] 2.1 `Spec{Name, Type, Entries []Entry{Path, Version, Enable}}` replaces `synth.PlatformInput`; `DecodeCRSpec` returns it (legacy `filter` ignored, empty version tolerated); delete `materialize.go` and `toInput`/`wireFromInput` in favour of `Spec` ↔ wire mapping.
- [x] 2.2 `generate.go`: CR `Spec` → `platformmodule.Roots` (missing version → the legacy-CR hint error, before registry I/O) → `Closure` via `NewRegistry(RegistryConfig{Registry, ClientType: "opm-cli", Env})` → `Generate(Input{ModulePath: opmodel.dev/platforms/cluster@v0, …})` → sha256 of files → `~/.opm/cache/platforms/<hash>/` (skip if present and equal; else staging + rename). Unit tests with a fixture `ModFileSource`, temp home.
- [x] 2.3 `resolve.go`: `Resolve` returns `(dir string, Resolution)`; flag = directory (file or non-module → shape error with `opm config init` hint), CR = generated dir, local = `config.PlatformDir`; `Resolution` gains `Dir` and `SkewPolicy` (CR only); `Describe` prints the directory. Update `resolve_test.go`.
- [x] 2.4 `seed.go`: `SpecFromPlatform(*platform.Platform)` through schema paths (`Registry` entries: key, `enable`, derived `version`; `metadata.name`, `type`); `cluster.go` seeds from `Spec`; `EnsureClusterPlatformForCatalog` unchanged. Update `cluster_test.go`.

## 3. internal/workflow/render

- [x] 3.1 `kernel.go`: `renderEnv{kernel, platform *platform.Platform, resolution, spec Spec, skew kernel.SkewPolicy}`; `resolvePlatformEnv` = `Resolve` → `AcquirePlatformFromDir(dir, {Registry})` → `SpecFromPlatform`; skew chosen per design (CR field > config key > warn); provenance line names the directory and a `refuse` policy.
- [x] 3.2 `render.go`: `FromInstanceFile` acquires with `AcquireInstanceFromDir(dir, opts, WithValues(sources...))` from `LoadSourceFromFile` per `-f` file; delete `unifyValuesFiles` and the `ProcessModuleInstance` call; `compileInstance` → `renderInstance` calling `Kernel.Render`; `Result.Warnings` from `RenderResult.Warnings` (+ D19 warning).
- [x] 3.3 `types.go`: drop `MatchPlan`, `Components`; `PlatformSpec` becomes `platform.Spec`; doc comments.
- [x] 3.4 `printValidationError` handles `*kernel.RenderError` (message + diagnostics rows) and `*oerrors.SkewError`; decide the verbose pair listing by reading `internal/cmd/module`'s verbose printer, then fix `verbose_output_test.go`. Update `render_test.go`, `module_test.go`.
- [x] 3.5 `internal/workflow/apply/apply.go`: seed call passes `result.PlatformSpec` (new type); no behaviour change; test for "seed carries derived versions".

## 4. Flags, docs

- [x] 4.1 `internal/cmdutil/flags.go`: `--platform` help = "Path to a platform module directory (overrides the cluster Platform and ~/.opm/platform/)". README/command docs sentences mentioning the platform file.
- [x] 4.2 `CLAUDE.md`: render path notes (Render, cache dir, skew key); remove materialize wording.

## 5. Integration and verification

- [x] 5.1 `tests/integration/platform-build/main.go` (renamed from `platform-materialize`): seeded local module → `AcquirePlatformFromDir` → derived versions; `render-parity/main.go`: both paths through `Render` against one platform dir, digests equal. Update the Taskfile entries that run them.
- [x] 5.2 Grep the repo for `PlatformInput`, `Materialize`, `MatchPlan`, `CompileInput`, `platform.cue` (outside archive and the config change's own hits); zero remaining. Also retire everything `cli-config-platform-module` left transitional: `config.LoadLegacyPlatformFile` and `schema/platform.cue` (`#PlatformFile`), `platform.DecodeFile` and `platform.LegacyDefaultPlatformFile` (with the three `tests/integration` mains that seed it), the `--platform` help text naming `~/.opm/platform.cue` (4.1), the transitional notes in `hack/platform/platform.cue`, `internal/platform/resolve.go` and the no-source error text, and `QUICKSTART.md`'s claim that renders use `~/.opm/platform/` (true only once this change lands).
- [x] 5.3 `task fmt lint test` green; `task test:e2e` against kind with `hack/kind-platform.yaml` applied (CR source path: generated module, render, apply, write-if-absent not triggered) and one run with the CR absent (fallback + seed from the local module).
  Verified 2026-09-04: `task fmt`, `task lint`, unit tests and `task test:integration` (kind `opm-dev`, GHCR) green; e2e green except `TestE2E_ThinEditor_ValuesRoundTrip` and `TestE2E_Delete_OperatorOwnedDelegates`, which wait for the in-cluster operator (released `v1.0.0-alpha.14`, still on the materialize path) to reconcile a core-v2 platform and time out; the CLI side of both (CR → generated module under `hack/cache/platforms/`, render, apply) succeeds, and no operator release yet carries the render switch. CR-source and CR-absent runs verified by hand with `opm instance apply` (dry-run through the generated module; fallback, apply and write-if-absent seed carrying derived versions and `enable`, then the CR restored from `hack/kind-platform.yaml`).
