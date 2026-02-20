## Why

`module.Load()` currently stops at AST inspection — it returns a `core.Module` with only `Name`, `DefaultNamespace`, and `PkgName` populated, leaving `Components`, `Config`, `Values`, and `Value` empty. Downstream pipeline phases must therefore re-load the CUE module from disk, and `release.Builder.Build()` computes release identity via a fragile CUE AST overlay instead of reading the identity that the CUE schema already computes. Additionally, `build/component.Component` is an identical duplicate of `core.Component`, and the `OPMNamespace` constant in the CLI does not match the canonical value defined in the catalog, silently producing wrong UUIDs.

## What Changes

- `module.Load()` performs full CUE evaluation (via `cueCtx.BuildInstance`) after AST inspection, populating `Module.Components`, `Module.Config`, `Module.Values`, and a new unexported `Module.value` field (the base `cue.Value` for the full module).
- `core.Component` is extended to align with the `#Component` CUE schema: adds `ApiVersion`, `Kind`, `Metadata *ComponentMetadata`, `Blueprints map[string]cue.Value`, `Spec cue.Value`, and a `Validate() error` receiver method. `ComponentMetadata` is introduced as a sub-type carrying `Name`, `Labels`, and `Annotations`.
- `internal/build/component/` package is deleted. All consumers are updated to use `core.Component`. **BREAKING**: removes `internal/build/component` as an import path.
- `release.Builder.Build()` signature changes from `Build(modulePath string, ...)` to `Build(mod *core.Module, ...)`. It uses `mod.value` (the already-evaluated base `cue.Value`) directly, eliminating a second `load.Instances()` call.
- The CUE overlay (`generateOverlayAST`, `overlay.go`) is deleted. Release UUID is computed in Go using `uuid.NewSHA1(OPMNamespace, fqn+":"+name+":"+namespace)`.
- **BREAKING**: `OPMNamespace` constant is corrected from `c1cbe76d-...` to `11bc6112-a6e8-4021-bec9-b3ad246f9466` (matching `OPMNamespace` in `catalog/v0/core/common.cue`). All previously computed release and module UUIDs will differ.
- Module UUID (`Module.Metadata.UUID`) and module labels are read directly from the CUE evaluation result (`metadata.uuid`, `metadata.labels`) instead of being computed separately in Go.
- `Module.Values` holds the module's `values` field (a plain struct of suggested config inputs). It is used by `Build()` as the fallback config when no `--values` flag is provided, and may be displayed by `opm mod status`. It is not part of the build flow when user values are explicitly supplied.
- `core` package gains a new `OPMNamespace` constant and a `ComputeReleaseUUID` helper used by `release.Builder.Build()` for release identity.
- `release.Options.PkgName` field is removed (pkgName comes from `mod.PkgName()`).

## Capabilities

### New Capabilities

- `module-cue-evaluation`: `module.Load()` performs full CUE evaluation, returning a `*core.Module` with `Components` (schema-level), `Config`, `Values`, and an internal base `cue.Value` populated. The module is self-describing before the build phase.
- `core-component`: Extended `core.Component` type aligned with `#Component` CUE schema, including `Blueprints`, `Spec`, `ComponentMetadata` sub-type, and a `Validate()` receiver method for structural correctness checks at extraction time.

### Modified Capabilities

- `release-identity-labeling`: The identity field is renamed from `metadata.identity` to `metadata.uuid` (aligning with the catalog schema). UUID computation moves from a CUE AST overlay to Go using the corrected `OPMNamespace`. **BREAKING**: all previously generated UUIDs are invalidated.
- `module-receiver-methods`: `module.Load()` contract expands — it now performs CUE evaluation in addition to AST inspection. `Validate()` may additionally check that `Components` is non-empty when the module defines components.

## Impact

- **Packages changed**: `internal/core/`, `internal/build/module/`, `internal/build/release/`, `internal/build/pipeline.go`
- **Package removed**: `internal/build/component/` (10 import sites updated)
- **SemVer**: MINOR — new internal capabilities, no user-facing CLI flag changes. The OPMNamespace correction is a pre-production breaking change accepted intentionally.
- **Dependencies**: No new external dependencies. Go `uuid` package (already in `go.mod` via inventory code) used for `uuid.NewSHA1`.
- **Test fixtures**: `testdata/` modules in `internal/build/` may need `module.cue` updates if the extended `metadata.uuid` field is required by the loader.
- **Experiment**: All runtime-critical design claims were validated before implementation in `experiments/module-full-load/` (43 tests, fully detached from `internal/`). See `design.md § Experiment Validation` for the decision-to-test mapping.
