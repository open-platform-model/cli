## Context

See proposal.md for motivation. Relevant facts:

- `pkg/module.ModuleMetadata` has the eight fields and JSON tags of `opm/schema.ModuleMetadata`; `pkg/module.InstanceMetadata` has five of the library's six (no `fqn,omitempty`).
- `internal/workflow/render/render.go` builds `Result.Instance` by copying `Name`, `Namespace`, `UUID`, `Labels` from `inst.Metadata`, then applies the namespace override; `Result.Module` comes from `decodeModuleMetadata(inst.Package.LookupPath(schema.Module))`, a `Decode` of the embedded module's `metadata` into the CLI type.
- The library's `module.decodeModuleMetadata` is unexported and `module.Instance` exposes no accessor for the embedded module's metadata, so a CLI-side decode of that subtree stays.
- `CanonicalModuleRef` is a method on the CLI type with two callers (`internal/workflow/apply/apply.go`, `thineditor.go`); `log_output.go` reads `Result.Instance.Namespace` and `Result.Module.Version`, names that exist on the library types.
- Nothing marshals `Result` to JSON (`grep json.Marshal` under the render, module and instance command packages is empty), so extra fields on the library types are invisible.

## Goals / Non-Goals

**Goals:**
- One declaration of each metadata shape, the library's.
- `spec.module` on the written CR is byte-identical before and after.

**Non-Goals:**
- Changing what `Result` exposes beyond the field types.
- Adding a library accessor for the embedded module's metadata.

## Research & Decisions

### Which library names to use

**Options considered**:
1. `opm/schema.ModuleMetadata` directly.
2. `opm/module.ModuleMetadata` / `module.InstanceMetadata`, the library's aliases of the same types.
**Decision**: the `opm/module` aliases.
**Rationale**: the CLI already imports `opm/module` at the sites that hold artifacts; the alias exists for exactly this use.

### Home of `CanonicalModuleRef`

**Context**: a method cannot be added to a foreign type.
**Decision**: `render.CanonicalModuleRef(m module.ModuleMetadata) (path, version string)` in `internal/workflow/render`, with `ensureVPrefix` and the existing tests moved beside it.
**Rationale**: `Result` lives there and both callers already import `render`.

### Whole-struct assignment for the instance

**Decision**: `result.Instance = *inst.Metadata` (nil-guarded), then the namespace override.
**Rationale**: no field list to keep in sync; `Annotations` and `FQN` arrive as the library decodes them.

## Risks / Trade-offs

- [A consumer of `Result.Instance.Labels` relied on `Annotations` being empty] → no reader of `Result.Instance.Annotations` exists; the apply workflow stamps its own labels.
- [Go importers of `pkg/module` break] → the proposal marks the removal as breaking for `pkg/` importers.
