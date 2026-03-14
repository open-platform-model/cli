## Context

`internal/runtime/modulerelease` defines `ModuleRelease` and `ReleaseMetadata` — pure data types with only `cuelang.org/go/cue` and `pkg/module` as dependencies. It is the most-imported internal package in the render group (15 files across 7 packages). Moving it to `pkg/render/` is a prerequisite for changes 3 (bundlerelease) and 4 (engine), both of which reference these types.

This change depends on change 1 (move-match-to-pkg-render) having already created the `pkg/render/` package.

## Goals / Non-Goals

**Goals:**
- Move `ModuleRelease`, `ReleaseMetadata`, and `NewModuleRelease` to `pkg/render/`
- Rename `ReleaseMetadata` → `ModuleReleaseMetadata` for disambiguation
- Update all 15 internal callers

**Non-Goals:**
- Change any behavior or add new methods
- Move `BundleRelease` (that's change 3)
- Move test fixtures

## Decisions

### Rename ReleaseMetadata to ModuleReleaseMetadata
In `internal/runtime/modulerelease`, the name `ReleaseMetadata` was unambiguous. In `pkg/render/`, it would collide conceptually with `BundleReleaseMetadata`. Rename to `ModuleReleaseMetadata` for clarity.

**Rationale**: Both metadata types will coexist in `pkg/render/`. Explicit naming prevents confusion.

### Update BundleRelease field type reference
`bundlerelease.BundleRelease` has a field `Releases map[string]*modulerelease.ModuleRelease`. After the move, `bundlerelease.go` imports `pkg/render` and the field becomes `map[string]*render.ModuleRelease`.

### Single-file approach
All modulerelease types fit in one file (`pkg/render/modulerelease.go`). No need to split.

## Risks / Trade-offs

- **[Medium risk] Large blast radius**: 15 files need import updates + the `ReleaseMetadata` → `ModuleReleaseMetadata` rename. All mechanical but easy to miss a reference. Use `go build ./...` to catch any misses.
- **[Low risk] Stale test references**: Test files use struct literals with field names. The rename from `ReleaseMetadata` to `ModuleReleaseMetadata` must be updated everywhere a struct literal is constructed.
