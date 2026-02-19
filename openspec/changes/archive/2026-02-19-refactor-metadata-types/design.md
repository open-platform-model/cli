## Context

The render pipeline produces metadata about the module and its release through several types defined in `internal/build/release/types.go`. Today there are three metadata types with overlapping, ambiguous responsibilities:

- **`release.ModuleReleaseMetadata`** — mixes module fields (ModuleName, Version, Identity as module UUID) with release fields (Name, Namespace, ReleaseIdentity). Re-exported as `build.ModuleReleaseMetadata`. Lives on `RenderResult.Release`.
- **`release.TransformerMetadata`** — subset of release metadata projected for transformer execution. Renames `ReleaseIdentity` → `Identity`, creating ambiguity. Re-exported as `transform.TransformerModuleReleaseMetadata`.
- **`release.Metadata`** — internal CUE extraction target with all fields. Re-exported as `build.ReleaseMetadata`.

The internal `release.Metadata` is projected into the other two types via `ToModuleReleaseMetadata()` and `ReleaseMetadataForTransformer()`. The `Identity` field means "module UUID" on `ModuleReleaseMetadata` but "release UUID" on `TransformerMetadata`.

The existing render-pipeline data model spec (`openspec/specs/render-pipeline/data-model.md`) already defines `RenderResult.Module ModuleMetadata` as the intended shape, but the implementation hasn't caught up yet.

## Goals / Non-Goals

**Goals:**

- Split metadata into two focused types: `module.ModuleMetadata` (module identity) and `release.ReleaseMetadata` (release identity)
- Eliminate `TransformerMetadata` by ensuring the two new types collectively cover all its fields
- Add `json:"..."` tags to both types for future serialization
- Add `Annotations` and `Components` fields to both types (annotations start empty, populated later)
- Bring `RenderResult` in line with the existing render-pipeline spec (separate `Module` and `Release` fields)
- Remove the re-export of `release.Metadata` as `build.ReleaseMetadata` (name now taken by the new type)

**Non-Goals:**

- CUE schema changes — annotation extraction from CUE is a separate change
- Changing the CUE `#TransformerContext` definition — `ToMap()` continues to produce the same CUE output
- Modifying `inventory.ModuleMetadata` or `inventory.ReleaseMetadata` — those are separate types in a separate package

## Decisions

### D1: Two types replace three

**Decision**: Create `module.ModuleMetadata` and rename `release.ModuleReleaseMetadata` to `release.ReleaseMetadata`. Delete `release.TransformerMetadata`.

**Rationale**: Module-level fields (Name, FQN, Version, module UUID) belong on a module type. Release-level fields (release name, namespace, release UUID) belong on a release type. The transformer context doesn't need its own type — it can pull from both.

**Alternatives considered**:

- *Keep a single combined type*: Rejected because it perpetuates the confusion about which UUID is which and what "Name" means.
- *Three types (add ModuleMetadata, keep TransformerMetadata)*: Rejected because TransformerMetadata becomes fully redundant once the other two types exist.

### D2: `UUID` field naming for both types

**Decision**: Both types use `UUID string` as the field name. `ModuleMetadata.UUID` holds the module identity UUID. `ReleaseMetadata.UUID` holds the release identity UUID (deterministic UUID5 from fqn+name+namespace).

**Rationale**: `UUID` is unambiguous within each type's context. The current naming (`Identity` meaning different things on different types) was the primary source of confusion.

### D3: Keep `release.Metadata` as the internal CUE extraction target

**Decision**: The internal `release.Metadata` struct remains unchanged. It continues to be the target for CUE field extraction in `extractReleaseMetadata()`. Projection methods on `BuiltRelease` create the two new public types from it.

**Rationale**: The internal type serves a different purpose (1:1 CUE extraction). Changing it would complicate the extraction code for no benefit. The re-export as `build.ReleaseMetadata` is removed since that alias name is now taken by the new public type.

### D4: `TransformerContext` holds both types

**Decision**: `TransformerContext` replaces its single `ModuleReleaseMetadata *TransformerModuleReleaseMetadata` field with two fields: `ModuleMetadata *module.ModuleMetadata` and `ReleaseMetadata *release.ReleaseMetadata`. The `ToMap()` method continues to produce the same CUE output structure by pulling from both types.

**Rationale**: This keeps the CUE contract unchanged while eliminating `TransformerMetadata`. The `ToMap()` function is the bridge between Go types and CUE structure — it can compose output from multiple sources.

### D5: `RenderResult` gets two metadata fields

**Decision**: `RenderResult` changes from `Release ModuleReleaseMetadata` to `Release ReleaseMetadata` + `Module ModuleMetadata`.

**Rationale**: Aligns with the existing render-pipeline data model spec. Consumers that need module-level data (version, FQN, module UUID) access `result.Module`. Consumers that need release-level data (release name, namespace, release UUID) access `result.Release`.

### D6: Components field on both types

**Decision**: Both `ModuleMetadata` and `ReleaseMetadata` carry `Components []string`. The same component name list is placed on both.

**Rationale**: Components are a property of both the module (which components exist) and the release (which components were rendered). Having it on both allows consumers to access it through whichever type they already hold.

## Risks / Trade-offs

- **Risk: Mechanical churn across ~15 files** → Mitigation: All changes are type renames and field path updates. No behavioral changes. Existing tests catch regressions.
- **Risk: `release.ReleaseMetadata` stutters as a package-qualified name** → Accepted: The re-export as `build.ReleaseMetadata` doesn't stutter, and that's how most consumers reference it. Mild stutter within the release package is acceptable.
- **Risk: Consumers forget to update field paths (e.g., `result.Release.Version` → `result.Module.Version`)** → Mitigation: This is a compile-time error. The `ReleaseMetadata` type doesn't have a `Version` field, so any stale access fails to compile.
- **Trade-off: Annotations start empty** → Accepted: Adding CUE extraction is a separate concern. The field exists for future use and for types that populate annotations from other sources (e.g., transformer context component metadata already has annotations).
