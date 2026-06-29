## Why

Enhancement [0002](../../../../enhancements/0002/) renames the OPM `Release` family to `Instance`. Slice **X2** was originally scoped to rename `BundleRelease` → `BundleInstance` (D7). X2 planning established that the entire bundle path is **unreachable dead code**: no `internal/cmd` command targets it; `internal/workflow/render/render.go` parses a bundle file into a `*bundle.Release` only to reject it one line later (`"bundle releases are not yet supported"`); and the struct's consumers — `pkg/render/bundle_renderer.go` (`NewBundle`/`Bundle.Execute`/`BundleResult`) and `pkg/render/process_bundlerelease.go` (`ProcessBundleRelease`, a validate-then-`not implemented yet` stub) — have **zero production callers** (referenced only from `pkg/render/{matchplan,process}_test.go`). The only producer of `kind: "BundleRelease"` is the **deprecated** `catalog/core/v1alpha1/bundlerelease` module; no bundle kind exists in `core`, `catalog_opm`, or `catalog_kubernetes`.

Per enhancement decision **D15** (supersedes D7), X2 therefore **removes** the bundle path rather than renaming it. Renaming dead code to a prettier name is busywork, and a renamed-but-still-dead `BundleInstance` is a maintenance trap — it reads as a working construct and invites someone to wire it up against machinery that returns "not implemented yet." With no bundle support planned, there is nothing to preserve. `BundleInstance` is reintroduced — as a real, rendered construct — only if and when bundle support is actually built (a future enhancement).

This is slice X2 of the bundled, atomic per-repo CLI PR (X1–X4): co-implemented with X1 (module-instance types/loader, already committed), X3 (command group), and X4 (label domain + inventory), and bulk-archived together. It is a **BREAKING** removal of an internal Go API surface; per D8/D15 (the CLI has no external API consumers) it ships with no compatibility shim.

## What Changes

- **BREAKING** — delete `pkg/bundle/` entirely (`bundle.go`, `release.go`): the `Bundle`, `BundleMetadata`, `Release`, `ReleaseMetadata` types.
- **BREAKING** — delete `pkg/render/bundle_renderer.go` (`Bundle` struct, `NewBundle`, `Bundle.Execute`, `BundleResult`) and `pkg/render/process_bundlerelease.go` (`ProcessBundleRelease`).
- **BREAKING** — remove the bundle parse path from `internal/instancefile/get_instance_file.go`: the `KindBundleRelease` const, the `Bundle *bundle.Release` field on `FileRelease`, the `KindBundleRelease` switch arm, and the `bareBundleRelease` / `mustBundleReleaseMetadata` / `bestEffortBundleMetadata` helpers.
- **BREAKING** — remove bundle awareness from the loader: drop `kindBundleRelease` and the bundle arm from `pkg/loader/instance_kind.go` (`DetectInstanceKind` no longer accepts `"BundleRelease"`), and remove `rejectBundleShape` / `bundleNotSupported` / the `kindBundleRelease` const from `pkg/loader/synth.go`. A `kind: "BundleRelease"` file now errors via `DetectInstanceKind`'s default arm → `unknown instance kind: "BundleRelease"`; the dedicated friendly message is **not** retained (Option B — no residual bundle awareness).
- Remove the now-redundant bundle reject branch in `internal/workflow/render/render.go`.
- Delete the bundle test fixture `internal/cmd/release/testdata/bundle_release.cue` and the bundle-only tests (`pkg/render/process_test.go` `TestProcessBundleRelease_*`, the `NewBundle` use in `pkg/render/matchplan_test.go`, and bundle assertions in loader/instancefile tests).
- Delete X1's `// TODO(0002 X2)` breadcrumbs rather than flipping them (`instancefile`, `synth.go`, `instance_file.go`, `instance_kind.go`).

## Capabilities

### New Capabilities

_None._ This slice removes a capability and trims bundle requirements from four others.

### Modified Capabilities

- `bundle-release-processing` _(**REMOVED capability**)_: the whole parse-stage/process-stage bundle contract is removed — its three requirements no longer have any backing code.
- `engine-rendering`: **REMOVE** the two bundle requirements ("Bundle renderer renders a BundleRelease into resources", "Bundle renderer uses Release map") — the `pkg/render` `Bundle` struct/`BundleResult` are deleted. (The module-instance `Release`-named references elsewhere in this spec are X1's rename gap, not touched here.)
- `loader-api`: **MODIFY** "DetectInstanceKind identifies instance type" to drop the `BundleRelease kind detection` scenario, and **REMOVE** "LoadBundleReleaseFromValue builds a BundleRelease" (X1 explicitly deferred both bundle parts to X2).
- `validation-gates`: **REMOVE** "Bundle Gate validates consumer values against #bundle.#config" — there is no bundle processing path to gate.
- `pkg-types`: **REMOVE** "Bundle and Release types", and **MODIFY** "Core types exported in pkg/" to drop the `pkg/bundle/` package line and the "External tool imports pkg/bundle" scenario.

### Coordination seams (documented, not delta'd here)

- `rel-commands` carries a `BundleRelease file is rejected with clear error` scenario asserting the message `"bundle releases are not yet supported"`. That message is deleted by this slice, but the whole `rel-commands` capability is renamed to `inst-commands` by **X3**; X3 owns dropping that now-obsolete scenario in the same atomic PR.
- `errors-domain` says `ConfigError` carries a context of `"module" or "bundle"`. The type still permits any context string (not contradictory), so this slice leaves it; a vocabulary tidy can drop `"bundle"` in a later hygiene pass.
- `render-pipeline` notes a non-functional "support future Bundle rendering" aspiration — left intact; bundle remains a possible *future* capability.

## Impact

- **Affected packages**: `pkg/bundle` (deleted), `pkg/render` (`bundle_renderer.go`, `process_bundlerelease.go` deleted; `process_test.go`, `matchplan_test.go` trimmed), `pkg/loader` (`instance_kind.go`, `synth.go`), `internal/instancefile`, `internal/workflow/render`.
- **Out of scope (later slices, same PR)**: the `rel-commands` → `inst-commands` rename incl. the bundle-rejection scenario (X3); label domain + inventory + example fixtures (X4). The module-instance `Release`-named *field* renames (`Resource.Release`, `ModuleResult.Release`) remain X1-gap/X4 territory.
- **Upstream dependency**: none new — this is a pure intra-`cli` deletion on top of X1. Not gated on `library`.
- **API/UX**: internal Go API removal (no external consumers). User-visible change: a bundle `.cue` file now fails with `unknown instance kind: "BundleRelease"` instead of the previous friendly "not yet supported" message.
- **SemVer**: MAJOR (breaking removal); ships on enhancement 0002's `v1.0.0-alpha.N` line per D13, bundled in the atomic CLI PR.
