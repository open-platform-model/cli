## Why

`ModuleRelease` and `BundleRelease` types currently live in `pkg/render/`, bundled with rendering logic. The existing specs (`pkg-types`, `core-modulerelease`) planned separate `pkg/modulerelease/` and `pkg/bundlerelease/` packages, but having dedicated packages for 1-2 types each is over-segmentation. Colocating release types with their parent domain types (`pkg/module/`, `pkg/bundle/`) is simpler, reduces package count, and aligns with how consumers think about these concepts — a release *is* a module or bundle concern.

## What Changes

- Move `pkg/render/modulerelease.go` → `pkg/module/release.go` (types: `ModuleRelease`, `ModuleReleaseMetadata`, `NewModuleRelease`, accessor methods)
- Move `pkg/render/bundlerelease.go` → `pkg/bundle/release.go` (types: `BundleRelease`, `BundleReleaseMetadata`)
- Update all imports across `pkg/render/`, `internal/releasefile/`, `internal/workflow/`, `internal/cmd/module/` to reference `module.ModuleRelease` / `bundle.BundleRelease` instead of `render.ModuleRelease` / `render.BundleRelease`
- `pkg/bundle/` gains a new dependency on `pkg/module/` (for `BundleRelease.Releases map[string]*module.ModuleRelease`)
- Delete the original files from `pkg/render/`

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `pkg-types`: Changes `ModuleRelease` location from `pkg/modulerelease/` to `pkg/module/` and `BundleRelease` from `pkg/bundlerelease/` to `pkg/bundle/`
- `core-modulerelease`: Changes `ModuleRelease` location from `pkg/modulerelease/` to `pkg/module/`

## Impact

- **pkg/module/**: Gains `ModuleRelease`, `ModuleReleaseMetadata`, `NewModuleRelease()`, plus `MatchComponents()` and `ExecuteComponents()` accessor methods. No new external dependencies (already uses `cue`).
- **pkg/bundle/**: Gains `BundleRelease`, `BundleReleaseMetadata`. Adds new import of `pkg/module` (one-way, no cycle risk).
- **pkg/render/**: Loses type ownership, gains imports of `pkg/module` and `pkg/bundle`. All function signatures referencing these types update to cross-package references.
- **~8 files** in `internal/` need import path updates.
- **SemVer**: PATCH — types move packages but keep identical names and signatures; no external API existed for the `render` package types.
