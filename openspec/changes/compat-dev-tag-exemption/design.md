## Context

See proposal.md for motivation. The gate lives in `internal/publish/compat.go`: `gateCompat` (publish) and `RegistryCheck` (`check.go:135`) both call `predecessorVersions(published, tag, major)` then `compatScan`. `CatalogGateOutcomes` carries four counters (`CompatCompared`, `CompatRefused`, `CompatAlpha`, `CompatNew`) rendered by `plan.go:renderCatalogGates` and `check.go`'s report; `Plan.CompatChecked` already distinguishes "not completed" from counts. The effective tag comes from `resolveVersion` (`identity.go`) as `"v" + Version`; `catalog_opm/.tasks/branch-tag.sh` stamps `v<M>.<m>.<p>-0.dev.<count>.g<sha>` into the working tree before publish, so the dev shape reaches the gate as an ordinary `p.Tag`.

Constraints: 0011 D26 clause 1 is the policy; D23's literal scan (release prereleases included, newest first) stays. Principle VII: no flag, the dev shape is recognised, never declared.

## Goals / Non-Goals

**Goals:**
- A dev-tagged build is neither judged nor a baseline, and says so in the summary.
- Release-tag behaviour is byte-for-byte unchanged (existing `compat_test.go` cases stay green untouched).
- `publish` and `registry check --compat` share one classifier and one window.

**Non-Goals:**
- Any change to what is compared for non-dev tags (clause 2 is `compat-prerelease-line-exemption`).
- Comparator fixes (library).
- Changing `branch-tag.sh` or the workflow.

## Decisions

### Research & Decisions

### Dev-tag recognition
**Context**: The gate needs a predicate over a semver tag.
**Explored**: `branch-tag.sh` line 7 documents `-0.dev.<commit_ct>.g<short_sha>`; `library/opm/materialize`'s retired `highestStable` skipped `-dev.*`; cli's own release tags never carry `dev`.
**Options considered**:
1. Prefix match on `-0.dev.`: exact to today's script, brittle to a script change.
2. Any dot-separated prerelease identifier equal to `dev`: covers `0.dev.N`, `dev.N`, `alpha.1.dev.3`; cannot collide with release prereleases, which use `alpha`/`beta`/`rc` counters.
3. A flag on publish: rejected, Principle VII and the workflow would have to know it.
**Decision**: Option 2, as `isDevTag(tag string) bool` in `compat.go` splitting `semver.Prerelease(tag)` on `.`.
**Rationale**: Robust to the script's exact shape while unambiguous; alpha/beta/rc identifiers are never `dev`.

### Where the skip lives
**Context**: Two callers, one policy.
**Options considered**:
1. Skip inside `compatScan`: the walk would still be entered and counters would be ambiguous.
2. Skip in each caller before the scan, with a shared outcome marker: `gateCompat` and `RegistryCheck` both check `isDevTag` first, set `g.CompatDevExempt = true`, and return without scanning; `predecessorVersions` filters dev tags for everyone.
**Decision**: Option 2.
**Rationale**: The scan stays a pure walk; the outcome struct carries the exemption so rendering has one source of truth; the window filter is a one-line addition inside the existing loop, so the baseline rule needs no second code path.

### Rendering
`renderCatalogGates` and the check report print `dev-exempt` on the compat row when `CompatDevExempt` is set; `CompatChecked` is set true for a dev build (the gate ran and decided), so the row is never "not completed". Existing wording for other builds is unchanged.

Signatures:

```go
// isDevTag reports whether tag's prerelease segment carries a dev identifier
// (branch-tag.sh's v<M>.<m>.<p>-0.dev.<count>.g<sha>); D26 clause 1.
func isDevTag(tag string) bool

type CatalogGateOutcomes struct {
    // existing counters ...
    CompatDevExempt bool // D26: dev build, neither judged nor a baseline
}
```

`predecessorVersions` gains `if isDevTag(v) { continue }` inside the loop.

### Error handling
No new error paths. A dev build with an unreachable registry still reports the lookup's `ConnectivityError` from `lookupPublished` (registry.go) before the gate runs; the gate itself performs no I/O for a dev build.

### Example output

Dry run of a dev-stamped tree:

```
  tag             v2.0.0-0.dev.1787900000.gdead000
  compat gate     dev-exempt (dev builds are neither judged nor a baseline)
```

Release tag, unchanged:

```
  compat gate     39 compared, 2 refused, 4 alpha-exempt, 0 new
```

Exit codes unchanged: 0 clean, 2 refused, 3 connectivity.

## Risks / Trade-offs

- [A dev build ships an incompatible shape that a release later inherits] → the release build is compared against the latest release, not the dev build, so the break is caught at the release gate; nothing is lost, only deferred to the tag that matters.
- [Someone tags a release with `dev` in it] → excluded from the gate; documented in the requirement and the `isDevTag` comment; release-please never emits such a tag.
- [`registry check --compat` on old dev builds already published] → now reports `dev-exempt`; previously they were compared against older dev builds, which was the non-deterministic verdict this change removes.
