## Why

The build pipeline's metadata types are tangled: `ModuleReleaseMetadata` mixes module-level and release-level concerns into a single struct, `TransformerMetadata` duplicates fields with confusing renames (`ReleaseIdentity` → `Identity`), and the `Identity` field means different things depending on which type you're looking at. This makes the types hard to reuse, hard to reason about, and out of alignment with the render-pipeline spec (which already defines a separate `Module` field on `RenderResult`).

Splitting into two focused types — `ModuleMetadata` (module identity, version, FQN) and `ReleaseMetadata` (release name, namespace, release UUID) — eliminates the confusion, makes `TransformerMetadata` obsolete, and brings the implementation in line with the existing spec contract.

## What Changes

- **Rename** `release.ModuleReleaseMetadata` to `release.ReleaseMetadata` with fields: `Name`, `Namespace`, `UUID` (release UUID, was `ReleaseIdentity`), `Labels`, `Annotations`, `Components`. All fields get `json:"..."` tags.
- **Add** `module.ModuleMetadata` with fields: `Name`, `DefaultNamespace`, `FQN`, `Version`, `UUID` (module UUID, was `Identity`), `Labels`, `Annotations`, `Components`. All fields get `json:"..."` tags.
- **Delete** `release.TransformerMetadata` — obsoleted by `ModuleMetadata` + `ReleaseMetadata` together containing all its fields.
- **Delete** `release.ReleaseMetadataForTransformer()` projection method.
- **Split** `RenderResult.Release` into two fields: `RenderResult.Release ReleaseMetadata` and `RenderResult.Module ModuleMetadata`.
- **Remove** `build.ReleaseMetadata` alias for `release.Metadata` (internal type no longer re-exported; name taken by new type).
- **Update** `TransformerContext` to hold both `ModuleMetadata` and `ReleaseMetadata` instead of `TransformerModuleReleaseMetadata`.
- **Update** all consumers to access module-level fields (Version, FQN, module UUID) from `result.Module` instead of `result.Release`.
- `Annotations` fields start empty — CUE extraction is a separate future change.

## Capabilities

### New Capabilities

_None — this is a structural refactoring, not a new capability._

### Modified Capabilities

- `render-pipeline`: The `RenderResult` data model changes from a single `Release` field to separate `Module` and `Release` fields. `ModuleMetadata` is expanded with `UUID`, `FQN`, `DefaultNamespace`, and `Annotations`. A new `ReleaseMetadata` type is introduced. `TransformerMetadata` is removed.

## Impact

- **Packages modified**: `internal/build/release`, `internal/build/module`, `internal/build`, `internal/build/transform`, `internal/cmd/mod`, `internal/cmdutil`, `internal/kubernetes`
- **Type aliases updated**: `internal/build/release_adapter.go` — remove old alias, add new re-exports
- **Function signatures changed**: `kubernetes.Apply()`, `kubernetes.Diff()`, `kubernetes.DiffPartial()` — parameter type rename
- **Test files updated**: ~8 files with struct literal changes (mechanical)
- **SemVer**: PATCH — all affected types are internal (no public API)
- **Breaking**: None — all types are in `internal/` packages
