# ci-workflow — Delta

## ADDED Requirements

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
