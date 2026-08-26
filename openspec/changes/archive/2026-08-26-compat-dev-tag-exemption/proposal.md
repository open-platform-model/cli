## Why

`catalog_opm`'s `branch-publish` workflow is refused by the compatibility gate on every dev push of a branch that tightens a beta member, and the verdict depends on which branch pushed a dev tag last rather than on the content (cli issue 165). Measured against `golang.org/x/mod/semver`: `-0.dev.*` orders below `-alpha.N`, so a dev tag's predecessor window contains only older dev tags, while a release tag's window is headed by the latest alpha and never reaches dev tags unless a member is absent from every release. Dev builds are throwaway iteration artifacts; judging them blinds the branch and baselining on them makes the gate non-deterministic. Enhancement 0011 D26 clause 1 amends D9/D23 to put dev builds outside the gate in both directions.

## What Changes

- `opm catalog publish` skips the compatibility compare when the effective tag is a dev prerelease (a `dev` identifier in the prerelease segment, the `branch-tag.sh` shape `v2.0.0-0.dev.<count>.g<sha>`), and the plan's `compat gate` row reports the skip as `dev-exempt` instead of counts.
- The predecessor scan filters dev tags out of the window for every build; alpha, beta and rc release tags stay, newest first, exactly as before.
- `opm catalog registry check --compat` applies both rules to the fetched build's version so the two commands agree.
- Regression tests pin the semver ordering (`-0.dev.*` below `-alpha.N`) and the unchanged behaviour for release tags.
- No new flags. PATCH per SemVer: existing refusals on release tags are unchanged; only dev tags, which no release ever compares against, change behaviour.

Out of scope: the prerelease-module-line exemption (D26 clause 2, separate change `compat-prerelease-line-exemption`) and the comparator's nested-provenance and matchN false positives (library change).

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `artifact-publishing`: the compatibility gate requirement gains the dev-build exemption and the dev-free predecessor window.
- `catalog-registry-check`: `--compat` on a dev-tagged build reports `dev-exempt`; its predecessor window excludes dev tags.

## Impact

- `internal/publish/compat.go`: `gateCompat`, `predecessorVersions`, `CatalogGateOutcomes` (new dev-exempt state), a dev-tag classifier.
- `internal/publish/plan.go` and `internal/publish/check.go`: `compat gate` / `compat` row rendering.
- `internal/publish/compat_test.go`, `check_test.go`: new hermetic multi-build cases.
- Downstream: `catalog_opm/.github/workflows/branch-publish.yml` passes again once its pinned `opm` is bumped to the release carrying this change (delivery, not part of this change).
- Implements enhancement 0011 D26 (clause 1); see `enhancement.yaml`.
