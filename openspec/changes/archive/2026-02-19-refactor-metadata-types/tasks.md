## 1. Define New Types

- [x] 1.1 Add `ModuleMetadata` struct to `internal/build/module/types.go` with fields: Name, DefaultNamespace, FQN, Version, UUID, Labels, Annotations, Components — all with `json:"..."` tags
- [x] 1.2 Rename `ModuleReleaseMetadata` to `ReleaseMetadata` in `internal/build/release/types.go` with fields: Name, Namespace, UUID (was ReleaseIdentity), Labels, Annotations, Components — all with `json:"..."` tags. Remove ModuleName, Version, Identity fields.
- [x] 1.3 Delete `TransformerMetadata` struct and `ReleaseMetadataForTransformer()` method from `internal/build/release/types.go`

## 2. Update Projection Methods

- [x] 2.1 Replace `BuiltRelease.ToModuleReleaseMetadata(moduleName string)` with two methods: `ToReleaseMetadata() ReleaseMetadata` and `ToModuleMetadata(moduleName string) module.ModuleMetadata` in `internal/build/release/types.go`
- [x] 2.2 Populate all fields on both types from `release.Metadata` — `ReleaseMetadata.UUID` from `Metadata.ReleaseIdentity`, `ModuleMetadata.UUID` from `Metadata.Identity`, component names from `BuiltRelease.Components` map keys

## 3. Update Type Aliases and Re-exports

- [x] 3.1 Update `internal/build/release_adapter.go`: remove `ReleaseMetadata = release.Metadata` alias, rename `ModuleReleaseMetadata` alias to `ReleaseMetadata = release.ReleaseMetadata`, add `ModuleMetadata = module.ModuleMetadata` alias
- [x] 3.2 Remove `TransformerModuleReleaseMetadata` alias from `internal/build/transform/context.go`

## 4. Update RenderResult

- [x] 4.1 Change `RenderResult` in `internal/build/types.go` from `Release ModuleReleaseMetadata` to `Release ReleaseMetadata` + `Module ModuleMetadata`
- [x] 4.2 Update pipeline result construction in `internal/build/pipeline.go` to populate both `Release` and `Module` fields

## 5. Update TransformerContext

- [x] 5.1 Replace `ModuleReleaseMetadata *TransformerModuleReleaseMetadata` field in `TransformerContext` with `ModuleMetadata *module.ModuleMetadata` and `ReleaseMetadata *release.ReleaseMetadata` in `internal/build/transform/context.go`
- [x] 5.2 Update `NewTransformerContext()` to populate both fields from `BuiltRelease`
- [x] 5.3 Update `ToMap()` to compose CUE output from both types (same output shape: `#moduleReleaseMetadata` with name, namespace, fqn, version, identity, labels)

## 6. Update Consumers — Command Layer

- [x] 6.1 Update `internal/cmd/mod/apply.go`: change `result.Release.Version` → `result.Module.Version`, `result.Release.ReleaseIdentity` → `result.Release.UUID`, `result.Release.ModuleName` → `result.Module.Name`, `result.Release.Identity` → `result.Module.UUID`
- [x] 6.2 Update `internal/cmdutil/output.go`: change `result.Release.Version` → `result.Module.Version`

## 7. Update Consumers — Kubernetes Package

- [x] 7.1 Update `kubernetes.Apply()` signature: `meta build.ModuleReleaseMetadata` → `meta build.ReleaseMetadata` in `internal/kubernetes/apply.go`
- [x] 7.2 Update `kubernetes.Diff()` signature: same rename in `internal/kubernetes/diff.go`
- [x] 7.3 Update `kubernetes.DiffPartial()` signature: same rename in `internal/kubernetes/diff.go`

## 8. Update Tests

- [x] 8.1 Update `internal/build/pipeline_test.go`: split assertions across `Module` and `Release` fields on `RenderResult`
- [x] 8.2 Update `internal/build/transform/context_test.go`: use `ModuleMetadata` and `ReleaseMetadata` instead of `TransformerModuleReleaseMetadata`
- [x] 8.3 Update `internal/build/transform/context_annotations_test.go`: same changes
- [x] 8.4 Update `internal/cmdutil/render_test.go`: use new `ReleaseMetadata` type and add `Module` field
- [x] 8.5 Update `internal/kubernetes/integration_test.go`: use `ReleaseMetadata` type (file is actually diff_integration_test.go — no separate integration_test.go exists)
- [x] 8.6 Update `internal/kubernetes/diff_integration_test.go`: use `ReleaseMetadata` type
- [x] 8.7 Update `tests/integration/deploy/main.go`: use `ReleaseMetadata` with `UUID` field
- [x] 8.8 Update `tests/integration/inventory-ops/main.go`: use `ReleaseMetadata` with `UUID` field

## 9. Validation

- [x] 9.1 Run `task test` — all unit tests pass
- [x] 9.2 Run `task check` — fmt + vet + test pass (pre-existing lint warnings remain, none from this change)
