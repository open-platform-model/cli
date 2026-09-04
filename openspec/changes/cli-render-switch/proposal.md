## Why

The CLI still renders through the library's retired pipeline: every platform source decodes into `synth.PlatformInput`, materializes via `SynthesizePlatform` → `Materialize`, and `Kernel.Compile` runs the match-and-execute. Library `v1.0.0-alpha.24` (`library-render-cutover`, 0019 Phase B) deleted all of it: `opm/materialize` and `opm/compile` no longer exist, `Kernel.Render(RenderInput{Instance, Platform, RuntimeName, Skew})` is the sole render verb, and its only platform input is a source-carrying module directory acquired with `AcquirePlatformFromDir`. The CLI is pinned at `alpha.23` and cannot take any later library release: twelve files fail to compile on the bump. The operator finished the same switch (PR 119); the CLI is the last consumer on the old path.

This change is `cli-render-switch`. It re-pins the library, rewires platform resolution and the render call, and absorbs what was sketched as `cli-platform-cr-generation`: rendering against the cluster Platform CR requires turning that CR into a module first, and the render path cannot ship without it. `cli-config-platform-module` (init/vet, the local platform module) is the sibling that lands first in branch order.

## What Changes

- **BREAKING (flag semantics)** `--platform` takes a platform **module directory** (a `cue.mod` plus `platform.cue`), not a data file. A file argument fails naming the expected shape and pointing at `opm config init`. Precedence is unchanged: `--platform` > cluster Platform CR > local default `~/.opm/platform/`.
- **Cluster CR → module.** When the cluster Platform CR is the source, the CLI decodes `spec.type` and `spec.registry` into registry entries, derives the dependency closure and generates a platform module through the library helper (`opm/helper/platformmodule`, byte-identical to what the operator generates), under the OPM home cache at `~/.opm/cache/platforms/<content-hash>/` (the hash of the generated files, so an unchanged CR reuses the directory and the write is idempotent). Module path `opmodel.dev/platforms/cluster@v0`, core pinned at the library's verified release. Legacy CR tolerance is preserved: a subscription without `version` fails at generation with the existing re-apply hint.
- **Render call.** `AcquirePlatformFromDir` replaces materialize; `Kernel.Render` replaces `Compile`. Instance-file commands acquire the instance with `AcquireInstanceFromDir`, layering `-f` values files through the library's values option (`library-platform-module-generator`), so `-f` keeps its meaning. `Result` drops `MatchPlan` and `Components` (no reader outside one test); `Warnings` carry the kernel's render warnings (skew under warn, unhandled optional traits).
- **Skew policy.** New optional `skewPolicy: "warn" | "refuse"` in `config.cue` (default `warn`, 0019 D18) applies when the platform is local or flag-sourced; when the cluster CR is the source its `spec.skewPolicy` wins, matching the operator. A refusal exits as a validation failure naming the path and both versions.
- **Seeding.** The write-if-absent seed (apply fallback) is decoded from the *built* platform value: `#registry` keys, `enable` and the derived `version`, plus `metadata.name` and `type`. Still carried on the render result, no re-read. `opm operator install`'s catalog seed is unchanged (it builds a CR spec, not a platform).
- **Errors.** `RenderError` unwraps to typed causes: unresolved demands and unmatched components stay validation failures; skew refusal, transform errors and over-subscribed providers are reported with the kernel's message and the diagnostics kept beside it.
- `go.mod` bumps to the library release carrying `opm/helper/platformmodule` and the values option. `internal/platform/materialize.go` is deleted; the two `tests/integration` mains (`platform-materialize` → `platform-build`, `render-parity`) move onto `Render`.

Out of scope: `opm config init`/`vet` and the local module template (`cli-config-platform-module`); a bump command for the cluster CR (`opm platform …`); any change to `hack/kind-platform.yaml` (a CR, still valid).

## Affected commands and packages

- Commands: `opm instance apply|diff|build|vet`, `opm module apply|build` (every render-bearing command; `--platform` help text), `opm config` (schema key only).
- Packages: `internal/platform` (resolution, CR decode, generation, seeding), `internal/workflow/render` (kernel env, render call, result), `internal/workflow/apply` (seed source), `internal/config` (`skewPolicy` schema + template comment), `internal/cmdutil` (flag help), `internal/cmd/module` (verbose test), `tests/integration`.

## SemVer classification

MAJOR by behaviour: `--platform` changes shape, the local platform is a module (sibling change), `-f` semantics are preserved. Released with `cli-config-platform-module` as one release; each commit follows the repo's conventional-commit rules.

## Complexity justification

Net removal: the CLI-side platform ingestion (`synth.PlatformInput`, materialize wrapper, match-plan surfaces) goes; the one addition, CR-to-module generation, is a call into the shared helper plus a cache directory. The alternative, a CLI-local generator, was rejected by the library-first decision.

## Capabilities

### New Capabilities

None.

### Modified Capabilities

- `platform-resolution`: precedence over module directories (flag = directory, cluster CR = generated module, local default = module); "Materialization mirrors the operator" is replaced by "Acquisition mirrors the operator" (same helper, same acquire); the seed document is decoded from the built platform; legacy CR tolerance fails at generation instead of synthesis.
- `kernel-render`: `Render` is the single render verb (no `Match`/`Compile`/`Finalize` sequence); instance files enter via `AcquireInstanceFromDir` with layered values; skew policy and warnings surface; the parity check runs both paths through `Render`.
- `config-types`: optional `skewPolicy` key.

## Impact

- `internal/platform/{resolve,spec,cluster}.go` rewritten around directories and entries; `materialize.go` deleted; new `generate.go` (CR entries → helper → cache dir) and `seed.go` (spec from built platform).
- `internal/workflow/render/{kernel,render,module,types}.go`; `internal/workflow/apply/apply.go` (seed call unchanged in shape).
- `internal/config/schema/config.cue`, `templates.go` (comment), `internal/cmdutil/flags.go`.
- Tests: `internal/platform/*_test.go`, `internal/workflow/render/*_test.go`, `internal/cmd/module/verbose_output_test.go`, `tests/integration/{platform-build,render-parity}/main.go`.
- Depends on: `library` release with `opm/helper/platformmodule` + `AcquireInstanceFromDir` values option; `cli-config-platform-module` for `config.PlatformDir`.
- `enhancement.yaml` declares 0019 D7, D8, D18 (CLI surface of the skew policy and shares-nothing consumption), mirroring the operator's declaration.
