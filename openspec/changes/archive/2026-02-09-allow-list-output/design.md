## Context

The transformer workflow produces Kubernetes resources from OPM components. Each transformer's `#transform` function receives a component and context, then returns an `output`. The CUE schema constrains `output: {...}` (a struct), which accepts both single resources and maps of resources — a keyed struct where each value is a K8s object is still a struct.

The Go executor (`internal/build/executor.go:226-266`) already handles three output shapes:

1. **Single resource** — struct with `apiVersion` (e.g., Deployment)
2. **Map of resources** — struct without `apiVersion`, keyed by name (e.g., PVC per volume)
3. **List of resources** — `[...]` array of resources (not currently exercised by any transformer)

The sibling catalog change (`catalog/openspec/changes/allow-list-output`) has:
- Added `"transformer.opmodel.dev/list-output": true` annotations to three plural resources: `#VolumesResource`, `#ConfigMapsResource`, `#SecretsResource`
- Converted ConfigMap and Secret transformers from single-resource to map output
- The annotation propagates to `#Component.metadata.annotations` automatically via existing CUE comprehension in `component.cue`

This CLI change needs to extract that annotation during release building and propagate it through the transformer context.

**Relevant code paths:**

- `internal/build/release_builder.go:134-190` — `extractComponent()` builds `LoadedComponent`
- `internal/build/context.go` — `TransformerContext`, `TransformerComponentMetadata`, `NewTransformerContext()`, `ToMap()`
- `internal/build/executor.go:136-269` — `executeJob()` injects component/context via FillPath, decodes output
- `internal/build/matcher.go` — matching logic (unaffected)

## Goals / Non-Goals

**Goals:**

- Extract component annotations from `metadata.annotations` during release building
- Propagate annotations through `TransformerComponentMetadata` into the CUE transformer context
- Add test coverage for map output scenarios (now exercised by PVC, ConfigMap, and Secret transformers)
- Formalize the executor's existing three-case output decoding as the documented contract

**Non-Goals:**

- Changing the CUE `#Transformer.#transform.output` constraint — `{...}` already permits map output, which is the only multi-output shape used in practice
- Changing the matcher logic — the annotation is about output shape, not matching criteria
- Supporting list (`[...]`) output in CUE — no transformer uses this today; can be added later if needed
- Modifying existing transformers — the catalog change already handles this

## Decisions

### 1. Annotation lives on resources, propagates via existing CUE inheritance

**Decision:** The `"transformer.opmodel.dev/list-output": true` annotation is set on plural resources in the catalog (e.g., `#VolumesResource`, `#ConfigMapsResource`, `#SecretsResource`). It propagates to `#Component.metadata.annotations` via the existing annotation comprehension in `component.cue`. The CLI reads it from the propagated `metadata.annotations` on the loaded component.

**Rationale:** Resources are the entities that are intrinsically plural — they know their `#spec` uses a map pattern. The composition struct (`#Volumes`, `#ConfigMaps`, etc.) and the component itself don't need to know. The existing annotation inheritance mechanism handles propagation without any changes to `#Component` or core CUE definitions.

**Alternative considered:** Adding a top-level `listOutput` field to `#Component` — rejected because it would require modifying a closed struct in core, and each composition struct would need to manually set it. The annotation approach works with no core schema changes.

### 2. Propagate annotations via `TransformerComponentMetadata`

**Decision:** Add an `Annotations map[string]string` field to `LoadedComponent` and `TransformerComponentMetadata`. Extract annotations in `extractComponent()` from `metadata.annotations`. Propagate through `NewTransformerContext()` into `ToMap()` for CUE FillPath injection.

**Implementation path (Go `internal/build/`):**

1. `module.go:LoadedComponent` — add `Annotations map[string]string`
2. `release_builder.go:extractComponent()` — iterate `metadata.annotations` fields, populate `comp.Annotations`
3. `context.go:TransformerComponentMetadata` — add `Annotations map[string]string`
4. `context.go:NewTransformerContext()` — copy `component.Annotations` to metadata
5. `context.go:ToMap()` — include `"annotations"` in component metadata map when non-empty
6. `executor.go` — no changes needed; output decoding already handles all three cases

**Rationale:** This follows the established pattern for labels, resources, and traits — extract during component loading, carry through context, inject via FillPath. Transformers that need the annotation can access `#context.#componentMetadata.annotations`.

**Alternative considered:** Directly inspecting the component CUE value for the annotation in the executor — rejected because it bypasses the established context injection pattern and would not be visible to CUE transformers.

### 3. No CUE schema changes needed for `output` constraint

**Decision:** Keep `#Transformer.#transform.output` as `{...}`. Do not loosen to `{...} | [...{...}]`.

**Rationale:** The existing `{...}` constraint already accepts both single resources and maps of resources — a keyed struct is still a struct. All three plural transformers (PVC, ConfigMap, Secret) produce map output that passes CUE validation under `{...}`. The stale comment "Must be a single provider-specific resource" in `transformer.cue` should be updated, but the constraint itself is correct.

List output (`[...]`) is not exercised by any current transformer. If needed in the future, the constraint can be loosened then. Per Principle VII (YAGNI), we don't add it now.

**Alternative considered:** Proactively loosening to `{...} | [...{...}]` — rejected because no transformer uses list output, and the constraint change would be speculative complexity.

### 4. Annotation value stored as string in Go

**Decision:** Store annotation values as `map[string]string` in Go, even though the CUE type is `bool`. The CUE value `true` is extracted as the string `"true"`.

**Rationale:** This is consistent with Kubernetes annotation conventions where all values are strings. It avoids type-switching complexity in Go and keeps the `Annotations` map uniform. Consumers check `annotations["transformer.opmodel.dev/list-output"] == "true"`.

## Risks / Trade-offs

**[Risk] Annotation is convention, not enforced** → The annotation documents intent but doesn't constrain output shape. A transformer matched to a component without the annotation could still produce map output and it would work. This is acceptable — the annotation is for consumer awareness, not enforcement.

**[Risk] Stale comment in `transformer.cue`** → The comment "Must be a single provider-specific resource" is incorrect for 3 of 12 transformers. The catalog should update this comment. Not a blocker for the CLI change but should be tracked.

**[Trade-off] Generic `Annotations` map vs. dedicated `ListOutput` field** → Using a generic map is more extensible but requires string comparison instead of boolean check. We chose extensibility — future annotations can reuse the same propagation path without adding new fields.

**[Trade-off] Not adding list output support now** → We only formalize what exists (map output). List output can be added later if a transformer needs it. This follows YAGNI.
