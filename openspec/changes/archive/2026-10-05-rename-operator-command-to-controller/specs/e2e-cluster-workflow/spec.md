## REMOVED Requirements

### Requirement: The job applies to operator-facing pull requests, cascade pull requests and release pull requests
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-controller-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

### Requirement: The cluster is prepared with the local loop's own tasks and the pinned operator module
**Reason**: Renamed with the product: the operator is now the controller (`opm controller`, `opm-controller-system`, `spec.owner: controller`). Restated under the new title.
**Migration**: None.

## MODIFIED Requirements

### Requirement: The suite runs with the cluster required and leaves evidence on failure

When the job applies, it SHALL run `go test ./tests/e2e/... -v` with the same `-timeout` as `task test:e2e`, with `OPM_E2E_REQUIRE_CLUSTER=1` set, and with the tools the cluster tests probe for present on the runner (`kubectl`, `task`, `crane`), so that every cluster-backed test runs and none takes a fallback path meant for a developer machine. When a controller-owned test fails, the test itself SHALL log, before its cleanup deletes the evidence, the test's ModuleInstance with its status, the cluster Platform with its status and the controller's logs, and a test that times out waiting for the controller to reconcile SHALL name the ModuleInstance's `Ready` condition in its failure message. Later tests (the controller lifecycle test tears the controller, CRDs and every ModuleInstance down and reinstalls) leave nothing of that state for a step after the suite. After any failed step, including a preparation step before the suite starts, the job SHALL also print the controller's logs, the cluster Platform and every ModuleInstance with their status, and recent cluster events.

#### Scenario: A cluster test cannot skip

- **WHEN** the cluster is unreachable when the suite starts
- **THEN** the cluster-backed tests fail rather than skip, and the job fails

#### Scenario: Lifecycle test takes the full-rollout path

- **WHEN** the controller lifecycle test checks whether the pinned image is pullable
- **THEN** `crane` is present, the image is reported pullable, and the test asserts a completed rollout

#### Scenario: Failure leaves diagnostics

- **WHEN** a controller-owned test fails because the controller did not reconcile
- **THEN** that test's output contains the controller's logs and the status of the Platform and of the test's ModuleInstance as they were when it failed, and its failure message names the ModuleInstance's `Ready` condition

#### Scenario: Preparation failure leaves diagnostics

- **WHEN** `task cluster:controller` fails before the suite starts
- **THEN** the job's diagnostics step prints the controller's logs, the cluster Platform, every ModuleInstance and recent events

## ADDED Requirements

### Requirement: The job applies to controller-facing pull requests, cascade pull requests and release pull requests

The workflow SHALL run one job, named `E2E (kind, embedded operator)` (a name the `main` ruleset requires as a status check, kept although nothing is embedded), triggered by `pull_request` targeting `main` with the activity types `opened`, `synchronize`, `reopened`, `labeled` and `unlabeled`, and by `workflow_dispatch`. The job SHALL do the cluster-backed work (it *applies*) when at least one of these holds:

- the run was started by `workflow_dispatch`;
- the pull request's head branch starts with `release-please--`;
- the pull request's head branch is the cascade branch `deps/cascade`;
- the pull request currently carries the label `deps-cascade`;
- the pull request changes a file under `internal/controller/`, `internal/cmd/controller/`, `templates/` or `hack/platform/`;
- the pull request changes one of the job's own inputs: the workflow file itself, its decision script `.github/scripts/e2e-cluster-applies.sh`, `Taskfile.yml` (which defines `cluster:create` and `cluster:controller`), `hack/fixtures.sh` (which seeds the registry), `hack/kind-config.yaml`, `hack/kind-platform.yaml`, `hack/kind-controller-rbac.yaml`, `hack/opm-config.cue`, any Go file directly under `tests/e2e/` (the cluster-backed tests and the suite's shared helpers such as `TestMain` and `runOPMWithEnv`), or anything under `tests/e2e/testdata/controller-owned/`.

The decision SHALL be computed from the pull request's state when the job runs (its head branch, its current labels and its full list of changed files), never from the triggering event alone, so runs for the same head commit, head branch and labels always reach the same decision. A pull request that is no longer open (read live: closed or merged) SHALL NOT apply, whatever else holds, so the label changes release-please makes after a release pull request merges start no cluster run.

#### Scenario: Controller pin bump applies

- **WHEN** a pull request changes `internal/controller/pin.go`
- **THEN** the job boots a cluster, installs the pinned controller module and runs the e2e suite

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
- **THEN** the job applies, so a change that breaks `task cluster:controller` fails on that pull request rather than on the next release pull request

#### Scenario: Controller command change applies

- **WHEN** a pull request changes only `internal/cmd/controller/install.go`
- **THEN** the job applies, because `task cluster:controller` runs that command and the lifecycle test exercises it

#### Scenario: Merged release pull request does not apply

- **WHEN** release-please swaps the labels of a release pull request after it merged, and the `labeled` and `unlabeled` events start runs
- **THEN** each run reports "not applicable: pull request is closed" and creates no cluster

#### Scenario: Unrelated pull request

- **WHEN** a pull request changes only `internal/cmd/mod/` files and carries no `deps-cascade` label
- **THEN** the job does not create a cluster

### Requirement: The cluster is prepared with the local loop's own tasks and the pinned controller module

When the job applies, it SHALL install kind at the version and checksum `pr.yml`'s `integration` job uses, create the `opm-dev` cluster with `task cluster:create` (the node image pinned in `Taskfile.yml`), and prepare it with `task cluster:controller`, which installs the controller module version the CLI pins (no `--version`), applies `hack/kind-platform.yaml` and `hack/kind-controller-rbac.yaml`, and waits for the cluster Platform to be Ready for its current generation. The job SHALL NOT restate those steps in the workflow.

The job SHALL run a registry container seeded from the tree with `hack/fixtures.sh seed`, joined to kind's docker network. The CLI on the runner SHALL resolve `testing.opmodel.dev` from it on `localhost:5000`, and the controller SHALL resolve `testing.opmodel.dev` from it through `KIND_CUE_REGISTRY` on its in-cluster address, set as a value of the controller's instance. Core, the catalogs and the controller module SHALL resolve from GHCR on both sides.

#### Scenario: The pinned controller module is what runs

- **WHEN** the job has prepared the cluster
- **THEN** the ModuleInstance `opm-controller` in `opm-controller-system` records the `PinnedModuleVersion` of the pull request's head commit, and the controller Deployment runs the controller release `PinnedControllerVersion` names

#### Scenario: Bumped fixture before it is published

- **WHEN** a pull request bumps the fixture `tests/e2e/testdata/controller-owned` pins to a version GHCR does not hold yet
- **THEN** the controller resolves that fixture from the job's registry and the controller-owned tests run against it

#### Scenario: Stalled Platform fails the preparation

- **WHEN** the pinned controller cannot build the cluster Platform
- **THEN** the job fails in the preparation step, before the suite starts, naming the Platform's stall reason
