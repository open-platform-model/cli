## 1. Refactor `release/types.go` — Replace `Metadata` with typed fields

- [x] 1.1 Add `ReleaseMetadata ReleaseMetadata` and `ModuleMetadata module.ModuleMetadata` fields to `BuiltRelease` struct
- [x] 1.2 Remove the `Metadata Metadata` field from `BuiltRelease`
- [x] 1.3 Delete the `Metadata` struct entirely
- [x] 1.4 Delete `BuiltRelease.ToReleaseMetadata()` method
- [x] 1.5 Delete `BuiltRelease.ToModuleMetadata()` method

## 2. Refactor `release/metadata.go` — Split extraction into two functions

- [x] 2.1 Rename `extractReleaseMetadata` to only extract release-level fields: Name (from opts), Namespace (from opts), UUID (from `#opmReleaseMeta.identity`), Labels (from `#opmReleaseMeta.labels` with fallback to `metadata.labels`), return `ReleaseMetadata`
- [x] 2.2 Add `extractModuleMetadata(v cue.Value) module.ModuleMetadata` that extracts: Name (from `metadata.name`), DefaultNamespace (from `metadata.defaultNamespace`), FQN (from `#opmReleaseMeta.fqn`, fallback `metadata.fqn`, fallback `metadata.apiVersion`), Version (from `#opmReleaseMeta.version`, fallback `metadata.version`), UUID (from `metadata.identity`), Labels (same source as release labels for behavioral parity)
- [x] 2.3 Split `extractMetadataFallback` into `extractReleaseMetadataFallback` and `extractModuleMetadataFallback` to match the two new functions

## 3. Update `release/builder.go` — Populate both fields on `BuiltRelease`

- [x] 3.1 Replace the single `extractReleaseMetadata` call with calls to both `extractReleaseMetadata` and `extractModuleMetadata`
- [x] 3.2 After extracting components, collect component names into `[]string` and set `ReleaseMetadata.Components` and `ModuleMetadata.Components` on the returned `BuiltRelease`
- [x] 3.3 Update the `BuiltRelease` literal in the return statement to use the two new fields

## 4. Update `pipeline.go` — Read fields directly from `BuiltRelease`

- [x] 4.1 Update debug log lines (`release.Metadata.Name`, `release.Metadata.Namespace`) to use `release.ReleaseMetadata.Name` and `release.ReleaseMetadata.Namespace`
- [x] 4.2 Replace `release.ToReleaseMetadata()` with `release.ReleaseMetadata` in the `RenderResult` literal
- [x] 4.3 Replace `release.ToModuleMetadata(moduleMeta.Name, moduleMeta.DefaultNamespace)` with `release.ModuleMetadata` in the `RenderResult` literal

## 5. Update `transform/context.go` — Read directly from `BuiltRelease` fields

- [x] 5.1 Replace `rel.ToModuleMetadata(rel.Metadata.Name, "")` with a copy of `rel.ModuleMetadata`
- [x] 5.2 Replace `rel.Metadata.Name` with `rel.ReleaseMetadata.Name` for `TransformerContext.Name`
- [x] 5.3 Replace `rel.Metadata.Namespace` with `rel.ReleaseMetadata.Namespace` for `TransformerContext.Namespace`
- [x] 5.4 Replace the `rel.ToReleaseMetadata()` call with a copy of `rel.ReleaseMetadata`

## 6. Update unit tests — `internal/build/`

- [x] 6.1 `pipeline_test.go`: Replace `Metadata: release.Metadata{...}` with `ReleaseMetadata: release.ReleaseMetadata{...}` and `ModuleMetadata: module.ModuleMetadata{...}`, remove `rel.ToModuleMetadata(...)` call, assert fields directly from struct
- [x] 6.2 `transform/context_test.go`: Replace `Metadata: release.Metadata{...}` with `ReleaseMetadata: release.ReleaseMetadata{...}` and `ModuleMetadata: module.ModuleMetadata{...}`
- [x] 6.3 `transform/context_annotations_test.go`: Replace `Metadata: release.Metadata{...}` with `ReleaseMetadata: release.ReleaseMetadata{...}` and `ModuleMetadata: module.ModuleMetadata{...}` in both test functions
- [x] 6.4 `transform/executor_test.go`: Replace `Metadata: release.Metadata{...}` with `ReleaseMetadata: release.ReleaseMetadata{...}` and `ModuleMetadata: module.ModuleMetadata{...}` in both test functions

## 7. Enrich `values_resolution_test.go`

- [x] 7.1 `TestBuild_StubsValuesCue_WhenValuesFlagsprovided`: Change `release.Metadata.Name` → `release.ReleaseMetadata.Name`; add assertions for `release.ModuleMetadata.Name` (`"test-module-values-only"`), `release.ModuleMetadata.FQN` (`"example.com/test-module-values-only@v0#test-module-values-only"`), `release.ModuleMetadata.Version` (`"1.0.0"`), `release.ModuleMetadata.DefaultNamespace` (`"default"`)
- [x] 7.2 `TestBuild_NoValuesCue_WithValuesFlag_Succeeds`: Same field path update and same module metadata assertions using test-module-no-values fixture values (`name: "test-module-no-values"`, same FQN pattern, version `"1.0.0"`, defaultNamespace `"default"`)
- [x] 7.3 `TestBuild_WithValuesCue_NoValuesFlag_Succeeds`: Same field path update and module metadata assertions using test-module fixture values (`name: "test-module"`, `fqn: "example.com/test-module@v0#test-module"`, version `"1.0.0"`, defaultNamespace `"default"`)

## 8. Validation

- [x] 8.1 Run `task test` — all tests must pass
- [x] 8.2 Run `task check` — no new lint warnings introduced
