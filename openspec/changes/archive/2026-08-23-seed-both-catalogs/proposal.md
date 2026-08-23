# Proposal: seed-both-catalogs

## Why

`catalog_opm` now publishes two CUE modules instead of one: `opmodel.dev/catalogs/opm@v2` (the
abstraction catalog) and `opmodel.dev/catalogs/k8s@v1` (the raw Kubernetes passthrough surface,
extracted from the `k8s-` prefixed family that lived inside the abstraction catalog). The seeded
`~/.opm/platform.cue` still names one catalog and its spec still says so explicitly, so a fresh
`opm config init` produces a platform that cannot match any raw contract. The escape hatch a module
author reaches for when an abstraction does not fit is simply absent from the default.

A platform subscribes to each catalog it wants, and the seeded default should name both. This
restores the pre-consolidation shape: the CLI default seeded two catalogs before enhancement 0010
D47 merged the catalogs into one.

## What Changes

- **The seeded `~/.opm/platform.cue` names both catalogs**, each with its own explicit pinned
  scalar `version`. **BREAKING** for the seeded artifact's shape: the file now carries two
  `registry` entries where the spec previously required exactly one.
- **`config.DefaultCatalogPath` becomes plural.** `DefaultCatalogPaths` becomes the source of truth
  naming both catalogs; `DefaultCatalogPath` survives as a derived alias to the OPM entry so the
  install-side single-catalog callers (explicitly out of scope, below) need no changes. See
  design.md.
- The mirror contract recorded in `CLAUDE.md` grows a fourth *file*: `internal/config/templates.go`,
  `hack/platform.cue`, `hack/kind-platform.yaml` (previously mirroring the pin without being named in
  the contract), and the operator's sample Platform each gain the second subscription and stay
  aligned in the same commit.

Explicitly **not** in this change:

- `opm operator install` resolving a version per catalog. Install resolves its subscription from
  the registry rather than from a hand-pinned literal, and the requirements governing that live in
  the `operator-install-platform` change, which is complete but not yet archived. Extending install
  to two catalogs is a follow-up written against those requirements once they land in the main
  specs.
- Any runtime change. `core`'s `#Platform.#registry` is keyed by module path and already admits any
  number of subscriptions; the materializer iterates them and the matcher is catalog-agnostic. A
  hand-authored platform file with two entries works today.

## Capabilities

**New Capabilities**: none.

**Modified Capabilities**:

- `config-commands` — the seeded platform's subscription set changes from exactly one entry to two,
  and the scenario that asserts the absence of a second catalog entry is replaced.

## Impact

- `internal/config/templates.go` — `DefaultCatalogPath`/`DefaultCatalogPaths` and
  `DefaultPlatformTemplate`.
- `internal/platform/catalog.go` — the re-export of the catalog path constants.
- `internal/cmd/config/init_test.go` — currently asserts the seeded file contains exactly one
  registry entry and no second catalog path.
- `hack/platform.cue`, `hack/kind-platform.yaml`, and the operator's sample Platform — mirror peers
  of the seeded template.
- `CLAUDE.md` — the mirror-contract note.

Not touched: `internal/cmd/operator/install.go` and its tests keep resolving `DefaultCatalogPath`
unchanged (design.md).

**Sequencing constraint — satisfied.** The seeded pin is a hand-bumped literal and `opm config init`
is normatively offline, so it cannot resolve "latest". The template's stated invariant is that an
immutable published tag never dangles, which holds only for a version that exists. This change
required the first release of `opmodel.dev/catalogs/k8s@v1` to be published to GHCR before merging;
that release (`1.0.0-alpha.1`) and a compatible `opmodel.dev/catalogs/opm@v2` build (`2.0.0-alpha.5`)
are now published and verified (2026-08-22).
