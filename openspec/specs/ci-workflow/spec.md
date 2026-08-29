## ADDED Requirements

### Requirement: Lint runs on every push
The CI workflow SHALL run `golangci-lint` and `go vet` on every push to any branch.

#### Scenario: Lint passes
- **WHEN** a push is made to any branch
- **THEN** the lint job runs and exits zero if no lint errors are found

#### Scenario: Lint fails
- **WHEN** a push introduces a lint violation
- **THEN** the lint job exits non-zero and the commit is marked failed

### Requirement: Unit tests run on every push
The CI workflow SHALL run `go test ./internal/...` on every push to any branch.

#### Scenario: Unit tests pass
- **WHEN** a push is made to any branch
- **THEN** the unit test job runs and exits zero if all tests pass

#### Scenario: Unit tests fail
- **WHEN** a push introduces a failing unit test
- **THEN** the unit job exits non-zero and the commit is marked failed

### Requirement: Lint and unit run in parallel
The CI workflow SHALL run the lint and unit jobs concurrently with no dependency between them.

#### Scenario: Parallel execution
- **WHEN** the CI workflow triggers
- **THEN** both jobs start simultaneously without waiting for the other

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

A workflow SHALL publish the repository's fixture modules to GHCR through `opm module publish`, using the binary built from the commit under test so a fixture that violates any publish gate fails CI. It SHALL trigger on pushes to `main` that touch the fixture trees or the publish script, and SHALL additionally offer `workflow_dispatch` so a new coordinate can be published from a branch before the consumers that pin it merge.

The workflow SHALL request `packages: write` at the job level only, and SHALL skip cleanly on forks, which cannot obtain that permission.

Idempotency SHALL live in the caller: the script SHALL probe GHCR for the resolved tag and skip fixtures already published, because publish itself always refuses an already-published tag and never skips.

#### Scenario: Unchanged fixture republished

- **WHEN** the workflow runs and every fixture's declared version is already on GHCR
- **THEN** each SHALL be reported as skipped by the caller-side filter and the workflow SHALL succeed

#### Scenario: Bumped fixture publishes

- **WHEN** a fixture's identity version is bumped and the workflow runs
- **THEN** that fixture SHALL be published at its new tag

#### Scenario: Fork run

- **WHEN** the workflow triggers on a fork
- **THEN** the publish job SHALL be skipped rather than fail on a missing token

### Requirement: Fixture publish gates run on every pull request

The PR workflow SHALL run the fixture publish gates in dry-run mode, without pushing, so a fixture that would fail the publish workflow fails the pull request first. A dry run whose only refusal is already-published SHALL pass, since a PR's committed version may legitimately be live; any other refusal SHALL fail the job.

The PR workflow's registry mapping SHALL route `testing.opmodel.dev` to GHCR, as the fixtures the tests resolve live there.

#### Scenario: Fixture with a gate violation

- **WHEN** a PR makes a fixture's `metadata.modulePath` disagree with its identity package
- **THEN** the fixture-gates job SHALL fail, naming the disagreement

#### Scenario: Fixture at an already-published version

- **WHEN** a PR touches a fixture without bumping its version
- **THEN** the fixture-gates job SHALL treat the already-published refusal as acceptable and pass
