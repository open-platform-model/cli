## Context

X1 is the foundation slice of the CLI half of enhancement [0002](../../../../enhancements/0002/) (Release → Instance). The upstream `core@v1` wire contract (`kind: "ModuleInstance"`, `module-instance.opmodel.dev/*` labels) is published and the operator already consumes it. The CLI is a **parallel Go implementation** — it has no Go dependency on the `library` kernel (that adoption is enhancement 0006). So X1's only upstream gate is the published `core@v1` tag, and its job is purely to rename the CLI's own module-instance type + loader surface and adopt the `instance.cue` file convention.

The whole CLI rename (X1–X4) lands as **one atomic per-repo PR, bulk-archived** — a pure rename has no mergeable intermediate state because the type web doesn't compile half-renamed. X1 is authored as a standalone, self-contained OpenSpec change for small-batch spec hygiene (CLI CONSTITUTION §VIII), but it is *implemented* alongside X2/X3/X4 and is not expected to produce a green full test suite on its own.

## Goals / Non-Goals

**Goals**

- Rename the prepared module-instance Go type and its loader/kind-detection surface to `Instance` vocabulary.
- Flip the **module** wire kind string to `"ModuleInstance"` (core@v1 compatibility).
- Adopt the D9 instance-file convention (`instance.cue`), hard-replacing `release.cue`.
- Leave a clean X1/X2 seam so the bundle rename is a separable, additive follow-up within the same PR.

**Non-Goals**

- Any `Bundle*` rename — `pkg/bundle`, `BundleRelease`, `process_bundlerelease`, `kindBundleRelease`, the bundle branches of `GetInstanceFile`/synth (X2).
- The `internal/cmd/release/` command group, flags, aliases, help text (X3).
- The label domain, `internal/inventory` behavior, `pkg/ownership`, `examples/releases/**`, integration/e2e fixtures (X4).
- Any adoption of the `library` kernel (enhancement 0006).

## Decisions

### D-X1.1 — Module wire kind string flips to `"ModuleInstance"`; `"BundleRelease"` is untouched

The detection switch becomes `case "ModuleInstance", "BundleRelease":`. The const `KindModuleRelease Kind = "ModuleRelease"` becomes `KindModuleInstance Kind = "ModuleInstance"`; `KindBundleRelease Kind = "BundleRelease"` is left verbatim for X2. This is what makes the CLI interoperable with `core@v1` (which emits `"ModuleInstance"`). Bundle files keep emitting `"BundleRelease"` until X2 flips them in the same PR.

**Why now and not deferred:** nothing in today's ecosystem emits `"ModuleInstance"` *until* a consumer pins `core@v1`; the `modules/`+`releases/` re-pin is separately tracked. But the CLI must be ready the moment they do, and the wire string is intrinsic to the kind-detection capability X1 owns — splitting it out would leave a second rename pass over the same switch. It rides with X1.

### D-X1.2 — Instance-file convention is a hard flip (D9): `instance.cue` only, no `release.cue` fallback

Inside a package directory the loader resolves `instance.cue`; if absent it errors (`… does not contain instance.cue`). `release.cue` is **not** accepted as a fallback. Justification: D8's hard-rename / no-alias conclusion plus the CLI having no external users (no compatibility owed). A dual-detect alias would be dead complexity (CLI CONSTITUTION §VII, YAGNI). The mirror of the operator's already-shipped `InstanceFileNotFound` behavior.

### D-X1.3 — `releasefile` → `instancefile` package rename keeps the bundle branches intact (X1/X2 seam)

`internal/releasefile/` is renamed to `internal/instancefile/` and `GetReleaseFile` → `GetInstanceFile` in X1, because the package name carries the `release` token (D10) and its module branch is X1's. But the package also dispatches `BundleRelease` and returns `*bundle.Release` / `*bundle.ReleaseMetadata`. Those bundle symbols are **left unchanged** — X1 renames the *container* and the *module* path; X2 renames the *bundle* path. The result is a transitional file where an `instancefile` package still references `bundle.Release`; this is intentional and reconciled when X2 lands in the same PR. A `// Was: …` / `// TODO(X2): …` breadcrumb marks each deferred bundle site so the seam is explicit, not accidental drift.

### D-X1.4 — The type-rename compile web is updated in place, reference-only, even in X3/X4-owned packages

`module.Release` / `module.ReleaseMetadata` are referenced by ~28 files across `pkg/render`, `internal/workflow/*`, `internal/inventory`, `internal/cmd/*`. Renaming the type forces updating every reference or the module doesn't compile. X1 performs these as **mechanical reference updates only** — it does not rename those packages' own types, capabilities, or behavior (that's X3/X4). Spec ownership is unchanged: X1 owns the `module.Instance` type + loader capabilities; the touched lines elsewhere are compile consequences, not new X1 requirements. Under the one-PR model this is invisible at merge.

### D-X1.5 — Capability-dir renames use the operator's archive mechanic

Each renamed capability is authored as a delta under the **new** dir name with `## ADDED Requirements` containing the full updated (instance-vocab) requirement blocks, prefixed by a `<!-- Renamed from \`old-name\` (0002 D10) -->` breadcrumb. The old spec dir is removed via `git mv`at archive time. This mirrors the opm-operator X-wave (`release-*` → `modulepackage-*`) so the two repos' archives read consistently.`loader-api` and `validation-gates` keep their names (no `release` token) and use `## MODIFIED Requirements` with full updated content.

## Risks / Trade-offs

- **Transitional mixed vocabulary** (D-X1.3): between X1 and X2 the `instancefile` package and `synth.go` mix `Instance` and `Release`/`Bundle` names. Mitigation: breadcrumbs at every seam; both slices land in one PR, so no intermediate commit is published in this state.
- **Full suite not green at X1 alone**: command/integration/e2e fixtures still assert `"ModuleRelease"` / `release.cue` until X3/X4. Mitigation: this is the accepted consequence of the atomic-PR model the user selected (Option 1); X1's own package tests (`pkg/loader`, `pkg/module`, `internal/instancefile`) are the green gate for this slice.
- **Wire-string flip ahead of consumers** (D-X1.1): the CLI will require `"ModuleInstance"` while `modules/`+`releases/` still emit `"ModuleRelease"` (separately tracked). Accepted: that ripple is out of 0002's CLI scope by design, and `core@v1`-pinned inputs are the forward target.
- **Inherited spec/code drift (pre-existing, NOT introduced by X1)**: several renamed capability specs name `pkg/loader` symbols that the parallel CLI does not implement — `LoadInstancePackage` / `LoadModuleInstanceFromValue` / `LoadBundleReleaseFromValue` (actual API: `LoadInstanceFile` + `LoadModulePackage`) — and `module-instance-receiver-methods` describes `Instance.ValidateValues` / `Instance.Validate` methods that do not exist (validation lives in `ParseModuleInstance` via `validate.Config`). The corpus is also internally inconsistent (some specs say `core.ModuleRelease`, others `pkg/module.Release`). This is residue of the paused `simplify-render-single-build` / promote-factory-engine refactor (the "Loading IS building" wording is its signature), present before 0002 began; X1's mechanical rename faithfully carried it. Out of scope to fix here — reconcile with whichever effort settles the `pkg/loader` public API (resume single-build, or enhancement 0006 kernel adoption, which replaces this surface wholesale). Archiving X1 does not certify these specs as accurate.

## Migration Notes

- `git mv pkg/module/release.go pkg/module/instance.go`
- `git mv pkg/loader/release_kind.go pkg/loader/instance_kind.go`
- `git mv internal/releasefile/ internal/instancefile/` (+ `get_release_file.go` → `get_instance_file.go`, and `_test.go`)
- Authored instance files in package dirs: `release.cue` → `instance.cue`.
- Verification gate for this slice: `task fmt` + `task lint` + `go test ./pkg/loader/... ./pkg/module/... ./internal/instancefile/...` green. Full `task test` green is gated on the bundled X2–X4 work.
