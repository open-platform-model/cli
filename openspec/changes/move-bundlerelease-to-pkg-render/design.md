## Context

`internal/runtime/bundlerelease` defines `BundleRelease` and `BundleReleaseMetadata`. It depends on `internal/runtime/modulerelease` (for the `Releases` map field type) and `pkg/bundle`. After change 2, `modulerelease` types are already in `pkg/render/`, so `bundlerelease` can now move there cleanly.

## Goals / Non-Goals

**Goals:**
- Move `BundleRelease` and `BundleReleaseMetadata` to `pkg/render/bundlerelease.go`
- Remove `//nolint:revive` stutter comment
- Update all 5 internal callers

**Non-Goals:**
- Add any new functionality
- Change the `Releases` map type (it's already `*render.ModuleRelease` after change 2)

## Decisions

### Same-package field reference simplification
The `Releases` field was `map[string]*modulerelease.ModuleRelease`. After both types are in `pkg/render/`, it becomes `map[string]*ModuleRelease` — a same-package reference. This simplifies the code.

### Remove stutter suppression
`bundlerelease.BundleReleaseMetadata` stuttered. `render.BundleReleaseMetadata` does not. Remove the `//nolint:revive` annotation.

## Risks / Trade-offs

- **[Low risk]**: 5 files to update, all mechanical.
