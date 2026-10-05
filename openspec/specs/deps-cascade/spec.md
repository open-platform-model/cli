# deps-cascade Specification

## Purpose
`task deps:cascade` moves the cli's upstream pins (library, the opm-operator module, and the opm catalog and core in the templates and test trees) to the newest published versions the release cascade allows. Its title and body tasks describe the result for the rolling `deps/cascade` pull request. The design is in workspace RELEASING.md, section "The cascade". The shared resolver in `open-platform-model/.github` answers every version question.

## Requirements

### Requirement: The cascade task moves the cli's upstream pins to published versions

`task -x deps:cascade` SHALL move the following pins in the working tree only. It SHALL NOT commit, branch or push.

- **library.** `github.com/open-platform-model/library` in `go.mod` SHALL move to the newest version the Go proxy serves in its major, through `go get <module>@<exact version>` followed by `go mod tidy`.
- **opm-operator module.** The operator module pin in `internal/operator/pin.go` (`PinnedModuleVersion` and `PinnedOperatorVersion`) SHALL move only through `hack/operator-pin <v>`, the program behind `task operator:pin VERSION=<v>`, built from the merge base like `opm`. The target is the newest published release of `opmodel.dev/modules/opm_operator` in the pinned major whose stated operator version, read from the module's source without a render, has a `MAJOR.MINOR` not above the cli's own, the same rule install applies to a target module version.
- **Catalog and core as a consistent set.** This covers the `cue.mod/module.cue` files of the three templates, `hack/platform`, `examples`, `tests/fixtures/modules/podinfo`, `tests/e2e/testdata/operator-owned`, `internal/instinit/testdata/initvalues`, `internal/workflow/render/testdata/skip-unprovided`, `tests/e2e/testdata/duplicate-identities`, `tests/integration/module-apply/testdata`, `tests/fixtures/valid/simple-module` and `tests/fixtures/valid/module-with-debug-values`.
  - `opmodel.dev/catalogs/opm@v4` SHALL move to the newest published catalog in its major, resolved against `templates/minimal`'s catalog.
  - `opmodel.dev/core@v2` SHALL move to the core version that the file's resulting catalog pins. A file without a catalog SHALL use `templates/minimal`'s resulting catalog.
- **Kind Platform.** The bare `version:` under `opmodel.dev/catalogs/opm@v4:` in `hack/kind-platform.yaml` SHALL be raised to `hack/platform`'s catalog after the move when it is lower. A version above it SHALL stay, with a warning, since no pin moves backwards.

How every move behaves:

- No pin SHALL move backwards. No pin SHALL move past an in-date `.cascade-hold` `max`. When a hold on core is below the core a newer catalog pins, the catalog SHALL stay too, with a warning.
- No pin SHALL move to a version that is tagged but not yet published.
- No pin listed for a file in `.cascade-frozen` SHALL change in that file. If `cue mod tidy` raises such a pin, the task SHALL fail, naming the file and the key.
- `cue mod get` SHALL name only the moved `opmodel.dev` keys, at exact versions. It and `cue mod tidy` SHALL run only in a module where a pin moved.
- Third-party pins, including `cue.dev/x/k8s.io@v0`, SHALL NOT be named. A third-party pin that `tidy` or `go mod tidy` raised SHALL be reported as a warning.
- The task SHALL use its own registry mapping, `testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`, for both `CUE_REGISTRY` and `OPM_REGISTRY`, whatever the environment sets.
- The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: Older pins move to the published set

- **WHEN** library, the operator module, the catalog and core are older than the newest published versions, and `task -x deps:cascade` runs
- **THEN** `go.mod`, `go.sum`, `internal/operator/pin.go`, every listed `cue.mod/module.cue` and `hack/kind-platform.yaml` carry the resolved versions
- **AND** core in each file equals the core that file's catalog pins
- **AND** the task exits 0

#### Scenario: Core already ahead of the catalog's pin stays

- **WHEN** a file pins a core newer than the core its catalog pins
- **THEN** that file's core is unchanged, and a warning names both versions

#### Scenario: A kind catalog ahead of hack/platform stays

- **WHEN** `hack/kind-platform.yaml` pins a catalog newer than `hack/platform`'s, and nothing else moves
- **THEN** the file is unchanged, the task exits 3, and a warning names both versions

#### Scenario: A hold on core holds the catalog

- **WHEN** `.cascade-hold` holds `opmodel.dev/core@v2` in date at the tree's core, and the newest catalog pins a newer core
- **THEN** no catalog or core pin changes, the task exits 3, and a warning says the catalog is held too

#### Scenario: An unpublished newest tag leaves the pin

- **WHEN** the newest catalog tag is not yet served by GHCR, and every older tag is not newer than the current pin
- **THEN** no catalog pin changes

#### Scenario: A frozen file is left byte-unchanged

- **WHEN** `.cascade-frozen` lists a moved test tree's `cue.mod/module.cue` for both `opmodel.dev` keys, and the other pins are older
- **THEN** that file is byte-unchanged, the other files move, and the task exits 0

#### Scenario: The third-party k8s pin is never named

- **WHEN** the catalog moves in `hack/platform`
- **THEN** `cue mod get` names only the `opmodel.dev` keys, and `cue.dev/x/k8s.io@v0` changes only if `cue mod tidy` raised it, in which case a warning names it

### Requirement: The cascade task resolves before it edits and reports by exit code

`task -x deps:cascade` SHALL refuse with exit 1 when the working tree has tracked or untracked changes, unless `CASCADE_ALLOW_DIRTY=1` is set. It SHALL validate `.cascade-frozen` and `.cascade-hold` through the resolver's `check-files` before anything else.

Order of work:

- Every resolver call that decides a target SHALL run before the first file edit.
- A failing target-deciding call, or any answer other than "move" or "stay", SHALL end the task with a code other than 0 and 3, and the tree SHALL be unchanged.
- A failure after the first edit (a tool, or a resolver call that decides no target, such as the docs-bundle check) SHALL end the task with a code other than 0 and 3. The tree may then be partly edited; callers discard it.

Exit codes:

- 0 when the working tree changed;
- 3 when it did not;
- any other code on error.

The task SHALL NOT turn a failure into a fallback version.

Source: workspace RELEASING.md, section "The cascade", "The receiver".

#### Scenario: Nothing to move

- **WHEN** every pin already equals its resolved target and `task -x deps:cascade` runs on a clean tree
- **THEN** it exits 3 and `git status --porcelain` is empty

#### Scenario: A resolver error leaves the tree clean

- **WHEN** the resolver fails on the first pin the task resolves
- **THEN** the task exits with a code other than 0 and 3, and `git status --porcelain` is empty

#### Scenario: A dirty tree is refused

- **WHEN** the tree holds an untracked file and `CASCADE_ALLOW_DIRTY` is not set
- **THEN** the task exits 1 and changes nothing

### Requirement: Template and fixture versions advance once per pull request

The three templates (`opmodel.dev/templates/<name>@v1`) and the podinfo fixture (`testing.opmodel.dev/modules/cli/podinfo@v0`) SHALL get a new version through `opm module version set` whenever their tree differs from the merge base of `CASCADE_BASE` (default `origin/main`), apart from the `Version:` line itself.

- The new version SHALL be one patch above the merge base's declared version when that version is published.
- It SHALL be the merge base's version when that version is not yet published.
- A second run in the same pull request SHALL NOT advance any version again.

- The task SHALL write a version only when it is greater than the file's. A file version above the target SHALL stay, with a warning.

The podinfo pin in `examples/cue.mod/module.cue` and `tests/e2e/testdata/operator-owned/cue.mod/module.cue` SHALL follow the fixture's new version in the same run, as a text rewrite made after their catalog and core moved. No `cue mod get` or `cue mod tidy` SHALL follow that rewrite. When a consumer's catalog or core moves while it pins a podinfo version that is not published, the task SHALL point it at the merge base's published podinfo for `cue mod get` and `tidy`, then rewrite it to the fixture's new version.

Source: workspace RELEASING.md, section "The cascade", "The receiver" ("Version advances happen once per PR").

#### Scenario: First run advances each changed module once

- **WHEN** the catalog moves in the templates and the podinfo fixture, and their declared versions are published
- **THEN** each of the four identity files carries the next patch version
- **AND** the podinfo pin in both consumers equals the fixture's new version

#### Scenario: Second run does not advance again

- **WHEN** the result of the first run is committed on the same branch and `task -x deps:cascade` runs again against the same base
- **THEN** it exits 3 and no identity file changes

#### Scenario: A consumer pinning the unpublished fixture still moves

- **WHEN** a first run advanced the podinfo fixture and its consumers, the result is committed, and a second run against the same base moves the consumers' catalog
- **THEN** the second run exits 0, the consumers carry the new catalog and core, and their podinfo pin is the fixture's new version

### Requirement: Title and body tasks describe the cascade diff

`task -x deps:cascade:title` SHALL print the shared resolver's one-line title for the diff against `CASCADE_BASE` (default `origin/main`). It SHALL use the path-class map `.tasks/cascade/classes`, which classes these paths as `test` and everything else as `shipped`:

- `hack/platform/`;
- `hack/kind-platform.yaml`;
- `examples/`;
- `tests/`;
- any `testdata/` directory;
- `*_test.go`.

`task -x deps:cascade:body` SHALL print the resolver's body for the same diff. `.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed:

- `github.com/open-platform-model/library` from `go.mod`;
- `github.com/open-platform-model/opm-operator` from `PinnedOperatorVersion` in `internal/operator/pin.go`;
- `opmodel.dev/modules/opm_operator@v0` from `PinnedModuleVersion` in `internal/operator/pin.go`, so a module release that deploys the same operator still shows as a moved pin;
- `opmodel.dev/catalogs/opm@v4` from `templates/minimal/cue.mod/module.cue`;
- `opmodel.dev/core@v2` from `templates/minimal/cue.mod/module.cue`.

Each pin SHALL be of class `shipped`, with no label.

Source: workspace RELEASING.md, section "The cascade", "Title from diff class".

#### Scenario: A shipped move titles as fix(deps)

- **WHEN** a run moved library, the operator module to a version deploying a newer operator, the catalog and core
- **THEN** `task -x deps:cascade:title` prints `fix(deps): bump 5 upstream pins`

#### Scenario: Test-only paths title as test(fixtures)

- **WHEN** the only changed paths are under `hack/platform/`, `examples/` and `tests/`
- **THEN** the title's type is `test(fixtures)`

#### Scenario: The body lists each moved pin

- **WHEN** a run moved four pins
- **THEN** `task -x deps:cascade:body` prints the title and labels markers, one table row per moved pin, and `## Notes` as the last section

#### Scenario: A module-only pin move is reported

- **WHEN** a run moved only the operator module pin, to a module version that deploys the same operator version
- **THEN** `task -x deps:cascade:title` prints a `fix(deps)` title counting one moved pin, and `task -x deps:cascade:body` prints one table row for `opmodel.dev/modules/opm_operator@v0` and none for `github.com/open-platform-model/opm-operator`

### Requirement: The cascade task is tested offline in required CI and fully in a network job

`task -x deps:cascade:test` SHALL run the cascade task against the stub resolver in sandbox copies of the tree, without touching the checkout.

- Its offline set, selected by `CASCADE_TEST_SET=offline`, SHALL run without registry or proxy access. It SHALL check:
  - the stub checksum;
  - that `pins.sh` reads the worktree and `HEAD` the same;
  - the no-op, error and dirty-tree scenarios.
- The full set SHALL add:
  - the older-pins scenario, including its second, idempotent run;
  - the frozen-file scenario;
  - the title and body scenario, when `CASCADE_RESOLVER_REAL` names the real resolver.

- The offline set SHALL also check that a core ahead of its catalog's pin stays with a warning, and that a hold on core holds the catalog.
- The full set SHALL also check a second catalog move on a branch whose consumers already pin the unpublished fixture.

The offline set SHALL run as a step of the `Lint` job in `.github/workflows/pr.yml`, the job workspace RELEASING.md "Rulesets on main" names as the cli's required check. The full set SHALL run in `.github/workflows/cascade-task.yml`, job `Cascade task (network)`, which is not a required check.

#### Scenario: Offline set runs on every pull request

- **WHEN** a pull request runs the `Lint` job
- **THEN** the job runs `task -x deps:cascade:test` with `CASCADE_TEST_SET=offline` and fails when any offline scenario fails

#### Scenario: The checkout is never modified by the test

- **WHEN** `task -x deps:cascade:test` finishes, pass or fail
- **THEN** `git status --porcelain` in the checkout is what it was before the run

### Requirement: The cascade task leaves non-cascade files alone and runs no dependency code

`task deps:cascade` SHALL NOT edit any of these:

- `.cascade-frozen` or `.cascade-hold`;
- `release-please-config.json`, `.release-please-manifest.json` or `CHANGELOG.md`;
- anything under `.github/`;
- `.opm-docs-version` or `docs-kit.cue`;
- any `language.version`;
- `hack/fixtures.sh` or `tests/fixtures/fixtures.go`.

It SHALL NOT publish, seed a real registry or push. It SHALL NOT build or run a program that links a moved Go dependency: the only Go programs it runs are the `opm` and the `hack/operator-pin` it builds, before any pin moves, from the merge base with `CASCADE_BASE` (default `origin/main`), never from the work tree, which in merge mode may already carry an earlier run's library move.

It SHALL append a warning to the cascade warnings file, without failing, in each of these cases:

- a CUE upstream that differs from the merge base declares a `language.version` newer than the `cue` version `.github/workflows/pr.yml` installs;
- a new major of a pinned upstream is available.

#### Scenario: Release and settings files stay untouched by a full run

- **WHEN** a run moves every pin
- **THEN** the changed paths include nothing under `.github/`, no release-please file, no `CHANGELOG.md`, no `.cascade-*` file and no `.opm-docs-version`

#### Scenario: A library move runs no library code

- **WHEN** a run moves library to a newer version
- **THEN** the task runs `go get` and `go mod tidy` for it and does not run `hack/docskit-dump` or any other program built from the moved tree

#### Scenario: Merge mode builds opm from the merge base

- **WHEN** a run on a `deps/cascade` branch that already pins a moved library, with `main` merged in, needs a version advance
- **THEN** the task builds `opm` from the merge base's tree, which pins `main`'s library, and runs `go` for nothing else but `get`, `mod tidy` and the builds of `opm` and `hack/operator-pin` from the merge base

#### Scenario: A newer language version is a warning

- **WHEN** a moved catalog declares a `language.version` newer than the `cue` that `pr.yml` installs
- **THEN** the pin still moves, and the warnings file names the module, its version and the two language versions
