## MODIFIED Requirements

### Requirement: Fixtures live on the testing domain and are published

A test fixture module SHALL declare a module path under `testing.opmodel.dev/modules/cli/`, and SHALL NOT declare one under `opmodel.dev/`. Each fixture tree SHALL carry an `identity/` package as the single source of its path and version, with `metadata.name`, `metadata.modulePath`, and `metadata.version` derived from it. Fixture trees SHALL be published to GHCR by repository CI through `opm module publish`, so every consumer (examples, e2e testdata, the kind dev cluster, a fresh clone) resolves them from a public registry with no local registry involved.

A consumer that names a fixture version SHALL pin every dependency it shares with that fixture exactly as CUE resolves it for that version: what `cue mod get <fixture>@<version>` followed by `cue mod tidy` writes. CUE keeps a dependency the consumer already lists at its listed version, so a consumer re-pinned to a new fixture version but left on an older core still passes `cue mod tidy --check` and evaluates against the older core.

The rule is mechanical, not stylistic: CUE resolves modules by longest-prefix match on the module path, so a fixture declared under `opmodel.dev/` forces that entire prefix, core and the catalogs included, onto whatever registry serves the fixture.

#### Scenario: No fixture occupies the production namespace

- **WHEN** `tests/fixtures/modules/*/cue.mod/module.cue` is inspected
- **THEN** every declared `module:` path SHALL begin with `testing.opmodel.dev/modules/cli/`

#### Scenario: A fixture passes the publish gates

- **WHEN** `opm module publish --dry-run` runs over a fixture tree
- **THEN** the plan SHALL resolve with no refusals, deriving the registry repository and tag from the fixture's own identity package

#### Scenario: Consumers resolve fixtures without a local registry

- **WHEN** the examples, the e2e testdata, or an integration program resolves its fixture module with only the default GHCR registry mapping configured and no local registry running
- **THEN** resolution SHALL succeed

#### Scenario: A version bump is an identity edit

- **WHEN** a fixture's version changes
- **THEN** the edit SHALL be to that fixture's `identity/identity.cue`, and every consumer naming the version SHALL be re-pinned to match
- **AND** every core and catalog pin the consumer shares with the fixture SHALL move to what CUE resolves for the new version

## ADDED Requirements

### Requirement: Consumer pins are checked against CUE's resolver

`hack/fixtures.sh consumers <dir>...` SHALL check each named consumer in a scratch copy: `cue mod get <fixture>@<pinned version>` for every `testing.opmodel.dev` dependency it pins, then `cue mod tidy`, then a comparison with the committed `cue.mod/module.cue`. It SHALL resolve through `CUE_REGISTRY`, defaulting to GHCR. It SHALL check every named consumer before it exits, SHALL print a `FAIL <consumer>: <reason>` line for each failure (a difference, any `cue` error, a missing `cue.mod/module.cue`, a consumer that pins no fixture), and SHALL exit non-zero when any consumer failed. A git-tracked `cue.mod/module.cue` outside the fixtures directory that pins a `testing.opmodel.dev` dependency but is not named SHALL also fail. The scratch copies SHALL be removed on every exit path. With `FIX=1` a differing consumer's `module.cue` SHALL be overwritten with the resolved one and SHALL NOT fail. The script SHALL stay byte-identical with opm-operator's copy; each repo names its own consumers.

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
