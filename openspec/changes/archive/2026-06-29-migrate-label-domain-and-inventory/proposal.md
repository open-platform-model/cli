# Migrate label domain and inventory (Release → Instance, cli X4)

## Why

Slice **X4** of enhancement [0002](../../../../enhancements/0002/) — the `Release` → `Instance` rename — and the **final cli slice** before the atomic bulk-archive PR. X1 (`pkg/module` types + loader + instance-file convention), X2 (deleted the dead `BundleRelease` stub), and X3 (command group + every user-facing string) are committed on branch `0002-cli-x1-rename-module-instance` (HEAD `8e6311e`) and held from archive. X4 lands in the **same atomic PR** and bulk-archives with X1–X4.

X3 drew its boundary at **"Go symbols vs. user-facing strings"** (D-X3.7): every string a user *reads* when invoking `opm instance …` was renamed in X3; every remaining `Release`-named *identifier* — inventory record types, the label domain, the logger, cmdutil flag bundles, render file-opts — was explicitly deferred to X4. X4 closes that residue so that, post-PR, no `Release` token referring to the renamed concept survives outside `// Was:` breadcrumbs (D8 hard-rename, no alias). **X4 is the slice that drives the whole cli `task test` to green for the first time in the wave** — X1–X3 each intentionally left fixtures asserting `"ModuleRelease"` / `release.cue`, reconciled here.

## What changes

Despite the slice name, X4 owns the **entire residual `Release`-named surface**, not just labels and inventory (D-X4.1). Five buckets:

1. **Label domain (D4).** `pkg/core/labels.go` `LabelModuleRelease{Name,Namespace,UUID}`: `module-release.opmodel.dev/*` → `module-instance.opmodel.dev/*`, plus consumers in `internal/inventory/*`, `internal/kubernetes/*`, `pkg/ownership/ownership.go`, and selector strings. Must match opm-operator **O3**, already migrated to `module-instance.opmodel.dev/*`. **Hard cutover, no back-compat read selector** (D-X4.3).

2. **Inventory + identity Go symbols.** `ReleaseInventoryRecord`, `ReleaseName`, `ReleaseID`, `ReleaseMetadata`, `ReleaseNamespace`, `ReleaseSummary`, `ReleaseInfo`, `ReleaseNotFoundError`, and the `releaseName`/`releaseID` identity vars — across `internal/{inventory,kubernetes,workflow/{apply,query}}` — → `Instance*`.

3. **Logger + output cluster (~122 refs).** `internal/output/log.go` `ReleaseLogger` → `InstanceLogger`; the `releaseLog` call-var across ~16 files (`internal/cmd/instance/*`, `internal/cmd/module/*`, `internal/cmdutil/manifest_output.go`, `internal/kubernetes/*`, `internal/workflow/*`) → `instanceLog`.

4. **cmdutil flags + render file-opts.** `ReleaseSelectorFlags` → `InstanceSelectorFlags`, `ReleaseFileFlags` → `InstanceFileFlags`, `ResolveReleaseIdentifier` → `ResolveInstanceIdentifier`; the `--release-name`/`--release-id` flags → `--instance-name`/`--instance-id` (**hard rename, no alias** — the last user-facing strings, D-X4.2); `FromReleaseFile`/`ReleaseFilePath`/`ReleaseFileOpts` in `internal/workflow/render` + `pkg/render` → `*Instance*` (**rename now**, D-X4.4).

5. **On-disk moves + go-green.** `git mv examples/releases/**/release.cue` → `examples/instances/**/instance.cue` (garage, jellyfin, mc_java_fleet); `tests/integration/rel-{list,tree}` → `inst-*`; `tests/e2e/testdata/vet-errors/release/`; remaining `"ModuleRelease"` fixture literals (~12 files) and the `render_test.go` stale fixture X2 flagged. Verify `pkg/loader/instance_kind.go`'s default-arm literal.

`// Was:` breadcrumbs at every rename site (D11/D12).

## Out of scope

- **`modules/` + `releases/` `release.cue` sweep** — the D9 ripple into out-of-`affects` repos; tracked separately.
- **The X1-gap `pkg/loader` public-API accuracy gap** (`LoadInstancePackage`/`LoadModuleInstanceFromValue` naming residue from the paused `simplify-render-single-build`) — a spec-accuracy gap, not a rename gap; reconciled with whichever effort settles that API (single-build resume or 0006).
- **Hand-renaming main-spec capability folders** `release-*` → `instance-*` — those ride the bulk-archive spec-sync (D-X3.4), not this change. See design risk re: the malformed-main-spec hazard.

## Affected packages & commands

`pkg/core`, `pkg/ownership`, `pkg/render`, `internal/inventory`, `internal/kubernetes`, `internal/output`, `internal/cmdutil`, `internal/workflow/{apply,query,render}`, `internal/cmd/{instance,module}`; commands `opm instance {apply,delete,status,tree,events,list}` and `opm module {apply,status,events,list}` (the `--instance-name`/`--instance-id` flags).

## SemVer

**MAJOR** — breaking flag rename (`--release-name`/`--release-id` removed) and label-domain cutover on a published wire/label contract. Ships on the enhancement-wide `v1.0.0-alpha.N` prerelease line (D13). Acceptable per [cli-no-external-users] and the alpha line.

## Dependencies & gate

Depends on **X1** (committed); not independently gated on `library`. Held from archive (one-PR bulk-archive model). Gate: `go build ./...` + `go vet ./...` + `task lint` (0 new) + full `task test` **green** + `openspec validate --strict`.
