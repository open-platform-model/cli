# test-fixture-lineage

## Purpose

Keeps the repo's fixtures and examples on the current published schema line and its tests free of sibling-checkout dependencies. (Test-infrastructure capability; precedent: `validation-gates`, `kind-cluster-tasks`, `ci-workflow`.)

## Requirements

### Requirement: Tests depend only on repo-local fixtures

No test or integration program SHALL read fixtures from a sibling repository checkout. Vendored copies SHALL carry a provenance header naming their origin.

#### Scenario: render-parity is self-contained

- **WHEN** the render-parity program runs in a standalone clone of this repo (no sibling checkouts)
- **THEN** it SHALL locate its module fixture under this repo's `tests/fixtures/` and proceed to the registry-gated comparison

### Requirement: Maintained fixtures and examples import only the current published schema line
Fixtures and examples consumed by tests or presented as current documentation SHALL import only the current published schema line (`opmodel.dev/core@v2` and the versioned `opmodel.dev/catalogs/opm@v4` packages). Artifacts kept for a retired line SHALL live under an explicitly marked legacy location with a note naming the line they document, and SHALL NOT be consumed by any test.

#### Scenario: No retired-schema imports outside legacy

- **WHEN** the repo is grepped for retired-line imports (`opmodel.dev/core@v1`, `core/v1alpha1`, `modulerelease@v1`, `opm/v1alpha1`) outside the marked legacy location
- **THEN** there SHALL be no matches in fixtures, examples, or test inputs

#### Scenario: Vet fixtures exercise current schema

- **WHEN** the module vet tests run
- **THEN** their fixtures SHALL be `core@v2`-line modules exercising the same behaviors (valid module, debug values) as before the port

### Requirement: A maintained module fixture's identity package declares Version as a plain string literal
A maintained module fixture's identity package SHALL declare `Version` as a plain string literal with no default arm and no local `#VersionType`, and its `metadata.version` SHALL reference that literal directly. A fixture SHALL load through the kernel's module loader with no interpolation or other workaround in `metadata`.

#### Scenario: Fixture identity is a literal

- **WHEN** `tests/fixtures/modules/*/identity/identity.cue` is read
- **THEN** each declares `Version: "<semver>"` with no `|`, no `*` and no `#VersionType`
- **AND** the matching `module.cue` declares `version: id.Version`
- **AND** `opm module build` on the fixture passes the loader shape gate

### Requirement: A fixture does not depend on a catalog mechanism the catalog has withdrawn
A fixture SHALL NOT depend on a catalog mechanism the catalog has withdrawn. When a catalog member or contract type a fixture exercises is removed from the published catalog, the fixture is removed with it rather than kept unvetted; a replacement fixture is added when the replacement mechanism exists.

#### Scenario: No fixture exercises a withdrawn catalog mechanism

- **WHEN** `tests/fixtures/` is grepped for the legacy secret vocabulary (`res.#Secret`, `$secretName`, `$dataKey`, `#AutoSecrets`, `opm-secrets`)
- **THEN** there SHALL be no matches

### Requirement: A test fixture module declares its path under the testing domain
A test fixture module SHALL declare a module path under `testing.opmodel.dev/modules/cli/`, and SHALL NOT declare one under `opmodel.dev/`.
The rule is mechanical, not stylistic: CUE resolves modules by longest-prefix match on the module path, so a fixture declared under `opmodel.dev/` forces that entire prefix, core and the catalogs included, onto whatever registry serves the fixture.

#### Scenario: No fixture occupies the production namespace

- **WHEN** `tests/fixtures/modules/*/cue.mod/module.cue` is inspected
- **THEN** every declared `module:` path SHALL begin with `testing.opmodel.dev/modules/cli/`

### Requirement: Each fixture tree carries an identity package and is published to GHCR by repository CI
Each fixture tree SHALL carry an `identity/` package as the single source of its path and version, with `metadata.name`, `metadata.modulePath`, and `metadata.version` derived from it.
Fixture trees SHALL be published to GHCR by repository CI through `opm module publish`, so every consumer (examples, e2e testdata, the kind dev cluster, a fresh clone) resolves them from a public registry with no local registry involved.

#### Scenario: A fixture passes the publish gates

- **WHEN** `opm module publish --dry-run` runs over a fixture tree
- **THEN** the plan SHALL resolve with no refusals, deriving the registry repository and tag from the fixture's own identity package

#### Scenario: Consumers resolve fixtures without a local registry

- **WHEN** the examples, the e2e testdata, or an integration program resolves its fixture module with only the default GHCR registry mapping configured and no local registry running
- **THEN** resolution SHALL succeed

### Requirement: A consumer that names a fixture version pins every shared dependency as CUE resolves it
A consumer that names a fixture version SHALL pin every dependency it shares with that fixture exactly as CUE resolves it for that version: what `cue mod get <fixture>@<version>` followed by `cue mod tidy` writes.
CUE keeps a dependency the consumer already lists at its listed version, so a consumer re-pinned to a new fixture version but left on an older core still passes `cue mod tidy --check` and evaluates against the older core.

#### Scenario: A version bump is an identity edit

- **WHEN** a fixture's version changes
- **THEN** the edit SHALL be to that fixture's `identity/identity.cue`, and every consumer naming the version SHALL be re-pinned to match
- **AND** every core and catalog pin the consumer shares with the fixture SHALL move to what CUE resolves for the new version

### Requirement: `hack/fixtures.sh consumers` checks each named consumer in a scratch copy against CUE's resolution
`hack/fixtures.sh consumers <dir>...` SHALL check each named consumer in a scratch copy: `cue mod get <fixture>@<pinned version>` for every `testing.opmodel.dev` dependency it pins, then `cue mod tidy`, then a comparison with the committed `cue.mod/module.cue`.
`hack/fixtures.sh consumers <dir>...` SHALL resolve through `CUE_REGISTRY`, defaulting to GHCR.

#### Scenario: Consumers that follow their fixture pass

- **WHEN** every named consumer pins what CUE resolves for its fixture versions
- **THEN** the subcommand prints `ok` for each and exits zero

#### Scenario: Stale shared pin is reported with the diff

- **WHEN** a consumer pins the fixture's version but an older core than the fixture requires
- **THEN** the subcommand prints the unified diff and a `FAIL` line naming the consumer, and exits non-zero

### Requirement: `hack/fixtures.sh consumers` checks every named consumer, prints a FAIL line for each failure and exits non-zero
`hack/fixtures.sh consumers <dir>...` SHALL check every named consumer before it exits, SHALL print a `FAIL <consumer>: <reason>` line for each failure (a difference, any `cue` error, a missing `cue.mod/module.cue`, a consumer that pins no fixture), and SHALL exit non-zero when any consumer failed. The scratch copies SHALL be removed on every exit path. (Requirement: With FIX=1 `hack/fixtures.sh consumers` writes the resolved pins and does not fail)

#### Scenario: Unresolvable fixture version is a named failure

- **WHEN** a consumer pins a fixture version the configured registry does not hold
- **THEN** the subcommand prints CUE's error and a `FAIL` line naming the consumer and the version, still checks the remaining consumers, and exits non-zero without leaving its scratch copy behind

### Requirement: A tracked cue.mod/module.cue that pins a fixture and is not named fails the check
A git-tracked `cue.mod/module.cue` outside the fixtures directory that pins a `testing.opmodel.dev` dependency but is not named SHALL also fail. (Requirement: `hack/fixtures.sh consumers` checks every named consumer, prints a FAIL line for each failure and exits non-zero)

#### Scenario: Unlisted consumer

- **WHEN** a tracked `cue.mod/module.cue` outside the fixtures directory pins a fixture and is not among the named consumers
- **THEN** the subcommand prints a `FAIL` line naming that file and exits non-zero

### Requirement: With FIX=1 `hack/fixtures.sh consumers` writes the resolved pins and does not fail
With `FIX=1` a differing consumer's `module.cue` SHALL be overwritten with the resolved one and SHALL NOT fail. (Requirement: `hack/fixtures.sh consumers` checks each named consumer in a scratch copy against CUE's resolution)

#### Scenario: Fix mode writes the resolved pins

- **WHEN** the subcommand runs with `FIX=1` and a consumer differs
- **THEN** that consumer's `module.cue` equals the resolved one afterwards and the run exits zero

### Requirement: `hack/fixtures.sh` stays byte-identical with opm-operator's copy
`hack/fixtures.sh` SHALL stay byte-identical with opm-operator's copy; each repo names its own consumers.

#### Scenario: The script is byte-identical with opm-operator's copy

- **WHEN** `hack/fixtures.sh` is compared with opm-operator's copy
- **THEN** the two files SHALL be byte-identical, and each repo names its own consumers

### Requirement: A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason
Each `opmodel.dev/core@v2` or `opmodel.dev/catalogs/*` pin in a test input that tests resolve from a registry SHALL be moved to the release cascade's target, unless `.cascade-frozen` lists it with a reason.
- **Test inputs** include a test module's `cue.mod/module.cue` and a module file a test writes from a Go literal. (Requirement: A core pin already newer than the cascade target stays) (Requirement: A pin that is old only because a tree was copied or an upstream release moved on is not frozen)

#### Scenario: Every old core or catalog pin in a test module is accounted for

- **WHEN** the tracked `cue.mod/module.cue` files under `tests/` and `internal/` are read, and their `opmodel.dev/core@v2` and `opmodel.dev/catalogs/*` pins are compared with the cascade target (the newest published catalog, and the core that catalog pins)
- **THEN** each one that is older than its target is at or under a path listed in `.cascade-frozen`, with that module path among its pins

#### Scenario: Bumped test trees still pass

- **WHEN** the module vet, module eval, render, instance-init, skip-unprovided and duplicate-identity tests run against the bumped trees, with the canonical registry mapping, which maps both `testing.opmodel.dev` and `opmodel.dev` to `ghcr.io/open-platform-model`
- **THEN** they pass without skipping for an unresolvable core

### Requirement: The targets of a catalog pin and a core pin follow the release cascade
- **The target.** A catalog pin's target SHALL be the newest published release in its major. A core pin's target is the core version that the input's catalog pins, or, for an input that pins no catalog, the core that `templates/minimal`'s catalog pins. The source is workspace RELEASING.md, section "The cascade", "Consistent set". (Requirement: A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason)

#### Scenario: The target of a catalog pin is the newest published release in its major

- **WHEN** a catalog pin in a test input is moved to the release cascade's target
- **THEN** the target is the newest published release in its major

### Requirement: A core pin already newer than the cascade target stays
- **No backwards moves.** A core pin already newer than that target SHALL stay. (Requirement: The targets of a catalog pin and a core pin follow the release cascade)

#### Scenario: A core pin newer than its target stays

- **WHEN** a core pin is already newer than that target
- **THEN** the pin stays

### Requirement: A `cue.mod` pin moves by `cue mod get` then `cue mod tidy` and is never bumped by a hand edit
- **How a pin moves.** A `cue.mod` file moves by `cue mod get` then `cue mod tidy`, which `task deps:cascade` runs for the inputs it lists. A Go literal moves by editing the literal. A `cue.mod` file SHALL NOT be bumped by a hand edit. (Requirement: A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason)

#### Scenario: A cue.mod pin moves by cue mod get then cue mod tidy

- **WHEN** a pin in a `cue.mod` file moves
- **THEN** it moves by `cue mod get` then `cue mod tidy`, and the `cue.mod` file SHALL NOT be bumped by a hand edit

### Requirement: A `.cascade-frozen` entry names the file or directory, the frozen module paths and a one-sentence reason
- **`.cascade-frozen` entries.** Each entry SHALL name the repo-relative file or directory, the module paths frozen there, and a one-sentence reason saying why the pin must stay old. The file uses the format in workspace RELEASING.md, section "Cascade files".
A literal that a test only compares as expected output and never resolves from a registry (a golden) MAY also be listed, with a reason saying so, so that it is not mistaken for a stale pin.

#### Scenario: A deliberately old pin carries its reason

- **WHEN** `.cascade-frozen` is read
- **THEN** every entry has a non-empty `path` that exists in the repo, a non-empty `pins` list, and a non-empty `reason`
- **AND** it lists `tests/e2e/instance_build_test.go` with `opmodel.dev/core@v2` and `opmodel.dev/catalogs/opm@v4`, whose tests need an older core to be refused, the collision floor, and an older catalog to show version skew
- **AND** it lists `internal/cmd/platform/check_test.go` with `opmodel.dev/core@v2`, whose tests need older cores to be refused and the collision floor
- **AND** it lists `internal/instinit/render_test.go` with `opmodel.dev/core@v2`, whose golden asserts rendered text that carries the old core version

#### Scenario: A golden may be listed with a reason

- **WHEN** a literal that a test only compares as expected output and never resolves from a registry (a golden) is listed in `.cascade-frozen`
- **THEN** its reason says so, so that it is not mistaken for a stale pin

### Requirement: A pin is frozen only when the test depends on that exact old version
A pin SHALL be frozen only when the test depends on that exact old version. Examples:

- a platform that must be refused as too old;
- a platform that must show catalog version skew;
- a floor that the test re-pins up to. (Requirement: A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason)

#### Scenario: A frozen pin is one the test needs at that old version

- **WHEN** a pin is frozen
- **THEN** the test depends on that exact old version, such as a platform that must be refused as too old, a platform that must show catalog version skew, or a floor that the test re-pins up to

### Requirement: A pin that is old only because a tree was copied or an upstream release moved on is not frozen
A pin that is old only because a tree was copied from an older tree, or because an upstream release moved on, SHALL NOT be frozen. Between an upstream release and the release cascade that follows it, a pin is behind but not in breach. (Requirement: A test pin is moved to the release cascade's target unless `.cascade-frozen` lists it with a reason)

#### Scenario: A pin that is old only because a tree was copied is not frozen

- **WHEN** a pin is old only because a tree was copied from an older tree, or because an upstream release moved on
- **THEN** the pin is not frozen

### Requirement: `hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture
`hack/fixtures.sh` SHALL treat each directory under a catalog fixture root as a catalog fixture: the root is `CATALOGS_DIR`, defaulting to the `catalogs/` sibling of the module fixture root when that directory exists, and absent otherwise.
With no catalog root the script SHALL behave as it does for module fixtures alone.

#### Scenario: No catalog root leaves the module flow unchanged

- **WHEN** `hack/fixtures.sh check` runs in this repo, which has no `tests/fixtures/catalogs`
- **THEN** it gates only the module fixtures through `opm module publish --dry-run`, as before

### Requirement: A catalog fixture goes through `opm catalog publish` wherever a module fixture goes through `opm module publish`
A catalog fixture SHALL go through `opm catalog publish` (and `opm catalog version set` for a pre-release tag) wherever a module fixture goes through `opm module publish`, so `check` runs every catalog publish gate and enforces changed-implies-bumped, `seed` and `publish` push it, and `pins` lists it, catalogs before modules. (Requirement: `hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture)

#### Scenario: A catalog fixture publishes through the catalog pipeline

- **WHEN** a repo carries `<fixtures>/catalogs/<name>/` with an identity package and `hack/fixtures.sh seed` runs
- **THEN** the catalog is published with `opm catalog publish` at the version its identity package declares, before any module fixture

#### Scenario: A changed catalog fixture must be bumped

- **WHEN** `hack/fixtures.sh check` finds a catalog fixture changed since `BASE_REF` at a version the upstream registry already holds
- **THEN** it prints a `FAIL` line naming the fixture and `opm catalog version set`, and exits non-zero

### Requirement: `hack/fixtures.sh consumers` does not report a `cue.mod` under the catalog root as an unlisted consumer
`consumers` SHALL NOT report a `cue.mod` under the catalog root as an unlisted consumer. (Requirement: `hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture)

#### Scenario: A cue.mod under the catalog root is not an unlisted consumer

- **WHEN** `hack/fixtures.sh consumers` finds a `cue.mod` under the catalog root
- **THEN** it SHALL NOT report it as an unlisted consumer

### Requirement: `tests/fixtures/fixtures.go` reads a catalog fixture's coordinate from its identity package
`tests/fixtures/fixtures.go` SHALL read a catalog fixture's coordinate from its identity package through `LoadCatalog` and `MustCatalog`. Both files SHALL stay byte-identical with opm-operator's copies. (Requirement: `hack/fixtures.sh` treats each directory under a catalog fixture root as a catalog fixture)

#### Scenario: A catalog fixture's coordinate comes from its identity package

- **WHEN** `tests/fixtures/fixtures.go` reads a catalog fixture's coordinate
- **THEN** it SHALL read it from the fixture's identity package through `LoadCatalog` and `MustCatalog`

#### Scenario: The fixture files are byte-identical with opm-operator's copies

- **WHEN** `hack/fixtures.sh` and `tests/fixtures/fixtures.go` are compared with opm-operator's copies
- **THEN** both files SHALL be byte-identical
