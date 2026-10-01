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
The PR workflow SHALL set `OPM_REGISTRY` and `CUE_REGISTRY` as workflow-level environment mapping `testing.opmodel.dev` and `opmodel.dev` to `ghcr.io/open-platform-model`, followed by `registry.cue.works`. Only the `unit`, `fixtures` and `e2e` jobs SHALL run a registry service: each seeds a job-local `registry:2` from the tree with `hack/fixtures.sh seed` and maps `testing.opmodel.dev` to it (the mixed mapping), because the examples and the e2e testdata pin the tree's fixture version, which GHCR holds only after the merge publishes it. Every other dependency, core and the catalogs included, SHALL resolve from GHCR in every job.

#### Scenario: Tests resolve fixtures and core from GHCR
- **WHEN** an integration test loads a module that imports `opmodel.dev/core` or a `testing.opmodel.dev` fixture
- **THEN** the import resolves from GHCR through the workflow-level mapping

#### Scenario: Seeded jobs resolve the tree's fixture version
- **WHEN** a pull request bumps a fixture to a version GHCR does not hold yet and a unit or e2e test resolves it
- **THEN** the fixture resolves from the job-local registry seeded from the tree, and core and the catalogs resolve from GHCR

### Requirement: Lint and unit mirror the push workflow
The `lint` job SHALL run golangci-lint v2.11.3 and the `unit` job SHALL run `go test ./internal/...`, both on Go 1.26.0, exactly as the push-triggered CI workflow does.

#### Scenario: Lint violation fails the PR
- **WHEN** the pull request introduces a lint violation
- **THEN** the `lint` job exits non-zero

#### Scenario: Unit failure fails the PR
- **WHEN** the pull request introduces a failing unit test
- **THEN** the `unit` job exits non-zero

### Requirement: Template publish gates run dry-run
The `template-gates` job SHALL build `opm` from the pull request, install the cue version that the cli's `go.mod` requires for `cuelang.org/go`, and run `.github/scripts/publish-templates.sh --dry-run`, so every gate runs over every template tree without pushing. For every template the gates SHALL be, in order:
- the identity: its `Version` SHALL be a stable SemVer `X.Y.Z`, and its `ModulePath` SHALL be `opmodel.dev/templates/<directory>@v<major of Version>`;
- the tree layout: it SHALL hold no symbolic link, no special file and no nested `cue.mod`;
- `opm module tidy --check`;
- `opm module vet`, which renders the template's `debugValues` against a platform generated from its own pins;
- the module zip that `cue mod publish --out` builds from the tree, the same `modzip.CreateFromDir` file selection `opm module publish` zips with, SHALL hold exactly the tree's files, so the published module is the one vet rendered;
- the `opm module publish --dry-run` gates, which SHALL either pass or refuse only because the template's version is already published;
- the version order: an unpublished version SHALL be above the highest stable version GHCR holds for the template's major, and an already-published version SHALL be that highest version, because `opm module init` resolves it.

An already-published version SHALL pass only when the artifact GHCR holds at that version contains exactly the tree's files. The script SHALL fetch the published module zip from GHCR with an anonymous pull token, verify it against its manifest digest, and compare it with the tree's module zip file by file, by path and bytes, comments included.

A template whose tree differs from its published artifact SHALL fail the job, because published versions are immutable. The comparison SHALL NOT depend on git history, so a change that reached `main` without a pull request, or under a later release tag, fails every later pull request's job until a bump lands. A failure to fetch, list, verify, build or compare SHALL fail the job and SHALL NOT count the template as unpublished. Any other refusal or a failed vet SHALL fail the job. The job SHALL report every failing template, not only the first, including when a template's identity cannot be evaluated.

#### Scenario: Template gate violation fails the PR
- **WHEN** a template tree violates a publish gate other than already-published
- **THEN** the `template-gates` job exits non-zero naming the refusal

#### Scenario: Already-published template passes
- **WHEN** a template's declared version is already on GHCR and is the highest stable version there for its major, it passes every other gate, and the published artifact holds exactly the tree's files
- **THEN** the `template-gates` job exits zero

#### Scenario: Template that does not vet fails the PR
- **WHEN** a template's `debugValues` do not render, for example a field left as an unresolved disjunction
- **THEN** `opm module vet` prints the render error
- **AND** the `template-gates` job exits non-zero naming the template

#### Scenario: Changed template at an already-published version fails the PR
- **WHEN** a template's tree differs in any file, its `cue.mod` pins and comments included, from the artifact GHCR holds at the template's declared version
- **THEN** the `template-gates` job exits non-zero, prints the difference, and names the template and `opm module version set` as the fix

#### Scenario: A change pushed to main without a bump fails later pull requests
- **WHEN** a commit that changed a template without bumping it reached `main` directly, possibly with a release cut on top of it, and a pull request that does not touch the template is opened afterwards
- **THEN** the pull request's `template-gates` job exits non-zero naming that template

#### Scenario: A change and its revert are not a change
- **WHEN** a template was changed and the change was reverted, so its tree again equals its published artifact
- **THEN** the template passes and the job does not fail for it

#### Scenario: A fetch error never passes as unpublished
- **WHEN** GHCR holds a template's declared version but the published artifact or the template's version list cannot be fetched, or the artifact does not match its manifest digest
- **THEN** the `template-gates` job exits non-zero naming the template

#### Scenario: A file the module zip omits fails the PR
- **WHEN** a template tree holds a symbolic link, a special file, a nested `cue.mod`, or any other file its module zip leaves out, whether or not its version is already published
- **THEN** the `template-gates` job exits non-zero naming the template and the file

#### Scenario: A version init would not resolve fails the PR
- **WHEN** a template declares a prerelease version, an unpublished version not above the highest published stable version of its major, or a published version below that highest one
- **THEN** the `template-gates` job exits non-zero naming the template and the highest published version

#### Scenario: A module path that does not match its directory fails the PR
- **WHEN** the identity `ModulePath` of the template in `templates/<directory>/` is not `opmodel.dev/templates/<directory>@v<major>`
- **THEN** the `template-gates` job exits non-zero naming the expected path

#### Scenario: A broken identity does not hide other failures
- **WHEN** one template's identity package cannot be evaluated and another template fails a gate
- **THEN** the `template-gates` job exits non-zero naming both templates

### Requirement: Fixture gates, seed and render parity
The `fixtures` job SHALL check out with `fetch-depth: 0`, build `opm`, install cue v0.17.1, and run `hack/fixtures.sh check` with `BASE_REF` set to `origin/<base branch>`: every fixture MUST pass the publish gates dry-run against GHCR, and a fixture changed since the merge-base MUST carry a version GHCR does not hold yet, because published versions are immutable. It SHALL then run `hack/fixtures.sh seed` to publish the tree's fixtures into a job-local `registry:2` service on `localhost:5000` with a fresh `CUE_CACHE_DIR`. Against the mixed mapping (`testing.opmodel.dev` local, everything else GHCR) it SHALL then run `hack/fixtures.sh consumers examples tests/e2e/testdata/operator-owned`, which MUST pass (see `test-fixture-lineage`), and `tests/integration/render-parity` with `OPM_ITEST_RENDER_PARITY=1`. `task test:fixtures` SHALL run the same steps against a local registry.

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

#### Scenario: Consumer left on a stale core pin
- **WHEN** a PR bumps a fixture whose own `cue.mod` moved core, and re-pins `tests/e2e/testdata/operator-owned` to the new fixture version but leaves its core pin unchanged
- **THEN** the `fixtures` job SHALL fail, printing the core pin diff and a `FAIL tests/e2e/testdata/operator-owned` line

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
