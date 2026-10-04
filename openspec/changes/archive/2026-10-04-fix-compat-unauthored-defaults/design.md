## Context

`checkDefaults` (`internal/compat/compat.go`) runs at every walk position and asks `cue.Value.Default()` whether each side has a default. See proposal.md (Why) for the refusal it produces. The cause is in CUE's evaluator (`adt.Vertex.Default`, cue v0.17.1): a value has a default in two unrelated cases.

1. Its base value is a disjunction with marked defaults (`*"a" | string`). The author wrote this.
2. Its base value is an open list. `Default()` returns the list closed at its fixed elements: `[]` for `[...string]`, `[string]` for `[...string] & [_, ...]`. Nobody wrote this; it is how CUE exports an open list.

`checkDefaults` treats both the same. For the catalog's `subjects` the case-2 "default" is `[#RoleSubjectSchema]`, which is not concrete, so the comparison falls to mutual subsumption under `cue.Schema(), cue.Raw()`, and that fails in one direction when the field's constraint marker changes from `!` to `?` ("value not an instance"), although both renderings are identical. The tests written for this change measured the bug as wider than the catalog case: a plain `xs: [...string]` made `xs?: [...string]` was refused the same way, adding an authored default to an open list (`[...string]` to `*["a"] | [...string]`) was refused as `default changed` against the implicit `[]`, and narrowing `[...string]` to `[...string] & [_, ...]` reported a spurious `default changed` beside its correct `domain narrowed`.

## Goals / Non-Goals

**Goals:**
- Compare defaults only where an author wrote one; the implicit open-list default never reports.
- Keep every existing verdict, including every real default change and removal.

**Non-Goals:**
- `required()`'s use of the implicit default (a regular open-list field counts as defaulted). Switching it to authored defaults would add `field added without optional or default` refusals for regular `[...T]` fields; that is a stricter rule and its own change (proposal Impact).
- The open-list element blind spot (`[...string]` to `[...int]` undetected), pinned in the tests as a CUE limitation.

## Decisions

### Authored default helper

`checkDefaults` calls `authoredDefault(v)` on both sides instead of `v.Default()`:

```go
// authoredDefault is v's default when an author marked one with *.
func authoredDefault(v cue.Value) (cue.Value, bool) {
	d, ok := v.Default()
	if !ok || implicitListDefault(v) {
		return d, false
	}
	return d, true
}

// implicitListDefault: v is a plain open list, not a disjunction.
func implicitListDefault(v cue.Value) bool {
	if v.IncompleteKind() != cue.ListKind {
		return false
	}
	n := v.Len()
	return n.Err() == nil && !n.IsConcrete()
}
```

The rest of `checkDefaults` is unchanged: prior has none, nothing to check; prior has one and next has none, `default removed`; both have one, compare.

## Research & Decisions

### Telling an authored default from the implicit one

**Context**: the public CUE API has no "is this default authored" call, and `Default()` returns `true` for both cases.

**Explored**: a probe over `[...string]`, `[...string] & [_, ...]`, `*["a"] | [...string]`, `*[] | [...string]`, `[...string] | *["a"]`, `*[...string] | int`, a default reached through a definition reference (`#C: *["a"] | [...string]`, `w: #C`), and a list unified with `*["a"] | _` across two conjuncts. Measured per shape: `Default()`, `Expr()`, `Syntax(cue.Raw())`, `Allows(cue.AnyIndex)` and `Len()`.

**Options considered**:
1. `Expr()` reports `OrOp`. Rejected: `Expr()` drops a default that a non-default disjunct subsumes, so `*["a"] | [...string]` reports `NoOp`; it also works per conjunct, not per evaluated value.
2. Look for a `*` in the emitted syntax. Rejected: `Syntax(cue.Raw())` keeps references (`#C`), so a default behind a reference is invisible; `cue.All()` expands them but the walk would have to stop at nested literals by hand.
3. Compare the default with "v closed" element by element. Rejected: on a disjunction `List()` iterates the default's elements, so `*["a"] | [...string]` looks implicit and a change to `*["b"] | [...string]` would be missed.
4. `Len()`. It evaluates on a list vertex only: a plain open list yields `int & >=n` (no error, not concrete), a closed list a concrete count, and any disjunction an error ("len not supported"). Every authored shape in the probe errors; both implicit shapes do not.

**Decision**: option 4, restricted to values whose incomplete kind is exactly a list (a mixed kind such as `*[...string] | int` is a disjunction and authored).

**Rationale**: it follows the evaluator's own rule (implicit default exists exactly for an open-list base value) through public API. It depends on `Len()`'s behaviour on a disjunction; tests pin both sides, so a CUE upgrade that changes it fails a test rather than silently dropping default checks.

## Risks / Trade-offs

- [`Len()` starts resolving a disjunction to its default in a later CUE] → the authored-list-default tests fail (`default changed` and `default removed` on `*["a"] | [...string]` would vanish). They are the pin.
- [`*[] | [...string]` to `[...string]` reports `default removed`, though the effective default is `[]` on both sides] → kept: the author removed a written default, and the rule is about authored defaults. Nothing in the catalogs uses this spelling today.

## Migration Plan

None. Ships in the next cli release; catalog_opm then bumps `.opm-cli-version` for `add-subjectless-roles`.
