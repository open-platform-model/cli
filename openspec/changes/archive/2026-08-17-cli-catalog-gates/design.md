# Design — cli-catalog-gates

## Overview

Three gates appended to the pipeline's catalog path plus one new command, all composing shipped pieces: `opm/compat` (the walk, level ladder, provenance strip), the pipeline's loader/refusal/registry plumbing, and core's two gate definitions. The one genuinely new mechanism is predecessor selection, implemented per D9's literal rule (settled in exploration; the decision tree below is the specification).

## Research & Decisions

### Predecessor selection: the literal rule, not `HighestStable`

**Context**: D9:158 defines the predecessor as "the last published build that shipped a primitive of that `name` at that `apiVersion`"; D9:187 points at `highestStable` — the deleted subscription float's stable-skipping selector. The two rules coincide only on a prerelease-only history (today's, by luck). D9:172's "never needs more than one prior build" holds only while members are never removed — and member removal is invisible to the gate (the walk visits the *published* build's members).
**Options considered**:
1. `HighestStable` — wrong the day a stable ships (compares `2.1.0-alpha.3` against `2.0.0`, missing breaks that consumers pinned to prereleases — D14-blessed — can see).
2. Immediate predecessor only, absent → pass — leaves the remove-then-readd-incompatible laundering hole open, and inherits D35's bypassed-build blindness.
3. The literal rule: per gated package, scan published versions strictly below the effective version (same major, prereleases included), newest → oldest; each member compares against the newest build carrying it; a member found nowhere passes.
**Decision**: option 3. A beta+ `name@apiVersion` key is a permanent claim on its own history; the only escape is the apiVersion bump.
**Rationale**: it is what D9's rule sentence says; it refuses incompatible re-introduction; it degrades gracefully across gate-bypassed builds; it needs no separate prerelease policy (recency is the ordering). Cost: identical to option 2 in the no-removals common case (one prior build per package); the full-history probe fires only for members absent from the immediate predecessor, at one CUE-cached module fetch per historical build.
**Record**: a 0011 amendment (`Amends: D9`) strikes the `:187` note; the library spec's "Predecessor Selection" requirement is re-scoped (`HighestStable` currently has zero callers).

```
per member (beta/GA, non-transformer):
  versions ← ModuleVersions(path@vN) < effective, newest→oldest
  walk versions: load <kindpkg>@v (module zip cached per v)
    member present? → predecessor found → StripProvenance both →
                      compat.CheckAtLevel → violations? refusal 9 : pass
  exhausted without a hit → pass (new at this key)
```

### The member walk is filesystem-driven; the compare operand is the definition

**Context**: `#Catalog` exposes only `#transformers`; transformers reference ~36 of 70 contract members and zero blueprints — a value walk cannot enumerate the tree. Refusal 9's example shows a `metadata.labels.*` violation path, so the compared operand is the whole member value, not `spec`.
**Decision**: enumerate `<kind>/<apiVersion>/` directories (contract kinds) plus the flat `transformers/` package from the tree being published; within each loaded package, iterate definitions and filter by the concrete `kind` discriminator every core kind carries. The compat operands are the member *definitions* post-`StripProvenance`; `spec` remains a non-concrete schema throughout (the compat walk was built for definition values — `cue.Schema()` subsume at leaves).
**Cross-catalog exclusion for free**: a directory walk only visits the catalog's own definitions (0010 D17's scope), where a value walk would drag referenced foreign primitives in.

### Concrete evaluation, and why the pipeline's filter is not reused

**Context**: two measured blind spots — `#TraitOptionalGate` rule 1 (unstated posture = incomplete value; invisible to plain vet) and `#CatalogMemberFQNGate`'s missing `declaredAPIVersion` (invisible to plain *and* `-c` vet; only concrete evaluation reports it). The pipeline's `conformIdentity` deliberately drops incomplete-value errors (openness is D4's separate gate).
**Decision**: the two catalog gates build their gate values as non-hidden fields and validate with `cue.Concrete(true)` **without** the incomplete-filter — here incompleteness is the finding. `#TraitOptionalGate` is handed the `optional` **field** only (handing the member drags `spec` into concreteness). CUE's errors route through the grouped funnel; refusal 11 is never hand-written. The pre-D49 wording in 06-operational's refusal-11 sketch (bare `kindPrefix[kind]` equality) is superseded by the shipped gate (`+ "/" + declaredAPIVersion` for contract kinds) — moot in practice since the message is CUE's, noted for the history event.
**Carve-out boundary stated explicitly**: D34's alpha exemption applies to the *compat* gate only; all 120 members pass through the FQN gate and all traits through the posture gate regardless of level.

### `registry check`: the consumer's check, on demand

**Context**: D7 — base mode is "the same check a consumer performs", identity-shaped; the library's materialize-time `verifyCatalogIdentity` is its shipped twin. `--compat` reuses the D9 machinery against a published build. D35 requires the aid-not-guarantee sentence in help text (graduation-gated).
**Decision**: `opm catalog registry check <path@version> [--compat]`. Base: pull (the same `load.Instances` route as the predecessor scan — a published catalog's root and subpackages load standalone), verify declared identity concrete and agreeing with the fetched coordinate, report the member inventory per kind/apiVersion. `--compat`: run the predecessor comparison for the *fetched* build exactly as publish would have. Exit codes: 0 clean, 2 findings, 3 connectivity. Help text carries D35's sentence verbatim.
**The identity operand question resolved**: the check reads the pulled catalog's `metadata.*` (like the library twin), not the identity package — a published catalog's identity package "is never evaluated as a package by any consumer" (core's own doc).

### Connectivity vs refusal across the walk

**Decision**: any failed enumeration or package load during predecessor search or `registry check` is a `*ConnectivityError` — the artifact was never judged; no partial verdict is rendered. The `gates.go` "only registry round-trip" comment is rewritten to name the walk. A *missing package at a given version* is not an error (it is the scan's negative signal); only transport-level failures abort.

## Technical Notes

- Gate wiring: kind-guarded block after `gateOverride` in `Run` — `gateMemberFQN`, `gateTraitOptional` (local, no registry), then `gateCompat` (registry). Plan `Render` gains a per-gate outcome section (members checked / compared / passed / refused counts).
- Schema resolution: `#CatalogMemberFQNGate` and `#TraitOptionalGate` via the same `SchemaCache().Get` + `LookupPath` route as `IdentityPackage`; `Options` gains the two sibling schema fields.
- Test harness: hermetic — publish builds 1..N of a fixture catalog into the in-process immutable registry, then gate build N+1. Cases pinned from the decision tree: clean pass; field-removed refusal (9's shape); alpha member skipped by compat but caught by FQN gate on a filing error; remove-then-readd refused (the option-2 divergence case); prerelease-predecessor compared (the option-1 divergence case); new-package pass; connectivity abort; posture unstated/pinned refusals; the D49-opacity bound documented in a test comment.
- Real-tree smoke: `catalog_opm/src` must pass the member and posture gates as-is (66 compat / 120 FQN / 27 posture) — run against GHCR in the registry-backed tier.

## Amendment (measured during implementation): the D30 exemption is a violation filter, not StripProvenance

Two comparator facts surfaced only against the real tree (`catalog_opm/src` vs the live GHCR history), and both forced a deviation from the "StripProvenance both operands" wording above — same rule, different application point:

1. **StripProvenance cannot rebuild a core-typed member.** Its `Syntax(cue.All(), cue.InlineImports(true))` round-trip emits a reference to core's hidden `#KebabToPascal` helper (reached via `metadata.#definitionName`) without inlining it, and the rebuild fails (`reference "#KebabToPascal" not found`). Every real member is core-typed; the hermetic fixtures that passed do not import core. The gate therefore applies D30 by dropping violations at exactly `metadata.catalogVersion` and `metadata.description` — the same denylist at the same scope, applied to the walk's output instead of its input.
2. **The walk's leaf subsume false-positives on byte-identical schemas** carrying `matchN` validators or pending comprehensions (`#Image`'s if-guards): measured, 33 of 66 unchanged members reported `domain narrowed`. The gate short-circuits with an identical-modulo-provenance syntax comparison (the strip's AST scrub without the rebuild) — a correct rule in its own right: a member restamped by a release cannot have changed shape. The walk runs only for members whose scrubbed syntax actually differs; its residual noise on *changed* comprehension-bearing members is a recorded library limitation.

Both facts land in the library follow-up (task 6.3) alongside the predecessor-selection re-scope.

## Open items logged, not solved here

- Removal of a beta+ member is refused by nothing (the walk sees only published members). Logged as an 0011 open question with the laundering-adjacent evidence.
- Plan `--json` remains deferred; the per-gate outcome section is text-only.
