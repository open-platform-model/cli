## MODIFIED Requirements

### Requirement: The cascade task is tested offline in required CI and fully in a network job

`task -x deps:cascade:test` SHALL run the cascade task against the stub resolver in sandbox copies of the tree, without touching the checkout.

- Its offline set, selected by `CASCADE_TEST_SET=offline`, SHALL run without registry or proxy access. It SHALL check:
  - the stub checksum;
  - that `pins.sh` reads the worktree and `HEAD` the same;
  - that every `older` row of `testdata/older.tsv` is older than the tree's pin, and every `oldest` row older than its `older` row;
  - the no-op, error and dirty-tree scenarios.
- The full set SHALL add:
  - the older-pins scenario, including its second, idempotent run;
  - the frozen-file scenario;
  - the title and body scenario, when `CASCADE_RESOLVER_REAL` names the real resolver.

- The offline set SHALL also check that a core ahead of its catalog's pin stays with a warning, and that a hold on core holds the catalog.
- The full set SHALL also check a second catalog move on a branch whose consumers already pin the unpublished fixture.

The setup of the older-pins, frozen-file and second-move scenarios SHALL lower the library to a version made from the tree's own library source under a lower version name, never to a published older release, so that the sandbox compiles whatever library API the cli uses. A setup that fails SHALL fail its scenario with the failed step and the reason in the FAIL line.

The offline set SHALL run as a step of the `Lint` job in `.github/workflows/pr.yml`, the job workspace RELEASING.md "Rulesets on main" names as the cli's required check. The full set SHALL run in `.github/workflows/cascade-task.yml`, job `Cascade task (network)`, which is not a required check.

#### Scenario: Offline set runs on every pull request

- **WHEN** a pull request runs the `Lint` job
- **THEN** the job runs `task -x deps:cascade:test` with `CASCADE_TEST_SET=offline` and fails when any offline scenario fails

#### Scenario: The checkout is never modified by the test

- **WHEN** `task -x deps:cascade:test` finishes, pass or fail
- **THEN** `git status --porcelain` in the checkout is what it was before the run

#### Scenario: The full set passes when the cli uses the newest library's API

- **WHEN** the cli imports a library package that no published library below the tree's pin holds, and `task -x deps:cascade:test` runs with `CASCADE_TEST_SET=all`
- **THEN** the older-pins, frozen-file and second-move scenarios pass

#### Scenario: A stale older row fails the offline set

- **WHEN** an `older` row of `testdata/older.tsv` is not older than the tree's pin, and `task -x deps:cascade:test` runs with `CASCADE_TEST_SET=offline`
- **THEN** the test prints a FAIL line that names the row and exits non-zero
