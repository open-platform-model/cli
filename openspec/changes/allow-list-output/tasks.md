## 1. LoadedComponent Annotation Extraction

- [ ] 1.1 Add `Annotations map[string]string` field to `LoadedComponent` in `internal/build/module.go`
- [ ] 1.2 Extract `metadata.annotations` in `extractComponent()` in `internal/build/release_builder.go` — iterate annotation fields and populate `comp.Annotations` (convert bool values to string)
- [ ] 1.3 Initialize `Annotations` to empty map (not nil) when no annotations exist

> **Sibling change**: The `"transformer.opmodel.dev/list-output": true` annotation is set on plural resources (`#VolumesResource`, `#ConfigMapsResource`, `#SecretsResource`) in the catalog repo (`catalog/openspec/changes/allow-list-output`). It propagates to `#Component.metadata.annotations` via existing CUE comprehension. This CLI change reads it from there.

## 2. TransformerContext Annotation Propagation

- [ ] 2.1 Add `Annotations map[string]string` field to `TransformerComponentMetadata` in `internal/build/context.go`
- [ ] 2.2 Copy `component.Annotations` to `TransformerComponentMetadata.Annotations` in `NewTransformerContext()`
- [ ] 2.3 Include `annotations` in the `componentMetadata` map in `ToMap()` (omit when empty, matching labels/resources/traits pattern)
- [ ] 2.4 Include `annotations` in the `compMetaMap` in `executor.go` `executeJob()` FillPath block (matching the existing pattern for labels/resources/traits)

## 3. Tests

- [ ] 3.1 Add unit test for `extractComponent()`: component with `"transformer.opmodel.dev/list-output": true` annotation — verify `LoadedComponent.Annotations` contains `"transformer.opmodel.dev/list-output": "true"`
- [ ] 3.2 Add unit test for `extractComponent()`: component without annotations — verify `Annotations` is empty map
- [ ] 3.3 Add unit test for `NewTransformerContext()`: verify annotations propagate from `LoadedComponent` to `TransformerComponentMetadata`
- [ ] 3.4 Add unit test for `ToMap()`: verify annotations included in output map when present, omitted when empty
- [ ] 3.5 Add executor test: transformer producing map output `{name: {apiVersion: ...}}` — verify multiple resources decoded (formalizes existing behavior for PVC/ConfigMap/Secret transformers)
- [ ] 3.6 Add executor test: transformer producing single resource output — verify single resource decoded (regression test)

## 4. Validation

- [ ] 4.1 Run `task fmt` — all Go files formatted
- [ ] 4.2 Run `task test` — all tests pass
- [ ] 4.3 Run `task check` — fmt + vet + test all pass
- [ ] 4.4 End-to-end: build a test module with `#Volumes` and `#ConfigMaps` (e.g., `testing/jellyfin`) after updating catalog dependency — confirm annotation appears in transformer context and map resources (PVCs, ConfigMaps) are correctly produced

## 5. Housekeeping

- [ ] 5.1 Review `TODO.md` — check if any items are addressed or impacted by this change and update accordingly
