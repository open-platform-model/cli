## 1. Module-instance type (`module-instance-type`)

- [x] 1.1 `git mv pkg/module/release.go pkg/module/instance.go`
- [x] 1.2 Rename type `Release` → `Instance` and `ReleaseMetadata` → `InstanceMetadata`; update the `MatchComponents` receiver and all field/doc comments; flip `#ModuleRelease` → `#ModuleInstance` in doc comments. Add `// Was: Release` / `// Was: ReleaseMetadata` breadcrumbs (D11/D12).
- [x] 1.3 Update `pkg/module/parse.go`: `ParseModuleRelease` → `ParseModuleInstance`, return `*Instance`, no input mutation; breadcrumb the old name.

## 2. Loader kind detection + instance-file convention (`loader-api`, `instance-file-loading`, D9)

- [x] 2.1 `git mv pkg/loader/release_kind.go pkg/loader/instance_kind.go`
- [x] 2.2 Rename `DetectReleaseKind` → `DetectInstanceKind`; flip the switch to `case "ModuleInstance", "BundleRelease":` (module string flips per D-X1.1; `"BundleRelease"` stays for X2 — breadcrumb it).
- [x] 2.3 Rename `resolveReleaseFile` → `resolveInstanceFile`; change directory detection from `release.cue` to `instance.cue` only, no fallback (D-X1.2); update error strings to `… does not contain instance.cue`.
- [x] 2.4 In `pkg/loader/` (file-loading entrypoints): `LoadReleaseFile` → `LoadInstanceFile`, `LoadReleasePackage` → `LoadInstancePackage`, `LoadModuleReleaseFromValue` → `LoadModuleInstanceFromValue`; leave `LoadBundleReleaseFromValue` unchanged (X2 — breadcrumb). Update all call sites.

## 3. Module synthesis (`module-synthetic-instance`, `instance-building`)

- [x] 3.1 In `pkg/loader/synth.go`: `SynthesizeModuleReleaseFromPackage` → `SynthesizeModuleInstanceFromPackage`; rename `SynthesisResult`/option doc comments from `#ModuleRelease` wrapper to `#ModuleInstance`; breadcrumb old names.
- [x] 3.2 Leave `kindBundleRelease = "BundleRelease"` const and the bundle-synthesis-unsupported path verbatim (X2 — breadcrumb the deferral).
- [x] 3.3 Verify `catalogModulePath`/catalog-pin behavior is unchanged by this slice. **FLAG (out of scope, do not fix here):** synth specs reference `opmodel.dev/core/v1alpha1@v1` / `…/modulerelease@v1` but `core@v1` ships a single `core` package — pre-existing path drift; record as a follow-up, not an X1 task.

## 4. Instance-file package (`internal/instancefile`, D-X1.3)

- [x] 4.1 `git mv internal/releasefile/ internal/instancefile/` and `get_release_file.go` → `get_instance_file.go` (+ `_test.go`); change `package releasefile` → `package instancefile`.
- [x] 4.2 `GetReleaseFile` → `GetInstanceFile`; const `KindModuleRelease` → `KindModuleInstance` (value `"ModuleInstance"`); retype `ModuleParseData.Metadata` to `*module.InstanceMetadata`; rename `bareModuleRelease`/`mustModuleReleaseMetadata` → instance forms.
- [x] 4.3 Leave the bundle branch intact (X1/X2 seam): `KindBundleRelease`, `FileRelease.Bundle *bundle.Release`, `bareBundleRelease`, `mustBundleReleaseMetadata` stay verbatim. Add `// TODO(0002 X2): rename bundle path` breadcrumbs at each deferred site.
- [x] 4.4 Update the `internal/releasefile` reference in `cli/CLAUDE.md` package map to `internal/instancefile` (D12 doc consistency).

## 5. Receiver-method + processing surfaces (`module-instance-receiver-methods`, `module-instance-processing`)

- [x] 5.1 Rename `ProcessModuleRelease` → `ProcessModuleInstance` (`pkg/render/process_modulerelease.go` → `process_moduleinstance.go` via `git mv`); update doc comments; leave `process_bundlerelease.go` for X2.
- [x] 5.2 Update receiver methods `Instance.ValidateValues` / `Instance.Validate` (formerly on `Release`); preserve call sequence and side-effect-free contract.

## 6. Compile-web reference updates (D-X1.4 — reference-only)

- [x] 6.1 Update every `module.Release` / `module.ReleaseMetadata` / `*Release` reference to the `Instance` forms across consumers: `pkg/render/{execute,module_renderer}.go`, `internal/workflow/{apply,query,render}/*`, `internal/inventory/*`, `internal/cmd/{module,release}/*`. Reference-only — do **not** rename those packages' own types, capabilities, or the `internal/cmd/release` command group (X3) or inventory label behavior (X4).
- [x] 6.2 Keep `FileRelease` struct name and `internal/workflow/render.FromReleaseFile` orchestrator name verbatim (dual-purpose container / X3 workflow surface; per subagent flags + D-X1.3). Transform only the `Instance`-vocab call sites inside them.

## 7. X1-scope test fixtures (loader / module / instancefile packages only)

- [x] 7.1 In `pkg/loader/*_test.go` (`module_release_test.go`, `release_file_test.go`, `synth_test.go`) and `internal/instancefile/*_test.go`: flip module fixtures `kind: "ModuleRelease"` → `"ModuleInstance"` and any authored `release.cue` → `instance.cue`; keep `kind: "BundleRelease"` fixtures unchanged (X2). Rename the test files where the name carries `release` (`module_release_test.go` → `module_instance_test.go`).
- [x] 7.2 Do **not** touch `internal/cmd/release/testdata/*`, `tests/integration/*`, or `tests/e2e/*` fixtures — those are X3/X4 scope (full-suite reconciliation happens in the bundled PR).

## 8. Verification gate (this slice)

- [x] 8.1 `task fmt` clean.
- [x] 8.2 `task lint` passes (0 issues) for touched packages.
- [x] 8.3 `go test ./pkg/loader/... ./pkg/module/... ./internal/instancefile/...` green (X1's package-level gate; full `task test` green is gated on X2–X4 in the same PR).
- [x] 8.4 `grep -rn 'module\.Release\b\|module\.ReleaseMetadata\|DetectReleaseKind\|releasefile' pkg internal` returns only deliberate breadcrumbs and deferred-bundle sites — no stray un-renamed module-instance references.
- [x] 8.5 `openspec validate rename-module-instance-types-and-loader --strict` passes.

## Implementation notes / deviations

- **D10 completeness (beyond task list):** `pkg/loader/release_file.go` (and `release_file_test.go`) also `git mv`'d to `instance_file.go` — the file housed `LoadInstanceFile` but the task list only enumerated `release_kind.go`. Renaming it completes D10 ("rename every release-named file"); no behavior change.
- **goconst (8.2 nuance):** the module wire string is now the package const `kindModuleInstance` in `synth.go` (mirroring the existing `kindBundleRelease`), used by the `DetectInstanceKind` switch. This removes a goconst finding whose *shape* pre-existed on `main` (the old `"ModuleRelease"`/`"BundleRelease"` switch literals). Touched files are lint-clean; the package retains pre-existing goconst debt in untouched files (`provider.go` `"kubernetes"`, etc.), so "0 issues" holds for touched files, not the whole pre-existing baseline.
- **Catalog-pin drift (3.3 flag, unresolved by design):** `synth.go` still imports `opmodel.dev/core/v1alpha1/modulerelease@v1` and applies `#ModuleRelease` (catalog-side wire contract). Left verbatim with a `FLAG` breadcrumb at `loadSynthWrapper`; tracked as a separate catalog-pin follow-up, not an X1 task.
- **Transitional mixed vocabulary (D-X1.3):** `internal/instancefile` and `synth.go` intentionally mix `Instance` (X1) and `Bundle*`/`Release` (X2) names; every deferred site carries a `// Was:` or `// TODO(0002 X2)` breadcrumb. Reconciled when X2 lands in the same atomic PR.
