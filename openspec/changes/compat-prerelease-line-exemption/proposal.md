## Why

Enhancement 0011 D9 keys the compatibility promise to a member's own `apiVersion` (0010 D34), so a `v1beta1` member on the `2.0.0-alpha.N` module line is already bound to additive-only evolution. `catalog_opm`'s constitution permits `feat!:` to tighten a published constraint while the line is alpha, and enhancement 0019 lands three such slices (`catalog-name-constraints`, D15 removal, D19 carve-outs). PR 51 is refused today on `#ExposeTrait.spec.expose.name` (cli issue 165). 0011 D26 clause 2 resolves the collision: while the module's own version is a prerelease, beta/GA members are exempt, visibly, and the gate arms at the first stable tag of the major.

Ordering: this change lands after `compat-dev-tag-exemption` and after the library change fixing the comparator's nested-provenance and matchN false positives; without the latter the exemption would hide comparator defects that will refuse legitimately additive changes on a stable line.

## What Changes

- `opm catalog publish`: when the effective tag carries a non-dev prerelease identifier (`-alpha.N`, `-beta.N`, `-rc.N`), beta and GA members are counted into a new `prerelease-exempt` bucket instead of being compared. The `compat gate` row renders the bucket next to `alpha-exempt`.
- `opm catalog registry check --compat` applies the same rule to the fetched build's version.
- The stable-line path is untouched: every existing refusal case in `compat_test.go` runs against stable effective tags and stays as is; the new tests pin that a stable tag after a prerelease line is compared against the prerelease predecessors as before.
- No new flags. PATCH per SemVer: a prerelease-line publish stops refusing on compat; nothing a stable line depends on changes.

## Capabilities

### New Capabilities

(none)

### Modified Capabilities

- `artifact-publishing`: the compatibility gate requirement gains the prerelease-module-line exemption and its visible bucket.
- `catalog-registry-check`: `--compat` reports `prerelease-exempt` for prerelease-line builds.

## Impact

- `internal/publish/compat.go` (`gateCompat`, `eligibleByPackage` or a wrapper, `CatalogGateOutcomes.CompatPrerelease`), `plan.go`, `check.go`, tests.
- Downstream: `catalog_opm` `ci.yml` dry-run on PR 51 passes with `2 prerelease-exempt`; `catalog_opm` bumps its pinned `opm` afterwards (delivery).
- Implements enhancement 0011 D26 (clause 2); see `enhancement.yaml`.
