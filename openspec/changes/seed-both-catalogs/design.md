## Context

The sequencing constraint in proposal.md is now satisfied: `opmodel.dev/catalogs/k8s@v1` published
`1.0.0-alpha.1` and `opmodel.dev/catalogs/opm@v2` published `2.0.0-alpha.5` to GHCR on 2026-08-22,
verified against the registry directly (not just the release tag). These are the two pins Group 1
and Group 2 seed.

`config.DefaultCatalogPath` today is a `const string` with exactly two direct consumers of the
*single-catalog* shape that stay out of scope for this change: `internal/cmd/operator/install.go`
(two call sites, `ResolveCatalogVersion` and `EnsureClusterPlatformForCatalog`) and four test files
(`operator_test.go`, and `internal/platform/catalog_test.go` / `cluster_test.go`) that assert against
it. `opm operator install` resolves its subscription from the registry rather than a hand-pinned
literal (see proposal.md - Not in this change), so it has no use for a second catalog path — it
needs to keep naming exactly one.

Separately, the hand-bumped-pin mirror set documented in `cli/CLAUDE.md` and in
`DefaultPlatformTemplate`'s own comment names three files (`templates.go` as source,
`hack/platform.cue`, the operator's sample Platform). A fourth file carries the identical pin and
mirrors the same two documented ones by its own header comment, but isn't named in that contract:
`cli/hack/kind-platform.yaml`, consumed by `task cluster:operator` in the kind dev flow.

## Goals / Non-Goals

**Goals:**
- Give `config.DefaultCatalogPath`'s single-catalog callers (`install.go` and its tests) a zero-diff
  path through this change, so the proposal's stated out-of-scope boundary around
  `opm operator install` holds in practice, not just in wording.
- Make the mirror-pin contract match the files that actually carry the pin.

**Non-Goals:**
- Changing how `opm operator install` resolves or seeds catalog subscriptions (tracked separately
  against `operator-install-platform`, per proposal.md).
- Any runtime/materializer change (per proposal.md, `#Platform.#registry` already admits N
  subscriptions).

## Decisions

### `DefaultCatalogPaths` as source of truth, `DefaultCatalogPath` as a derived alias

`internal/config` gets:

```go
var DefaultCatalogPaths = []string{
	"opmodel.dev/catalogs/opm@v2",
	"opmodel.dev/catalogs/k8s@v1",
}

// DefaultCatalogPath is the entry `opm operator install` resolves a
// single-catalog Platform against. Derived from DefaultCatalogPaths, not a
// second literal — each catalog path is still spelled exactly once.
var DefaultCatalogPath = DefaultCatalogPaths[0]
```

`internal/platform/catalog.go` re-exports both names unchanged in kind (re-export, not a second
source of truth). `install.go` and its four test files need **no edits** — `platform.DefaultCatalogPath`
keeps resolving to the OPM catalog path exactly as it does today.

Both become `var`, not `const`: a slice cannot be a Go constant. No caller in the codebase depends on
`DefaultCatalogPath` being a compile-time constant (no use in a `const` expression, array size, or
`case` on a typed constant), so this is a safe widening.

**Alternatives considered:**
- *Positional index at each call site* (`DefaultCatalogPaths[0]`, no alias) — rejected: pushes
  `// opm — the abstraction catalog` comments onto ~6 call sites instead of one declaration, and
  `install.go`/its tests would need to change even though the proposal scopes them out.
- *Keyed map* (`map[string]string{"opm": ..., "k8s": ...}`) — rejected: introduces a catalog-family
  string-key vocabulary (`"opm"`, `"k8s"`) that doesn't exist anywhere else in the domain model, and
  map iteration order isn't stable, so the template renderer would still need a fixed-order list
  alongside it.

### The mirror-pin contract grows to four files

`cli/hack/kind-platform.yaml` is added as a fourth mirror peer alongside `hack/platform.cue` and the
operator's sample Platform, both in the `cli/CLAUDE.md` mirror-contract note and as an explicit task
item (previously it mirrored the pin by convention, undocumented). No behavior changes — it already
gets applied by `task cluster:operator` today; this only makes the existing dependency visible so a
future pin bump doesn't miss it.

### Pins

```
opmodel.dev/catalogs/opm@v2:  2.0.0-alpha.5   (bumped from 2.0.0-alpha.3)
opmodel.dev/catalogs/k8s@v1:  1.0.0-alpha.1   (new)
```

Both confirmed published (non-dev tags) on GHCR before this design was written, satisfying the
proposal's never-dangles invariant.

## Risks / Trade-offs

- **Derived alias goes stale if `DefaultCatalogPaths`'s order changes.** `DefaultCatalogPath` always
  means index 0 by construction — reordering the slice silently repoints
  `opm operator install`. Mitigated by the doc comment on `DefaultCatalogPath` stating what it names
  and why, and by `install.go`'s tests still asserting against the OPM path's literal value, not just
  the alias.
- **`kind-platform.yaml` was already drifting silently before this change** (undocumented mirror).
  Adding it to the contract here is a one-time catch-up; if a fifth undocumented mirror exists
  elsewhere, this change doesn't discover it. No further audit is in scope.
