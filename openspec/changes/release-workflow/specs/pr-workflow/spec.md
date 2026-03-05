## ADDED Requirements

### Requirement: Full test suite runs on pull requests
The PR workflow SHALL run lint, unit tests, registry tests, integration tests, and e2e tests when a pull request is opened or updated.

#### Scenario: All checks pass
- **WHEN** a pull request is opened or synchronized
- **THEN** all five job types run and each exits zero

#### Scenario: Any check fails
- **WHEN** any job exits non-zero
- **THEN** the pull request is marked failed and the failing job is identified

### Requirement: All jobs run in parallel
The PR workflow SHALL run lint, unit, registry, integration, and e2e jobs concurrently with no inter-job dependencies.

#### Scenario: Parallel execution
- **WHEN** the PR workflow triggers
- **THEN** all five jobs start simultaneously

### Requirement: Integration tests use an ephemeral kind cluster
The integration job SHALL create a kind cluster named `opm-dev` at job start and delete it at job end, regardless of test outcome.

#### Scenario: Cluster created before tests
- **WHEN** the integration job starts
- **THEN** a kind cluster is created using `hack/kind-config.yaml` before any test runs

#### Scenario: Cluster deleted after tests
- **WHEN** the integration job finishes (pass or fail)
- **THEN** the kind cluster is deleted

### Requirement: Registry tests have CUE registry access
The registry test job SHALL set the `OPM_REGISTRY` environment variable to allow resolution of `opmodel.dev/core`.

#### Scenario: Registry env var set
- **WHEN** the registry job runs `go test ./internal/builder/...`
- **THEN** `OPM_REGISTRY` is set to the configured registry value

### Requirement: Workflow targets self-hosted runner
The PR workflow SHALL specify `runs-on: self-hosted` for all jobs.

#### Scenario: Self-hosted runner assignment
- **WHEN** the PR workflow triggers
- **THEN** all jobs are assigned to the self-hosted runner pool

### Requirement: Workflow is initially disabled
The PR workflow SHALL use only `workflow_dispatch` as an active trigger. Pull request triggers SHALL be present but commented out.

#### Scenario: Manual trigger works
- **WHEN** a user manually dispatches the workflow
- **THEN** all test jobs run

#### Scenario: PR does not trigger workflow automatically
- **WHEN** a pull request is opened before the self-hosted runner is available
- **THEN** the workflow does not run automatically
