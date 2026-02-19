## Why

`BuiltRelease` carries an internal `Metadata` grab-bag that mixes module-level
fields (`FQN`, `Version`, `Identity`) with release-level fields (`Name`,
`Namespace`, `ReleaseIdentity`). This forces `ToModuleMetadata()` to project
module data from a release struct — the very coupling the previous
`refactor-metadata-types` change was meant to eliminate. `BuiltRelease` should
carry the two clean public types (`ReleaseMetadata`, `ModuleMetadata`) directly,
populated by the builder from the CUE value it already holds.

## What Changes

- **Add** `ReleaseMetadata` and `ModuleMetadata` fields to `BuiltRelease`,
  replacing the internal `Metadata` field
- **Delete** `release.Metadata` struct (the internal grab-bag)
- **Split** `extractReleaseMetadata` in `release/metadata.go` into two functions:
  `extractReleaseMetadata` and `extractModuleMetadata`, each populating the
  corresponding public type from the CUE value
- **Delete** `BuiltRelease.ToReleaseMetadata()` — no longer needed; callers
  read `BuiltRelease.ReleaseMetadata` directly
- **Delete** `BuiltRelease.ToModuleMetadata()` — no longer needed; callers
  read `BuiltRelease.ModuleMetadata` directly
- **Simplify** `pipeline.go`: `RenderResult.Release` and `RenderResult.Module`
  are assigned directly from `BuiltRelease` fields, not from projection methods
- **Simplify** `transform/context.go`: `NewTransformerContext` reads from
  `BuiltRelease.ReleaseMetadata` and `BuiltRelease.ModuleMetadata` directly,
  removing the incorrect `rel.Metadata.Name` fallback
- **Update** all tests that construct `BuiltRelease{Metadata: release.Metadata{...}}`
  to use `ReleaseMetadata` and `ModuleMetadata` fields instead — and enrich
  `values_resolution_test.go` to assert module-level fields (FQN, Version)

No public API breakage — all changes are within `internal/build/` and its
sub-packages. SemVer: **PATCH**.

## Capabilities

### New Capabilities

_(none — this is a structural refactor within the existing render pipeline)_

### Modified Capabilities

- `render-pipeline`: The `BuiltRelease` internal type now carries `ReleaseMetadata`
  and `ModuleMetadata` directly. The data-model spec needs updating to reflect
  that `BuiltRelease` (the builder output) is the typed carrier of both metadata
  types, and that the internal `Metadata` struct no longer exists.

## Impact

- `internal/build/release/types.go` — structural change to `BuiltRelease`,
  deletion of `Metadata`, deletion of two projection methods
- `internal/build/release/metadata.go` — split one extraction function into two
- `internal/build/release/builder.go` — populate both fields on `BuiltRelease`
- `internal/build/pipeline.go` — assign fields directly, remove projection calls
- `internal/build/transform/context.go` — read directly from `BuiltRelease`
  fields, remove incorrect fallback
- `internal/build/pipeline_test.go` — update `BuiltRelease` construction
- `internal/build/transform/context_test.go` — update `BuiltRelease` construction
- `internal/build/transform/context_annotations_test.go` — update `BuiltRelease`
  construction
- `internal/build/transform/executor_test.go` — update `BuiltRelease` construction
- `internal/build/values_resolution_test.go` — update field paths + enrich
  assertions with FQN, Version, DefaultNamespace
