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
