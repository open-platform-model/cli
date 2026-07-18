# Proposal: cli-kernel-adoption (enhancement 0006 slice C2)

## Why

The CLI still renders through its own pipeline (`pkg/render` + the match path in `pkg/loader`) while the operator renders through the `library` kernel — so the render digest the CLI writes to `status.lastAppliedRenderDigest` (C1) and the digest the operator computes on its first post-handoff reconcile are only equal by luck, not by construction. Enhancement 0006 D9 closes this by deleting the CLI's pipeline and rendering through the same kernel the operator runs; D30 makes CLI≡operator digest parity the accepted→implemented gate for this slice. The same slice clears the ground first (D39): `~/.opm` still carries a `providers` import block, a pinned `cue.mod` dependency set, and a two-phase bootstrap config loader — all machinery that exists solely to feed a `#Provider` concept the 0001 core rewrite removed.

## What Changes

- **BREAKING** — delete `pkg/render` and the match/synthesis path in `pkg/loader`; every render (`instance apply/build/diff`, `module build/apply`) goes through the `library` kernel (`SynthesizeInstance`/`LoadInstancePackage` → `Validate` → `Match` → `Plan` → `Compile` → `Finalize`). The CLI has a single user (D14): no compat window.
- **BREAKING** — `~/.opm` simplification (D39): `config.cue` loses `providers` and `cacheDir` (scalar data only); `~/.opm/cue.mod/` is retired; the two-phase config load (`BootstrapRegistry` regex scrape → registry-scoped evaluation) collapses to a single pass; `GlobalConfig.Providers`, `GlobalConfig.CueContext`, `pkg/loader.LoadProvider`, `pkg/provider`, and provider auto-resolution are deleted.
- **BREAKING** — `--provider` flag is removed; superseded by `--platform <file>` (D21).
- New platform resolution (D11/D12/D17/D21/D22): precedence `--platform <file>` > cluster `Platform` CR > local default `~/.opm/platform.cue`; warn-on-fallback with source provenance reported on every command; offline `build`/`render` never read the cluster; solo-cluster write-if-absent of the un-owned singleton `cluster` Platform via create-only (409 = success-noop, D22). All three sources decode into `synth.PlatformInput` and run the operator's own `SynthesizePlatform` → `Materialize` path.
- `opm config init` seeds `~/.opm/platform.cue` subscribed to `opmodel.dev/catalogs/opm` + `opmodel.dev/catalogs/kubernetes` with explicit prerelease-tolerant ranges; no `cue mod tidy` step. `opm config vet` validates both files.
- 0002 carryover: the synthesis path emits `kind: "ModuleInstance"` via the kernel (retiring `loadSynthWrapper`'s `#ModuleRelease` application and the `…/modulerelease@v1` import-path drift) — the last production `#ModuleRelease` reference in `cli/`.
- `status.lastAppliedSourceDigest` upgrades from a module-reference identity digest to the kernel's content digest (C1 left this as a stopgap).
- Add `github.com/open-platform-model/library` to `go.mod` (kernel only — no `opm-operator` import, D13). CUE v0.17.1 already landed in C1 (D36), so MVS is satisfied.
- D30 parity gate: an integration check that a CLI render and an operator render of the same instance against the same Platform spec produce identical digests. Precondition (D36): `library` and `opm-operator` are on the same CUE line before the experiment runs.

## Capabilities

### New Capabilities

- `kernel-render`: the CLI renders instances through the `library` kernel; render/source/config digests come from kernel outputs; CLI≡operator digest parity is structural.
- `platform-resolution`: platform-source precedence, the `--platform` flag surface, provenance reporting, the `~/.opm/platform.cue` local default, and solo-cluster Platform write-if-absent.

### Modified Capabilities

- `config`: single-pass load (no registry bootstrap phase); `providers`/`cacheDir` removed from the schema; default template becomes scalar-only.
- `config-commands`: `config init` writes `config.cue` + `platform.cue`, creates no `cue.mod`, runs no `cue mod tidy`; `config vet` validates both files.
- `config-types`: `GlobalConfig` drops `Providers` and `CueContext`.
- `instance-building`: loading/synthesis/values-validation route through the kernel; the synth wrapper requirement set is replaced by kernel synthesis emitting `#ModuleInstance`.
- `module-synthetic-instance`: synthetic builds emit `kind: "ModuleInstance"` via kernel synthesis (0002 carryover).
- `instance-inventory`: `lastAppliedSourceDigest` becomes the kernel content digest; digest fields documented as kernel-derived.

### Removed Capabilities

Delta specs with `## REMOVED Requirements` retire the CLI-pipeline capabilities wholesale: `render-pipeline`, `engine-rendering`, `pkg-render-match`, `component-matching`, `transformer-match-plan-execute`, `provider-loader`, `provider-loading`, `provider-auto-resolution`, `provider-match`, `core-provider`.

## Impact

- **SemVer**: MAJOR (removed flag `--provider`, removed config fields, removed packages). Accepted under D14 (single user, no compat burden).
- **Code**: deletes `pkg/render`, `pkg/provider`, most of `pkg/loader`; rewires `internal/workflow/render` + `internal/workflow/apply`; simplifies `internal/config` (loader, templates, resolver); new `internal/platform` (resolution); `go.mod` gains `library`.
- **Cross-repo gate**: enhancement 0001's library kernel slice — **satisfied** (shipped; operator already consumes it). D36 sequencing: the D30 parity experiment additionally requires `library`/`opm-operator` on CUE v0.17.x line.
- **Conflict**: the active change `cue-binary-integration` exists to run `cue mod tidy` in `config init` — D39 removes that need. That change should be re-scoped or withdrawn; this proposal does not depend on it.
- **User migration**: one-time — existing `~/.opm` users re-run `opm config init` (or hand-edit); the old `providers` block and `cue.mod` are ignored-then-removed. No product deprecation window (D14).
