## Context

Slice X2 of enhancement [0002](../../../../enhancements/0002/). Originally "rename `BundleRelease` → `BundleInstance`" (D7); rescoped by enhancement decision **D15** (supersedes D7) to *remove* the bundle path entirely. X2 planning established the bundle surface is unreachable dead code: `internal/workflow/render/render.go` parses a bundle file into a `*bundle.Release` only to reject it one line later (`"bundle releases are not yet supported"`); the would-be consumers — `pkg/render/bundle_renderer.go` (`Bundle`/`NewBundle`/`Bundle.Execute`/`BundleResult`) and `pkg/render/process_bundlerelease.go` (`ProcessBundleRelease`, a validate-then-`not implemented yet` stub) — have **zero production callers** (tests only). The only producer of `kind: "BundleRelease"` is the deprecated `catalog/core/v1alpha1/bundlerelease`; no bundle kind exists in `core`/`catalog_opm`/`catalog_kubernetes`.

This lands in the single atomic per-repo CLI PR (X1–X4) and bulk-archives with the other cli slices. X1 (module-instance types/loader) is committed and left precise `// TODO(0002 X2)` breadcrumbs marking the bundle seam.

## Goals / Non-Goals

**Goals:**

- Delete the bundle path with no residual awareness (Option B): `pkg/bundle`, `pkg/render/bundle_renderer.go`, `pkg/render/process_bundlerelease.go`, the `internal/instancefile` bundle parse arm/helpers, the loader bundle detection (`DetectInstanceKind` arm, `rejectBundleShape`), the `render.go` reject branch, and all bundle tests/fixtures.
- Trim every spec requirement that the deleted code backed: remove the `bundle-release-processing` capability and the bundle requirements in `engine-rendering`, `loader-api`, `validation-gates`, `pkg-types`.
- Leave the codebase compiling and `task check`-green **at the level of the atomic PR** (X2 in isolation is not independently green by design — see Non-Goals).

**Non-Goals:**

- Renaming `BundleRelease` → `BundleInstance`. Superseded by D15. `BundleInstance` is reintroduced only if/when real bundle support is built.
- Keeping a friendly "bundle not supported" message. Option B removes detection; a bundle file now errors via `DetectInstanceKind`'s default arm → `unknown instance kind: "BundleRelease"`.
- The `rel-commands` → `inst-commands` rename and its bundle-rejection scenario — owned by **X3** (see D-X2.3).
- The module-instance *field* renames (`Resource.Release`, `ModuleResult.Release`, `BundleResult.ReleaseOrder`) — X1-gap / X4 territory, not bundle-specific.
- Independent green-ness of X2 alone. Like X1, X2 is one slice of an atomic PR; command/integration/e2e fixtures reconcile across X3/X4 in the same PR.

## Decisions

### D-X2.1 — Full removal, no residual bundle awareness (Option B)

Delete the bundle machinery *and* the detection/rejection guards (`DetectInstanceKind` bundle arm, `synth.go` `rejectBundleShape`/`bundleNotSupported`, `render.go` reject branch). The alternative (Option A — keep one friendly rejection) was rejected by the user: it would preserve a `release`-tokened code path the rename exists to retire, for the sake of a message that the generic `unknown instance kind` error adequately replaces. Trade-off accepted: a bundle file's error is now terser. Source: user decision 2026-06-28 (enhancement 0002 D15).

### D-X2.2 — X1's `// TODO(0002 X2)` breadcrumbs are deleted, not flipped

X1 left breadcrumbs in `internal/instancefile/get_instance_file.go`, `pkg/loader/{synth,instance_kind,instance_file}.go` anticipating a rename. Under removal they are deleted along with the code they annotated. The `FileRelease` struct keeps its name (X1 noted it doubles as an X3 workflow surface); only its `Bundle *bundle.Release` field is removed.

### D-X2.3 — `rel-commands` bundle-rejection scenario is X3's to drop

`rel-commands` asserts a `BundleRelease file is rejected with clear error` scenario quoting `"bundle releases are not yet supported"`. That message is deleted here, but the whole capability is renamed `rel-commands` → `inst-commands` by X3. To avoid two slices editing the same capability delta, X3 owns removing the now-obsolete scenario in the same atomic PR. X2 does not author a `rel-commands` delta. Recorded so the seam is not lost at archive time.

### D-X2.4 — X2's `pkg-types` delta incidentally corrects an X1 spec-sync gap

X1 renamed `module.Release` → `module.Instance` in code but did not author `pkg-types` / `engine-rendering` deltas, so those main specs still say `Release`/`ReleaseMetadata`. X2 must restate `pkg-types`' "Core types exported in pkg/" requirement (to drop the `pkg/bundle/` line), and writes the `pkg/module/` line with the accurate post-X1 names (`Instance`, `InstanceMetadata`). This is a side effect of removing the adjacent bundle line, not an expansion of X2's scope. The remaining `module.Release` references in `engine-rendering`'s non-bundle requirements are left for the broader X1-gap reconciliation; X2 only removes the two bundle requirements there.

### D-X2.5 — `errors-domain` and `render-pipeline` are left intact

`errors-domain` says `ConfigError` carries a context of `"module" or "bundle"`; the type still permits any context string, so the requirement is not falsified by removal — left for a later vocabulary tidy. `render-pipeline` notes a non-functional "support future Bundle rendering" aspiration — bundle remains a possible *future* capability, so the aspiration stands. Neither gets a delta.

## Risks / Trade-offs

- **Terser error for a stray bundle file.** Low impact — bundles were never renderable; the file would have errored regardless, one step later.
- **Two slices touching overlapping specs in one PR.** Mitigated by clean ownership: X2 owns `bundle-release-processing`/`engine-rendering`/`loader-api`/`validation-gates`/`pkg-types`; X3 owns `rel-commands`. No capability is delta'd by both.
- **Reviving bundle support later costs reintroduction.** Accepted: D15's rationale is that renaming dead code is worse — a live-looking `BundleInstance` stub is a maintenance trap. A future bundle enhancement re-adds the surface deliberately, under the Instance vocabulary.

## Migration Notes

No external consumers (memory: the CLI has no external API consumers; D8/D15 hard-removal, no shim). User-visible delta: a `kind: "BundleRelease"` `.cue` file now fails with `unknown instance kind: "BundleRelease"` instead of `"bundle releases are not yet supported - use a #ModuleRelease file"`.
