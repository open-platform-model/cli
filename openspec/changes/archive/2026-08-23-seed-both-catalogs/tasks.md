# Tasks: seed-both-catalogs

Ordered so each group is green on its own: the constant first, then the template it feeds, then the
assertions, then the mirror peers that must not drift.

The sequencing constraint is satisfied: `opmodel.dev/catalogs/k8s@v1` (`1.0.0-alpha.1`) and
`opmodel.dev/catalogs/opm@v2` (`2.0.0-alpha.5`) are both published on GHCR (verified 2026-08-22). See
design.md for the pin values and the rationale behind the API shape below.

## 1. Catalog path constants

- [x] 1.1 In `internal/config`, add `DefaultCatalogPaths []string` naming both first-party catalogs (`opmodel.dev/catalogs/opm@v2`, `opmodel.dev/catalogs/k8s@v1`), each spelled exactly once. Change `DefaultCatalogPath` from a literal `const` to a `var` derived as `DefaultCatalogPaths[0]` — see design.md "DefaultCatalogPaths as source of truth, DefaultCatalogPath as a derived alias". Update its doc comment: it currently calls the abstraction catalog "the single first-party OPM catalog", which is no longer true, and it should now say what the derived alias names and why (`opm operator install` stays single-catalog).
- [x] 1.2 Update the re-export in `internal/platform/catalog.go` to re-export both `DefaultCatalogPaths` and `DefaultCatalogPath`, keeping it a re-export rather than a second source of truth.
- [x] 1.3 `go build ./...` passes with no edits needed in `internal/cmd/operator/install.go` or its tests (`operator_test.go`, `internal/platform/catalog_test.go`, `internal/platform/cluster_test.go`) — `platform.DefaultCatalogPath` keeps resolving to the OPM catalog path unchanged.

## 2. Seeded platform template

- [x] 2.1 Pin `opmodel.dev/catalogs/opm@v2` to `2.0.0-alpha.5` (bumped from `2.0.0-alpha.3`) and `opmodel.dev/catalogs/k8s@v1` to `1.0.0-alpha.1` — both published, per design.md.
- [x] 2.2 Extend `DefaultPlatformTemplate` in `internal/config/templates.go` so the rendered `registry` block carries both subscriptions, each with an explicit concrete `version`, interpolated from `DefaultCatalogPaths` (1.1) rather than written as literals.
- [x] 2.3 Extend the template's own comment: it explains why the pin is hand-bumped and offline; it now explains it for two pins.
- [x] 2.4 Rendered output is still valid data-only CUE with no imports, and still decodes through the embedded projection schema.

## 3. Assertions

- [x] 3.1 Update `internal/cmd/config/init_test.go`: it asserts the seeded file has exactly one registry entry and no second catalog path. It now asserts two entries keyed by both catalog paths, each with a concrete version.
- [x] 3.2 Keep the assertion that no `opmodel.dev/catalogs/kubernetes` entry appears — that path names a retired module, not the extracted catalog, and the distinction is easy to lose.
- [x] 3.3 Add a case covering the `config-commands` scenario "Seeded platform offers the raw escape hatch": a demand on a raw contract is matchable from the seeded default with no user edit. Implemented via `opmconfig.LoadPlatformFile` + a `cue.Value` lookup of the `k8s@v1` subscription, not a substring match.
- [x] 3.5 (not originally listed) `internal/platform/spec_test.go` also hardcodes the old single-entry/`2.0.0-alpha.3` shape of `config.DefaultPlatformTemplate` in `TestDecodeFile_DefaultTemplate` and `TestWireRoundTrip_FileToInputToCRSpec` — direct fallout of task 2.2, not covered by the proposal's Impact list. Updated to assert both entries/pins.
- [x] 3.6 `task test:unit` passes. `task test:integration` was attempted but blocked by pre-existing cluster state unrelated to this change (a namespace stuck `Terminating` from a prior run on the connected cluster) — not re-verified end-to-end; flagging rather than treating as green.

## 4. Mirror peers

- [x] 4.1 Add the second subscription to `hack/platform.cue` with the same pin.
- [x] 4.2 Add the second subscription to the operator's sample Platform with the same pin.
- [x] 4.3 Add the second subscription to `hack/kind-platform.yaml` with the same pin. It already mirrors `hack/platform.cue`/`templates.go` by its own header comment but was never named in the documented mirror contract — see design.md "The mirror-pin contract grows to four files".
- [x] 4.4 Update the mirror-contract note in `cli/CLAUDE.md`: it names three files that must be bumped in one commit; the set is now four files, each carrying two pins.
- [x] 4.5 (not originally listed) `opm-operator/internal/controller/platform_controller_test.go` and `opm-operator/test/integration/reconcile/registry_helpers_test.go` each carry a `testCatalogVersion()` helper whose own comment says "the default tracks the pin in config/samples" — a self-declared mirror of the sample Platform bumped in 4.2. Updated both to `2.0.0-alpha.5`; `go build ./...` and `go vet` pass in `opm-operator`. `opm-operator/internal/controller/platform_test.go`'s three `2.0.0-alpha.3` literals are untouched — inline test fixture values with no declared mirror relationship, left alone rather than guessed at. See note below on `library/modules/opm_platform`.
- [x] 4.6 `task fmt`, `task lint`, `task test:unit` all pass in `cli`. `task test:integration`/`task test:e2e` not verified — see 3.6.

## Found but not touched: `library/modules/opm_platform`

`opm-operator`'s sample Platform previously commented that its pin was "kept aligned with the CLI's
DefaultPlatformTemplate seed and the library's `opm_platform` fixture (three pins, one value)".
`library/opm/kernel/synth_platform_test.go` and the `library/modules/opm_platform` fixture it reads
both still pin `2.0.0-alpha.3` — a fifth mirror site, in a fourth repo, never named in this
proposal's Impact list. Left as-is: `library/` is its own repo with its own OpenSpec workspace and
branch model, and this loosely-worded three-way "kept aligned" convention (vs. the CLI's tightly
specified mirror contract) isn't something to guess at from here. Flagged for a human decision —
either a small follow-up change in `library/`, or a decision that its pin is independent by design.
