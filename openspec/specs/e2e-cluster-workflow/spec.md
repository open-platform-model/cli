## Purpose

`.github/workflows/e2e-cluster.yml` runs the cluster-backed e2e suite in CI against the operator the CLI embeds, so a change that breaks the CLI-to-operator path, or an upstream publish that breaks the embedded operator, fails a pull request instead of reaching a release.

## Requirements

### Requirement: The job applies to operator-facing pull requests, cascade pull requests and release pull requests

The workflow SHALL run one job, named `E2E (kind, embedded operator)`, triggered by `pull_request` targeting `main` with the activity types `opened`, `synchronize`, `reopened`, `labeled` and `unlabeled`, and by `workflow_dispatch`. The job SHALL do the cluster-backed work (it *applies*) when at least one of these holds:

- the run was started by `workflow_dispatch`;
- the pull request's head branch starts with `release-please--`;
- the pull request's head branch is the cascade branch `deps/cascade`;
- the pull request currently carries the label `deps-cascade`;
- the pull request changes a file under `internal/operator/`, `internal/cmd/operator/`, `templates/` or `hack/platform/`;
- the pull request changes one of the job's own inputs: the workflow file itself, its decision script `.github/scripts/e2e-cluster-applies.sh`, `Taskfile.yml` (which defines `cluster:create` and `cluster:operator`), `hack/fixtures.sh` (which seeds the registry), `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-operator-rbac.yaml`, `hack/opm-config.cue`, any Go file directly under `tests/e2e/` (the cluster-backed tests and the suite's shared helpers such as `TestMain` and `runOPMWithEnv`), or anything under `tests/e2e/testdata/operator-owned/`.

The decision SHALL be computed from the pull request's state when the job runs (its head branch, its current labels and its full list of changed files), never from the triggering event alone, so runs for the same head commit, head branch and labels always reach the same decision. A pull request that is no longer open (read live: closed or merged) SHALL NOT apply, whatever else holds, so the label changes release-please makes after a release pull request merges start no cluster run.

#### Scenario: Operator pin bump applies

- **WHEN** a pull request changes `internal/operator/dist/install.yaml` and `internal/operator/manifest.go`
- **THEN** the job boots a cluster, installs the embedded operator and runs the e2e suite

#### Scenario: Release pull request applies

- **WHEN** release-please opens or updates a pull request from the branch `release-please--branches--main--components--opm`
- **THEN** the job runs the cluster-backed suite even if no listed path changed

#### Scenario: Label added after opening

- **WHEN** a pull request that changes no listed path is opened, and the `deps-cascade` label is added afterwards
- **THEN** the `labeled` event starts a new run, which applies and runs the suite

#### Scenario: Unrelated label does not change the outcome

- **WHEN** an unrelated label is added to a pull request
- **THEN** the new run reaches the same decision as the previous run for that head commit

#### Scenario: Cascade branch applies without its label

- **WHEN** the `deps-cascade` label is removed from a pull request whose head branch is `deps/cascade`
- **THEN** the `unlabeled` event starts a new run, which still applies and runs the suite

#### Scenario: Change to the preparation task applies

- **WHEN** a pull request changes only `Taskfile.yml`
- **THEN** the job applies, so a change that breaks `task cluster:operator` fails on that pull request rather than on the next release pull request

#### Scenario: Operator command change applies

- **WHEN** a pull request changes only `internal/cmd/operator/install.go`
- **THEN** the job applies, because `task cluster:operator` runs that command and the lifecycle test exercises it

#### Scenario: Merged release pull request does not apply

- **WHEN** release-please swaps the labels of a release pull request after it merged, and the `labeled` and `unlabeled` events start runs
- **THEN** each run reports "not applicable: pull request is closed" and creates no cluster

#### Scenario: Unrelated pull request

- **WHEN** a pull request changes only `internal/cmd/mod/` files and carries no `deps-cascade` label
- **THEN** the job does not create a cluster

### Requirement: The job always reports a result

The workflow SHALL NOT use a workflow-level path or branch filter beyond the `main` base branch, and the job SHALL NOT be skipped through a job-level condition. When the job does not apply, it SHALL finish successfully within its first steps and SHALL print which conditions it checked and that none held. This keeps the check reporting on every pull request, so it can be made a required status check without leaving unrelated pull requests pending.

#### Scenario: Not applicable passes quickly

- **WHEN** the job does not apply
- **THEN** it exits zero without installing kind, and its log names the head branch, the labels and the reason it did not apply

#### Scenario: Required check never pending

- **WHEN** the check is required by a ruleset and a pull request touches none of the listed paths
- **THEN** the check reports success for that pull request's head commit

### Requirement: The cluster is prepared with the local loop's own tasks and the embedded operator

When the job applies, it SHALL install kind at the version and checksum `pr.yml`'s `integration` job uses, create the `opm-dev` cluster with `task cluster:create` (the node image pinned in `Taskfile.yml`), and prepare it with `task cluster:operator`, which installs the operator from the manifest the CLI embeds (no `--version`), applies `hack/kind-platform.yaml` and `hack/kind-operator-rbac.yaml`, and waits for the cluster Platform to be Ready for its current generation. The job SHALL NOT restate those steps in the workflow.

The job SHALL run a registry container seeded from the tree with `hack/fixtures.sh seed`, joined to kind's docker network. The CLI on the runner SHALL resolve `testing.opmodel.dev` from it on `localhost:5000`, and the operator SHALL resolve `testing.opmodel.dev` from it through `KIND_CUE_REGISTRY` on its in-cluster address. Core and the catalogs SHALL resolve from GHCR on both sides.

#### Scenario: Embedded operator is what runs

- **WHEN** the job has prepared the cluster
- **THEN** the controller Deployment's image is the image reference in `internal/operator/dist/install.yaml` of the pull request's head commit

#### Scenario: Bumped fixture before it is published

- **WHEN** a pull request bumps the fixture `tests/e2e/testdata/operator-owned` pins to a version GHCR does not hold yet
- **THEN** the operator resolves that fixture from the job's registry and the operator-owned tests run against it

#### Scenario: Stalled Platform fails the preparation

- **WHEN** the embedded operator cannot build the cluster Platform
- **THEN** the job fails in the preparation step, before the suite starts, naming the Platform's stall reason

### Requirement: The suite runs with the cluster required and leaves evidence on failure

When the job applies, it SHALL run `go test ./tests/e2e/... -v` with the same `-timeout` as `task test:e2e`, with `OPM_E2E_REQUIRE_CLUSTER=1` set, and with the tools the cluster tests probe for present on the runner (`kubectl`, `task`, `crane`), so that every cluster-backed test runs and none takes a fallback path meant for a developer machine. When an operator-owned test fails, the test itself SHALL log, before its cleanup deletes the evidence, the test's ModuleInstance with its status, the cluster Platform with its status and the operator's logs, and a test that times out waiting for the operator to reconcile SHALL name the ModuleInstance's `Ready` condition in its failure message. Later tests (the operator lifecycle test tears the operator, CRDs and every ModuleInstance down and reinstalls) leave nothing of that state for a step after the suite. After any failed step, including a preparation step before the suite starts, the job SHALL also print the operator's logs, the cluster Platform and every ModuleInstance with their status, and recent cluster events.

#### Scenario: A cluster test cannot skip

- **WHEN** the cluster is unreachable when the suite starts
- **THEN** the cluster-backed tests fail rather than skip, and the job fails

#### Scenario: Lifecycle test takes the full-rollout path

- **WHEN** the operator lifecycle test checks whether the pinned image is pullable
- **THEN** `crane` is present, the image is reported pullable, and the test asserts a completed rollout

#### Scenario: Failure leaves diagnostics

- **WHEN** an operator-owned test fails because the operator did not reconcile
- **THEN** that test's output contains the operator's logs and the status of the Platform and of the test's ModuleInstance as they were when it failed, and its failure message names the ModuleInstance's `Ready` condition

#### Scenario: Preparation failure leaves diagnostics

- **WHEN** `task cluster:operator` fails before the suite starts
- **THEN** the job's diagnostics step prints the operator's logs, the cluster Platform, every ModuleInstance and recent events
