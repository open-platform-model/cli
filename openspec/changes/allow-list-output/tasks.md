## 1. CUE Schema Updates (opmodel.dev/core)

- [ ] 1.1 Update `#Transformer.#transform.output` from `{...}` to `{...} | [...{...}]` in `transformer.cue`
- [ ] 1.2 Add `annotations?: [string]: bool` to `#TransformerContext.#componentMetadata` in `transformer.cue`
- [ ] 1.3 Run `cue vet ./...` to confirm schema changes validate

> **Sibling change**: The `"transformer.opmodel.dev/list-output": true` annotation on `#VolumesResource` is added in the catalog repo (`catalog/openspec/changes/allow-list-output`). That change sets the annotation on the resource; it propagates to `#Component.metadata.annotations` via existing CUE comprehension. This CLI change consumes it.

## 2. LoadedComponent Annotation Extraction

- [ ] 2.1 Add `Annotations map[string]string` field to `LoadedComponent` in `internal/build/module.go`
- [ ] 2.2 Extract `metadata.annotations` in `extractComponent()` in `internal/build/release_builder.go` — iterate annotation fields and populate `comp.Annotations`
- [ ] 2.3 Initialize `Annotations` to empty map (not nil) when no annotations exist

## 3. TransformerContext Annotation Propagation

- [ ] 3.1 Add `Annotations map[string]string` field to `TransformerComponentMetadata` in `internal/build/context.go`
- [ ] 3.2 Copy `component.Annotations` to `TransformerComponentMetadata.Annotations` in `NewTransformerContext()`
- [ ] 3.3 Include `annotations` in the `componentMetadata` map in `ToMap()` (omit when empty, matching labels/resources/traits pattern)
- [ ] 3.4 Include `annotations` in the `compMetaMap` in `executor.go` `executeJob()` FillPath block (matching the existing pattern for labels/resources/traits)

## 4. Tests

- [ ] 4.1 Add unit test for `extractComponent()`: component with `"transformer.opmodel.dev/list-output": true` annotation — verify `LoadedComponent.Annotations` is populated
- [ ] 4.2 Add unit test for `extractComponent()`: component without annotations — verify `Annotations` is empty map
- [ ] 4.3 Add unit test for `NewTransformerContext()`: verify annotations propagate from `LoadedComponent` to `TransformerComponentMetadata`
- [ ] 4.4 Add unit test for `ToMap()`: verify annotations included in output map when present, omitted when empty
- [ ] 4.5 Add executor test: transformer producing list output `[{apiVersion: ...}, ...]` — verify multiple resources decoded
- [ ] 4.6 Add executor test: transformer producing map output `{name: {apiVersion: ...}}` — verify multiple resources decoded (formalizes existing behavior)

## 5. Validation

- [ ] 5.1 Run `task fmt` — all Go files formatted
- [ ] 5.2 Run `task test` — all tests pass
- [ ] 5.3 Run `task check` — fmt + vet + test all pass
- [ ] 5.4 End-to-end: build a test module with `#Volumes` (e.g., `testing/jellyfin`) and confirm annotation appears in transformer context and PVC resources are correctly produced

## 6. Housekeeping

- [ ] 6.1 Review `TODO.md` — check if any items are addressed or impacted by this change and update accordingly
