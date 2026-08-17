# Tasks — cli-catalog-gates

> Depends on the merged pipeline; consumes `opm/compat` for the first time. Local gates (2) land before registry-touching ones (3) so each is green in isolation. Bare-`@` ban on every commit.

## 1. Member enumeration

- [x] 1.1 `internal/publish`: filesystem walk of `<kind>/<apiVersion>/` dirs + flat `transformers/`; per-package `loadPackage` reuse; definition iteration filtered by the concrete `kind` discriminator; member model {kind, name, apiVersion, value, pkgPath}.
- [x] 1.2 Tests over a fixture tree: counts per kind, fragments/schemas beside members excluded, cross-catalog references never visited.

## 2. Local gates (no registry)

- [x] 2.1 `Options` gains the `#CatalogMemberFQNGate` / `#TraitOptionalGate` schema fields, resolved beside `IdentityPackage` in `cmdutil`.
- [x] 2.2 `gateMemberFQN`: unify every member, concrete evaluation, NO incomplete-filter; CUE error → refusal via the grouped funnel. Pin the measured blind spot: missing `declaredAPIVersion` is caught.
- [x] 2.3 `gateTraitOptional`: the `optional` field only; unstated and pinned both refused; `v1alpha1` traits included (carve-out is compat-only — assert it).
- [x] 2.4 Wire both into `Run` behind `KindCatalog`; `Plan.Render` per-gate outcome counts.

## 3. The compat gate

- [x] 3.1 Predecessor scan per design's tree: version enumeration below effective (same major, prereleases included, newest first), per-package backward probe, per-member resolution, exhausted → pass. Missing-package-at-version = negative signal; transport failure = `ConnectivityError`.
- [x] 3.2 Comparison: `StripProvenance` both operands, `compat.CheckAtLevel` with the member's own apiVersion; alpha members skipped before any registry work; transformers never enter.
- [x] 3.3 Refusal 9 rendering: caller-attached header (member, apiVersion, predecessor coordinate), path-located violation lines via `AlignColumns`, the fixed closing action.
- [x] 3.4 Hermetic multi-build tests: clean; field-removed; remove-then-readd refused; prerelease predecessor compared; new package passes; connectivity abort mid-walk yields no partial verdict. Update the `gates.go` round-trip comment.

## 4. registry check

- [x] 4.1 `opm catalog registry check <path@version> [--compat]` under a new `registry` subgroup: pull, identity concreteness + coordinate agreement, member inventory report; `--compat` runs 3.x against the fetched build. Exit 0/2/3.
- [x] 4.2 Help text carries the D35 aid-not-guarantee sentence verbatim (graduation-gated); constructor + e2e output tests.

## 5. Real-tree smoke + gates

- [x] 5.1 Registry-backed test: `catalog_opm/src` passes member + posture gates as-is; compat gate against the live GHCR history runs clean.
- [x] 5.2 `task fmt vet lint test` green.

## 6. Record + cross-repo corrections

- [x] 6.1 `enhancements/0011/`: slice → done + `openspec_ref` + history event; log the open question (beta+ member removal refused by nothing); note the refusal-11 pre-D49 wording as superseded by the shipped gate.
- [x] 6.2 0011 amendment decision (`Amends: D9`, via the enhancements workflow): strike the `highestStable` implementation note; demote the one-prior-build claim to the no-removals fast path; state the literal rule as implemented.
- [x] 6.3 Library follow-up (small separate PR): `catalog-compatibility` spec's "Predecessor Selection" re-scoped; `HighestStable` re-documented as the float selector — its disposition is settled: `cli-template-modules` makes it template resolution's version selector (its first true caller), so it stays.
