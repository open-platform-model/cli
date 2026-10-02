## ADDED Requirements

### Requirement: The shared fixture flow publishes catalog fixtures

`hack/fixtures.sh` SHALL treat each directory under a catalog fixture root as a catalog fixture: the
root is `CATALOGS_DIR`, defaulting to the `catalogs/` sibling of the module fixture root when that
directory exists, and absent otherwise. A catalog fixture SHALL go through `opm catalog publish`
(and `opm catalog version set` for a pre-release tag) wherever a module fixture goes through
`opm module publish`, so `check` runs every catalog publish gate and enforces changed-implies-bumped,
`seed` and `publish` push it, and `pins` lists it, catalogs before modules. `consumers` SHALL NOT
report a `cue.mod` under the catalog root as an unlisted consumer. With no catalog root the script
SHALL behave as it does for module fixtures alone. `tests/fixtures/fixtures.go` SHALL read a catalog
fixture's coordinate from its identity package through `LoadCatalog` and `MustCatalog`. Both files
SHALL stay byte-identical with opm-operator's copies.

#### Scenario: No catalog root leaves the module flow unchanged

- **WHEN** `hack/fixtures.sh check` runs in this repo, which has no `tests/fixtures/catalogs`
- **THEN** it gates only the module fixtures through `opm module publish --dry-run`, as before

#### Scenario: A catalog fixture publishes through the catalog pipeline

- **WHEN** a repo carries `<fixtures>/catalogs/<name>/` with an identity package and `hack/fixtures.sh seed` runs
- **THEN** the catalog is published with `opm catalog publish` at the version its identity package declares, before any module fixture

#### Scenario: A changed catalog fixture must be bumped

- **WHEN** `hack/fixtures.sh check` finds a catalog fixture changed since `BASE_REF` at a version the upstream registry already holds
- **THEN** it prints a `FAIL` line naming the fixture and `opm catalog version set`, and exits non-zero
