## 1. Delete the `pkg/bundle` package (`pkg-types`)

- [x] 1.1 `git rm pkg/bundle/bundle.go pkg/bundle/release.go` (removes `Bundle`, `BundleMetadata`, `Release`, `ReleaseMetadata`). Directory is now empty.
- [x] 1.2 Confirmed no remaining import of `github.com/opmodel/cli/pkg/bundle` (grep clean across `pkg`/`internal`/`cmd`).

## 2. Delete the `pkg/render` bundle renderer + processor (`engine-rendering`, `bundle-release-processing`)

- [x] 2.1 `git rm pkg/render/bundle_renderer.go` (`Bundle` struct, `NewBundle`, `Bundle.Execute`, `BundleResult`).
- [x] 2.2 `git rm pkg/render/process_bundlerelease.go` (`ProcessBundleRelease`, `cuecontextMarker`). `cuecontextMarker` had no other reference — gone with the file (grep-confirmed).
- [x] 2.3 Removed `TestProcessBundleRelease_*` from `pkg/render/process_test.go` (+ `pkg/bundle` import) and `TestBundleRenderer_RenderReturnsNonNilEmptySlices` + the `render.NewBundle(...)` setup from `pkg/render/matchplan_test.go` (+ import). Non-bundle assertions kept.

## 3. Remove the bundle parse path from `internal/instancefile` (`bundle-release-processing`)

- [x] 3.1 Removed `KindBundleRelease` const, the `Bundle *bundle.Release` field, the bundle switch arm, the `bareBundleRelease` / `mustBundleReleaseMetadata` / `bestEffortBundleMetadata` helpers, the `pkg/bundle` import, and the `// TODO(0002 X2)` breadcrumbs. `FileRelease` struct name kept (X3 surface).
- [x] 3.2 Package doc comment now reads "a #ModuleInstance" only, with a removal breadcrumb (D15).
- [x] 3.3 Removed `TestGetInstanceFile_BundleReleasePartial` and `TestGetInstanceFile_FailsWhenBundleMetadataNotConcrete`.

## 4. Remove loader bundle detection (`loader-api`, `validation-gates`)

- [x] 4.1 `pkg/loader/instance_kind.go`: `DetectInstanceKind` switch is now `case kindModuleInstance:` only; bundle kinds fall through to `unknown instance kind`. Breadcrumb comment updated.
- [x] 4.2 `pkg/loader/synth.go`: removed `kindBundleRelease` const, `rejectBundleShape`, `bundleNotSupported`, and the call site. `errors` stdlib import stays used (line ~246). Synth path compiles and rejects non-module input via normal validation.
- [x] 4.3 `pkg/loader/instance_file.go`: `LoadInstanceFile` doc comment drops the `#BundleRelease` reference and the X2 breadcrumb.
- [x] 4.4 `synth_test.go`: removed `TestSynthesizeModuleInstanceFromPackage_BundleRejected`. `module_instance_test.go`: the `BundleRelease` row now asserts `wantErr` + `unknown instance kind` (regression coverage for the rejection). `instance_file_test.go`: removed the `valid BundleRelease file` subtest.

## 5. Remove the redundant reject branch + fixtures (`rel-commands` seam → X3)

- [x] 5.1 Removed the `KindBundleRelease` reject branch from `internal/workflow/render/render.go`. `fmt` + `internalinstancefile` imports stay used.
- [x] 5.2 `render_test.go`: `TestRenderFromReleaseFile_RejectsBundleRelease` now asserts `unknown instance kind` (the new upstream failure). Passes.
- [x] 5.3 `git rm internal/cmd/release/testdata/bundle_release.cue` (was orphaned — test helpers create temp fixtures inline). **Deviation (intentional, per the task's conditional):** `TestReleaseVetCmd_RejectsBundleRelease` does **not** depend on that fixture (it tests arg-count enforcement), so it is left for X3 to rename/drop with the `rel-commands` → `inst-commands` rename (D-X2.3). Not touched here.

## 6. Doc + comment cleanup (D12)

- [x] 6.1 `pkg/core/resource.go`: dropped the "rendering a BundleRelease … multiple ModuleReleases" sentence from the `Resource` doc comment. The `Release` *field* name is left (X1-gap/X4).
- [x] 6.2 No `pkg/bundle` reference in `CLAUDE.md` / `README.md` / `AGENTS.md` — nothing to change.

## 7. Verification gate (this slice)

- [x] 7.1 `gofmt` clean on all touched packages (after re-formatting `get_instance_file.go`).
- [x] 7.2 Lint: the bundle-removal diff introduces **0 new issues**. One pre-existing `goconst` (`pkg/loader/provider.go:26` `"kubernetes"`) is in an untouched file and reproduces at the X1 commit (`git stash`-verified) — not X2's to fix.
- [x] 7.3 `go build ./...` and `go vet ./...` both green.
- [x] 7.4 `pkg/render`, `pkg/loader` (incl. registry-backed synth tests), and `internal/instancefile` all green. **`internal/workflow/render` has one pre-existing X1-gap failure** — `TestRenderFromReleaseFile_ValidValuesDoNotPanicAcrossRuntimes` uses a stale `kind: "ModuleRelease"` fixture (verbatim in X1 commit `7a9175d:internal/workflow/render/render_test.go:127`); the X2 bundle test in that file passes. Full-suite green is gated on the X3/X4 fixture reconciliation in the same atomic PR (matches X1's documented "not green alone" model).
- [x] 7.5 Grep sweep: no surviving bundle *code*. Remaining `bundle` tokens are D15 removal-breadcrumbs, rejection-test names, the X3-owned `cmd/release` test (D-X2.3), and deliberately-left `pkg/errors`/`cmdutil` prose (D-X2.5).
- [x] 7.6 `openspec validate remove-bundle-stub --strict` passes.

## Implementation notes / deviations

- **5.3 — `TestReleaseVetCmd_RejectsBundleRelease` left in place.** The task said remove it *if it depends on the fixture*; it does not (arg-count test), so per D-X2.3 it stays for X3's `rel-commands` rename. The orphaned testdata fixture was still removed.
- **7.2 / 7.4 — two pre-existing X1-gap items surfaced, neither introduced by X2.** (a) `goconst` in untouched `pkg/loader/provider.go`; (b) the stale `kind: "ModuleRelease"` fixture in `internal/workflow/render/render_test.go`. Both reproduce at the X1 HEAD without X2's changes and belong to the broader X1-gap / X3-X4 reconciliation that completes in the bundled atomic PR. Recorded so the verify pass does not misattribute them.
- **Spec-delta accuracy note:** X2's `pkg-types` `MODIFIED` delta writes the `pkg/module/` line with post-X1 names (`Instance`, `InstanceMetadata`) — incidentally correcting an X1 spec-sync gap, since the requirement had to be restated to drop the adjacent `pkg/bundle` line (D-X2.4).
- **Held from archive.** Per the atomic-PR / bulk-archive model, this change stays active alongside X1/X3/X4 and is bulk-archived with them; the `enhancements/0002/config.yaml` history event is recorded at bulk-archive time.
- **Post-verify polish (`openspec-verify-change`, APPROVE / no critical/major).** Two reviewer SUGGESTIONs applied since they fit X2's dead-code thesis and X3 (a rename slice) would not catch them: (S2) moved the `kindModuleInstance` const from `synth.go` to `instance_kind.go` (its only consumer after `rejectBundleShape` was deleted); (S1) aligned the now-unreachable `default` arm in `get_instance_file.go` to `DetectInstanceKind`'s `"unknown instance kind"` wording (was the divergent `"unsupported instance kind"`), keeping it as a defensive guard. Build/vet/fmt/`pkg/loader`+`internal/instancefile` tests green after. The two remaining reviewer nitpicks are intentional deferrals (D-X2.5 `errors-domain` comment; D-X2.3 `TestReleaseVetCmd_RejectsBundleRelease` name → X3).
