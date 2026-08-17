# Proposal — cli-catalog-gates

> Slice `cli-catalog-gates` of enhancement `0011`. Decisions D7, D9, D22 (+ 0010 D17, D21, D25, D28, D34, D35, D42, D49). Depends on `cli-publish-pipeline` (extends its gate run) and `library-compat-comparator` (consumes `opm/compat`).

## Why

`opm catalog publish` currently gates identity and coordinates but not *content*: a catalog can ship a build that silently breaks a contract it already published — remove a field from a `v1beta1` trait, and every module compiled against the previous build still matches on that key and breaks at render. D9's answer is the compatibility gate: for every beta/GA member, compare against the last published build that shipped that `name` + `apiVersion` under D27's additive-only rule, via the library's measured field-wise comparator (`opm/compat` — built for exactly this, currently consumed by nothing publish-side). D22 adds the structural gate: every member of the tree — all four kinds — unified against core's `#CatalogMemberFQNGate` (refusal 11: wrong-kind filing, wrong-depth filing, stale authored `fqn`, stale `catalogVersion`), plus `#TraitOptionalGate` refusing traits with an unstated or pinned `optional` posture. D7 adds the out-of-band aid: `opm catalog registry check [--compat]` runs the consumer's identity check — and optionally the compat comparison — against a *published* catalog on demand.

**This change also settles a rule 0011 left ambiguous, and corrects the record.** D9's text defines the predecessor as "the last published build that shipped a primitive of that `name` at that `apiVersion`" — but its implementation note points at `highestStable`, the *subscription-float* selector, whose stable-skipping semantics are a different rule that only coincides on today's prerelease-only history. The gate implements D9's literal rule — a per-package backward scan through published builds, prereleases included — which additionally closes the remove-then-readd laundering hole and sees through gate-bypassed intermediate builds. The correction lands as an amendment decision in 0011 and a small delta to the library's `catalog-compatibility` spec (which mislabels `HighestStable` as predecessor selection; it has zero callers).

## What Changes

- **The compat gate on `opm catalog publish`** (and its `--dry-run`): for each member of the current tree's `<kind>/<apiVersion>/` packages — transformers excluded structurally (no `apiVersion`, D44), alpha members excluded by policy (D34) — find the predecessor by the literal rule: enumerate published versions strictly below the effective version (same major, prereleases included, newest first); per gated package, load the published subpackage at each build (`<pkgpath>@<version>` — measured loadable standalone; one module fetch per build, CUE-cached) until every member is resolved or history is exhausted; a member found nowhere passes (the apiVersion-bump escape). Found pairs compare via `compat.CheckAtLevel` over `StripProvenance`'d definition values; violations render as refusal 9 with the caller-attached header (member, apiVersion, predecessor coordinate) and path-located violation lines.
- **The member gate**: filesystem/package-driven walk (measured: a value-driven walk reaches ~half the contract members and no blueprints — `#Catalog` exposes only `#transformers`) over all `<kind>/<apiVersion>/` packages plus flat transformers; every member unified against `#CatalogMemberFQNGate` with **concrete evaluation** (the pipeline's incomplete-value filter must not be reused — here incompleteness is the finding, measured: a missing `declaredAPIVersion` passes both vet modes and surfaces only concretely); every trait's `optional` **field** (never the member — dragging `spec` in breaks on schema non-concreteness) unified against `#TraitOptionalGate`. CUE's errors are the refusal (refusal 11 is not hand-written). The alpha carve-out does **not** apply here — `v1alpha1` members are FQN- and posture-gated.
- **`opm catalog registry check [--compat]`**: base mode pulls a published catalog by path@version and runs the consumer's identity verification out of band (declared identity concrete; `modulePath`/`version` agree with the fetched coordinate — the CLI twin of the library's shipped materialize-time check) and reports what the catalog contains (member listing per kind/apiVersion). `--compat` additionally runs the predecessor comparison against the *published* build. Help text carries D35's framing verbatim: an aid, not a guarantee — enforcement exists only producer-side.
- **Connectivity semantics**: the gate turns publish's single registry round-trip into 1 + N package loads; any failure mid-walk is a `ConnectivityError` ("the artifact was never judged"), never a refusal; the `gates.go` ordering comment is updated.
- **No `opm catalog vet`**: the carried note from the pipeline change is dropped — nothing in 0011 (decisions, plan, graduation) backs it; `catalog publish --dry-run` and `registry check` cover both directions. Recorded.

## Capabilities

### New Capabilities

- `catalog-registry-check`: out-of-band verification of a published catalog, with the aid-not-guarantee contract.

### Modified Capabilities

- `artifact-publishing`: `opm catalog publish` gains the compatibility gate and the member/posture gates; the refusal catalog gains 9 and 11; the connectivity contract covers the member walk.

## Impact

- **SemVer: MINOR** (a new command; stricter catalog publish is the feature).
- **Packages**: `internal/publish` (catalog-only gates appended to `Run`, kind-guarded like the existing precedents; predecessor scan; member walk; plan `Render` extension for per-gate outcomes), `internal/cmd/catalog/` (`registry` subgroup + `check`), `library/opm/compat` consumed for the first time.
- **Cost**: publish of `catalog_opm` today = ~5 gated packages × 2-build history; member gates cover 120 members, compat gates 66 post-carve-out.
- **Cross-repo coordination recorded on landing**: (1) a 0011 amendment decision (`Amends: D9`) striking the `highestStable` implementation note and demoting the one-prior-build claim to a no-removals fast path; (2) a library `catalog-compatibility` spec delta re-scoping "Predecessor Selection" (`HighestStable` has zero callers — delete or re-document); (3) an open question logged: removal of a beta+ member is itself refused by nothing.
- **Known bound (documented)**: the scan probes by the current D49 filing convention; pre-D49 builds are opaque to it — acceptable, that horizon is entirely alpha-era.
