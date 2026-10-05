## MODIFIED Requirements

### Requirement: Consumer pins are checked against CUE's resolver

`hack/fixtures.sh consumers <dir>...` SHALL check each named consumer in a scratch copy: `cue mod get <fixture>@<pinned version>` for every `testing.opmodel.dev` dependency it pins, then `cue mod tidy`, then a comparison with the committed `cue.mod/module.cue`. It SHALL resolve through `CUE_REGISTRY`, defaulting to GHCR. It SHALL check every named consumer before it exits, SHALL print a `FAIL <consumer>: <reason>` line for each failure (a difference, any `cue` error, a missing `cue.mod/module.cue`, a consumer that pins no fixture), and SHALL exit non-zero when any consumer failed. A git-tracked `cue.mod/module.cue` outside the fixtures directory that pins a `testing.opmodel.dev` dependency but is not named SHALL also fail. The scratch copies SHALL be removed on every exit path. With `FIX=1` a differing consumer's `module.cue` SHALL be overwritten with the resolved one and SHALL NOT fail. The script SHALL stay byte-identical with opm-controller's copy; each repo names its own consumers.

#### Scenario: Consumers that follow their fixture pass

- **WHEN** every named consumer pins what CUE resolves for its fixture versions
- **THEN** the subcommand prints `ok` for each and exits zero

#### Scenario: Stale shared pin is reported with the diff

- **WHEN** a consumer pins the fixture's version but an older core than the fixture requires
- **THEN** the subcommand prints the unified diff and a `FAIL` line naming the consumer, and exits non-zero

#### Scenario: Unresolvable fixture version is a named failure

- **WHEN** a consumer pins a fixture version the configured registry does not hold
- **THEN** the subcommand prints CUE's error and a `FAIL` line naming the consumer and the version, still checks the remaining consumers, and exits non-zero without leaving its scratch copy behind

#### Scenario: Unlisted consumer

- **WHEN** a tracked `cue.mod/module.cue` outside the fixtures directory pins a fixture and is not among the named consumers
- **THEN** the subcommand prints a `FAIL` line naming that file and exits non-zero

#### Scenario: Fix mode writes the resolved pins

- **WHEN** the subcommand runs with `FIX=1` and a consumer differs
- **THEN** that consumer's `module.cue` equals the resolved one afterwards and the run exits zero

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
SHALL stay byte-identical with opm-controller's copies.

#### Scenario: No catalog root leaves the module flow unchanged

- **WHEN** `hack/fixtures.sh check` runs in this repo, which has no `tests/fixtures/catalogs`
- **THEN** it gates only the module fixtures through `opm module publish --dry-run`, as before

#### Scenario: A catalog fixture publishes through the catalog pipeline

- **WHEN** a repo carries `<fixtures>/catalogs/<name>/` with an identity package and `hack/fixtures.sh seed` runs
- **THEN** the catalog is published with `opm catalog publish` at the version its identity package declares, before any module fixture

#### Scenario: A changed catalog fixture must be bumped

- **WHEN** `hack/fixtures.sh check` finds a catalog fixture changed since `BASE_REF` at a version the upstream registry already holds
- **THEN** it prints a `FAIL` line naming the fixture and `opm catalog version set`, and exits non-zero
