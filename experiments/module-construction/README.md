# Experiment: module-construction

**Goal**: Find a way to construct the `cue.Value` passed as `#module` into
`#ModuleRelease` during the build pipeline without relying on `mod.Raw` — an
opaque blob that gives the builder no control over what is passed in.

**Context**: The current builder does:

```go
releaseSchema.FillPath(cue.MakePath(cue.Def("module")), mod.Raw)
```

`mod.Raw` is the entire evaluated CUE package value. The builder has no control
over what fields it contains or which values are included. The goal is to
eliminate this and construct the value explicitly from the Module struct's fields.

---

## The Critical Dependency

The `#ModuleRelease` schema has:

```cue
_#module: #module & {#config: values}
components: _#module.#components
```

This means CUE resolves component fields (e.g. `spec.container.image`) by
unifying the module's `#config` with the injected `values`. Because
`components.cue` contains cross-references like:

```cue
spec: {
    container: image: #config.image    // cross-reference
    scaling: count:   #config.replicas // cross-reference
}
```

...these references must remain intact through whatever construction strategy
is used. **This is the gate condition for all approaches.**

---

## Four Approaches Tested

### Approach A — FillPath Assembly

Extract `#config`, `#components`, and `metadata` individually from `mod.Raw`
via `LookupPath`, then fill them into the `#Module` schema one by one.

```go
assembled := moduleSchema.
    FillPath(cue.ParsePath("metadata"),    rawModule.LookupPath("metadata")).
    FillPath(cue.ParsePath("#config"),     rawModule.LookupPath("#config")).
    FillPath(cue.ParsePath("#components"), rawModule.LookupPath("#components"))
```

**Result**: ❌ FAILS at gate (A3).

When `#components` is extracted via `LookupPath` and reinjected into a new
parent, the `#config.image` reference inside it loses its binding. CUE's
cross-references are resolved relative to the evaluation scope where the
value was originally defined — moving a sub-value to a new parent does not
rebind its internal references.

Error: `#ModuleRelease.components.app.spec.container.image: non-concrete value string`

What DOES work in A:

- Extraction compiles cleanly
- Assembled value satisfies `#Module` schema
- CUE-computed fields (FQN, UUID, labels) evaluate on the assembled value

---

### Approach B — Selective Raw Transform

Keep `mod.Raw` as the base — never decompose it. Apply targeted FillPath
overrides before injection.

```go
releaseSchema.FillPath(cue.MakePath(cue.Def("module")), mod.Raw).
    FillPath("values", selectedValues)...
```

**Result**: ✅ WORKS end-to-end.

Because `mod.Raw` is never decomposed, the `#config` ↔ `#components`
cross-references are always intact. The builder can control which values are
injected (module defaults vs user overrides) with full correctness.

**Key finding — concrete fields are immutable**: Attempting to override a
metadata field that is already concrete in `mod.Raw` (e.g. `defaultNamespace:
"default"` set in `module.cue`) via FillPath **conflicts**:

```text
metadata.defaultNamespace: conflicting values "production" and "default"
```

CUE unification of two different concrete strings always fails. FillPath can
only narrow open/abstract fields — it cannot override concrete values.

This means Approach B gives control over values selection but NOT over
already-concrete metadata fields in `mod.Raw`.

---

### Approach C — Compile + Inject Hybrid

Compile Go-native metadata as a CUE text literal, inject `#config` and
`#components` as `cue.Value` via FillPath.

```go
metaCUE := ctx.CompileString(`{apiVersion: "...", name: "...", ...}`)
assembled := moduleSchema.
    FillPath(cue.ParsePath("metadata"),    metaCUE).
    FillPath(cue.ParsePath("#config"),     rawModule.LookupPath("#config")).
    FillPath(cue.ParsePath("#components"), rawModule.LookupPath("#components"))
```

**Result**: ❌ FAILS at gate (C3). Same breakage as A.

Even though `#config` and `#components` both come from the same `rawModule`
evaluation and thus share an evaluation context, injecting them separately
into a new parent (the `#Module` schema) still breaks the cross-references.
The act of extracting them via `LookupPath` and reinserting via `FillPath`
severs the binding — regardless of the metadata source.

**Bonus finding** (C4 — passes): CUE-computed fields DO evaluate correctly on
the hybrid-assembled value:

- `metadata.fqn = "test.dev/simple-module@v0#Simple"` ✅
- `metadata.uuid = "829407b0-f32c-529f-b482-68f58cdc34cf"` ✅
- `metadata.labels["module.opmodel.dev/name"] = "simple"` ✅

This means Approach C gives full metadata control AND correct computed fields —
but it cannot be used alone because cross-refs break.

---

### Approach D — Module.Encode() API Pattern

The Module struct holds `Config` and `Components` as `cue.Value` (not
decomposed), plus `Metadata` as Go-native fields. An `Encode()` method
reconstructs a `cue.Value` for the builder.

Two internal strategies tested:

- `encodeViaSchema`: uses `#Module` schema as base (same as A internally)
- `encodeViaCompile`: compiles a minimal CUE struct from Go strings (same as C internally)

**Result**: ❌ FAILS at gate for both strategies.

Both D strategies internally separate `#config` and `#components` before
injecting them into a new parent — the same operation that breaks cross-refs
in A and C. The API design (Encode method) does not change the CUE semantics.

**What DOES work in D**:

- `TestD_GoFieldModificationReflectsInEncoded` ✅ — modifying a Go field
  before calling `Encode()` produces a CUE value reflecting that change.
  This IS the control benefit, just not sufficient alone.
- `TestD_ComponentsAsUndecomposedValue` ✅ — holding `#components` as a
  single `cue.Value` (not decomposed into `map[string]*Component`) preserves
  the full structure for iteration.

---

## Summary of Findings

| Approach | Assembly | Gate (cross-refs) | End-to-end | Metadata control |
|----------|----------|-------------------|------------|-----------------|
| **A** FillPath into schema | [x] works | [ ] BROKEN | [ ] fails | Inherits from Raw |
| **B** Selective Raw | N/A (no decomposition) | [x] INTACT | [x] works | Limited (concrete fields immutable) |
| **C** Compile + inject | [x] works | [ ] BROKEN | [ ] fails | [x] full control |
| **D** Encode method | [x] works | [ ] BROKEN | [ ] fails | [x] full control |

**Root cause**: `LookupPath` on a CUE value returns a sub-value whose internal
references are relative to the original evaluation scope. When this sub-value
is injected into a new parent via `FillPath`, the references are not rebound to
the new parent's sibling fields. The `#config.image` reference in `#components`
was bound at evaluation time to the `#config` in the original package scope —
that binding cannot be transferred by moving the value.

---

## Conclusion

**Only Approach B works end-to-end.** The module value passed to
`FillPath(#module, ...)` must be the original evaluated CUE package value —
decomposing it and reassembling always breaks cross-references.

**The redesign direction**: rather than removing `mod.Raw`, the correct path is:

1. Keep `mod.Raw` (renamed perhaps to make intent clearer)
2. Remove the *builder's direct dependency* on `mod.Raw` by having the Module
   struct expose a method like `ModuleValue() cue.Value` that returns the raw
   CUE value — this gives the builder an explicit, controlled access point
3. For metadata control: apply `FillPath` overrides to the module value for
   OPEN fields only (values, abstract metadata fields), using Approach C/D
   for metadata that needs Go-controlled construction
4. For values control: the builder already has full control — it selects
   `selectedValues` and passes them via `FillPath("values", selectedValues)`
   at the release level, completely independent of what's in the module

The "construct CUE on the fly" goal is achievable for everything EXCEPT the
`#components` ↔ `#config` cross-reference binding, which requires the original
evaluation scope to be preserved.
