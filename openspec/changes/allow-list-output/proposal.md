## Why

The transformer workflow's CUE schema constrains `#transform.output` to `{...}` (a struct), which technically only permits single-resource or map-of-resource outputs. However, the Go executor (`internal/build/executor.go`) already handles three output shapes: lists, single structs, and maps. This mismatch means map outputs (like the PVC transformer producing one PVC per volume) work by coincidence rather than by contract. A module author who wants a transformer to produce a list output—or who wants to make it explicit that a component produces multiple resources—has no way to express this in the schema. Adding a `"transformer.opmodel.dev/list-output"` annotation to `#Component` provides a declarative opt-in for components whose resources naturally produce collections, making the implicit behavior explicit and enabling CUE-level validation of multi-resource transformer outputs.

## What Changes

- Add an optional `"transformer.opmodel.dev/list-output"?: bool` annotation to the `#Component` CUE definition (in `opmodel.dev/core`) that signals the component's transformers may produce list or map outputs
- Update the `#Transformer.#transform.output` CUE type to allow `{...} | [...]` when the annotation is true, so list outputs pass CUE validation
- Ensure the Go executor's existing three-case output decoding (list, single resource, map) is tested and documented to match the updated schema contract
- Update transformer matching or execution to read the annotation from the component and propagate it to the transformer context

**SemVer**: MINOR — adds a new optional annotation with no default behavior change; no existing behavior changes.

## Capabilities

### New Capabilities

- `list-output-schema`: CUE schema changes to `#Component` and `#Transformer` that allow components to declare multi-resource output support via the `"transformer.opmodel.dev/list-output"` annotation, and transformers to produce list or map outputs that pass CUE validation.

### Modified Capabilities

- `render-pipeline`: The executor's output decoding already handles lists and maps, but the contract changes—transformer outputs are now validated against the component's list-output annotation rather than being silently coerced. The `Resource` collection step needs to respect the annotation.

## Impact

- **CUE schema** (`opmodel.dev/core`): `#Component` gains `"transformer.opmodel.dev/list-output"?: bool` annotation; `#Transformer.#transform.output` type constraint loosens from `{...}` to `{...} | [...]` when the annotation is set
- **Go executor** (`internal/build/executor.go`): Existing list/map handling is already implemented; needs tests covering annotation propagation and edge cases
- **Go release builder** (`internal/build/release_builder.go`): May need to extract and propagate the annotation from loaded components
- **Existing modules**: No breakage — annotation is optional; omitting it preserves current struct-only behavior
- **Provider transformers**: PVC transformer and similar map-producing transformers can opt in to formalize what they already do
