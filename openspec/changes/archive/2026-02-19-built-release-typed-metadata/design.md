## Context

`BuiltRelease` (in `internal/build/release/types.go`) is the output of
`Builder.Build()` — the fully evaluated CUE module with concrete values injected.
It currently carries an internal `Metadata` struct that is a grab-bag of both
module-level fields (`FQN`, `Version`, `Identity`) and release-level fields
(`Name`, `Namespace`, `ReleaseIdentity`, `Labels`).

This mixed struct is the source of a coupling bug: `ToModuleMetadata()` projects
module-level metadata *from a release struct*, which means module metadata is
derived from release internals rather than from the module itself. The field
`rel.Metadata.Name` is the *release* name, not the canonical *module* name —
making `NewTransformerContext` silently use the wrong value when `--name` overrides
the default.

The previous `refactor-metadata-types` change introduced the public types
`ReleaseMetadata` and `ModuleMetadata` on `RenderResult`, but left `BuiltRelease`
carrying the old internal struct — meaning the projection methods
(`ToReleaseMetadata`, `ToModuleMetadata`) are still bridges over the same coupling.

The fix: populate `BuiltRelease` with the two clean public types directly, built
from the CUE value that `Builder.Build()` already produces. The projection methods
and the `Metadata` struct are then deleted entirely.

## Goals / Non-Goals

**Goals:**

- `BuiltRelease` carries `ReleaseMetadata` and `ModuleMetadata` as direct fields
- Module metadata is extracted from the CUE value by the `release` package itself,
  not inferred from release fields
- `NewTransformerContext` reads the correct module name (from `ModuleMetadata.Name`)
  without any workaround
- `pipeline.go` populates `RenderResult.Release` and `RenderResult.Module` by
  direct field access, not method calls
- `task test` passes with no new failures

**Non-Goals:**

- Changing the public `RenderResult` shape (already correct from prior change)
- Changing how `ModuleMetadata` fields are sourced from CUE (same CUE paths,
  same fallbacks — behavioral parity)
- Splitting module labels from release labels (same labels map used for both,
  as before)
- Extracting `Annotations` from CUE (still empty — future change)

## Decisions

### D1: Extract module metadata inside `Builder.Build()`, not in `pipeline.go`

**Decision:** Add `extractModuleMetadata(v cue.Value) module.ModuleMetadata`
alongside `extractReleaseMetadata` in `release/metadata.go`. Both are called
from `builder.go` before constructing `BuiltRelease`.

**Alternatives considered:**

- *Extract in `pipeline.go` after `Build()` returns* — `pipeline.go` would need
  to call a new `module.ExtractModuleMetadata(release.Value, ...)` function. This
  keeps the module package responsible but requires exporting more API surface and
  passing the CUE value back out. Rejected: the CUE extraction logic already lives
  in `release/metadata.go` and all CUE paths are already known there.

- *Enrich `MetadataPreview` / `module.ExtractMetadata` before `Build()`* — run a
  second CUE load before the overlay. Rejected: it duplicates CUE evaluation;
  `Builder.Build()` already produces the evaluated value that has all fields.

**Rationale:** The builder holds the CUE value; it should be responsible for
populating both metadata types. This keeps CUE path knowledge in one file
(`release/metadata.go`) and requires zero changes to `module` package API.

### D2: Both metadata types share the same `Labels` source

**Decision:** Both `ReleaseMetadata.Labels` and `ModuleMetadata.Labels` are
populated from the same CUE source (`#opmReleaseMeta.labels`, with fallback to
`metadata.labels`). No split.

**Rationale:** Behavioral parity. Splitting module labels (from `metadata.labels`)
from release labels (from `#opmReleaseMeta.labels`) is a separate, future change.
Doing it here would widen scope without a clear user-facing need.

### D3: `Components []string` is added to both types inside `builder.go`

**Decision:** After extracting components from the CUE value, `builder.go`
collects component names into a `[]string` and sets it on both
`ReleaseMetadata.Components` and `ModuleMetadata.Components` before returning
`BuiltRelease`.

**Rationale:** Both types carry components (established by the prior refactor).
The builder has the component map at construction time; no caller needs to re-add
them.

### D4: Delete `Metadata`, `ToReleaseMetadata()`, and `ToModuleMetadata()`

**Decision:** All three are removed. No deprecation period — these are
`internal/` types.

**Rationale:** Keeping them as thin wrappers around the new fields would add dead
code and leave the old coupling pattern visible as a template for future misuse.

## Risks / Trade-offs

- **Test construction verbosity** — Tests that built `BuiltRelease{Metadata: ...}`
  now need to fill two structs. Accepted: the tests become *more honest* about
  what the type carries.

- **Labels behavioral parity** — Both metadata types get the same labels map.
  The same struct pointer is not shared (both are copied), so mutation of one
  does not affect the other. Acceptable for current usage.

- **`DefaultNamespace` in `ModuleMetadata` from CUE** — `extractModuleMetadata`
  reads `metadata.defaultNamespace` from the CUE value. In the pipeline's Phase 1
  fallback path (`module.ExtractMetadata`), the same field is read from the same
  CUE path. Values will match. If the field is absent from CUE (test fixtures
  without `defaultNamespace`), the field is empty — consistent with current
  behaviour.
