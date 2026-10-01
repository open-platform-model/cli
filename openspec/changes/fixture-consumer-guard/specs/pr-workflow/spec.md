## MODIFIED Requirements

### Requirement: Registry mapping is set at workflow level
The PR workflow SHALL set `OPM_REGISTRY` and `CUE_REGISTRY` as workflow-level environment mapping `testing.opmodel.dev` and `opmodel.dev` to `ghcr.io/open-platform-model`, followed by `registry.cue.works`. Only the `unit`, `fixtures` and `e2e` jobs SHALL run a registry service: each seeds a job-local `registry:2` from the tree with `hack/fixtures.sh seed` and maps `testing.opmodel.dev` to it (the mixed mapping), because the examples and the e2e testdata pin the tree's fixture version, which GHCR holds only after the merge publishes it. Every other dependency, core and the catalogs included, SHALL resolve from GHCR in every job.

#### Scenario: Tests resolve fixtures and core from GHCR
- **WHEN** an integration test loads a module that imports `opmodel.dev/core` or a `testing.opmodel.dev` fixture
- **THEN** the import resolves from GHCR through the workflow-level mapping

#### Scenario: Seeded jobs resolve the tree's fixture version
- **WHEN** a pull request bumps a fixture to a version GHCR does not hold yet and a unit or e2e test resolves it
- **THEN** the fixture resolves from the job-local registry seeded from the tree, and core and the catalogs resolve from GHCR

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
