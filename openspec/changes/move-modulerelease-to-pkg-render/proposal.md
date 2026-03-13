## Why

The `internal/runtime/modulerelease` package defines `ModuleRelease` and `ReleaseMetadata` — pure domain types with zero CLI dependencies. The Kubernetes controller needs these same types for its render pipeline. This is the second step in building the shared `pkg/render/` package.

## What Changes

- Move `ModuleRelease`, `ReleaseMetadata`, and `NewModuleRelease` from `internal/runtime/modulerelease/` into `pkg/render/`
- **Rename** `ReleaseMetadata` to `ModuleReleaseMetadata` for clarity when it shares a package with `BundleReleaseMetadata`
- Update all 15 internal callers across 7 packages to import from `pkg/render/`

## Capabilities

### New Capabilities

### Modified Capabilities

- `module-release-processing`: The `ModuleRelease` runtime type moves from `internal/runtime/modulerelease` to `pkg/render`, and `ReleaseMetadata` is renamed to `ModuleReleaseMetadata`.

## Impact

- **Files moved**: `internal/runtime/modulerelease/release.go` → `pkg/render/modulerelease.go`
- **Type rename**: `ReleaseMetadata` → `ModuleReleaseMetadata` (15 files affected)
- **Internal callers updated** (15 files across 7 packages):
  - `internal/engine/` (3 files): execute.go, module_renderer.go, matchplan_test.go
  - `internal/runtime/bundlerelease/release.go`: field type reference
  - `internal/releasefile/get_release_file.go`: constructor call + struct literal
  - `internal/releaseprocess/` (3 files): module.go, synthesize.go, module_test.go
  - `internal/workflow/render/` (5 files): types.go, render.go, values.go, render_test.go, log_output_test.go
  - `internal/workflow/apply/apply_test.go`
  - `internal/cmd/module/verbose_output_test.go`
- **No breaking public API changes** — `internal/runtime/modulerelease` was never importable externally
- **SemVer**: MINOR (new public API surface)
