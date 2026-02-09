## Why

The transformer workflow produces Kubernetes resources from OPM components. Some resources are inherently plural — `#VolumesResource`, `#ConfigMapsResource`, and `#SecretsResource` all use map-based `#spec` patterns where each entry becomes a separate Kubernetes object. Their transformers (PVC, ConfigMap, Secret) already produce map output (one K8s resource per map entry), and the Go executor already handles this. However, there is no signal on the component telling consumers whether to expect single or multi-resource output from its transformers.

The sibling catalog change (`catalog/openspec/changes/allow-list-output`) adds a `"transformer.opmodel.dev/list-output": true` annotation on plural resources. This annotation propagates to `#Component.metadata.annotations` via existing CUE inheritance. The CLI needs to extract this annotation during release building and propagate it through the transformer context so it is available to transformers and downstream consumers.

## What Changes

- Extract component annotations (specifically `"transformer.opmodel.dev/list-output"`) from `component.metadata.annotations` during release building
- Propagate annotations through `TransformerComponentMetadata` into the `#TransformerContext` so CUE transformers can access them via `#context.#componentMetadata.annotations`
- Ensure the Go executor's existing three-case output decoding (single resource, map of resources, list of resources) is tested and documented to match the multi-output contract
- No CUE schema changes needed in `opmodel.dev/core` — the existing `output: {...}` constraint already accepts map output, and the annotation lives on resources in the catalog, not on `#Component` directly

**SemVer**: MINOR — adds annotation extraction and propagation; no existing behavior changes.

**Sibling change**: `catalog/openspec/changes/allow-list-output` — adds the annotation to `#VolumesResource`, `#ConfigMapsResource`, and `#SecretsResource`; converts ConfigMap and Secret transformers to map output.

## Capabilities

### New Capabilities

- `list-output-schema`: CLI-side annotation extraction and propagation. The CLI reads `"transformer.opmodel.dev/list-output"` from `component.metadata.annotations` (set by plural resources in the catalog) and makes it available in the transformer context.

### Modified Capabilities

- `render-pipeline`: The executor's output decoding already handles single, map, and list output shapes. The contract is formalized — `TransformerComponentMetadata` gains an `Annotations` field that carries the annotation through to transformer execution context.

## Impact

- **Go release builder** (`internal/build/release_builder.go`): `extractComponent()` gains annotation extraction from `metadata.annotations`
- **Go context** (`internal/build/context.go`): `LoadedComponent` and `TransformerComponentMetadata` gain `Annotations` fields; `NewTransformerContext()` and `ToMap()` propagate them
- **Go executor** (`internal/build/executor.go`): No code changes needed — existing three-case output decoding already works. Needs test coverage for map and list cases.
- **CUE schema** (`opmodel.dev/core`): No changes needed — `output: {...}` already permits map output, and annotation inheritance on `#Component` already exists
- **Existing modules**: No breakage — annotation extraction is additive
- **Affected transformers** (in catalog): PVC, ConfigMap, and Secret transformers all produce map output; the annotation formalizes this
