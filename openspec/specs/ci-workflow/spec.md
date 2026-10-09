# Capability: ci-workflow

## Purpose

The push-triggered GitHub Actions workflow (`.github/workflows/ci.yml`) runs golangci-lint and `go test ./internal/...` as two independent jobs on every branch push, so a broken commit is caught before it reaches a pull request. This capability also covers `.github/workflows/publish-fixtures.yml`, which publishes the repository's test-fixture modules to GHCR through `opm module publish` when a fixture changes on main.

## Requirements

### Requirement: Lint runs on every push
The CI workflow SHALL run `golangci-lint` (the version the file `.golangci-lint-version` names, with `govet` enabled as one of its linters in `.golangci.yml`, so no separate `go vet` step exists) on every push to any branch.

#### Scenario: Lint passes
- **WHEN** a push is made to any branch
- **THEN** the lint job runs and exits zero if no lint errors are found

#### Scenario: Lint fails
- **WHEN** a push introduces a lint violation
- **THEN** the lint job exits non-zero and the commit is marked failed

### Requirement: Unit tests run on every push
The CI workflow SHALL run `go test ./internal/...` on Go 1.26.0 on every push to any branch.

#### Scenario: Unit tests pass
- **WHEN** a push is made to any branch
- **THEN** the unit test job runs and exits zero if all tests pass

#### Scenario: Unit tests fail
- **WHEN** a push introduces a failing unit test
- **THEN** the unit job exits non-zero and the commit is marked failed

### Requirement: Lint and unit run in parallel
The CI workflow SHALL run the lint and unit jobs concurrently with no dependency between them, and SHALL cancel an in-progress run for the same ref when a newer push arrives (`concurrency` keyed on workflow and ref with `cancel-in-progress`).

#### Scenario: Parallel execution
- **WHEN** the CI workflow triggers
- **THEN** both jobs start simultaneously without waiting for the other

#### Scenario: Superseded run cancelled
- **WHEN** a second push to the same branch arrives while the workflow is running
- **THEN** the earlier run is cancelled and only the newest push is checked

### Requirement: Workflow targets GitHub-hosted runner
The CI workflow SHALL specify `runs-on: ubuntu-latest` for all jobs.

#### Scenario: GitHub-hosted runner assignment
- **WHEN** the workflow triggers
- **THEN** all jobs are assigned to the `ubuntu-latest` runner pool

### Requirement: Workflow is active immediately
The CI workflow SHALL use `push` to any branch and `workflow_dispatch` as active triggers.

#### Scenario: Manual trigger works
- **WHEN** a user manually dispatches the workflow from the GitHub UI
- **THEN** the workflow runs lint and unit jobs

#### Scenario: Push triggers workflow
- **WHEN** a commit is pushed
- **THEN** the workflow runs automatically

### Requirement: Fixture modules are published to GHCR by CI

A workflow (`.github/workflows/publish-fixtures.yml`) SHALL publish the repository's fixture modules to GHCR through `opm module publish`, using the binary built from the commit under test so a fixture that violates any publish gate fails CI. It SHALL trigger on pushes to `main` that touch `tests/fixtures/modules/**`, `hack/fixtures.sh` or the workflow file itself, and SHALL additionally offer `workflow_dispatch` so a new coordinate can be published from a branch before the consumers that pin it merge.

The workflow SHALL request `packages: write` at the job level only, and SHALL skip cleanly on forks, which cannot obtain that permission.

Idempotency SHALL live in the caller: `hack/fixtures.sh publish` SHALL attempt every fixture and treat a publish whose only refusal is that GHCR already holds the tag as "nothing to do", because publish itself always refuses an already-published tag and never skips.

#### Scenario: Unchanged fixture republished

- **WHEN** the workflow runs and every fixture's declared version is already on GHCR
- **THEN** each SHALL be reported as already present by the caller-side filter and the workflow SHALL succeed

#### Scenario: Bumped fixture publishes

- **WHEN** a fixture's identity version is bumped and the workflow runs
- **THEN** that fixture SHALL be published at its new tag

#### Scenario: Fork run

- **WHEN** the workflow triggers on a fork
- **THEN** the publish job SHALL be skipped rather than fail on a missing token

### Requirement: The linter configuration is verified without the network
Every `Lint` job that runs golangci-lint (in `.github/workflows/ci.yml` and `.github/workflows/pr.yml`) SHALL verify `.golangci.yml` against the golangci-lint JSON schema committed in the repository, and SHALL NOT download a schema at run time. The golangci-lint action's own configuration check, which downloads the schema, SHALL be off. The verification SHALL give the same result when no network is reachable, and `task lint` SHALL run the same verification before the linters.

#### Scenario: A valid configuration passes with the linter's website unreachable
- **WHEN** the configuration check runs for a valid `.golangci.yml` and `golangci-lint.run` cannot be reached
- **THEN** the check exits zero

#### Scenario: An invalid configuration fails the job
- **WHEN** `.golangci.yml` holds a key the schema does not allow
- **THEN** the configuration check exits non-zero and prints the schema violation

#### Scenario: The action's own check is refused
- **WHEN** a workflow step uses the golangci-lint action without turning its configuration check off
- **THEN** the configuration check exits non-zero and names the workflow

### Requirement: One file names the linter version
The file `.golangci-lint-version` SHALL be the only place that names the golangci-lint version CI installs; `go.mod`, which the action reads first, SHALL NOT name one. Every use of the golangci-lint action, in any letter case and quoted or not, SHALL read it and SHALL NOT name a version of its own, every use SHALL pin the same action commit, and each workflow that uses the action SHALL run the configuration check. The committed schema SHALL be the one for that version's minor line, SHALL be the only schema file in its directory, and SHALL match its recorded SHA-256 checksum. The configuration check SHALL fail when any of these does not hold, and when the installed golangci-lint is of another minor line than the file names.

#### Scenario: A workflow names its own version
- **WHEN** a step that uses the golangci-lint action sets a `version` input, or does not read `.golangci-lint-version`
- **THEN** the configuration check exits non-zero and names the workflow

#### Scenario: The version moves to a minor line without a committed schema
- **WHEN** `.golangci-lint-version` names a version whose minor line has no committed schema file
- **THEN** the configuration check exits non-zero and names the missing file

#### Scenario: The committed schema was edited
- **WHEN** the bytes of the committed schema do not match its recorded checksum
- **THEN** the configuration check exits non-zero

#### Scenario: The two workflows pin different action commits
- **WHEN** `pr.yml` and `ci.yml` use the golangci-lint action at different commit SHAs
- **THEN** the configuration check exits non-zero and prints both

#### Scenario: The installed linter is of another minor line
- **WHEN** the golangci-lint on `PATH` reports a minor line other than the one `.golangci-lint-version` names
- **THEN** the configuration check exits non-zero and prints both versions

#### Scenario: go.mod names a linter version
- **WHEN** `go.mod` holds a golangci-lint module line
- **THEN** the configuration check exits non-zero, because the action would take the version from there
