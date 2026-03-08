## Why

ModuleRelease is currently an internal, ephemeral construct — built in-memory during the render pipeline and never exposed to users. This prevents three important workflows: (1) GitOps-style deployment repos where release intent is version-controlled as declarative files, (2) multi-release management where the same module is deployed with different configurations without juggling values files and CLI flags, and (3) auditability through diffable, reviewable release definitions. Additionally, `opm mod vet` requires a `values.cue` file even though `#Module` already has a `debugValues` field designed for exactly this purpose.

## What Changes

- **New `opm release` command group** (alias: `rel`) with subcommands: `vet`, `build`, `apply`, `diff`, `status`, `tree`, `events`, `delete`, `list`
- **Polymorphic release command surface** — `opm release` is designed to handle both `#ModuleRelease` and `#BundleRelease` files. This change implements `ModuleRelease` only; `BundleRelease` files are detected and rejected with a clear "not yet supported" error.
- **Predefined `<name>_release.cue` files** — users can author `#ModuleRelease` definitions directly as CUE files with inline values
- **Hybrid module resolution** — release files can import modules from registry (`#module: jellyfin`) or have `#module` filled from a local directory via `--module` flag
- **Positional argument UX** — render commands (`vet`, `build`, `apply`, `diff`) take a `.cue` file path; cluster-query commands (`status`, `tree`, `events`, `delete`) take a release name or UUID (auto-detected)
- **Cluster-query commands migrate from `mod` to `release`** — `status`, `tree`, `events`, `delete`, `list` move to `opm release` since they operate on releases, not modules
- **`opm mod build/apply` become aliases** — internally delegate to the release pipeline, constructing an ephemeral release from flags (`--values`, `--namespace`, `--release-name`)
- **`opm mod vet` uses `debugValues` by default** — when no `-f` flag is provided, uses the module's `debugValues` field instead of requiring `values.cue`

This is a **MINOR** version change. Existing `opm mod` commands continue to work. New `opm release` commands are additive.

## Capabilities

### New Capabilities

- `rel-commands`: The `opm release` command group (alias: `rel`) — file-based release rendering (vet/build/apply/diff) and cluster-query commands (status/tree/events/delete/list) with positional argument UX; polymorphic surface designed to handle both `#ModuleRelease` and `#BundleRelease` (implements `ModuleRelease` only in this change)
- `release-file-loading`: Loading and validating predefined `#ModuleRelease` from `<name>_release.cue` files, including hybrid module resolution (registry import vs `--module` flag)

### Modified Capabilities

- `mod-vet`: `opm mod vet` uses `debugValues` from the module by default when no `-f` values flag is provided
- `cmd-structure`: New `internal/cmd/release/` package added; cluster-query commands migrate from `mod` to `rel`
- `release-building`: Loading IS building (promote-factory-engine architecture); `pkg/loader/` gains release-file loading functions; no separate builder phase

## Impact

- **New package**: `internal/cmd/release/` — all `release` subcommands
- **New functions**: `pkg/loader/release_file.go` — `LoadReleaseFile()`, `LoadModulePackage()`, `LoadReleasePackageWithValue()`
- **Modified packages**: `internal/cmd/mod/` (deprecation aliases for migrated commands), `internal/cmd/root.go` (register `release` group), `internal/cmdutil/` (new flag groups, `RenderFromReleaseFile()`, `DebugValues` on `RenderReleaseOpts`)
- **Unchanged packages**: `internal/cmd/mod/build.go`, `internal/cmd/mod/apply.go` — already use `cmdutil.RenderRelease()`, no changes needed
- **Modified specs**: `cmd-structure`, `mod-vet`, `release-building`, `release-file-loading`
- **No breaking changes**: `opm mod` commands retain current behavior; migrated commands emit deprecation notices referencing `opm release`

> Note: `internal/builder/`, `internal/pipeline/`, and `internal/loader/` no longer exist after `promote-factory-engine`. All references to those packages in this change have been updated to use `pkg/loader/` and `pkg/engine/` instead.
