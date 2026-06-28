## Why

Enhancement [0002](../../../../enhancements/0002/) renames the OPM `Release` family to `Instance` across the whole stack. `core@v1` is published (`v1.0.0-alpha.1`), the `library` kernel and `opm-operator` have landed their renames, and the operator now stamps `module-instance.opmodel.dev/*` labels and emits `kind: "ModuleInstance"`. The CLI is the last actor still speaking the old `Release` / `ModuleRelease` vocabulary, so a CLI loading a `core@v1` module instance fails kind detection and a CLI render is no longer interoperable with the operator.

This change (slice **X1**) is the **foundation** of the CLI rename: the module-instance Go types, the loader/kind-detection surface, and the on-disk **instance-file convention** (`release.cue` → `instance.cue`, D9). It is the first authored slice of the bundled, atomic per-repo CLI PR (X1–X4); it is co-implemented with X2 (`BundleRelease`), X3 (command group), and X4 (label domain + inventory + example fixtures) and bulk-archived together. Everything `Bundle*`, every command surface, and the label/inventory domain are explicitly **out of scope here** and deferred to those slices.

This is a **BREAKING** rename of an internal Go API and an on-disk file-name convention. Per the enhancement (memory: the CLI has no external API consumers; D8 hard-rename / no-alias), it ships with no compatibility shims.

## What Changes

- **BREAKING** — `module.Release` → `module.Instance` and `module.ReleaseMetadata` → `module.InstanceMetadata` (`pkg/module/release.go` → `instance.go`), including the `MatchComponents` receiver, doc comments, and the `#ModuleRelease` → `#ModuleInstance` references in those comments. Every consumer of these types across `pkg/` and `internal/` updates its references (mechanical compile-web fix; ~28 files).
- **BREAKING** — kind detection `loader.DetectReleaseKind` → `loader.DetectInstanceKind` (`pkg/loader/release_kind.go` → `instance_kind.go`); the helper `resolveReleaseFile` → `resolveInstanceFile`.
- **BREAKING** — the module-instance **wire kind string** flips `"ModuleRelease"` → `"ModuleInstance"` to match `core@v1`. The `"BundleRelease"` string is **left untouched** (X2 owns it); detection continues to accept it.
- **BREAKING (D9)** — the authored instance-file convention changes from `release.cue` to `instance.cue`. The loader detects `instance.cue` only inside a package directory; `release.cue` is dropped with no fallback.
- **BREAKING** — `internal/releasefile/` → `internal/instancefile/`; `GetReleaseFile` → `GetInstanceFile`; `ModuleParseData.Metadata` retyped to `*module.InstanceMetadata`; const `KindModuleRelease` → `KindModuleInstance` (value `"ModuleInstance"`). The `KindBundleRelease` const and all `bundle.Release` handling inside this package are **left for X2**, producing transitional mixed-vocabulary files reconciled in the same PR.
- **BREAKING** — the module-synthesis surface: `SynthesizeModuleReleaseFromPackage` and the `module-synthetic-release` capability → instance forms (`pkg/loader/synth.go`); the synthesized wrapper is shaped like `#ModuleInstance`. The `kindBundleRelease` const stays (X2).
- Old-name breadcrumbs (`// Was: …`, D11/D12) at every rename site; `git mv` for every renamed file/dir (D10).
- Loader/module/instancefile **package-level test fixtures** that assert the module kind flip from `"ModuleRelease"` to `"ModuleInstance"` and `release.cue` → `instance.cue`. Command, integration, and e2e fixtures are **not** touched here (X3/X4).

## Capabilities

### New Capabilities

_None._ This slice renames and modifies existing capabilities.

### Modified Capabilities

Renamed capability dirs (delta authored under the new name with a `Renamed from` breadcrumb; the spec dir is `git mv`'d at archive, mirroring the operator's X-wave mechanic):

- `module-instance-type` _(was `module-release-type`)_: `module.Instance` is the prepared module-instance type; `InstanceMetadata`; `MatchComponents` accessor.
- `module-instance-parsing` _(was `module-release-parsing`)_: `ParseModuleInstance` constructs a prepared `Instance` without mutating inputs.
- `module-instance-processing` _(was `module-release-processing`)_: `ProcessModuleInstance` is the public rendering entrypoint.
- `module-instance-receiver-methods` _(was `module-release-receiver-methods`)_: `Instance.ValidateValues` / `Instance.Validate` sequencing and side-effect freedom.
- `instance-file-loading` _(was `release-file-loading`)_: instance-file loader lives in `pkg/loader/`; raw parse-data inspection; concrete metadata during parse-only extraction.
- `instance-building` _(was `release-building`)_: loader validates consumer values and produces a concrete module instance; `LoadInstanceFile()`; synthesis-path catalog resolution.
- `module-synthetic-instance` _(was `module-synthetic-release`)_: synthesize a `#ModuleInstance` from a module-package directory; synthetic-instance metadata defaults; bundle-instance synthesis still unsupported.
- `mod-instance-optional` _(was `mod-release-optional`)_: `mod build`/`mod apply` work without an `instance.cue`; synthesis defaults.

Modified in place (no `release` token in the capability name):

- `loader-api`: `DetectInstanceKind`, `LoadModuleInstanceFromValue`, `SynthesizeModuleInstanceFromPackage`, the module-loader function names; `LoadBundleReleaseFromValue` is **left unchanged** (X2).
- `validation-gates`: the Module Gate validates consumer values against `#module.#config` against a `#ModuleInstance` wrapper; the Bundle Gate requirement is **left unchanged** (X2).

## Impact

- **Affected packages**: `pkg/module`, `pkg/loader`, `internal/instancefile` (was `internal/releasefile`); reference-only compile-web updates in `pkg/render`, `internal/workflow/*`, `internal/inventory`, `internal/cmd/*`.
- **Out of scope (later slices, same PR)**: `pkg/bundle` + `BundleRelease`/`process_bundlerelease` (X2); `internal/cmd/release/` command group (X3); `pkg/core/labels.go` label domain, `internal/inventory` behavior, `examples/releases/**`, integration/e2e fixtures (X4).
- **Upstream dependency**: `core@v1` (`v1.0.0-alpha.1`), already published. **Not** gated on `library` — the CLI has no Go dependency on the kernel; that coupling is enhancement 0006's concern.
- **API/UX**: internal Go API rename (no external consumers); the user-visible change is `release.cue` → `instance.cue` as the authored instance-file name.
- **SemVer**: MAJOR (breaking); ships on enhancement 0002's `v1.0.0-alpha.N` line per D13.
