## REMOVED Requirements

### Requirement: The cascade task moves the cli's upstream pins to published versions
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The cascade task moves pins in the working tree only", "The library pin moves to the newest version the Go proxy serves in its major", "The operator module pin moves only through hack/operator-pin", "The operator module target is the newest release whose operator version is not above the cli's own", "The catalog and core pins cover a fixed set of cue.mod/module.cue files", "The catalog pin moves to the newest published catalog in its major", "The core pin moves to the core that the file's resulting catalog pins", "The kind platform catalog is raised to hack/platform's catalog", "No pin moves backwards, past a hold, or holds a catalog back by a lower core hold", "No pin moves to a version that is tagged but not yet published", "A pin listed in .cascade-frozen does not change", "cue mod get names only the moved keys and runs only where a pin moved", "Third-party pins are not named and a raised one is reported as a warning", "The task uses its own registry mapping".

### Requirement: The cascade task resolves before it edits and reports by exit code
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "A dirty working tree is refused with exit 1", "The frozen and hold files are validated before anything else", "Every target-deciding resolver call runs before the first file edit", "A failure after the first edit ends the task with an error code", "Exit 0 means the working tree changed and exit 3 means it did not", "The cascade task does not turn a failure into a fallback version".

### Requirement: Template and fixture versions advance once per pull request
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "Templates and the podinfo fixture get a new version when their tree differs from the merge base", "The new version is one patch above the merge base's, or the merge base's when unpublished", "A second run in the same pull request advances no version again", "A version is written only when it is greater than the file version", "The podinfo pin in the two consumers follows the fixture version without cue mod get or tidy", "A consumer pinning an unpublished podinfo version is pointed at the merge base version for cue mod get and tidy".

### Requirement: Title and body tasks describe the cascade diff
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The title task prints the resolver title using the path-class map", "The body task prints the resolver body for the same diff", "The pins script reports the library and operator pins", "The pins script reports the catalog and core pins", "Each pin of the pins script is of class shipped with no label".

### Requirement: The cascade task is tested offline in required CI and fully in a network job
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The cascade test runs the task against the stub resolver in sandbox copies", "The offline set runs without registry or proxy access and checks fixed items", "The full set adds the older-pins, frozen-file and title and body scenarios", "The offline set also checks a core ahead of its catalog and a hold on core", "The full set also checks a second catalog move on a branch pinning the unpublished fixture", "The setup lowers the library to a version made from the tree and a failed setup fails its scenario", "The offline set runs as a step of the Lint job in pr.yml", "The full set runs in cascade-task.yml and is not a required check".

### Requirement: The cascade task leaves non-cascade files alone and runs no dependency code
**Reason**: Split into requirements that each state one behavior.
**Migration**: See "The cascade task edits none of the frozen, release, workflow, docs-version or fixture files", "The cascade task does not publish, seed a registry or push", "The cascade task runs no program that links a moved Go dependency", "The cascade task appends a warning without failing in two cases".

## ADDED Requirements

### Requirement: The cascade task moves pins in the working tree only

`task -x deps:cascade` SHALL move the following pins in the working tree only. It SHALL NOT commit, branch or push. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: Older pins move to the published set

- **WHEN** library, the operator module, the catalog and core are older than the newest published versions, and `task -x deps:cascade` runs
- **THEN** `go.mod`, `go.sum`, `internal/operator/pin.go`, every listed `cue.mod/module.cue` and `hack/kind-platform.yaml` carry the resolved versions
- **AND** core in each file equals the core that file's catalog pins
- **AND** the task exits 0

#### Scenario: The task edits the tree and nothing else
- **WHEN** `task -x deps:cascade` runs
- **THEN** it SHALL move the following pins in the working tree only and SHALL NOT commit, branch or push

### Requirement: The library pin moves to the newest version the Go proxy serves in its major

`task -x deps:cascade` SHALL move the following pins in the working tree only. **library.** `github.com/open-platform-model/library` in `go.mod` SHALL move to the newest version the Go proxy serves in its major, through `go get <module>@<exact version>` followed by `go mod tidy`. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: The library pin moves
- **WHEN** the Go proxy serves a newer library version in its major
- **THEN** `github.com/open-platform-model/library` in `go.mod` SHALL move to it through `go get <module>@<exact version>` followed by `go mod tidy`

### Requirement: The operator module pin moves only through hack/operator-pin

`task -x deps:cascade` SHALL move the following pins in the working tree only. **opm-operator module.** The operator module pin in `internal/operator/pin.go` (`PinnedModuleVersion` and `PinnedOperatorVersion`) SHALL move only through `hack/operator-pin <v>`, the program behind `task operator:pin VERSION=<v>`, built from the merge base like `opm`. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: The operator pin moves through the program
- **WHEN** the operator module pin moves
- **THEN** it SHALL move only through `hack/operator-pin <v>`, the program behind `task operator:pin VERSION=<v>`, built from the merge base like `opm`

### Requirement: The operator module target is the newest release whose operator version is not above the cli's own

`task -x deps:cascade` SHALL move the following pins in the working tree only. **opm-operator module.** The target is the newest published release of `opmodel.dev/modules/opm_operator` in the pinned major whose stated operator version, read from the module's source without a render, has a `MAJOR.MINOR` not above the cli's own, the same rule install applies to a target module version. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: The target stays within the cli MAJOR.MINOR
- **WHEN** the operator module pin moves
- **THEN** the target is the newest published release of `opmodel.dev/modules/opm_operator` in the pinned major whose stated operator version has a `MAJOR.MINOR` not above the cli's own

### Requirement: The catalog and core pins cover a fixed set of cue.mod/module.cue files

**Catalog and core as a consistent set.** The consistent set SHALL cover the `cue.mod/module.cue` files of the three templates, `hack/platform`, `examples`, `tests/fixtures/modules/podinfo`, `tests/e2e/testdata/operator-owned`, `internal/instinit/testdata/initvalues`, `internal/workflow/render/testdata/skip-unprovided`, `tests/e2e/testdata/duplicate-identities`, `tests/integration/module-apply/testdata`, `tests/fixtures/valid/simple-module` and `tests/fixtures/valid/module-with-debug-values`.

#### Scenario: The set of files
- **WHEN** the task moves the catalog and core as a consistent set
- **THEN** it SHALL cover the `cue.mod/module.cue` files of the three templates, `hack/platform`, `examples`, `tests/fixtures/modules/podinfo` and the other listed test trees

### Requirement: The catalog pin moves to the newest published catalog in its major

`task -x deps:cascade` SHALL move the following pins in the working tree only. **Catalog and core as a consistent set.** `opmodel.dev/catalogs/opm@v4` SHALL move to the newest published catalog in its major, resolved against `templates/minimal`'s catalog. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: The catalog pin moves
- **WHEN** a newer catalog is published in its major
- **THEN** `opmodel.dev/catalogs/opm@v4` SHALL move to it, resolved against `templates/minimal`'s catalog

### Requirement: The core pin moves to the core that the file's resulting catalog pins

`task -x deps:cascade` SHALL move the following pins in the working tree only. **Catalog and core as a consistent set.** `opmodel.dev/core@v2` SHALL move to the core version that the file's resulting catalog pins. A file without a catalog SHALL use `templates/minimal`'s resulting catalog. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: Core already ahead of the catalog's pin stays

- **WHEN** a file pins a core newer than the core its catalog pins
- **THEN** that file's core is unchanged, and a warning names both versions

### Requirement: The kind platform catalog is raised to hack/platform's catalog

`task -x deps:cascade` SHALL move the following pins in the working tree only. **Kind Platform.** The bare `version:` under `opmodel.dev/catalogs/opm@v4:` in `hack/kind-platform.yaml` SHALL be raised to `hack/platform`'s catalog after the move when it is lower. A version above it SHALL stay, with a warning, since no pin moves backwards. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: A kind catalog ahead of hack/platform stays

- **WHEN** `hack/kind-platform.yaml` pins a catalog newer than `hack/platform`'s, and nothing else moves
- **THEN** the file is unchanged, the task exits 3, and a warning names both versions

### Requirement: No pin moves backwards, past a hold, or holds a catalog back by a lower core hold

How every move behaves: No pin SHALL move backwards. No pin SHALL move past an in-date `.cascade-hold` `max`. When a hold on core is below the core a newer catalog pins, the catalog SHALL stay too, with a warning. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: A hold on core holds the catalog

- **WHEN** `.cascade-hold` holds `opmodel.dev/core@v2` in date at the tree's core, and the newest catalog pins a newer core
- **THEN** no catalog or core pin changes, the task exits 3, and a warning says the catalog is held too

### Requirement: No pin moves to a version that is tagged but not yet published

No pin SHALL move to a version that is tagged but not yet published. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: An unpublished newest tag leaves the pin

- **WHEN** the newest catalog tag is not yet served by GHCR, and every older tag is not newer than the current pin
- **THEN** no catalog pin changes

### Requirement: A pin listed in .cascade-frozen does not change

No pin listed for a file in `.cascade-frozen` SHALL change in that file. If `cue mod tidy` raises such a pin, the task SHALL fail, naming the file and the key. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: A frozen file is left byte-unchanged

- **WHEN** `.cascade-frozen` lists a moved test tree's `cue.mod/module.cue` for both `opmodel.dev` keys, and the other pins are older
- **THEN** that file is byte-unchanged, the other files move, and the task exits 0

### Requirement: cue mod get names only the moved keys and runs only where a pin moved

`cue mod get` SHALL name only the moved `opmodel.dev` keys, at exact versions. It and `cue mod tidy` SHALL run only in a module where a pin moved. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: cue mod get and tidy run only where a pin moved
- **WHEN** a pin moved in one module
- **THEN** `cue mod get` SHALL name only the moved `opmodel.dev` keys, at exact versions, and it and `cue mod tidy` SHALL run only in a module where a pin moved

### Requirement: Third-party pins are not named and a raised one is reported as a warning

Third-party pins, including `cue.dev/x/k8s.io@v0`, SHALL NOT be named. A third-party pin that `tidy` or `go mod tidy` raised SHALL be reported as a warning. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: The third-party k8s pin is never named

- **WHEN** the catalog moves in `hack/platform`
- **THEN** `cue mod get` names only the `opmodel.dev` keys, and `cue.dev/x/k8s.io@v0` changes only if `cue mod tidy` raised it, in which case a warning names it

### Requirement: The task uses its own registry mapping

The task SHALL use its own registry mapping, `testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`, for both `CUE_REGISTRY` and `OPM_REGISTRY`, whatever the environment sets. The source is workspace RELEASING.md, sections "The cascade" and "What each repo's task moves".

#### Scenario: The mapping holds whatever the environment sets
- **WHEN** the environment sets `CUE_REGISTRY` or `OPM_REGISTRY`
- **THEN** the task SHALL use its own registry mapping, `testing.opmodel.dev=ghcr.io/open-platform-model,opmodel.dev=ghcr.io/open-platform-model,registry.cue.works`, for both

### Requirement: A dirty working tree is refused with exit 1

`task -x deps:cascade` SHALL refuse with exit 1 when the working tree has tracked or untracked changes, unless `CASCADE_ALLOW_DIRTY=1` is set. Source: workspace RELEASING.md, section "The cascade", "The receiver".

#### Scenario: A dirty tree is refused

- **WHEN** the tree holds an untracked file and `CASCADE_ALLOW_DIRTY` is not set
- **THEN** the task exits 1 and changes nothing

### Requirement: The frozen and hold files are validated before anything else

`task -x deps:cascade` SHALL validate `.cascade-frozen` and `.cascade-hold` through the resolver's `check-files` before anything else. Source: workspace RELEASING.md, section "The cascade", "The receiver".

#### Scenario: The files are validated first
- **WHEN** `task -x deps:cascade` runs
- **THEN** it SHALL validate `.cascade-frozen` and `.cascade-hold` through the resolver's `check-files` before anything else

### Requirement: Every target-deciding resolver call runs before the first file edit

Order of work: Every resolver call that decides a target SHALL run before the first file edit. A failing target-deciding call, or any answer other than "move" or "stay", SHALL end the task with a code other than 0 and 3, and the tree SHALL be unchanged. Source: workspace RELEASING.md, section "The cascade", "The receiver".

#### Scenario: A resolver error leaves the tree clean

- **WHEN** the resolver fails on the first pin the task resolves
- **THEN** the task exits with a code other than 0 and 3, and `git status --porcelain` is empty

### Requirement: A failure after the first edit ends the task with an error code

A failure after the first edit (a tool, or a resolver call that decides no target, such as the docs-bundle check) SHALL end the task with a code other than 0 and 3. The tree may then be partly edited; callers discard it. Source: workspace RELEASING.md, section "The cascade", "The receiver".

#### Scenario: A late failure may leave a partly edited tree
- **WHEN** a tool fails after the first edit
- **THEN** the task SHALL end with a code other than 0 and 3, and the tree may then be partly edited; callers discard it

### Requirement: Exit 0 means the working tree changed and exit 3 means it did not

Exit codes SHALL be: 0 when the working tree changed; 3 when it did not; any other code on error. Source: workspace RELEASING.md, section "The cascade", "The receiver".

#### Scenario: Nothing to move

- **WHEN** every pin already equals its resolved target and `task -x deps:cascade` runs on a clean tree
- **THEN** it exits 3 and `git status --porcelain` is empty

### Requirement: The cascade task does not turn a failure into a fallback version

The task SHALL NOT turn a failure into a fallback version. Source: workspace RELEASING.md, section "The cascade", "The receiver".

#### Scenario: No fallback version
- **WHEN** the task meets a failure
- **THEN** it SHALL NOT turn it into a fallback version

### Requirement: Templates and the podinfo fixture get a new version when their tree differs from the merge base

The three templates (`opmodel.dev/templates/<name>@v1`) and the podinfo fixture (`testing.opmodel.dev/modules/cli/podinfo@v0`) SHALL get a new version through `opm module version set` whenever their tree differs from the merge base of `CASCADE_BASE` (default `origin/main`), apart from the `Version:` line itself. Source: workspace RELEASING.md, section "The cascade", "The receiver" ("Version advances happen once per PR").

#### Scenario: First run advances each changed module once

- **WHEN** the catalog moves in the templates and the podinfo fixture, and their declared versions are published
- **THEN** each of the four identity files carries the next patch version
- **AND** the podinfo pin in both consumers equals the fixture's new version

### Requirement: The new version is one patch above the merge base's, or the merge base's when unpublished

The new version SHALL be one patch above the merge base's declared version when that version is published. It SHALL be the merge base's version when that version is not yet published. Source: workspace RELEASING.md, section "The cascade", "The receiver" ("Version advances happen once per PR").

#### Scenario: The target version
- **WHEN** a template tree differs from the merge base
- **THEN** the new version SHALL be one patch above the merge base's declared version when that version is published, and the merge base's version when it is not yet published

### Requirement: A second run in the same pull request advances no version again

A second run in the same pull request SHALL NOT advance any version again. Source: workspace RELEASING.md, section "The cascade", "The receiver" ("Version advances happen once per PR").

#### Scenario: Second run does not advance again

- **WHEN** the result of the first run is committed on the same branch and `task -x deps:cascade` runs again against the same base
- **THEN** it exits 3 and no identity file changes

### Requirement: A version is written only when it is greater than the file version

The task SHALL write a version only when it is greater than the file's. A file version above the target SHALL stay, with a warning. Source: workspace RELEASING.md, section "The cascade", "The receiver" ("Version advances happen once per PR").

#### Scenario: A higher file version stays
- **WHEN** a file version is above the target
- **THEN** it SHALL stay, with a warning

### Requirement: The podinfo pin in the two consumers follows the fixture version without cue mod get or tidy

The podinfo pin in `examples/cue.mod/module.cue` and `tests/e2e/testdata/operator-owned/cue.mod/module.cue` SHALL follow the fixture's new version in the same run, as a text rewrite made after their catalog and core moved. No `cue mod get` or `cue mod tidy` SHALL follow that rewrite. Source: workspace RELEASING.md, section "The cascade", "The receiver" ("Version advances happen once per PR").

#### Scenario: The consumers follow the fixture
- **WHEN** the fixture gets a new version
- **THEN** the podinfo pin in `examples/cue.mod/module.cue` and `tests/e2e/testdata/operator-owned/cue.mod/module.cue` SHALL follow it in the same run, as a text rewrite made after their catalog and core moved

### Requirement: A consumer pinning an unpublished podinfo version is pointed at the merge base version for cue mod get and tidy

When a consumer's catalog or core moves while it pins a podinfo version that is not published, the task SHALL point it at the merge base's published podinfo for `cue mod get` and `tidy`, then rewrite it to the fixture's new version. Source: workspace RELEASING.md, section "The cascade", "The receiver" ("Version advances happen once per PR").

#### Scenario: A consumer pinning the unpublished fixture still moves

- **WHEN** a first run advanced the podinfo fixture and its consumers, the result is committed, and a second run against the same base moves the consumers' catalog
- **THEN** the second run exits 0, the consumers carry the new catalog and core, and their podinfo pin is the fixture's new version

### Requirement: The title task prints the resolver title using the path-class map

`task -x deps:cascade:title` SHALL print the shared resolver's one-line title for the diff against `CASCADE_BASE` (default `origin/main`). It SHALL use the path-class map `.tasks/cascade/classes`, which classes these paths as `test` and everything else as `shipped`: `hack/platform/`; `hack/kind-platform.yaml`; `examples/`; `tests/`; any `testdata/` directory; `*_test.go`. Source: workspace RELEASING.md, section "The cascade", "Title from diff class".

#### Scenario: A shipped move titles as fix(deps)

- **WHEN** a run moved library, the operator module to a version deploying a newer operator, the catalog and core
- **THEN** `task -x deps:cascade:title` prints `fix(deps): bump 5 upstream pins`

#### Scenario: Test-only paths title as test(fixtures)

- **WHEN** the only changed paths are under `hack/platform/`, `examples/` and `tests/`
- **THEN** the title's type is `test(fixtures)`

### Requirement: The body task prints the resolver body for the same diff

`task -x deps:cascade:title` SHALL print the shared resolver's one-line title for the diff against `CASCADE_BASE` (default `origin/main`). `task -x deps:cascade:body` SHALL print the resolver's body for the same diff. Source: workspace RELEASING.md, section "The cascade", "Title from diff class".

#### Scenario: The body lists each moved pin

- **WHEN** a run moved four pins
- **THEN** `task -x deps:cascade:body` prints the title and labels markers, one table row per moved pin, and `## Notes` as the last section

### Requirement: The pins script reports the library and operator pins

`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed: `github.com/open-platform-model/library` from `go.mod`; `github.com/open-platform-model/opm-operator` from `PinnedOperatorVersion` in `internal/operator/pin.go`; `opmodel.dev/modules/opm_operator@v0` from `PinnedModuleVersion` in `internal/operator/pin.go`, so a module release that deploys the same operator still shows as a moved pin. Source: workspace RELEASING.md, section "The cascade", "Title from diff class".

#### Scenario: A module-only pin move is reported

- **WHEN** a run moved only the operator module pin, to a module version that deploys the same operator version
- **THEN** `task -x deps:cascade:title` prints a `fix(deps)` title counting one moved pin, and `task -x deps:cascade:body` prints one table row for `opmodel.dev/modules/opm_operator@v0` and none for `github.com/open-platform-model/opm-operator`

### Requirement: The pins script reports the catalog and core pins

`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed: `opmodel.dev/catalogs/opm@v4` from `templates/minimal/cue.mod/module.cue`; `opmodel.dev/core@v2` from `templates/minimal/cue.mod/module.cue`. Source: workspace RELEASING.md, section "The cascade", "Title from diff class".

#### Scenario: Catalog and core are reported
- **WHEN** `.tasks/cascade/pins.sh <ref>` runs
- **THEN** it SHALL report `opmodel.dev/catalogs/opm@v4` and `opmodel.dev/core@v2`, each `v`-prefixed, from `templates/minimal/cue.mod/module.cue`

### Requirement: Each pin of the pins script is of class shipped with no label

`.tasks/cascade/pins.sh <ref>` SHALL report five pins, each `v`-prefixed. Each pin SHALL be of class `shipped`, with no label. Source: workspace RELEASING.md, section "The cascade", "Title from diff class".

#### Scenario: Class of a reported pin
- **WHEN** `.tasks/cascade/pins.sh <ref>` reports a pin
- **THEN** each pin SHALL be of class `shipped`, with no label

### Requirement: The cascade test runs the task against the stub resolver in sandbox copies

`task -x deps:cascade:test` SHALL run the cascade task against the stub resolver in sandbox copies of the tree, without touching the checkout.

#### Scenario: The checkout is never modified by the test

- **WHEN** `task -x deps:cascade:test` finishes, pass or fail
- **THEN** `git status --porcelain` in the checkout is what it was before the run

### Requirement: The offline set runs without registry or proxy access and checks fixed items

The offline set of `task -x deps:cascade:test`, selected by `CASCADE_TEST_SET=offline`, SHALL run without registry or proxy access. It SHALL check: the stub checksum; that `pins.sh` reads the worktree and `HEAD` the same; that every `older` row of `testdata/older.tsv` is older than the tree's pin, that the `oldest` catalog and core rows are older than their `older` rows, and that the `oldest` podinfo row is older than the tree's fixture version; the no-op, error and dirty-tree scenarios.

#### Scenario: A stale older row fails the offline set

- **WHEN** an `older` row of `testdata/older.tsv` is not older than the tree's pin, and `task -x deps:cascade:test` runs with `CASCADE_TEST_SET=offline`
- **THEN** the test prints a FAIL line that names the row and exits non-zero

### Requirement: The full set adds the older-pins, frozen-file and title and body scenarios

The full set of `task -x deps:cascade:test` SHALL add: the older-pins scenario, including its second, idempotent run; the frozen-file scenario; the title and body scenario, when `CASCADE_RESOLVER_REAL` names the real resolver.

#### Scenario: The full set adds scenarios
- **WHEN** `task -x deps:cascade:test` runs the full set
- **THEN** it SHALL add the older-pins scenario, including its second, idempotent run, the frozen-file scenario, and the title and body scenario when `CASCADE_RESOLVER_REAL` names the real resolver

### Requirement: The offline set also checks a core ahead of its catalog and a hold on core

The offline set SHALL also check that a core ahead of its catalog's pin stays with a warning, and that a hold on core holds the catalog.

#### Scenario: Offline checks on core
- **WHEN** the offline set runs
- **THEN** it SHALL also check that a core ahead of its catalog's pin stays with a warning, and that a hold on core holds the catalog

### Requirement: The full set also checks a second catalog move on a branch pinning the unpublished fixture

The full set SHALL also check a second catalog move on a branch whose consumers already pin the unpublished fixture.

#### Scenario: Full check on a second move
- **WHEN** the full set runs
- **THEN** it SHALL also check a second catalog move on a branch whose consumers already pin the unpublished fixture

### Requirement: The setup lowers the library to a version made from the tree and a failed setup fails its scenario

The setup of the older-pins, frozen-file and second-move scenarios SHALL lower the library to a version made from the tree's own library source under a lower version name, never to a published older release, so that the sandbox compiles whatever library API the cli uses. A setup that fails SHALL fail its scenario with the failed step and the reason in the FAIL line.

#### Scenario: The full set passes when the cli uses the newest library's API

- **WHEN** the cli imports a library package that no published library below the tree's pin holds, and `task -x deps:cascade:test` runs with `CASCADE_TEST_SET=all`
- **THEN** the older-pins, frozen-file and second-move scenarios pass

### Requirement: The offline set runs as a step of the Lint job in pr.yml

The offline set SHALL run as a step of the `Lint` job in `.github/workflows/pr.yml`, the job workspace RELEASING.md "Rulesets on main" names as the cli's required check.

#### Scenario: Offline set runs on every pull request

- **WHEN** a pull request runs the `Lint` job
- **THEN** the job runs `task -x deps:cascade:test` with `CASCADE_TEST_SET=offline` and fails when any offline scenario fails

### Requirement: The full set runs in cascade-task.yml and is not a required check

The full set SHALL run in `.github/workflows/cascade-task.yml`, job `Cascade task (network)`, which is not a required check.

#### Scenario: The network job runs the full set
- **WHEN** the workflow `cascade-task.yml` runs
- **THEN** the full set SHALL run in job `Cascade task (network)`, which is not a required check

### Requirement: The cascade task edits none of the frozen, release, workflow, docs-version or fixture files

`task deps:cascade` SHALL NOT edit any of these: `.cascade-frozen` or `.cascade-hold`; `release-please-config.json`, `.release-please-manifest.json` or `CHANGELOG.md`; anything under `.github/`; `.opm-docs-version` or `docs-kit.cue`; any `language.version`; `hack/fixtures.sh` or `tests/fixtures/fixtures.go`.

#### Scenario: Release and settings files stay untouched by a full run

- **WHEN** a run moves every pin
- **THEN** the changed paths include nothing under `.github/`, no release-please file, no `CHANGELOG.md`, no `.cascade-*` file and no `.opm-docs-version`

### Requirement: The cascade task does not publish, seed a registry or push

`task deps:cascade` SHALL NOT publish, seed a real registry or push.

#### Scenario: No publish or push
- **WHEN** the task runs
- **THEN** it SHALL NOT publish, seed a real registry or push

### Requirement: The cascade task runs no program that links a moved Go dependency

`task deps:cascade` SHALL NOT build or run a program that links a moved Go dependency: the only Go programs it runs are the `opm` and the `hack/operator-pin` it builds, before any pin moves, from the merge base with `CASCADE_BASE` (default `origin/main`), never from the work tree, which in merge mode may already carry an earlier run's library move.

#### Scenario: A library move runs no library code

- **WHEN** a run moves library to a newer version
- **THEN** the task runs `go get` and `go mod tidy` for it and does not run `hack/docskit-dump` or any other program built from the moved tree

#### Scenario: Merge mode builds opm from the merge base

- **WHEN** a run on a `deps/cascade` branch that already pins a moved library, with `main` merged in, needs a version advance
- **THEN** the task builds `opm` from the merge base's tree, which pins `main`'s library, and runs `go` for nothing else but `get`, `mod tidy` and the builds of `opm` and `hack/operator-pin` from the merge base

### Requirement: The cascade task appends a warning without failing in two cases

`task deps:cascade` SHALL append a warning to the cascade warnings file, without failing, in each of these cases: a CUE upstream that differs from the merge base declares a `language.version` newer than the `cue` version `.github/workflows/pr.yml` installs; a new major of a pinned upstream is available.

#### Scenario: A newer language version is a warning

- **WHEN** a moved catalog declares a `language.version` newer than the `cue` that `pr.yml` installs
- **THEN** the pin still moves, and the warnings file names the module, its version and the two language versions
