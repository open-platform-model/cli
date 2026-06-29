## Context

Slice X4 of enhancement [0002](../../../../enhancements/0002/) (`Release` → `Instance` rename) and the **final cli slice** before the atomic bulk-archive PR. X1 (`pkg/module` types + loader + instance-file convention), X2 (delete dead `BundleRelease` stub), and X3 (command group + every user-facing string) are committed on branch `0002-cli-x1-rename-module-instance` (HEAD `8e6311e`), held from archive. X4 lands in the **same atomic PR** and bulk-archives with X1–X4. It depends on X1 (committed); it is not independently gated on `library`.

X3's boundary (D-X3.7) was **"Go symbols vs. user-facing strings"**: it renamed every string a user *reads* when invoking `opm instance …` and deferred every remaining `Release`-named *identifier* to X4 — explicitly listing "the inventory/health/output symbols, `ReleaseSelectorFlags`, `ReleaseFileFlags`, `ResolveReleaseIdentifier`, the `releaseName`/`releaseID` identity vars + `--release-name`/`--release-id` flags," and `ReleaseLogger`/`releaseLog` "if they come from `internal/output` … left (X4)." X4 closes that residue. It is also the slice that drives the whole cli `task test` to green for the first time in the wave — X1–X3 each intentionally left fixtures asserting `"ModuleRelease"` / `release.cue`, reconciled here.

## Goals / Non-Goals

**Goals**

- Migrate the label domain `module-release.opmodel.dev/*` → `module-instance.opmodel.dev/*` (D4) and every selector/consumer, matching opm-operator O3.
- Rename the full residual `Release`-named Go surface: inventory record + identity types, the logger/output cluster, the cmdutil flag bundles, and the render file-opts.
- Hard-rename the `--release-name`/`--release-id` flags to `--instance-name`/`--instance-id` (the last user-facing strings).
- Move the example/integration/e2e trees and fixtures (`examples/releases/**/release.cue` → `examples/instances/**/instance.cue`, `rel-*` → `inst-*`) and reconcile every remaining `"ModuleRelease"` fixture literal so full `task test` is green.
- Author the four X3-deferred `mod-*` deltas plus a `cmdutil` delta, and the label/inventory/ownership deltas.

**Non-Goals**

- The `modules/` + `releases/` `release.cue` sweep (D9 ripple into out-of-`affects` repos) — tracked separately.
- The X1-gap `pkg/loader` public-API accuracy gap (paused `simplify-render-single-build` residue) — a spec-accuracy gap, not this rename.
- Hand-renaming main-spec capability folders `release-*` → `instance-*` — they ride the bulk-archive spec-sync (see Risks).
- Independent green-ness of X4 alone is not the bar; X1–X4 green together in one PR (X4 is the slice that gets there).

## Decisions

### D-X4.1 — Keep the slice name; document that it owns the full residual surface

The name `migrate-label-domain-and-inventory` (from `planned-changes.md` and the wave ordering) describes buckets 1–2 only. Buckets 3 (logger/output, ~122 refs) and 4 (cmdutil flags + render file-opts) are neither "label" nor "inventory." Renaming the slug now ripples into the enhancement doc and the wave map for no functional gain. **Keep the name**; the proposal and this design state explicitly that X4 is the catch-all tail that renames *every* remaining `Release`-named identifier per D-X3.7. The name is a label, not a scope fence.

### D-X4.2 — Hard-rename `--release-name`/`--release-id` → `--instance-name`/`--instance-id`, no alias

These are the last user-facing strings X3 deliberately left (they are flag *names*, i.e. Go-owned identifiers, not help text). A user who just typed `opm instance …` should not then pass `--release-name`. Consistent with D8 (hard rename, no back-compat alias) and X3's D-X3.1 (dropped `rel` with no alias). Justified by [cli-no-external-users] and the alpha line. The persistent-flag registration, `ReleaseSelectorFlags.AddTo`, validation, and all call sites move together.

### D-X4.3 — Label-domain hard cutover; no back-compat read selector

Inventory discovery and prune selectors switch to `module-instance.opmodel.dev/*` with **no** fallback that also matches old `module-release.opmodel.dev/*` secrets. Rationale: opm-operator O3 hard-cut the same three keys; D8 mandates no alias; there is no production cluster state to migrate (alpha line, and the operator/CLI share the label contract — a split-domain read would let a stale-labelled secret survive a prune). A `module-instance`-labelled CLI and a `module-instance`-labelled operator agree; a pre-existing `module-release`-labelled secret is treated as foreign (not OPM-discovered) — acceptable given no real deployments. *This is the one place a silent data-discovery gap could hide; called out so it is a conscious cutover, not an accident.*

### D-X4.4 — Rename the render file-opts now

`FromReleaseFile`/`ReleaseFilePath`/`ReleaseFileOpts` (`internal/workflow/render` + `pkg/render`) → `FromInstanceFile`/`InstanceFilePath`/`InstanceFileOpts`. The plan hedged "if not closed by the X1-gap"; 0006 has not landed, so X4 owns them — and renaming them is the only way to reach zero stray `Release` tokens. This surface is adjacent to the paused `simplify-render-single-build` refactor; if that effort (or 0006 kernel adoption) resumes and rewrites this surface wholesale, it will re-touch these names — an accepted future merge-cost, not a reason to leave a half-renamed surface in the merged PR.

### D-X4.5 — Maximal rename depth (resolved during implementation)

The plan's gate 6.1 ("no stray `Release` token") was in tension with the X1-gap exclusion: implementation revealed ~15 symbols beyond the enumerated buckets, split between a **public/consumer surface** (`GetReleaseStatus`/`PrintReleaseStatus`, `EvaluateReleaseHealth`/`QuickReleaseHealth`/`BuildReleaseSummary`/`RenderReleaseListOutput` — D-X3.7's "inventory/health/output symbols") and **deep render/synth internals** (`renderPreparedModuleRelease`, `resolveReleaseValues`, `SynthesizeModuleReleaseFromPackage`, the `internal/workflow/render` var threading, the `core.Resource.Release` / render `Result.Release` / `TreeResult.Release` fields) that enhancement 0006 / the paused `simplify-render-single-build` will rewrite wholesale. **Decision (user, during apply): maximal** — rename *everything*, including Camp B internals and the bare `Release` fields, to reach zero stray `Release` identifiers. Accepts churn in soon-to-be-rewritten render/synth code and a possible future merge-touch if 0006 resumes (same trade-off as D-X4.4, extended to the whole pipeline).

**Sole carve-out:** the catalog wire contract in `pkg/loader/synth.go` (and `tests/integration/inst-tree/testdata/instance.cue`) — the `opmodel.dev/core/v1alpha1/modulerelease@v1` import and `mr.#ModuleRelease` definition reference the *published catalog artifact*, not a CLI identifier; renaming them would break synthesis against the published catalog. Left verbatim with its existing X1 `FLAG` comment, tracked as the separate catalog-pin follow-up. Genuine `// Was:` breadcrumbs and history comments (`instance_kind.go` "was release.cue", `get_instance_file.go` the `release` token note) are also retained.

Also fixed in passing (D9 correctness, not pure cosmetics): `internal/cmdutil/path_guard.go` detected `release.cue` instead of the D9 `instance.cue` convention — corrected, with the file-rename moves (`examples/instances/**/instance.cue`, `tests/integration/inst-{list,tree}`, `tests/e2e/testdata/vet-errors/instance/`).

## Risks / Trade-offs

- **Malformed-main-spec archive hazard (D-X3.4, carried).** Several cli main specs carry delta-style `## ADDED Requirements` headers in main-spec position (e.g. `openspec/specs/release-workflow/spec.md`) — the exact pathology that forced the opm-operator O-wave to run `openspec archive --skip-specs` and defer a spec-hygiene pass. If the cli bulk-archive hits it, the `release-*` → `instance-*` capability **folder renames become a manual post-archive hygiene pass**, not an automatic spec-sync. Pre-acknowledged here so it is not a surprise at archive time; the repair itself is out of scope for X4 (it is a pre-existing main-spec defect, not introduced by this rename). Well-formed targets (e.g. `mod-apply`'s `## Purpose`/`## Requirements`) sync cleanly.
- **Large mechanical diff, atomic PR.** ~250+ identifier touch-sites across ~30 files plus tree moves. Mitigated by: it is a pure rename (no behavior change), every site carries a `// Was:` breadcrumb (D11/D12), and the whole-suite `task test` green gate is the correctness backstop — the first time the wave reaches it.
- **Shared-file edits with X3 in one PR.** `QUICKSTART.md` and `internal/workflow/query/inventory_test.go` are touched by both slices. Clean ownership (D-X3.7 / D-X3.8): X3 owns command *verbs* / user-read strings; X4 owns example *paths* / inventory *types* / label *keys* / flag *names*. No line owned by both.

## Migration Notes

Hard cutover — no migration shim. A user re-running `opm instance status`/`delete`/`list` against a cluster that holds inventory Secrets written by a pre-X4 CLI (old `module-release.opmodel.dev/*` labels) will not discover them; this is accepted (alpha, no production state, D-X4.3). The `--release-name`/`--release-id` flags are removed outright; cobra emits `unknown flag` guidance. Capability main-spec folder renames are deferred to the bulk-archive spec-sync (or its manual hygiene fallback).
