# Capability: ci-workflow

## Purpose

The push-triggered GitHub Actions workflow (`.github/workflows/ci.yml`) runs golangci-lint and `go test ./internal/...` as two independent jobs on every branch push, so a broken commit is caught before it reaches a pull request. This capability also covers `.github/workflows/publish-fixtures.yml`, which publishes the repository's test-fixture modules to GHCR through `opm module publish` when a fixture changes on main.

## Requirements

### Requirement: Lint runs on every push
The CI workflow SHALL run `golangci-lint` (v2.11.3, with `govet` enabled as one of its linters in `.golangci.yml`, so no separate `go vet` step exists) on every push to any branch.

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
