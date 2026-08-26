## Context

See proposal.md. After `compat-dev-tag-exemption`, `gateCompat` and `RegistryCheck` already classify the effective tag once (`isDevTag`) before the scan. `eligibleByPackage` (`compat.go`) partitions members: transformers out, unparseable apiVersion out, alpha into `CompatAlpha`, the rest into the walk. `CatalogGateOutcomes` is rendered by `plan.go:renderCatalogGates` and `check.go`.

Constraints: 0011 D26 clause 2. 0010 D34's member-level model is unchanged; the library's `CheckAtLevel` keeps keying on the member's apiVersion and must not learn about module versions (its doc comment says so explicitly; the module version is "an independent axis"). The policy therefore lives in the cli gate, not in `library/opm/compat`.

## Goals / Non-Goals

**Goals:**
- Prerelease-line builds pass the gate with the exemption counted and printed.
- Stable-line behaviour and every existing test unchanged.
- One classifier over the effective tag shared by publish and check.

**Non-Goals:**
- Teaching `library/opm/compat` about module versions.
- Changing which builds are predecessors (D23 stands; dev filtering is the sibling change).
- Comparator noise (library change).

## Decisions

### Research & Decisions

### Where the exemption is applied
**Context**: The exemption is per build, not per member, but the summary wants a per-member count.
**Options considered**:
1. Skip the whole gate like the dev case and print `prerelease-exempt`: cheap, but loses the counts the issue explicitly asks for and cannot distinguish "4 alpha members, 39 exempt" from "43 exempt".
2. Pass a `lineIsPrerelease bool` into `eligibleByPackage`; beta/GA members are counted into `g.CompatPrerelease` and excluded from `byPkg`, so the scan runs with an empty walk set, performs no registry loads, and the row reads `0 compared, 0 refused, 4 alpha-exempt, 39 prerelease-exempt, 0 new`.
**Decision**: Option 2.
**Rationale**: The counts stay honest and the code path is the existing partition with one more bucket; no registry I/O happens because `pkgs` is empty.

### Classifier
`semver.Prerelease(tag) != "" && !isDevTag(tag)` as `isReleasePrerelease(tag string) bool` next to `isDevTag`. Both callers compute it once.

Signatures:

```go
func isReleasePrerelease(tag string) bool
func eligibleByPackage(members []Member, lineIsPrerelease bool, g *CatalogGateOutcomes) (map[string][]Member, []string)

type CatalogGateOutcomes struct {
    // ...
    CompatPrerelease int // D26 clause 2: beta/GA members exempt on a prerelease module line
}
```

### Rendering
`renderCatalogGates` prints `%d compared, %d refused, %d alpha-exempt, %d prerelease-exempt, %d new`; `check.go` mirrors it with `violating`. The `prerelease-exempt` segment is always printed (0 on a stable line) so the row's shape is stable for log greps.

### Error handling
No new errors. `--version` assertion, "already holds" and connectivity refusals are unaffected and still precede or accompany the gate.

### Example output

```
  tag             v2.0.0-alpha.5
  compat gate     0 compared, 0 refused, 4 alpha-exempt, 39 prerelease-exempt, 0 new
```

```
  tag             v2.0.0
  compat gate     39 compared, 1 refused, 4 alpha-exempt, 0 prerelease-exempt, 0 new
```

Exit codes unchanged.

## Risks / Trade-offs

- [A real break ships on the alpha line and is only caught at `2.0.0`] → by D26 that is the accepted trade; the first stable tag compares against the newest alpha, so the break surfaces there with the alpha coordinate named, and the author fixes or bumps before the stable ships.
- [Exemption masks comparator noise] → sequenced after the library change; task 3.3 asserts that on a stable tag the known-noisy real members (`realtree_test.go`) compare clean, so the stable path is proven before the exemption lands.
- [Log consumers grep the old four-segment row] → `catalog_opm/ci.yml` greps only `already holds` and `1 refusal`; the row gains a segment, existing segments keep their order.
