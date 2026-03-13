## Why

The `internal/runtime/bundlerelease` package defines `BundleRelease` and `BundleReleaseMetadata` — pure domain types with zero CLI dependencies. The Kubernetes controller needs these types for its bundle reconciliation. This is the third step in building the shared `pkg/render/` package.

## What Changes

- Move `BundleRelease` and `BundleReleaseMetadata` from `internal/runtime/bundlerelease/` into `pkg/render/`
- Remove the `//nolint:revive` stutter comment on `BundleReleaseMetadata` (no longer stutters in `render` package)
- Update the `Releases` field type from `*modulerelease.ModuleRelease` to `*render.ModuleRelease` (already in `pkg/render/` after change 2)
- Update all 5 internal callers

## Capabilities

### New Capabilities

### Modified Capabilities

- `module-release-processing`: The `BundleRelease` runtime type moves from `internal/runtime/bundlerelease` to `pkg/render`.

## Impact

- **Files moved**: `internal/runtime/bundlerelease/release.go` → `pkg/render/bundlerelease.go`
- **Internal callers updated** (5 files):
  - `internal/engine/bundle_renderer.go`: parameter type
  - `internal/engine/matchplan_test.go`: struct literal
  - `internal/releasefile/get_release_file.go`: field type + struct literal
  - `internal/releaseprocess/bundle.go`: parameter type
  - `internal/releaseprocess/module_test.go`: struct literal
- **SemVer**: MINOR (new public API surface)
