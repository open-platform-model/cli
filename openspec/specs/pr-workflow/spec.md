# Capability: pr-workflow

## Purpose

`.github/workflows/pr.yml` is the gate every pull request against main must pass. It runs lint, unit tests, template publish gates, fixture gates with a seeded job-local registry and a render-parity check, integration tests against an ephemeral kind cluster, and the e2e suite, all as independent jobs, so a failure names itself without blocking the others.

## Requirements

### Requirement: Six independent jobs run on every pull request
The PR workflow SHALL run the jobs `lint`, `unit`, `template-gates`, `fixtures`, `integration` and `e2e` with no `needs` dependency between them, triggered by `pull_request` targeting `main` and by `workflow_dispatch`. A `concurrency` group keyed on workflow and ref with `cancel-in-progress` SHALL cancel a superseded run.

#### Scenario: All six jobs start in parallel
- **WHEN** a pull request is opened or synchronized
- **THEN** all six jobs start simultaneously and each exits zero when its checks pass

#### Scenario: Any job failing marks the PR failed
- **WHEN** any job exits non-zero
- **THEN** the pull request is marked failed and the failing job is identified

#### Scenario: A newer push cancels the running workflow
- **WHEN** a second push to the pull request arrives while the workflow is running
- **THEN** the earlier run is cancelled and only the newest push is checked

### Requirement: Registry mapping is set at workflow level
The PR workflow SHALL set `OPM_REGISTRY` and `CUE_REGISTRY` as workflow-level environment mapping `testing.opmodel.dev` and `opmodel.dev` to `ghcr.io/open-platform-model`, followed by `registry.cue.works`. No job SHALL run a registry service except `fixtures`, so anything the tests resolve MUST be published on GHCR.

#### Scenario: Tests resolve fixtures and core from GHCR
- **WHEN** a unit, integration or e2e test loads a module that imports `opmodel.dev/core` or a `testing.opmodel.dev` fixture
- **THEN** the import resolves from GHCR through the workflow-level mapping

### Requirement: Lint and unit mirror the push workflow
The `lint` job SHALL run golangci-lint v2.11.3 and the `unit` job SHALL run `go test ./internal/...`, both on Go 1.26.0, exactly as the push-triggered CI workflow does.

#### Scenario: Lint violation fails the PR
- **WHEN** the pull request introduces a lint violation
- **THEN** the `lint` job exits non-zero

#### Scenario: Unit failure fails the PR
- **WHEN** the pull request introduces a failing unit test
- **THEN** the `unit` job exits non-zero

### Requirement: Template publish gates run dry-run
The `template-gates` job SHALL build `opm` from the pull request, install cue v0.17.1, and run `.github/scripts/publish-templates.sh --dry-run`, so every publish gate runs over every template tree without pushing. A dry run whose only refusal is that the template's version is already published SHALL pass; any other refusal SHALL fail the job.

#### Scenario: Template gate violation fails the PR
- **WHEN** a template tree violates a publish gate other than already-published
- **THEN** the `template-gates` job exits non-zero naming the refusal

#### Scenario: Already-published template passes
- **WHEN** a template's declared version is already on GHCR and it passes every other gate
- **THEN** the `template-gates` job exits zero

### Requirement: Fixture gates, seed and render parity
The `fixtures` job SHALL check out with `fetch-depth: 0`, build `opm`, install cue v0.17.1, and run `hack/fixtures.sh check` with `BASE_REF` set to `origin/<base branch>`: every fixture MUST pass the publish gates dry-run against GHCR, and a fixture changed since the merge-base MUST carry a version GHCR does not hold yet, because published versions are immutable. It SHALL then run `hack/fixtures.sh seed` to publish the tree's fixtures into a job-local `registry:2` service on `localhost:5000` with a fresh `CUE_CACHE_DIR`, and run `tests/integration/render-parity` with `OPM_ITEST_RENDER_PARITY=1` against the mixed mapping (`testing.opmodel.dev` local, everything else GHCR).

#### Scenario: Fixture with a gate violation
- **WHEN** a PR makes a fixture's `metadata.modulePath` disagree with its identity package
- **THEN** the `fixtures` job SHALL fail, naming the disagreement

#### Scenario: Changed fixture at an already-published version
- **WHEN** a PR changes a fixture tree without bumping its identity version and GHCR already holds that tag
- **THEN** the `fixtures` job SHALL fail, naming the fixture and pointing at `opm module version set`

#### Scenario: Unchanged fixture at a published version
- **WHEN** a fixture is unchanged since the merge-base and its version is already on GHCR
- **THEN** the check SHALL report it as published and unchanged and pass

#### Scenario: Render parity runs against the seeded registry
- **WHEN** the seed step has published the tree's fixtures to the job-local registry
- **THEN** the render-parity program SHALL load each fixture from the tree and from the registry and require byte-identical renders

### Requirement: Integration tests use an ephemeral kind cluster with CRDs installed
The `integration` job SHALL install kind v0.31.0 (checksum-verified), create a cluster named `opm-dev` from `hack/kind-config.yaml` with the pinned node image, run `go run ./cmd/opm operator install --crds-only --context kind-opm-dev` so the ModuleInstance CRD exists, run the integration programs under `tests/integration/` (deploy, inventory-apply, inventory-ops, inst-list, migration, gates), and delete the cluster with `if: always()`.

#### Scenario: CRDs installed before any program runs
- **WHEN** the integration job starts
- **THEN** the kind cluster is created and the CRDs are installed before the first integration program runs

#### Scenario: Cluster deleted after tests
- **WHEN** the integration job finishes, whether the programs passed or failed
- **THEN** the kind cluster is deleted

### Requirement: E2E suite runs on every pull request
The `e2e` job SHALL run `go test ./tests/e2e/... -v`.

#### Scenario: E2E failure fails the PR
- **WHEN** an e2e test fails
- **THEN** the `e2e` job exits non-zero

### Requirement: Workflow targets GitHub-hosted runner
The PR workflow SHALL specify `runs-on: ubuntu-latest` for all jobs.

#### Scenario: GitHub-hosted runner assignment
- **WHEN** the PR workflow triggers
- **THEN** all jobs are assigned to the `ubuntu-latest` runner pool
